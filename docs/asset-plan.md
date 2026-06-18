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

## 設備
- キャパシタ: **実装済み**。接続中、発射倍率に **+0.1**（`Config.CapacitorFireRateBonus`）。`Component.Mods() Modifier` の修飾子システム経由で、タイル追加/削除時に再計算。画像 `tile_capacitor`（現状プレースホルダ＝tile_junkのコピー、本番アートが必要）。博士のタイルバンドルで `DoctorSpec.CapacitorChance`(=0.15) の確率で出現
  - 将来拡張: `Modifier` に `MaxHPAdd` 等を足せば増加装甲のような設備も同じ仕組みで追加可能

## ジャンク（何かが出る物。弾の発射を応用して作る）
- 風船サービス装置
- コーヒーメーカー
- トースター
- オルゴール
- ラバーダック設置装置
- 花火

## ジャンク（何も出ないもの）
- unusual banana
- 扇風機
- 電卓
- Wi-Fiアンテナ
- サグラダファミリア
- FAX
- ラーバランプ
- オイルヒーター
- 炊飯器
- 泉(現代アート)
- 愚か者には見えないキャノン砲
- NFT核ミサイル
- ツノ
- AI照準装置

---

### 現状（実装済み・参考）
- 武器: Cannon / Shotgun / Sniper / Laser（`core/weapon.go`、画像キー `tile_weapon_*`）
- ジャンク: 機械的効果は同一・画像は `tile_junk` 1枚共用（`core/turret_gen.go` の `junkDeviceNames`）
- 「設備」「弾が出るジャンク」「キャパシタ（電力ボーナス）」は**新カテゴリ**で未実装
