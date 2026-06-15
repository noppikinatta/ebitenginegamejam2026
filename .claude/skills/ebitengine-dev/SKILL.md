---
name: ebitengine-dev
description: Ebitengine (Ebiten) implementation guide focused on rendering performance and memory. Use this whenever implementing or modifying ANY code that touches Ebitengine — the Game interface (Update/Draw), GUI screens, ebiten.Image, sprites, DrawImage/DrawTriangles, text, vector graphics, shaders, or audio — and whenever the user reports FPS/TPS drops, stutter, slow rendering, too many draw calls, GC pauses, or growing memory usage (Ebitengineの実装・修正、GUI実装、描画が重い、パフォーマンス改善、メモリ使用量の調査 など). Consult this even for small Ebitengine changes, because the costly mistakes (per-frame image creation, pixel readbacks, batch breaking) look harmless in code review.
---

# Ebitengine 開発スキル（パフォーマンス・メモリ重視）

Ebitengine の性能モデルは3行で要約できる:

1. **描画は「ドローコマンド数」で決まる** — Ebitengine は `DrawImage`/`DrawTriangles` を内部でキューイングし、条件が揃えば1つのドローコマンドにバッチングする。バッチを壊す書き方が最大の性能リスク。
2. **CPU↔GPU 転送は桁違いに遅い** — `At`/`ReadPixels` は GPU からの読み戻し、`WritePixels` は GPU への転送。毎フレーム呼ぶ場所に書いてはいけない。
3. **Go の GC は毎フレームのアロケーションに弱い** — `Update`/`Draw` の中での `ebiten.NewImage`、スライス・マップ・クロージャの生成が積もるとGCポーズになる。

## リファレンス（必要になったら読む）

- `references/rendering-performance.md` — バッチング条件、テクスチャアトラスの仕組み、restoring（コンテキストロスト対策）が遅くなる理由、ピクセル操作のコスト、`ebitenginedebug` の使い方。**描画コードを書く・直すときは必ず読む。**
- `references/memory-and-resources.md` — `ebiten.Image` のライフサイクル、`SubImage`、`Deallocate`、Unmanaged画像、ミップマップ、text/v2・vector・audio のリソース管理。**画像や音声などリソースの生成・破棄に触れるときに読む。**

## 鉄則

1. `ebiten.NewImage` / `NewImageFromImage` は**ロード時に一度だけ**。`Update`/`Draw` の中では呼ばない。
2. スプライトシートは1枚の画像 + `SubImage` で持つ。`SubImage` はピクセルをコピーせず親のテクスチャ（同一アトラス）を共有するので、バッチングに最も有利。
3. 描画順は「同じ描画先・同じ Blend・同じ Filter・同じ描画元アトラス」が連続するように並べる。これが崩れるとドローコマンドが分割される。
4. `(*Image).At` / `ReadPixels` を毎フレーム呼ばない。当たり判定などでピクセル値が要るなら、ロード時に CPU 側（`image.Image` や独自構造）へ展開して保持する。
5. 一度描画元（render source）として使った画像に後から描き込まない。`A.DrawImage(B)` の後の `B.DrawImage(C)` や、`A→B`・`B→A` の循環描画は restoring の再計算を誘発する。screen を描画元にしない。
6. `Draw` ではゲーム状態を変更しない。`Update` は固定60TPS、`Draw` はモニタのリフレッシュレート依存で呼び出し回数が異なる。重い計算は `Update` 側か、さらに事前計算へ。
7. 毎フレームのアロケーションを避ける: 頂点・インデックスのスライスや `DrawImageOptions` は再利用できる構造にする（`op.GeoM.Reset()` で使い回す等）。
8. 縮小描画 + `FilterLinear` はミップマップを自動生成する（特にモバイルで高コスト）。不要なら `DisableMipmaps: true` か `FilterNearest` を検討。
9. 文字描画は text/v2 を使い、`GoTextFaceSource` を**1フォントにつき1つ**だけ作って使い回す（グリフキャッシュは Source が持つ）。毎フレーム Source を作るとキャッシュが効かない。
10. 効果音は `audioContext.NewPlayerFromBytes(bs)` を都度作ってよい（軽い）。BGM は1つの `audio.Player` を `Rewind` で使い回す。

## パフォーマンス問題を調査するときの手順

推測で直さない。必ず測ってから直す:

1. **現状把握**: `ebitenutil.DebugPrint` 等で `ebiten.ActualFPS()` / `ebiten.ActualTPS()` を表示。TPS低下なら `Update`（CPU/GC）、FPSのみ低下なら `Draw`（GPU/ドローコマンド）が主因。
2. **ドローコマンドを数える**: `go run -tags=ebitenginedebug .` で実際のドローコマンドのログを見る。1フレームのコマンド数と、分割されている箇所（dst/blend/filter/srcの切り替わり）を特定する。
3. **CPU側は pprof**: `runtime/pprof` か `net/http/pprof` でプロファイルを取り、`Update` 内のホットスポットとアロケーション（`-alloc_objects`）を見る。
4. 原因が描画なら `references/rendering-performance.md`、メモリ・GCなら `references/memory-and-resources.md` の該当節を参照して直す。
5. 修正後に同じ計測を繰り返し、数値で改善を確認してから報告する。

## このプロジェクトでの注意

- ゲームロジック（`core` 予定）と描画を分離するのが README の方針。Ebitengine への依存を core に持ち込まず、core 層は `ebiten.Image` を知らない設計にする。
