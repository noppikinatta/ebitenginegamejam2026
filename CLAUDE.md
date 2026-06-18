# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Ebitengine Game Jam 2026** entry. Theme: **"Disconnect"**.

Written in Go using the Ebitengine game engine. Supports both desktop and WebAssembly (WASM) builds.

**Module:** `github.com/noppikinatta/ebitenginegamejam2026`

## Game Concept

ジャンル：ヴァンサバライク（Vampire Survivors風アクション）。自機は**戦車**。

### バックストーリー

敵が攻めてきて戦車で発進しようとしたが、大勢の博士がそれぞれ思い思いの武装（や役に立たない装置）を勝手に追加した結果、砲塔がものすごく巨大化してしまった。

### コアアイデア

- 通常のヴァンサバライクとは逆の発想：**最初から大量の武装が砲塔にくっついている**
- 電力は**全タイルに均等配分**（`発電量 / 接続タイル数`）。タイルが増えるほど各武装が弱くなる
- レベルアップでは博士が砲塔に**タイルを追加**してくる（武装 or 無意味なガジェット）。どちらも電力を薄める
- プレイヤーは戦闘中に**タイルを切り離す（Disconnect）**ことで残存タイルに電力を再集中させる
  - 切り離しは「プラモデル用ニッパー」を消費する（数量限定）
- ねらい：博士が肥大化させる ⇄ プレイヤーがニッパーで絞る、という綱引きが run 全体のビルド計画になる

### Disconnect の設計方針：ポーズ画面でクリック切断

切り離しは**ポーズ中**に行う。**Space でポーズを切り替え**るとシミュレーション全体（敵・弾・移動）が停止し、戦車＋砲塔が画面中央に**ズームアップ・上向き固定**で描画される。**タイルをクリックして切断**する。

- ポーズで敵も止まるため、落ち着いてタイルを選んで切断できる（リアルタイムのカーソル操作が難しすぎたための再設計）
- **切断してもポーズは解除されない**ので、連続して複数タイルを切れる。カット操作は**マウスのみ**（キーボード不要）
- 1回の切断は1タイルだが、**その下流（ジェネレータから繋がらなくなったタイル）も巻き添えで切れる**（カスケード）
- 切ったタイル分の電力が残存タイルに再配分され、残った武装が強化される
- 切断は**ニッパー**を消費。初期3個。燭台（停止する破壊可能オブジェ）を壊すとドロップ、レベルアップで低確率入手

### ゲームループ

1. 戦車で敵を倒しながら経験値とニッパーを集める
2. レベルアップで博士3人から1人を選ぶ（タイル追加 or ニッパー入手）→ 砲塔が肥大し電力が薄まる
3. Space でポーズし、ズームした砲塔の不要タイルをクリックで切断（連続可）→ 残存武装に電力が再集中
4. 少ない武装でより強力に戦う構成を、ニッパーをやりくりしながら目指す

## Development Roadmap / Progress

標準的なヴァンサバライクを先に完成させ、後から配線ツリー（Disconnect）を**武装の energy 変調として**乗せる方針。ヘックスマップへのピボット後は H0〜H4 フェーズで実装。

- [x] **フェーズ0：ビルドを通す** — `ui` パッケージ作成、旧 `2025` import 修正、InGame/Result スタブ、埋め込みアセットのプレースホルダ整備。WASM ビルドと vet が通る
- [x] **フェーズ1：最小VSループ** — `core` パッケージで 戦車移動・自動武装・弾・敵・経験値ジェム・レベルアップ・スポーンを実装。単体テスト
- [x] **フェーズ2：レベルアップ選択（簡易版）** — `core/upgrade.go` の `Upgrade` + `World.ChooseUpgrade`
- [x] **H0：クリーンアップ** — `hexmap` テスト import 修正、固定 RNG シード可変化、デッドコード除去
- [x] **H1：ヘックスグリッド電力ソルバー** — `core/turret.go`。`Component` インターフェース、PurgeTile（旧版：BFS距離リング配布、後にフラット配分へ置換）
- [x] **H2：ランダム砲塔生成** — `core/turret_gen.go`。フロンティア成長アルゴリズム、BranchProb で枝分かれ制御
- [x] **H3：World統合** — `core/world.go` を `Turret` ベースに全面書き換え
- [x] **H4：クリックUI** — `scene/ingame.go` に砲塔オーバーレイ（後に再設計）
- [x] **レーザー武装** — `KindLaser`。砲塔タイルに固定された持続ビーム（毎フレーム最近接敵を追尾、経路上の敵を貫通DPS）。`geom.PointSegmentDistance` でカプセル判定。`World.ActiveBeams()` で描画用スナップショット
- [x] **再設計A：電力フラット化** — BFS距離リングソルバーを撤廃し `発電量/接続タイル数` の均等配分へ。`Component` を `Name()` のみに簡略化、`ProportionalWeapon`→`WeaponComponent` 改名、`Junk`（無意味ガジェット）追加。`Capacitor`/`ThresholdWeapon`/`PurgeWeapon` 削除。HUD に Pwr/Tile 表示
- [x] **再設計B：ポーズ画面でクリック切断** — `Player.Nippers`、`World.CutTile`（ニッパー消費＋カスケード）。当初は Shift+WASD+Space の戦闘中カーソル切断だったが難しすぎたため再設計：**Space でポーズ**（`InGame.paused`、シミュレーション停止）→ 砲塔をズーム・上向き描画 → **タイルをクリックで切断**。切断してもポーズ継続で連続カット可、マウスのみ。砲塔描画は `InGame.drawTurretTiles(cx,cy,size,theta)` に共通化し戦闘ミニチュアと共用
- [x] **再設計C：レベルアップ＝タイル追加** — `Turret.AddTile`（空き隣接にランダム配置）/`TileCount`。`rollChoices` を博士3人提案（武装/ジャンク/ニッパー）に置換、ソフトキャップ `maxTurretTiles`。scene はカード式UIに置換
- [x] **再設計D：燭台ドロップ** — `Enemy.DropsNipper`/`Pickup`。`spawnCandlestick`（停止・無害・周期スポーン）、`updatePickups`（収集で+1ニッパー）
- [ ] **H5：複数ジェネレータ対応** — 初版は中央1基のみ。後続バージョンで追加予定

### 既知の暫定対応・残課題

- **音声はプレースホルダ**。`asset/sound/bgm.ogg` は OggS ヘッダのみのダミー、`explosion.wav` は無音。`LoadSounds()` はデコード失敗を握りつぶす（ログして継続）よう変更済みなので落ちないが、本物の音源に差し替えるまで鳴らない。`LoadSounds()` はまだどこからも呼ばれていない
- **タイトル画像 `asset/img/title.png` もプレースホルダ**（枠だけの矩形）
- **言語CSVが空**。`asset/lang/english.csv` / `japanese.csv` は存在するが中身が無い。`scene/title.go` は `story-1` キーを参照（現状フォールバック表示）
- **バランス調整未実施** — 砲塔生成パラメータ（MaxTiles/BranchProb/WeaponDensity/JunkDensity）、電力量、ニッパー入手率、燭台周期、`maxTurretTiles` 等は初期値のまま。プレイテストで要調整

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

- `world.go` — `World` が全状態（Player/Enemies/Projectiles/Gems/Pickups、Choices、State、Tick、RNG、turret、各種タイマー）を保持。`NewWorld(seed, cfg)` は RNG → `GenerateTurret` → `ActiveWeapons` で初期化（`cfg` はバランス値；後述の data 注入）。`Update` は移動・武装・ビーム・弾・敵・ジェム・ニッパー収集・スポーン・燭台スポーンを毎tick回す。`CutTile(idx)` がニッパー消費のタイル切断（シーン側のポーズ中にクリックで呼ぶ）、`rollChoices`/`rollDoctorChoice` がレベルアップの博士提案
- `entity.go` — `Player`（戦車：FacingAngle/Nippers 含む）/ `Enemy`（DropsNipper で燭台）/ `Projectile` / `Gem` / `Pickup`（ニッパー）。位置は `geom.PointF`、当たり判定は円（半径）
- `weapon.go` — `WeaponKind`（Cannon/Shotgun/Sniper/Laser）+ `Weapon` と `Stats(p)`。**発射はアキュムレータ方式**：各 `Weapon.fireProgress` が毎tick `fireIncrement(p, fireMult)`（＝発射倍率、`BaseInterval/MinInterval` で上限クランプ）ぶん進み、`BaseInterval` に達したら発射。平均間隔 = `BaseInterval/fireMult`（`MinInterval` 下限）。照準（ロックオン圏内の最近接へ向く）と発射は独立し、**CIWS以外は敵不在でも前方へ発射**（`HoldWhenNoTarget` の武器だけ満タンで保持）。ダメージは `BaseDamage × LevelMult^Level`、射程・ビームは定数（energy スケールは廃止）。弾の生存は武器ごとにデータ化：`ProjLife = round(ProjMaxDist / ProjSpeed)`、当たり半径 `ProjRadius`。`beamTicksLeft`/`beamAngle` でビーム照射状態を保持
- `turret.go` — ヘックスグリッド砲塔。電力は**接続消費タイル数→発射倍率の区分線形補間**（`PowerMultiplier(curve, ConsumerTileCount())`、`Config.PowerCurve` の `PowerPoint{Tiles,Mult}` 列を両端クランプで補間）。`ComputePower`/`WeaponPower` は各タイルの**接続判定**として存続（値は描画の減光のみで参照）。`Component` インターフェースは `Name()` ＋ `Mods() Modifier`（戦車/砲塔への加算修飾子；Wire/WeaponComponent/Junk はゼロ、`Capacitor` は `FireRateAdd`）。接続中タイルの `Mods()` を合算した `Turret.Modifiers()` をタイル追加/削除時にキャッシュ再計算し、`World.FireRateMultiplier()` がカーブ値に加算（`Capacitor` ＝発射倍率 +0.1）。`Modifier` は将来 MaxHP 等の設備へ拡張可能。`distancesFrom` は接続判定・カスケード用。`PurgeTile`（+`propagatePurge` カスケード）、`AddTile`（ランダム隣接配置）、`TileCount`/`ConsumerTileCount`、`ActiveWeapons`、`MuzzleOffset`
- `turret_gen.go` — `GenerateTurret()` フロンティア成長アルゴリズム、`DefaultTurretGenConfig`、`pickComponent`（武装/Junk/Wire）、`junkDeviceNames`
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
