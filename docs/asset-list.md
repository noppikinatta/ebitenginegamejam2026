# アセット制作リスト

実装が出揃った時点での、必要アセットのファイル名一覧（制作・差し替え用）。
現状はすべて**プレースホルダ**が入っており、同名ファイルを上書きすれば差し替わる。
寸法は現プレースホルダの実寸（目安。比率を保てば多少前後してよい）。

- 画像は `asset/img/<name>.png`（ファイル名＝画像キー）。`drawing.Image("<name>")` で参照。
- 透過PNG推奨。特にジャンクは土台 `tile.png` の上に重ねるので**背景透過**にすると土台が活きる。

---

## UI / タイトル

| ファイル | 寸法 | 用途 |
|---|---|---|
| `title.png` | 460x300 | タイトル画面のメインビジュアル |
| `background.png` | 1280x720 | スクロール背景。レイアウトと同寸の1枚絵で、**上下左右が繋がる（シームレスにタイルする）**ように作る。ゲーム中はカメラに追従して1:1スクロール、オープニングの発進デモでは上→下にスクロール。`make bg-img` でプレースホルダ生成 |

## 自機・ピックアップ・弾

| ファイル | 寸法 | 用途 |
|---|---|---|
| `tank.png` | 48x64 | 自機（戦車）。**上向き**で authoring（砲塔は別途タイルで重なる） |
| `gem.png` | 8x8 | 経験値ジェム |
| `nipper.png` | 16x16 | ニッパー（切断回数アイテム） |
| `projectile.png` | 8x8 | 既定の弾（武器がスプライト未指定の時のフォールバック） |
| `proj_cannon.png` | **8x12** | キャノン弾（長方形・**進行方向へ回転**。上向き authoring） |
| `proj_shotgun.png` | 6x6 | ショットガン弾（丸） |
| `proj_sniper.png` | **4x16** | スナイパー弾（細長・**進行方向へ回転**。上向き authoring） |
| `proj_gatling.png` | 6x6 | ガトリング弾（丸） |
| `proj_grenade.png` | 14x14 | グレネード弾（丸・放物投擲） |
| `proj_ciws.png` | 6x6 | CIWS弾（丸） |
| `proj_missile.png` | **8x12** | ミサイル弾（やや大きめ・ホーミングで**敵方向へ回転**。上向き authoring） |
| `proj_balloon.png` | 16x16 | 風船ジャンクが出すコミカル弾 |
| `proj_steam.png` | 16x16 | コーヒーメーカーが立ち上らせる湯気 |
| `proj_toast.png` | 16x16 | トースターが飛ばすトースト |
| `proj_note.png` | 16x16 | オルゴールが漂わせる音符 |
| `proj_duck.png` | 16x16 | ラバーダック設置装置が撒くダック |
| `proj_firework.png` | 16x16 | 花火ジャンクが打ち上げる花火玉 |

## 砲塔タイル（土台 24x24）

| ファイル | 寸法 | 用途 |
|---|---|---|
| `tile.png` | 24x24 | 全武装/ジャンクが乗る**プレーンな土台タイル** |
| `tile_generator.png` | 24x24 | 中央ジェネレータ（切断不可・無料の主砲枠） |
| `tile_capacitor.png` | 24x24 | キャパシタ設備（発射倍率+0.1） |

## 武装バレル（24x48・縦長）

> **規約**：タイルより縦長の長方形スプライト。**上＝前方（照準方向）／最下段の 24x24 ブロック＝土台タイル**で authoring。
> 描画は pivot＝最下段タイル中心 `(w/2, h-12)` を土台に合わせ、一様スケール＋回転で旋回（`drawing.DrawSpriteAnchored`）。砲身の長さ＝スプライト高さで自由。

| ファイル | 寸法 | 武器 |
|---|---|---|
| `tile_weapon_cannon.png` | 24x48 | キャノン |
| `tile_weapon_shotgun.png` | 24x48 | ショットガン |
| `tile_weapon_sniper.png` | 24x48 | スナイパー |
| `tile_weapon_laser.png` | 24x48 | レーザー |
| `tile_weapon_gatling.png` | 24x48 | ガトリング |
| `tile_weapon_grenade.png` | 24x48 | グレネード |
| `tile_weapon_ciws.png` | 24x48 | CIWS |
| `tile_weapon_missile.png` | 24x48 | ミサイル |

## ジャンク（種類ごとに1枚）

> 平型は 24x24 で土台 `tile.png` の上に重ねて描画。**Five-storied Pagoda のみ縦長 24x72**（土台から立ち上がる背の高いフィクスチャ。バレル同様、最下段24x24＝土台で authoring）。
> ファイル名は `core.JunkImageKey(デバイス名)` のスラッグ。`make junk-img` でプレースホルダ再生成。

| ファイル | 寸法 | デバイス名 |
|---|---|---|
| `junk_unusual_banana.png` | 24x24 | Unusual Banana |
| `junk_electric_fan.png` | 24x24 | Electric Fan（扇風機） |
| `junk_calculator.png` | 24x24 | Calculator（電卓） |
| `junk_wi_fi_antenna.png` | 24x24 | Wi-Fi Antenna |
| `junk_five_storied_pagoda.png` | **24x72** | Five-storied Pagoda（五重塔・**Tall**） |
| `junk_fax_machine.png` | 24x24 | Fax Machine（FAX） |
| `junk_lava_lamp.png` | 24x24 | Lava Lamp |
| `junk_oil_heater.png` | 24x24 | Oil Heater（オイルヒーター） |
| `junk_rice_cooker.png` | 24x24 | Rice Cooker（炊飯器） |
| `junk_modern_art_fountain.png` | 24x24 | Modern Art Fountain（泉/現代アート） |
| `junk_concept.png` | 24x24 | Concept（概念） |
| `junk_nft_nuclear_missile.png` | 24x24 | NFT Nuclear Missile |
| `junk_horns.png` | 24x24 | Horns（ツノ） |
| `junk_high_end_gpu.png` | 24x24 | High-End GPU（ハイエンドGPU） |
| `junk_balloon_service_unit.png` | 24x24 | Balloon Service Unit（風船サービス装置） |
| `junk_coffee_maker.png` | 24x24 | Coffee Maker |
| `junk_toaster.png` | 24x24 | Toaster |
| `junk_music_box.png` | 24x24 | Music Box（オルゴール） |
| `junk_rubber_duck_dispenser.png` | 24x24 | Rubber Duck Dispenser |
| `junk_fireworks.png` | 24x24 | Fireworks（花火） |

## 敵・ボス

> 敵は描画時に半径×2のフットプリント。寸法は現プレースホルダ実寸。

| ファイル | 寸法 | 用途 |
|---|---|---|
| `enemy.png` | 32x32 | グラント（標準ザコ） |
| `enemy_swarmer.png` | 28x28 | スウォーマー（高速・低耐久・群れ） |
| `enemy_brute.png` | 52x52 | ブルート（低速・高耐久・大型） |
| `candlestick.png` | 32x32 | 燭台（停止・無害・ニッパーdrop） |
| `boss1.png` | 80x80 | 3分ボス（Prototype Hauler） |
| `boss2.png` | 80x80 | 6分ボス（Siege Engine） |
| `boss3.png` | 112x112 | 10分・最終ボス（The Disconnector） |

---

## サウンド

### BGM（自作前提・生wavを直接コミット＆埋め込み）

2曲構成。差し替えは該当ファイルを置換するだけ。

| ファイル | 用途 | 鳴るタイミング |
|---|---|---|
| `asset/sound/bgm_title.wav` | タイトル曲 | オープニング＋タイトル |
| `asset/sound/bgm_game.wav` | ゲーム曲 | ゲーム中＋リザルト |

> 同じ曲を共有するシーン間（オープニング↔タイトル、ゲーム↔リザルト）は鳴り直さずシームレスに継続。曲が変わる切替時（タイトル→ゲーム開始、ゲーム→オープニング復帰）のみ頭から再生。

### SE（フリー素材可・`raw/` は非コミット→`se.pak`）

発射音は**武器ごとに別ファイル**。`raw/` に置いて `make sound-pak` で `se.pak` に格納。

| ファイル | 用途 |
|---|---|
| `asset/sound/raw/fire_cannon.wav` | キャノン発射 |
| `asset/sound/raw/fire_shotgun.wav` | ショットガン発射 |
| `asset/sound/raw/fire_sniper.wav` | スナイパー発射 |
| `asset/sound/raw/fire_laser.wav` | レーザー発射 |
| `asset/sound/raw/fire_gatling.wav` | ガトリング発射 |
| `asset/sound/raw/fire_grenade.wav` | グレネード発射 |
| `asset/sound/raw/fire_ciws.wav` | CIWS発射 |
| `asset/sound/raw/fire_missile.wav` | ミサイル発射 |
| `asset/sound/raw/explosion.wav` | 爆発（爆発弾・グレネード等） |
| `asset/sound/raw/hit.wav` | 自機被弾 |

> SEは1個だけでも差し替え可（マージ方式）：当該wavを `raw/` に置いて `make sound-pak`。他は既存pakから維持。SE削除は `raw/` から消して `-rebuild`。

---

## 制作不要（対応済み）

- フォント `asset/font/Mplus2-Regular.ttf`（ライセンス `asset/font/license.md`）
- 多言語テキスト `asset/lang/english.csv` / `japanese.csv`

## 将来・未実装（今は不要）

弾を出すジャンク全6種は配線済み（`docs/asset-plan.md` 参照）。弾スプライト `proj_steam` / `proj_toast` / `proj_note` / `proj_duck` / `proj_firework` はプレースホルダ（`make proj-img` ＝ `tools/genprojimg` で生成した色付き円）。本番アートに差し替える場合は該当 PNG を上書きするだけ。

通常武器の弾もそれぞれ専用スプライト（`proj_cannon` / `proj_shotgun` / `proj_sniper` / `proj_gatling` / `proj_grenade` / `proj_ciws` / `proj_missile`、Laser はビームのため弾なし）を持つ。同じく `make proj-img` でプレースホルダ生成（丸弾は円、長方形弾は縦長カプセル）。割り当ては `core.Sprite*` 定数 → `data/weapon.go` の `WeaponParams.Sprite`／`ProjDrawW`／`ProjDrawH`／`ProjFaceVelocity`。**Cannon / Sniper / Missile は `ProjFaceVelocity` で進行方向へ回転**（弾は上向き authoring、scene が `Vel.Angle()+π/2` で回す＝戦車と同規約）。本番アートは該当 PNG を上書きするだけ。
