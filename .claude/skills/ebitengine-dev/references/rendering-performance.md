# Ebitengine の描画パフォーマンス

出典: 公式 Performance Tips (https://ebitengine.org/en/documents/performancetips.html)、
公式設計ドキュメント「How rendering works in Ebitengine」、および ebiten v2 ソースコードの doc comment。

## 目次

1. レンダリングの内部モデル（なぜバッチングされるのか）
2. バッチング（ドローコマンド統合）の条件
3. テクスチャアトラスとバッチが分割されるケース
4. `ebitenginedebug` でドローコマンドを観測する
5. Restoring（コンテキストロスト対策）と「描画元への書き込み」が遅い理由
6. ピクセル操作（At / ReadPixels / WritePixels / Set）のコスト
7. オフスクリーンの使いどころとトレードオフ
8. アンチエイリアスと vector パッケージ
9. その他（ディスクリートGPU等）

---

## 1. レンダリングの内部モデル

Ebitengine では screen も画像ファイルから作った画像もオフスクリーンも、すべて `ebiten.Image`
であり、描画とは「画像の上に画像を描く」操作。`DrawImage`/`DrawTriangles` は即座に GPU へ
描画せず、内部の**ドローコマンドキュー**に積まれ、フレーム末尾などでまとめて実行される。
この遅延実行のおかげで、連続する互換なコマンドは1つのドローコマンド（GPUドローコール）に
統合される。**性能は概ね「1フレームあたりの（統合後の）ドローコマンド数」で決まる。**

ドローコマンドが少ないほど速い。公式の examples/sprites は1万以上のスプライトを
1〜数個のドローコマンドで描く好例。

## 2. バッチングの条件

連続する描画呼び出しは、以下を**すべて**満たすとき1つのドローコマンドに統合される:

- すべて `DrawImage` または `DrawTriangles` であること
- 描画先（`A.DrawImage(B, op)` の `A`）が同じであること
- Blend が同じであること
- Filter が同じであること
- Address が同じであること（`DrawTriangles` のみ）
- （実質的な追加条件）描画元が同じテクスチャアトラス上にあること（§3）

実践上の帰結:

- **描画順をソートする**: エンティティ種別ごとに描画先・Filter・Blend が揃うよう、
  描画リストをまとめて出す。「背景→全スプライト→UI」のように層で揃え、
  層の中で Filter や Blend を切り替えない。
- **スプライトシート（1枚画像 + `SubImage`）を使う**: 描画元が同じ画像（同じアトラス）なら
  種類の違うスプライトを交互に描いてもバッチが続く。
- `ColorScale` や `GeoM` の違いはバッチを壊さない（頂点属性として処理される）。
  一方、`ColorM`（colorm パッケージ）や Blend / Filter の切り替えは壊す。

## 3. テクスチャアトラスとバッチが分割されるケース

Ebitengine の画像は通常、内部の**自動テクスチャアトラス**（巨大なテクスチャ）に同居する。
ただし次の場合はアトラスに乗らず、描画元が変わった時点でドローコマンドが分割される:

- **巨大な画像**（アトラスの最大サイズはグラフィックデバイス依存）
- アトラスを使い切った場合
- **オフスクリーン（描画先として使った画像）**: 高確率でアトラスを共有しない
- `NewImageWithOptions` で `Unmanaged: true` を指定した画像（意図的にアトラス外に置く）

つまり「小さな画像をたくさん `NewImage` で個別に作る」のは、アトラスが効いていれば
動くが、アトラス溢れ・断片化の原因になる。最初から1枚にまとめて `SubImage` で切るのが堅い。

## 4. `ebitenginedebug` でドローコマンドを観測する

実際に発行されたドローコマンドはビルドタグで観測できる（v2.3 以前は `ebitendebug`）:

```bash
go run -tags=ebitenginedebug .
# 公式exampleで試す場合:
go run -tags=ebitenginedebug github.com/hajimehoshi/ebiten/v2/examples/blocks@latest
```

出力例（`--` がフレーム区切り）:

```
--
draw-triangles: dst: 7 <- src: 1, colorm: <nil>, mode copy, filter: nearest, address: clamp_to_zero
draw-triangles: dst: 7 <- src: 2, colorm: <nil>, mode source-over, filter: nearest, address: clamp_to_zero
draw-triangles: dst: 8 (screen) <- src: 1, colorm: <nil>, mode clear, filter: nearest, address: clamp_to_zero
draw-triangles: dst: 8 (screen) <- src: 7, colorm: <nil>, mode copy, filter: screen, address: clamp_to_zero
--
```

読み方: 1行 = 統合後の1ドローコマンド。`dst`/`src` の数字は内部画像（≒アトラス）のID。
行数が多すぎる、または同じ dst に対して src/mode/filter が細かく切り替わっているなら、
描画順かリソースの持ち方（アトラス分散）に問題がある。修正の前後でこのログの行数を比較する。

## 5. Restoring と「描画元への書き込み」が遅い理由

Ebitengine はコンテキストロスト（GPU状態消失。特にブラウザ/モバイルで起きる）から復元する
ため、ほぼすべての描画関数呼び出しを記録している。このため:

- **一度描画元として使った画像のピクセルを後から変更すると**、復元用の依存グラフの
  再計算が走り、複雑で重い処理になる。

```go
A.DrawImage(B, op) // B は描画元になった
B.DrawImage(C, op) // B のピクセルを変更 → 避ける
```

- **循環描画も避ける**:

```go
A.DrawImage(B, op)
B.DrawImage(A, op) // 循環! 避ける
```

- **screen を描画元にしない**: screen は毎フレームクリアされる特別な画像で、
  描画元に使うと復元計算が複雑になる。ポストエフェクトをかけたいなら、
  ゲーム全体をオフスクリーンに描き、最後にオフスクリーン→screen に1回描く構成にする。

役割を固定するのが原則: 「描画元専用（素材）」と「描画先（オフスクリーン）」を分け、
描画先に使う画像はなるべく描画元として使い回さない（使うなら一方向のパイプラインにする）。

## 6. ピクセル操作のコスト

| API | 何が起きるか | 指針 |
|-----|------------|------|
| `At` / `RGBA64At` | キュー済みドローコマンドを全部解決し、GPU→CPUへピクセルを読み戻す | 毎フレーム禁止。ピクセル判定が要るならロード時にCPU側へ展開して保持 |
| `ReadPixels` | 同上（領域一括版） | スクリーンショット等の単発用途のみ |
| `WritePixels`（旧 `ReplacePixels`） | CPU→GPU転送。比較的重い | 毎フレーム大画像に使わない。手続き的生成はロード時/変化時のみ。毎フレーム必要なら領域を最小化するかシェーダで生成 |
| `Set` | 1ピクセルずつの書き込み | 可能なら常に `WritePixels` を使う（公式doc注記） |
| `Fill` | ドローコマンド1つ | 問題なし |

## 7. オフスクリーンの使いどころとトレードオフ

オフスクリーン（`NewImage` して描画先に使う画像）が有効なケース:

- 静的な合成結果のキャッシュ: 変化しないUIパネル、地形タイルの合成結果などを一度だけ
  オフスクリーンに描き、毎フレームはそれを1回 `DrawImage` する。
- ポストエフェクトの中間バッファ。

トレードオフ: オフスクリーンはアトラスに乗らない可能性が高く、それを描画元に使う時点で
ドローコマンドが分かれる。「毎フレーム描き直すオフスクリーン」は restoring の記録も増える。
**内容が変化しないフレームでは再描画しない**（dirty フラグを持つ）のが定石。

## 8. アンチエイリアスと vector パッケージ

- `DrawTrianglesOptions.AntiAlias` は内部ドローコール数を増やす。多用するなら
  `ebitenginedebug` で実数を確認する。なお v2.9 で `DrawTriangles` の `AntiAlias`/`FillRule`
  は非推奨になり、`vector.FillPath` / `vector.StrokePath` の利用が推奨。
- `vector.Path` はフラット化（曲線→線分変換）結果をキャッシュする。静的な形状の Path を
  毎フレーム作り直すとキャッシュが効かない。Path はフィールドに保持して使い回す。
- 静的なベクター図形は、毎フレーム `FillPath` するのではなく一度オフスクリーンに
  ラスタライズして `DrawImage` で使い回すことも検討する。

## 9. その他

- **Windows でディスクリートGPUを使わせる**: `NvOptimusEnablement` /
  `AmdPowerXpressRequestHighPerformance` シンボルのエクスポートで促せる（要Cgo）。
  `github.com/silbinarywolf/preferdiscretegpu` を blank import するのが手軽。
- `Update` は固定 60 TPS、`Draw` はリフレッシュレート依存（呼び出し頻度は一致しない）。
  状態変更は `Update` のみで行い、`Draw` は読み取り専用にする。
