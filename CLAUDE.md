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

## Development Roadmap / Progress

標準的なヴァンサバライクを先に完成させ、後から配線ツリー（Disconnect）を**武装の energy 変調として**乗せる方針。ヘックスマップへのピボット後は H0〜H4 フェーズで実装。

- [x] **フェーズ0：ビルドを通す** — `ui` パッケージ作成、旧 `2025` import 修正、InGame/Result スタブ、埋め込みアセットのプレースホルダ整備。WASM ビルドと vet が通る
- [x] **フェーズ1：最小VSループ** — `core` パッケージで 戦車移動・自動武装・弾・敵・経験値ジェム・レベルアップ・スポーンを実装。単体テスト
- [x] **フェーズ2：レベルアップ選択（簡易版）** — `core/upgrade.go` の `Upgrade` + `World.ChooseUpgrade`
- [x] **H0：クリーンアップ** — `hexmap` テスト import 修正、固定 RNG シード可変化、デッドコード除去
- [x] **H1：ヘックスグリッド電力ソルバー** — `core/turret.go`。`Component` インターフェース（Wire/Capacitor/ProportionalWeapon/ThresholdWeapon）、BFS距離リング電力配布アルゴリズム、PurgeTile/PurgeWeapon、13テスト
- [x] **H2：ランダム砲塔生成** — `core/turret_gen.go`。フロンティア成長アルゴリズム、BranchProb で枝分かれ制御、6テスト
- [x] **H3：World統合** — `core/world.go` を `Turret` ベースに全面書き換え。`core/tree.go` 削除、WeaponKind を `weapon.go` へ移動。buildDisconnectChoices でタイルパージ/武装パージ両択を生成
- [x] **H4：クリックUI** — `scene/ingame.go` に六角ブリックレイアウト砲塔オーバーレイ。左クリック=Cut（タイルパージ+速度ボーナス）、Shift+クリック=Disarm（武装のみパージ）、ホバー時ツールチップ、数字キーフォールバック
- [ ] **H5：複数ジェネレータ対応** — 初版は中央1基のみ。後続バージョンで追加予定

### 既知の暫定対応・残課題

- **音声はプレースホルダ**。`asset/sound/bgm.ogg` は OggS ヘッダのみのダミー、`explosion.wav` は無音。`LoadSounds()` はデコード失敗を握りつぶす（ログして継続）よう変更済みなので落ちないが、本物の音源に差し替えるまで鳴らない。`LoadSounds()` はまだどこからも呼ばれていない
- **タイトル画像 `asset/img/title.png` もプレースホルダ**（枠だけの矩形）
- **言語CSVが空**。`asset/lang/english.csv` / `japanese.csv` は存在するが中身が無い。`scene/title.go` は `story-1` キーを参照（現状フォールバック表示）
- **H4 UIの選択マッチング** — `handleLevelUpInput` でクリック時に選択肢をタイル座標文字列で検索しているが、UI の改善余地あり（H4は暫定実装）
- **バランス調整未実施** — 砲塔生成パラメータ（MaxTiles/BranchProb/WeaponDensity等）は初期値のまま。プレイテストで要調整

### Build/Verify この環境での注意

デスクトップ向け `go build ./...` / `make run` は X11・ALSA 等のネイティブライブラリが必要で、ヘッドレス環境では失敗する。**コードのコンパイル検証は WASM ビルドで行う**：

```bash
GOOS=js GOARCH=wasm go build -o=release/game.wasm app/main.go
GOOS=js GOARCH=wasm go vet ./...
```

`core` パッケージは Ebiten 非依存なので普通にテストできる：`go test ./core/...`

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
| `core/` | **ゲームロジック本体（Ebiten非依存）**。後述 |
| `scene/` | 各シーンとシーン配線。Ebiten の描画・入力アダプタ層。新シーンはここに追加 |
| `drawing/` | 描画ユーティリティ（後述） |
| `geom/` | `PointF`（2Dベクトル：Add/Subtract/Multiply/Angle/Abs/Distance/InnerProduct、極座標・image.Point 変換） |
| `lang/` | 多言語テキスト（後述） |
| `asset/` | `//go:embed` による埋め込みアセットと初期化（後述） |
| `ui/` | `Input` 型（`nyuuryoku` のマウス・キーボードをまとめる） |

### core パッケージ（ゲームロジック）

**Ebiten に依存しない**ので単体テストできるのが要点。シーン層（`scene/ingame.go`）は入力を `geom.PointF` の移動ベクトルに変換して `World.Update(move)` を毎フレーム呼び、`World` の状態を読んで描画するだけ。

- `world.go` — `World` が全状態（Player/Enemies/Projectiles/Gems、Choices、State、Tick、RNG、turret）を保持。`NewWorld(seed)` は RNG → `GenerateTurret` → `ActiveWeapons` で初期化
- `entity.go` — `Player`（戦車）/ `Enemy` / `Projectile` / `Gem`。位置は `geom.PointF`、当たり判定は円（半径）
- `weapon.go` — `WeaponKind`（Cannon/Shotgun/Sniper）+ `Weapon` と `StatsFromEnergy()`。**`energy` から戦闘数値（ダメージ/連射間隔/射程）を導出する関数がソルバー統合の唯一の接点**
- `turret.go` — ヘックスグリッド電力ソルバー。`Component` インターフェース（Wire/Capacitor/ProportionalWeapon/ThresholdWeapon）、BFS距離リング配布、CanPurgeTile/CanPurgeWeapon、ActiveWeapons
- `turret_gen.go` — `GenerateTurret()` フロンティア成長アルゴリズム、`DefaultTurretGenConfig`
- `upgrade.go` — `type Upgrade struct { Name, Desc string; Apply func(*World) }` (軽量選択肢モデル)
- `State`：`StatePlaying` / `StateLevelUp` / `StateGameOver`
- シーン再入場時の世界リセットは `InGame.OnStart()`（bamenn の `OnStarter`）で行う

### hexmap パッケージ

ヘックスグリッド座標系（cube coord: x+y+z=0）を提供。

- `index.go` — `Index{x,y}` キューブ座標。`IdxXY/IdxYZ/IdxZX` 生成、`Add/Mul/Distance`、6方向定数（Direction01〜Direction11）、`AppendAround`
- `size.go` — `Size{radius}` と `Contains(idx)`（原点からの距離≤半径かどうか）

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
