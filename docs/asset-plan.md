# 武器・設備・ジャンク アセット案（メモ）

> ステータス: **アイデアメモ（詳細は後日説明される予定）**。
> ここでは「どんな案があるか」だけを記録する。挙動・ステータス・画像キーは未確定。

> **武装バレルのアート規約**：タイルは24×24だが、武装バレルは**タイルより縦長の長方形スプライト**として別に描く（socket＝`tile_wire` 等の24×24は据え置き）。バレルは「上＝前方（照準方向）／最下段の 24×24 ブロック＝土台タイル」で authoring する。描画は `drawing.DrawSpriteAnchored` が pivot＝最下段タイル中心 `(iw/2, ih-12)`（12＝タイル半分）を socket 中心に合わせ、一様スケール `size/24`・回転 `theta` で旋回。砲身の長さ＝スプライト高さ（`ih`）で自由。現状の `tile_weapon_*.png` は 24×48 のプレースホルダ（武器ごとに色違い、土台＋砲身＋砲口）。

## 武器
- キャノン (Cannon) — 実装済
- ショットガン (Shotgun) — 実装済（4発・固定拡散）
- スナイパー (Sniper) — 実装済
- レーザー (Laser) — 実装済
- ガトリング (Gatling) — **実装済**。前方固定・ロックオンなし。8s毎に2ダメージ弾10発を3tick間隔ストリーム、ランダム拡散±0.2rad。画像 `tile_weapon_gatling`（プレースホルダ＝cannonコピー、本番アート必要）
- グレネード (Grenade) — **実装済**。外側固定・ロックオンなし。30s毎に速度2の無ダメージ弾を放射状に発射、寿命切れ(120px)で半径64に15ダメージ爆発。画像 `tile_weapon_grenade`（同上プレースホルダ）。爆発エフェクト実装済（オレンジ円がアルファ減衰、`vector.DrawFilledCircle`、core が `Explosion` をキュー）
- CIWS — **実装済**。ロックオン半径80・点防御。`HoldWhenNoTarget` で圏内に敵が入るまで満タン保持し、8s毎に2ダメージ弾10発を2tick間隔バースト（小ランダム拡散±0.1）。画像 `tile_weapon_ciws`（プレースホルダ＝cannonコピー）
- ミサイル (Missile) — **実装済**。ロックオン半径240・16s毎に発射。弾速2の低速弾を `ProjectileMover`（ホーミング）で毎tick敵方向へ操舵（seek／旋回力0.3・巡航6）。接触8ダメージ、未命中で寿命切れ時に半径48・10ダメージ爆発（グレネードより小）。画像 `tile_weapon_missile`（同上プレースホルダ）

> **弾の移動ロジック差し替え**：`core.ProjectileMover` インターフェース（`Steer(p, w)`）で弾の毎tick操舵を差し替え可能。`Projectile.Mover` に設定（`WeaponParams.Mover` 経由）。nil は直進。ホーミングは `core.NewHomingMover(turn, maxSpeed)`。将来「弾の仕組みを使うジャンク」（例：ゆらゆら登る風船）も別の `ProjectileMover` 実装として追加できる。`Projectile.PassThrough` で接触を無視するか（グレネード=true／ミサイル=false）を制御。

## 敵・ボス
スポーンは**ディレクタ方式**（`core` の `SpawnPhases` 時間帯別重み → `spawnPackOf` パック生成、HPは `HPBase×2^(tick/HPDoublingTicks)` で時間スケール）。ザコは描画時 `radius×2` のフットプリント。
- **グラント (Grunt)** — 実装済。バランス型の追尾。画像 `enemy`（既存）。HPBase10/速1.2/半径16/接触8。単体出現
- **スウォーマー (Swarmer)** — 実装済。高速・低耐久、**パック（3〜6体）**で出現。画像 `enemy_swarmer`（プレースホルダ＝ピンクの円）。HPBase5/速2.1/半径11/接触4
- **ブルート (Brute)** — 実装済。低速・高耐久・大型・痛い。画像 `enemy_brute`（プレースホルダ＝茶の円）。HPBase60/速0.7/半径26/接触18。3分以降に出現
- **燭台 (Candlestick)** — 実装済（停止・無害・ニッパー drop）。画像 `candlestick`（既存）
- **ボス** — 実装済。`Config.Bosses` で 3分/6分/10分に1体ずつ出現（HP固定・時間スケールなし）。`ActiveBoss()`＋HUD上部にHPバー＋名前。画像 `boss`（プレースホルダ＝赤/金リングの円、`radius×2`で最大表示）
  - 3分: Prototype Hauler（HP1200）／6分: Siege Engine（HP3000）／**10分: The Disconnector（HP8000・Final）→ 撃破で `StateCleared`（クリア）**
  - ※HP/速/ダメージ・スポーン重み・`HPDoublingTicks` 等はすべて初期値、要バランス調整。実アート未着手（grunt/candlestick以外はプレースホルダ）。ボス専用挙動（召喚・弾幕等）は未実装＝ただの大型追尾

## 設備
- キャパシタ: **実装済み**。接続中、発射倍率に **+0.1**（`Config.CapacitorFireRateBonus`）。`Component.Mods() Modifier` の修飾子システム経由で、タイル追加/削除時に再計算。画像 `tile_capacitor`（現状プレースホルダ＝tile_junkのコピー、本番アートが必要）。博士のタイルバンドルで `DoctorSpec.CapacitorChance`(=0.15) の確率で出現
  - 将来拡張: `Modifier` に `MaxHPAdd` 等を足せば増加装甲のような設備も同じ仕組みで追加可能

> **実装状況**：全ジャンクは `core/turret_gen.go` の `junkSpecs` レジストリ（`junkSpec{Name, Tall, Emitter}`）にデータ定義済み。装飾ジャンクは inert（電力を薄めるだけ）。表示名は `asset/lang/*.csv` の `junk-<slug>` キーで多言語化。画像は `tile_junk` 共用（Tall のみ `junk_tower`）。`Emitter`（`*EmitterSpec`）が設定されたジャンクのみ弾を出す。

## ジャンク（何かが出る物。弾の発射を応用して作る）

**発射の仕組み（実装済みの基盤）**：武器とは独立した軽量エミッタ系で `Projectile`＋`ProjectileMover` を再利用。
- `core/junk_emitter.go`：`EmitterSpec{Interval, Aim(EmitUp/Outward/Random), Speed, Life, Radius, Sprite, Mover, ExplodeRadius}`、ランタイム `junkEmitter{spec, timer}`（`Junk.emitter *junkEmitter`、値コピーでも timer 共有）、`World.updateJunkEmitters()`（`Update` で毎tick、`MuzzleOffset` から発射）、`emitAngle`。
- `Turret.ActiveEmitters()`：接続中の発射ジャンクを収集（`ComputePower` ベース＝junkも含む）。
- 発射は**固定 Interval（発射倍率の影響なし）**。弾は**コミカル演出＝0ダメージ・`PassThrough`（敵すり抜け）**で、ジャンクは依然「切るべき無用タイル」。
- `Projectile.Sprite`（画像キー、空＝既定弾）に加え `DrawW/DrawH`（描画寸法、0＝既定）・`FaceVelocity`（進行方向へ回転）を持つ。scene はサイズ未指定時に junk弾を 16px・既定弾を 8px でフォールバック描画。Mover は `core/projectile_mover.go`。

## 通常武器の弾（武器ごとに専用スプライト）— 実装済

各 `WeaponKind`（Laser 除く）に `core.Sprite*` 定数 → `data/weapon.go` の `WeaponParams.Sprite`／`ProjDrawW`／`ProjDrawH`／`ProjFaceVelocity` を割り当て。`emitPellets` が `Projectile` へ伝搬し、scene が `Projectile.Sprite`／`DrawW`／`DrawH` で描画、`FaceVelocity` の弾は `Vel.Angle()+π/2`（弾は上向き authoring、戦車と同規約）で回転。
- Cannon `proj_cannon`（8×12・**回転**）/ Shotgun `proj_shotgun`（6×6・丸）/ Sniper `proj_sniper`（4×16・**回転**）/ Gatling `proj_gatling`（6×6・丸）/ Grenade `proj_grenade`（14×14・丸）/ CIWS `proj_ciws`（6×6・丸）/ Missile `proj_missile`（8×12・ホーミング＋**回転**）。
- プレースホルダは `make proj-img`（`tools/genprojimg`）が丸弾＝円・回転弾＝縦長カプセルで生成。サイズ・色は初期値、要バランス調整。

**実装状況**
- 風船サービス装置 (Balloon Service Unit) — **実装済**。`NewRiseMover`（上昇＋サイン横揺れ）で画面上へ漂う。Interval 90tick、Sprite `proj_balloon`（プレースホルダ＝赤丸）。
- コーヒーメーカー (Coffee Maker) — **実装済**。外側へ噴出→`NewGravityMover`（下向き加速）で落下。Interval 70tick、Sprite `proj_coffee`
- トースター (Toaster) — **実装済**。上へポン→`NewGravityMover` でアーチ落下。Interval 120tick、Sprite `proj_toast`
- オルゴール (Music Box) — **実装済**。外側へ漂う音符（`NewRiseMover(lift=0)` で横揺れドリフト）。Interval 60tick、Sprite `proj_note`
- ラバーダック設置装置 (Rubber Duck Dispenser) — **実装済**。ランダム方向に短寿命のダックを撒く（`NewGravityMover`）。Interval 100tick、Sprite `proj_duck`
- 花火 (Fireworks) — **実装済**。上へ→寿命切れで `ExplodeRadius>0` の**0ダメージ爆発**演出（`SndExplosion` も鳴る）。Interval 150tick、Sprite `proj_firework`

> 共通 `gravityMover`（`core/projectile_mover.go` の `NewGravityMover`）と各 `proj_*` プレースホルダ画像（`tools/genprojimg` ＝ `make proj-img` で生成）を追加し、各 `EmitterSpec` を `junkSpecs` に紐付けて全6種が稼働。説明は共通 `junk-desc` を流用（個別説明は任意）。

## ジャンク（何も出ないもの）— 実装済（inert）
- unusual banana (Unusual Banana)
- 扇風機 (Electric Fan)
- 電卓 (Calculator)
- Wi-Fiアンテナ (Wi-Fi Antenna)
- 五重塔 (Five-storied Pagoda) — `Tall`
- FAX (Fax Machine)
- ラーバランプ (Lava Lamp)
- オイルヒーター (Oil Heater)
- 炊飯器 (Rice Cooker)
- 泉(現代アート) (Modern Art Fountain)
- 愚か者には見えないキャノン砲 (Invisible Cannon)
- NFT核ミサイル (NFT Nuclear Missile)
- ツノ (Horns)
- AI照準装置 (AI Targeting Device)

---

### 現状（実装済み・参考）
- 武器: Cannon / Shotgun / Sniper / Laser（`core/weapon.go`、画像キー `tile_weapon_*`）
- ジャンク: 機械的効果は同一（inert）・画像は `tile_junk` 1枚共用（`core/turret_gen.go` の `junkSpecs`、20種）。Tall のみ `junk_tower`
- キャパシタ（電力ボーナス）は実装済（設備カテゴリ）
- 「弾が出るジャンク」はエミッタ系（`core/junk_emitter.go`）実装済み。**全6種（風船・コーヒー・トースター・オルゴール・ラバーダック・花火）が稼働**
