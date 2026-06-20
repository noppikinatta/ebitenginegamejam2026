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
- [x] **再設計B：ポーズ画面でクリック切断** — `Player.Nippers`、`World.CutTile`（ニッパー消費＋カスケード）。当初は Shift+WASD+Space の戦闘中カーソル切断だったが難しすぎたため再設計：**Space でポーズ**（`InGame.paused`、シミュレーション停止）→ 砲塔をズーム・上向き描画 → **タイルをクリックで切断**。切断してもポーズ継続で連続カット可、マウスのみ。砲塔描画は `InGame.drawTurretTiles(cx,cy,size,theta)` に共通化し戦闘ミニチュアと共用。**ホバー中タイルの説明**を画面下パネルに表示（`drawPauseTileInfo`／`pauseTileInfo`＋`weaponDesc`：プレビュー画像＋名前＋一行説明、切る対象を把握できる）
- [x] **再設計C：レベルアップ＝タイル追加** — `Turret.AddTile`（空き隣接にランダム配置）/`TileCount`。`rollChoices` を博士3人提案に置換、ソフトキャップ `maxTurretTiles`。scene はカード式UIに置換。**提案は `OfferItem` のリスト**（`Upgrade{Doctor, Items, Apply}`／`OfferKind`＝AddWeapon/AddJunk/AddCapacitor/Upgrade/Nippers）：1提案内に**追加とアップグレードが混在**可能（ニッパー提案のみ単独）。scene は1行＝アイコン＋ラベル(Add/Upgrade)＋名前で描画（`drawOfferItem`/`offerIcon`/`offerLabel`）。`upgradeShare(DoctorSpec)` で項目ごとのupgrade確率を算出、`atCap` では全項目upgrade化
- [x] **再設計D：燭台ドロップ** — `Enemy.DropsNipper`/`Pickup`。`spawnCandlestick`（停止・無害・周期スポーン）、`updatePickups`（収集で+1ニッパー）
- [ ] **H5：複数ジェネレータ対応** — 初版は中央1基のみ。後続バージョンで追加予定

### 既知の暫定対応・残課題

- **音声はダミー（差し替え前提）だが配線済み・実際に鳴る**。**SEとBGMで扱いを分離**：
  - **SE**は**難読化した1ファイル `asset/sound/se.pak` に同梱**してコミット（エントリ名で格納）。**発射音は武器ごとに別ファイル**（`fire_cannon`/`fire_shotgun`/`fire_sniper`/`fire_laser`/`fire_gatling`/`fire_grenade`/`fire_ciws`/`fire_missile`）＋`explosion`/`hit` の計10種。core は `SndFire*` を**武器種ごとに emit**（`core.FireSound(WeaponKind)`、`world.go` の発射時）、scene の `soundSink` が各 `asset.SEFire*` へマップ。同tick・同武器種の多重発射は `DispatchSounds` で1回に間引くが、**別武器種は各々鳴る**。**生のwavはコミットしない**：素材は `asset/sound/raw/`（gitignore）に置き、`make sound-pak`（=`go run tools/sndpak/main.go asset/sound/raw asset/sound/se.pak`）で pak に固める。**packerはマージ方式**：既存 se.pak をベースに、`raw/` にあるファイルだけ同名エントリを上書きするので、**SEを1個だけ差し替える時は当該wavを `raw/` に置いて再パックするだけ**でよい（他はpakから維持、cleanクローンでrawが空でも可）。`-rebuild` で raw/ のみから再構築（SE削除はこれ）。pak は `sndpak` パッケージのXORキーストリームで軽く難読化されており（RIFFヘッダも隠れる）、**リネームしてそのまま再生はできない**＝フリー素材サイト等のSEを生のまま公開リポジトリに置かないための配慮（暗号ではなく速度バンプ。キーはソース内）
  - **BGMは自作前提なので隠す必要がなく、生wavを直接コミット＆`//go:embed`**（pak非経由・難読化なし）。**2曲構成**：`asset/sound/bgm_title.wav`（オープニング＋タイトル）と `asset/sound/bgm_game.wav`（ゲーム中＋リザルト）。本物が出来たら各ファイルを差し替えるだけ。`asset.PlayBGM(BGMTitle|BGMGame)` が**現在再生中と同じトラックなら何もしない**ので、同曲を共有するシーン遷移（オープニング↔タイトル、ゲーム↔リザルト）はシームレス継続、曲が変わる切替時のみ頭から再生。各シーンの `OnStart` で自分のトラックを要求（Opening/Title→`BGMTitle`、InGame/Result→`BGMGame`）
  - 仮音源はサイン波生成：`make sound-gen` が `gensound`（SE→`raw/`、BGM2曲→`asset/sound/`）→`sndpak -rebuild` を実行
  - `asset.LoadSounds()`（`app/main.go` 起動時）が `bgm_title.wav`/`bgm_game.wav` を直接デコード＋`sndpak.Unpack(se.pak)` でSE展開→各エントリをデコード。デコード/欠落は握りつぶす（ログして継続、pakが壊れていてもSE無音で起動、BGM失敗もBGM無しで継続）。SEは `context.NewPlayerFromBytes` で毎回生成し多重再生可、BGMはトラックごとに単一プレイヤーを `Rewind` で使い回しループ
- **タイトル画像 `asset/img/title.png` もプレースホルダ**（枠だけの矩形）
- **言語CSVは整備済み**。`asset/lang/english.csv` / `japanese.csv` に scene の全UI文言・武器名/説明・博士/ジャンク/ボス名をキーで定義（両言語キー集合一致、`lang/csv_test.go` で検証）。scene は `drawing.DrawTextByKey`/`DrawTextTemplate` で描画し、core由来の名前は scene の `loc.go`（`weaponName`/`doctorNameL`/`junkNameL`/`bossNameL`、未定義キーは `lang.TextWithDefault` で元文字列にフォールバック）でキー解決。デフォルト言語は english（L キーで日本語へ切替）。カード番号など語を含まない純数値のみ `fmt.Sprintf` のまま
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

`scene/sequence.go` — `bamenn.Sequence` でシーン遷移を構成。順序は **Opening → Title → InGame → Result** で、Result から勝利時=Opening / 敗北時=InGame(リトライ)・Opening(受容) へ分岐ループ。各シーンは `Init(...)` で次シーン参照を受け取り `SwitchWithTransition` でフェード遷移。`Result.Init` だけ `(inGame, opening, seq, tran)` と特殊（勝敗判定元＋分岐先のため）。

`CreateSequence` は `wrapperGame` を返す。これは `langSwitcher`（後述）を `Sequence` にかぶせ、全シーン共通で言語切替の入力と表示を処理するラッパー。

- `scene/opening.go` — オープニング・シネマティック。エイリアン徘徊＋テロップ→自機（武装なし）が下から登場し中央へ→博士のセリフで武装が画面外から1つずつ飛来して装着→完成でタイトルへ自動遷移（クリックでスキップ）。`OnStart` で毎回リセット。タイムラインは tick ベース、装飾用の固定砲塔配置 `openingWeapons`（実runの生成砲塔とは無関係）
- `scene/title.go` — タイトル画面。タイトル画像とストーリーテキスト（`lang.Text("story-1")`）を表示し、左クリックで次シーンへ。シーン実装の参考パターンになる（`Title` 構造体 + `NewTitle` + `Init`/`Update`/`Draw`/`Layout`）
- `scene/result.go` — 勝敗で分岐。勝利＝「エイリアンを倒し、自由を手に入れた」＋『オープニングに戻る』。敗北＝「…自由を失った。…」＋『リトライ』(InGame)／『結果を受け入れる』(Opening)。勝敗は `InGame.Outcome()`（`StateCleared`/`StateGameOver`）から取得。`sceneButton` で簡易クリックボタン
- `scene/lang.go` — `langSwitcher`。**L キー**で言語をトグルし、`DrawTriangles` のグラデ矩形＋テキストで現在言語を一時表示（alpha フェードアウト）
- `scene/tuning.go` — **scene（Ebiten）層の調整可能パラメータの集約先**。描画解像度（`screenW`/`screenH`）、背景スクロール速度（`bgScrollMul`/`opScrollSpeed`）、ワールド描画・カードレイアウト・パワーゲージ・**HPバー（位置＝通常は下段中央／ポーズ時は左上、被弾時の揺れ `hpShake*`）**・**浮遊ダメージ数字（`dmg*`）**・オープニング演出のタイムライン等の数値をここで一元管理。`data` パッケージ（core のバランス＝`core.Config`）の presentation 版カウンターパート。`combatTileSize` のような「自由に変えられない（`core.TurretTileSize` と一致必須）」定数は使用箇所に残す
- スクロール背景：`drawScrollBG(screen, ox, oy)`（`scene/ingame.go`）がレイアウト同寸の `asset.ImgBackground`（上下左右シームレス想定）を 2×2 タイルで敷き、`(ox,oy)` だけずらす。InGame はカメラに `bgScrollMul` 倍で追従、Opening は発進デモ中のみ `opScrollSpeed` で上→下スクロール。プレースホルダは `make bg-img`（`tools/genbgimg`）

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

- `world.go` — `World` が全状態（Player/Enemies/Projectiles/Gems/Pickups/Explosions、Choices、State、Tick、RNG、turret、各種タイマー、bossesSpawned）を保持。`NewWorld(seed, cfg)` は RNG → `GenerateTurret` → `ActiveWeapons` で初期化（`cfg` はバランス値；後述の data 注入）。`Update` は移動・武装・ビーム・爆発・弾・敵・ジェム・ニッパー収集・スポーン・ボススポーン・燭台スポーンを毎tick回す。`CutTile(idx)` がニッパー消費のタイル切断（シーン側のポーズ中にクリックで呼ぶ）、`rollChoices`/`rollDoctorChoice` がレベルアップの博士提案。**敵スポーンはディレクタ方式**：`currentPhase()` が現在の時間帯（`SpawnPhase`＝`UntilTick`/`Interval`/`Weights`）を選び、**インターバルも種類重みも時間帯ごと**に切替（`spawnEnemies` がそのバンドの `Interval` で `spawnTimer` を設定、`pickKind` で重み抽選）→ `spawnPackOf`（`EnemyStats.PackMin/Max` のパック生成）。HPは `makeEnemy` で `HPBase×2^(tick/HPDoublingTicks)` の時間スケール。バンドは `data/spawnPhases()` で調整。`spawnBosses` は `Config.Bosses` を時刻 `AtTick` で1体ずつ生成、`Final` ボス撃破で `killEnemy` が `State=StateCleared`（勝利）。`State` は `StatePlaying/StateLevelUp/StateGameOver/StateCleared`、終了状態はスティッキー（`damagePlayer` は Playing 時のみ GameOver 化）。`ActiveBoss()` がHUDのボスHPバー用スナップショット
- `death.go` — `DeathEvent{Pos, Kind, Radius, IsBoss, DropsNipper}` と `emitDeath`。`killEnemy` で emit（敵は従来どおり即 `compact` 除去＝`w.Enemies` は生存敵のみ）。scene が drain して**死亡地点に残るフェードアウトスプライト**（`scene/death.go` の `deathFX`：α減衰＋微膨張、`deathFadeTicks`/`deathGrow`）を生成。スプライト選択は `enemySpriteKeyFor(kind,isBoss,dropsNipper)` を live描画と共用
- `damage.go` — `DamageEvent{Pos, Amount, ToPlayer}` と `emitDamage`。`SoundEvent` と同じ「core はデータだけ貯める」パターン：被弾箇所（ビーム/弾接触/範囲爆発＝敵=白、`damagePlayer`＝自機=赤）で `World.DamageEvents` に積み、毎tickクリア。0ダメージ（花火等の演出爆発）は積まない。scene が drain して浮遊ダメージ数字（`scene/damage.go` の `damagePopup`）を生成：上±45°の扇へランダムに素早く飛び→静止→数秒後フェード。**1桁ずつ `drawing.DrawText`**（数字×フォントサイズの10枚だけキャッシュ、全数の組合せを作らない）、白文字キャッシュを `ColorScale` で着色。調整値は `scene/tuning.go` の `dmg*`
- `entity.go` — `Player`（戦車：FacingAngle/Nippers 含む）/ `Enemy`（DropsNipper で燭台）/ `Projectile` / `Gem` / `Pickup`（ニッパー）。位置は `geom.PointF`、当たり判定は円（半径）
- `weapon.go` — `WeaponKind`（Cannon/Shotgun/Sniper/Laser/Gatling/Grenade/CIWS/Missile）+ `Weapon` と `Stats(p)`。**発射はアキュムレータ方式**：各 `Weapon.fireProgress` が毎tick `fireIncrement(p, fireMult)`（＝発射倍率、`BaseInterval/MinInterval` で上限クランプ）ぶん進み、`BaseInterval` に達したら発射。平均間隔 = `BaseInterval/fireMult`（`MinInterval` 下限）。**照準と発射は独立**：`AimMode`（`AimLockOn`＝圏内最近接へ/CIWS・Missile／`AimForward`＝常に前方/Gatling／`AimOutward`＝タイルから放射状/Grenade）。**CIWS以外は敵不在でも発射**（`HoldWhenNoTarget` の武器＝CIWSだけ圏内に敵が入るまで満タン保持）。1ショットの弾数は `Pellets`＋`SpreadRad`（`SpreadRandom`＝乱数拡散/Gatling・CIWS、固定均等/Shotgun）、`BurstGap>0` で時間差連射（`pelletsLeft`/`pelletTimer`：Gatling10発3tick間隔、CIWS10発2tick間隔ストリーム）。ダメージは `BaseDamage × LevelMult^Level`。**爆発弾** `ExplodeRadius>0` は寿命切れ時に範囲 `ExplodeDamage`（`Projectile.PassThrough`＝接触無視/Grenade、接触ありで未命中時のみ爆発/Missile）。**ホーミング等の弾移動は `ProjectileMover`（`projectile_mover.go`）で差し替え**：`Steer(p, w)` が毎tick速度を操舵、`Projectile.Mover`（`WeaponParams.Mover` 由来、nil=直進）。`NewHomingMover(turn, maxSpeed)`＝最近接敵へ旋回力制限付きで操舵（seek）。将来の弾系ジャンク（揺れて登る風船等）も別 Mover として追加可能。弾の生存は `ProjLife = round(ProjMaxDist / ProjSpeed)`、当たり半径 `ProjRadius`。`beamTicksLeft`/`beamAngle` でビーム照射状態を保持。**バレル描画の向き**は `Weapon.aimRender`（毎tick `weaponAim` へ `stepAngle` で平滑化、`RenderAngle()` で公開）：戦闘ミニチュアは各バレルが自分の照準方向を向く（`drawTurretTiles` の `aimBarrels`、ポーズ表示は上向き固定）
- `turret.go` — ヘックスグリッド砲塔。電力は**接続消費タイル数→発射倍率の区分線形補間**（`PowerMultiplier(curve, ConsumerTileCount())`、`Config.PowerCurve` の `PowerPoint{Tiles,Mult}` 列を両端クランプで補間）。`ComputePower`/`WeaponPower` は各タイルの**接続判定**として存続（値は描画の減光のみで参照）。`Component` インターフェースは `Name()` ＋ `Mods() Modifier`（戦車/砲塔への加算修飾子；Wire/WeaponComponent/Junk はゼロ、`Capacitor` は `FireRateAdd`）。接続中タイルの `Mods()` を合算した `Turret.Modifiers()` をタイル追加/削除時にキャッシュ再計算し、`World.FireRateMultiplier()` がカーブ値に加算（`Capacitor` ＝発射倍率 +0.1）。`Modifier` は将来 MaxHP 等の設備へ拡張可能。`distancesFrom` は接続判定・カスケード用。**ジェネレータ（中央タイル）は切断不可の接続ルートだが、武器/ジャンク等のコンポーネントを普通に載せる**（`ConsumerTileCount` からは除外＝薄めないので中央武器は実質「無料の主砲」。`ActiveWeapons` には含まれ発射、`MuzzleOffset` は中央=戦車中心）。`PurgeTile`（+`propagatePurge` カスケード）、`AddTile`（ランダム隣接配置）、`TileCount`/`ConsumerTileCount`、`ActiveWeapons`、`MuzzleOffset`
- `turret_gen.go` — `GenerateTurret()` フロンティア成長アルゴリズム（中央ジェネレータタイルも `pickComponent` で武装/Junk/Wire を載せる）、`DefaultTurretGenConfig`、`pickComponent`（武装/Junk/Wire）、`junkDeviceNames`/`tallJunkNames`/`newJunk`
- `upgrade.go` — `type Upgrade struct { Name, Desc string; Apply func(*World) }` (軽量選択肢モデル)
- `State`：`StatePlaying` / `StateLevelUp` / `StateGameOver`
- シーン再入場時の世界リセットは `InGame.OnStart()`（bamenn の `OnStarter`）で行う

### hexmap パッケージ

ヘックスグリッド座標系（cube coord: x+y+z=0）を提供。

- `index.go` — `Index{x,y}` キューブ座標。`IdxXY/IdxYZ/IdxZX` 生成、`Add/Mul/Distance`、6方向定数（Direction01〜Direction11）、`AppendAround`
- `size.go` — `Size{radius}` と `Contains(idx)`（原点からの距離≤半径かどうか）

### drawing パッケージ

- `text.go` — `DrawText` / `DrawTextByKey` / `DrawTextTemplate` / `MeasureText`。`(文字列, フォントサイズ)` をキーに**描画済みテキスト画像をキャッシュ**する（影付き）。フレーム毎に `ebiten.Image` を作らないこと
- `img.go` — `drawing.Image("key")` で `asset` 側のロード済み画像マップから取得。見つからなければ赤い "IMAGE NOT FOUND" のフォールバック画像を返す。`WhitePixel` は `DrawTriangles` で塗り図形を描くための1px白テクスチャ。`DrawSprite`＝中心 pivot で w×h ボックスに合わせて描画（縦横独立スケール）。`DrawSpriteAnchored`＝**ソースpx基準の任意 pivot (ax,ay) を中心に一様スケール＋回転**して `(cx,cy)` へ配置（アスペクト維持）。砲塔バレルのような「タイルより縦長の長方形スプライト」を土台タイル中心で旋回させるために使う
- `rect.go` — `DrawRect`（`DrawTriangles` で矩形塗り）と `ColorF32` ヘルパー
- `gauge.go` — `GaugeDrawer`。`Current/Max` の割合でバー幅と色（Min→Max 補間）を描く HP/エネルギーゲージ用

### lang パッケージ

- `asset/lang/<language>.csv` を `<言語名>` として読み込む（ファイル名 = 言語名。現状 `english` / `japanese`）。デフォルトは **english**
- CSV は `key,value` の2列。`#` 始まりはコメント、value 内の `\n` リテラルは改行に変換される
- 取得は `lang.Text("key")`。プレースホルダ入りは `lang.ExecuteTemplate("key", data)` で Go `text/template` として評価（テンプレートはキャッシュされる）。キーが無ければ `NO_TMPL: ...` を返す。`lang.Has("key")` でキー有無を判定、`lang.TextWithDefault("key", def)` は未定義時に `def` を返す（core由来の英語名をCSV移行する際のフォールバックに使用）
- `lang.Switch()` で言語を循環切替（戻り値は表示用に先頭大文字化した言語名）

### asset パッケージ

- `embed.go` — フォント（Mplus2-Regular）、`lang/*.csv`、`img/*.png` を埋め込み、`init()` で `FontFace(size)`（サイズ別キャッシュ）・言語テンプレート・画像マップを構築。`Images()` / `LoadTemplates()` / `FontFace()` を公開
- `sound.go` — `bgm.wav`（ループ）と SE（`fire`/`explosion`/`hit`）`.wav` を埋め込み、48000Hz の `audio.Context` を作る。`LoadSounds()`（`app/main.go` で起動時に呼ぶ）でSEは**デコード済みPCMを保持**し `PlaySound` ごとに `NewPlayerFromBytes` で多重再生、BGMは単一ループプレイヤー。`PlaySound(Sound)` / `PlayBGM()` / `StopBGM()`。`Sound` は `BGM` / `SEFire` / `SEExplosion` / `SEPlayerHit` の enum。**core はEbiten非依存のまま音を扱う**：`core.SoundEvent`（`SndFire`/`SndExplosion`/`SndPlayerHit`）を `World.Update` 中に `emit` で `World.SoundEvents` へ貯め、scene が毎フレーム `core.DispatchSounds(events, sink)`（同tick重複は1回に間引き）で `core.SoundSink` 実装（`scene.soundSink`→`asset.PlaySound`）へ流す。`SoundSink` がテスト用フェイクで差し替え可能な注入点

### Drawing / Performance Conventions

Ebitengine 実装時は `.claude` の `ebitengine-dev` スキルの方針に従う。要点：

- テキストは必ず `drawing.DrawText` 系を使う（毎フレームの画像生成を避けるためキャッシュ済み）
- 塗り図形は `drawing.DrawRect` / `WhitePixel` 経由の `DrawTriangles` を使い、バッチ分断やフレーム毎の画像生成を避ける
- 画像は `asset` で一括ロードして `drawing.Image(key)` で取得する（描画ループ内で新規 `ebiten.Image` を作らない）

### Dependencies

- `github.com/hajimehoshi/ebiten/v2` v2.9 — game engine
- `github.com/noppikinatta/bamenn` — scene transition management
- `github.com/noppikinatta/nyuuryoku` — input handling abstraction
