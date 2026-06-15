# Ebitengine のメモリとリソース管理

出典: ebiten v2 ソースコードの doc comment（image.go, text/v2, vector, audio）および
公式 Performance Tips。

## 目次

1. `ebiten.Image` のライフサイクル（生成・解放）
2. `SubImage` の正しい使い方
3. `NewImageWithOptions` と Unmanaged 画像
4. ミップマップとメモリ
5. Go の GC 圧を下げる（毎フレームのアロケーション対策）
6. text/v2 のキャッシュ構造
7. audio のリソース管理
8. メモリの計測方法

---

## 1. `ebiten.Image` のライフサイクル

- `NewImage` / `NewImageFromImage` は GPU 上のテクスチャ領域（通常は共有アトラスの一部）を
  確保する。**ロード時・シーン構築時に一度だけ**呼び、`Update`/`Draw` の中では呼ばない。
  毎フレームの生成は GPUメモリ確保 + ピクセル転送 + アトラス断片化 + GC圧の四重苦になる。
- 解放は基本的に GC 任せでよいが、**シーン遷移などで大量の画像を確実に手放したい**場合は
  `(*Image).Deallocate()`（v2.7+）を呼ぶ。`Deallocate` 後もその画像は再利用可能
  （内部状態が再確保される）。`Dispose` は v2.7 で非推奨。
- `Deallocate` は SubImage に対しては何もしない（親を解放する必要がある）。
- 巨大な背景画像などを場面ごとに切り替える設計では、「保持し続けて使い回す」か
  「`Deallocate` して読み直す」かをメモリ予算で決める。中途半端に参照を残すと
  GC が回収できず、GPUメモリが積み上がる。

## 2. `SubImage` の正しい使い方

```go
sprite := sheet.SubImage(image.Rect(0, 0, 32, 32)).(*ebiten.Image)
```

- `SubImage` はピクセルを**コピーしない**。親と同じテクスチャ（同じアトラス）を共有する
  ビューなので、スプライトシート運用の基盤。バッチングにも最有利（描画元が全部同じ）。
- 戻り値は `image.Image` なので `*ebiten.Image` への型アサーションが必要。
- 呼び出し自体は安価だが小さな構造体を毎回アロケートする。タイルマップ等で
  毎フレーム数千回呼ぶなら、ロード時に `[]*ebiten.Image` へ切り出してキャッシュする。
- SubImage 経由でも「親が描画元として使われた後に親へ描き込む」と restoring の問題
  （rendering-performance.md §5）は同様に起きる。素材シートは読み取り専用に保つ。

## 3. `NewImageWithOptions` と Unmanaged 画像

```go
img := ebiten.NewImageWithOptions(bounds, &ebiten.NewImageOptions{Unmanaged: true})
```

- Unmanaged 画像は**自動アトラスに決して乗らない**。通常画像はアトラス上の配置を
  Ebitengine が自動管理するが、Unmanaged はそれを外す。
- 使いどころ: 巨大画像（どうせアトラスに乗らない）、毎フレーム描き直すオフスクリーン、
  `WritePixels` を頻繁に行う動的テクスチャなど、「アトラスの自動管理がむしろ邪魔」な画像。
  パフォーマンスとメモリを細かく制御したい場合の道具であり、デフォルトでは不要。

## 4. ミップマップとメモリ

- `FilterLinear` で `GeoM` が画像を**縮小**するとき、Ebitengine は自動的にミップマップを
  生成して使う。縮小描画の品質は上がるが、生成コストと追加のGPUメモリがかかり、
  **特にモバイルで高価**。
- 対策: `DrawImageOptions.DisableMipmaps = true`（v2.9+）、`FilterNearest` を使う、
  あるいは事前に縮小済みのアセットを用意して実行時の大幅な縮小を避ける。
  `FilterLinear` 以外では `DisableMipmaps` は無視される。

## 5. Go の GC 圧を下げる

`Update` は毎秒60回、`Draw` はそれ以上呼ばれうる。ここでのアロケーションは全部GC負荷になる:

- `ebiten.DrawImageOptions` はループ内で毎回 `&ebiten.DrawImageOptions{}` を作るのが
  普通のスタイルだが、数万スプライト規模では1個をフィールドに持ち
  `op.GeoM.Reset()` / `op.ColorScale.Reset()` で使い回す。
- `DrawTriangles` の頂点 (`[]ebiten.Vertex`) とインデックスのスライスは
  `vs = vs[:0]` で再利用する。容量が安定するとアロケーションゼロにできる。
- 毎フレームの `fmt.Sprintf`（デバッグ表示など）、マップ生成、クロージャ生成、
  `[]byte`/`image.Image` の変換も積もる。プロファイル（§8）で上位に出たら潰す。
- 文字列やスライスのフィールドへの事前確保 (`make(..., 0, cap)`) を活用する。

## 6. text/v2 のキャッシュ構造

- **`GoTextFaceSource` がグリフ画像キャッシュを持つ**（LRUで古いグリフは破棄される）。
  `GoTextFace`（サイズ等の設定）は軽量だが、Source を作り直すとキャッシュごと捨てられる。
  → **1フォントファイルにつき `GoTextFaceSource` は1個**をアプリ全体で共有し、
  サイズ違いは `GoTextFace` 側で表現する。
- `GoXFace`（`golang.org/x/image/font.Face` ラッパー）は Face インスタンス自身が
  キャッシュを持つ。これも使い回す。
- 毎フレーム同じ文字列を描くのは（キャッシュが効くので）問題ないが、
  完全に静的で巨大なテキストブロックはオフスクリーンに一度描く選択肢もある。

## 7. audio のリソース管理

- **効果音**: `audioContext.NewPlayerFromBytes(bs)` で都度プレイヤーを作ってよい。
  生成は安価で、再生中のプレイヤーはGCされない。1つのストリームは複数プレイヤーで
  共有できないため、同じSEの同時再生にはこの方式が正しい。

```go
func PlaySE(bs []byte) {
    p := audioContext.NewPlayerFromBytes(bs)
    p.Play() // 再生が終わればGCされる
}
```

- **BGM**: バイト列が大きいので全体を都度メモリ展開するのは高くつく。
  1つの `audio.Player` を作って `Rewind` で使い回す。デコード済みストリーム
  （`vorbis.DecodeF32` 等）から `NewPlayer` で作り、ループには `audio.NewInfiniteLoop`。

## 8. メモリの計測方法

- `runtime.ReadMemStats` / `runtime/metrics` を `Update` 内で定期取得して
  `HeapAlloc`・GC回数をデバッグ表示すると、毎フレームアロケーションの悪化にすぐ気づける。
- `net/http/pprof` を有効にして `go tool pprof -alloc_objects http://localhost:6060/debug/pprof/heap`
  でアロケーション元を特定する。
- GPU側（テクスチャ）の使用量は直接は見えないが、`ebitenginedebug` のログに出る
  内部画像IDの増え方や、作成した `ebiten.Image` の総ピクセル数×4バイトで概算できる。
