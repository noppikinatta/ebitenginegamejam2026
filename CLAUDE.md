# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Ebitengine Game Jam 2026** entry. Theme: **"Disconnect"**.

Written in Go using the Ebitengine game engine. Supports both desktop and WebAssembly (WASM) builds.

**Module:** `github.com/noppikinatta/ebitenginegamejam2026`

## Game Concept

ジャンル：ヴァンサバライク（Vampire Survivors風アクション）。自機は**戦車**。

### コアアイデア

- 通常のヴァンサバライクとは逆の発想：**最初から大量の武器が砲塔にくっついている**
- ただし配線が混乱しているため、最初は武器が扱いづらく、ジェネレータのエネルギーが分散して各武装が弱い
- レベルアップ時に武装パーツを**切り離す（Disconnect）**ことでパワーアップする
  - 不要なパーツを切り離す → 残った武装にエネルギーが集中 → パワーアップ
- プレイヤーは武装の動作を観察しながら、どのパーツを切り離すか判断する

### Disconnect の設計方針：配線トポロジー（ツリー）

切り離しの判断を「ハードな論理パズル」にも「キー暗記などの操作難度」にもせず、**配線トポロジーを使った軽量な空間パズル**にする。

- 砲塔は**ノードのツリー**。中央にジェネレータがあり、エネルギーが配線を通って外側の武装ノードへ流れる
- 1回の disconnect は1ノードを切るが、**その下流（先につながっていた武装）も巻き添えで切れる**
- そのため「弱い武器を素直に切る」だけでは済まない。残したい武器が切りたい武器の下流にぶら下がっていることがあり、**どの枝を切るかという空間判断**になる
- 切った枝が消費していたエネルギーが残った経路に再配分され、残存武装が強化される

ねらい：個々の判断は読みやすく（「ここを切るとこの先が全部死ぬ」は直感的）、ヴァンサバライクの即決テンポを壊さない。一方で run 全体ではビルド計画として深みが出る。

### タイミング

切り離しは**レベルアップ時（ポーズ中）**に選択する想定（リアルタイム除去はUI・入力・バランスが重くなるため当面採らない）。アクション性は戦車操作と弾幕回避で担保する。

### ゲームループ

1. 戦車で敵を倒しながら経験値を集める
2. レベルアップ時に「切り離す枝（ノード）」を選択
3. 切り離した分のエネルギーが残存経路に再配分されて強化される
4. より少ない武装でより強力に戦う構成を目指す

## Current State of the Codebase (重要)

これは去年のゲーム（`ebitenginegamejam2025`）から再利用できそうなコードを**断片的にコピーしてきた状態**で、まだ**コンパイルが通らない**。ゲーム本体（戦車・武装・敵・配線ツリー）は未実装で、現状あるのは共通基盤（描画・アセット・言語・シーン枠組み）のみ。

把握済みの未解決点：

- **`ui` パッケージが存在しない**。`app/main.go` / `scene/title.go` / `scene/sequence.go` が `ui.Input`（`Mouse` フィールドを持つ想定）を参照しているが、`ui/` ディレクトリ自体が無い → 要新規作成
- **`NewInGame` / `NewResult` が未定義**。`scene/sequence.go` から呼ばれているがシーン実装ファイルが無い
- **古いモジュールパスが残っている**。`scene/sequence.go` と `scene/lang.go` が `github.com/noppikinatta/ebitenginegamejam2025/...` を import している（`2026` に直す必要あり）
- **埋め込みアセットの実体が無い**。`asset/embed.go` は `img/*.png` を、`asset/sound.go` は `sound/bgm.ogg` と `sound/explosion.wav` を `//go:embed` するが、`asset/img/` は `.gitkeep` のみ、`asset/sound/` ディレクトリ自体が無い → `go build` 時に embed エラー
- **言語CSVが空**。`asset/lang/english.csv` / `asset/lang/japanese.csv` は存在するが中身が無い。`scene/title.go` は `story-1` キーを参照
- `app/main.go` のウィンドウタイトルが `"Ebitengine Game Jam 2025"` のまま
- `asset.LoadSounds()` は定義のみで init から呼ばれていない（音を使うなら明示呼び出しが必要）

## Commands

```bash
make run        # Run locally (go run app/main.go)
make build      # Build WASM for web (output: release/game.wasm)
make gen        # Run go generate
make test       # Run tests (verbose)
make test-cov   # Generate coverage HTML report and open it
```

Run a single test:
```bash
go test ./lang/... -run TestName -v
```

## Architecture

### Entry Point

`app/main.go` — 1280×720 ウィンドウを設定し、`ui.Input`（マウスは `nyuuryoku.NewMouse()`）を作り、`scene.CreateSequence(input)` でゲームを構築して `ebiten.RunGame` を呼ぶ。

### Scene System

`scene/sequence.go` — `bamenn.Sequence` でシーン遷移を構成。順序は **Title → InGame → Result → (Title)** のループ。各シーンは `Init(nextScene, sequence, transition)` で次シーンへの参照を受け取り、`SwitchWithTransition` でフェード遷移する。

`CreateSequence` は `wrapperGame` を返す。これは `langSwitcher`（後述）を `Sequence` にかぶせ、全シーン共通で言語切替の入力と表示を処理するラッパー。

- `scene/title.go` — タイトル画面。タイトル画像とストーリーテキスト（`lang.Text("story-1")`）を表示し、左クリックで次シーンへ。シーン実装の参考パターンになる（`Title` 構造体 + `NewTitle` + `Init`/`Update`/`Draw`/`Layout`）
- `scene/lang.go` — `langSwitcher`。**L キー**で言語をトグルし、`DrawTriangles` のグラデ矩形＋テキストで現在言語を一時表示（alpha フェードアウト）

### Packages

| Package | Role |
|---|---|
| `app/` | `main` パッケージ。エントリポイントのみ |
| `scene/` | 各シーンとシーン配線。新シーンはここに追加 |
| `drawing/` | 描画ユーティリティ（後述） |
| `geom/` | `PointF`（2Dベクトル：Add/Subtract/Multiply/Angle/Abs/Distance/InnerProduct、極座標・image.Point 変換） |
| `lang/` | 多言語テキスト（後述） |
| `asset/` | `//go:embed` による埋め込みアセットと初期化（後述） |
| `ui/` | **未作成**。`Input` 型（`nyuuryoku` のマウス等をまとめる）を置く想定 |

### drawing パッケージ

- `text.go` — `DrawText` / `DrawTextByKey` / `DrawTextTemplate` / `MeasureText`。`(文字列, フォントサイズ)` をキーに**描画済みテキスト画像をキャッシュ**する（影付き）。フレーム毎に `ebiten.Image` を作らないこと
- `img.go` — `drawing.Image("key")` で `asset` 側のロード済み画像マップから取得。見つからなければ赤い "IMAGE NOT FOUND" のフォールバック画像を返す。`WhitePixel` は `DrawTriangles` で塗り図形を描くための1px白テクスチャ
- `rect.go` — `DrawRect`（`DrawTriangles` で矩形塗り）と `ColorF32` ヘルパー
- `gauge.go` — `GaugeDrawer`。`Current/Max` の割合でバー幅と色（Min→Max 補間）を描く HP/エネルギーゲージ用

### lang パッケージ

- `asset/lang/<language>.csv` を `<言語名>` として読み込む（ファイル名 = 言語名。現状 `english` / `japanese`）。デフォルトは **english**
- CSV は `key,value` の2列。`#` 始まりはコメント、value 内の `\n` リテラルは改行に変換される
- 取得は `lang.Text("key")`。プレースホルダ入りは `lang.ExecuteTemplate("key", data)` で Go `text/template` として評価（テンプレートはキャッシュされる）。キーが無ければ `NO_TMPL: ...` を返す
- `lang.Switch()` で言語を循環切替（戻り値は表示用に先頭大文字化した言語名）

### asset パッケージ

- `embed.go` — フォント（Mplus2-Regular）、`lang/*.csv`、`img/*.png` を埋め込み、`init()` で `FontFace(size)`（サイズ別キャッシュ）・言語テンプレート・画像マップを構築。`Images()` / `LoadTemplates()` / `FontFace()` を公開
- `sound.go` — `bgm.ogg`（ループ）と `explosion.wav` を埋め込み、48000Hz の `audio.Context` を作る。`LoadSounds()`（明示呼び出し）/ `PlaySound(Sound)` / `StopSound(Sound)`。`Sound` は `BGM` / `SEExplosion` の enum

### Drawing / Performance Conventions

Ebitengine 実装時は `.claude` の `ebitengine-dev` スキルの方針に従う。要点：

- テキストは必ず `drawing.DrawText` 系を使う（毎フレームの画像生成を避けるためキャッシュ済み）
- 塗り図形は `drawing.DrawRect` / `WhitePixel` 経由の `DrawTriangles` を使い、バッチ分断やフレーム毎の画像生成を避ける
- 画像は `asset` で一括ロードして `drawing.Image(key)` で取得する（描画ループ内で新規 `ebiten.Image` を作らない）

### Dependencies

- `github.com/hajimehoshi/ebiten/v2` v2.9 — game engine
- `github.com/noppikinatta/bamenn` — scene transition management
- `github.com/noppikinatta/nyuuryoku` — input handling abstraction
