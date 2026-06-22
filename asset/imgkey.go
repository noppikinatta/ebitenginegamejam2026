package asset

// Image keys for embedded PNGs in asset/img. Each constant MUST equal the
// file's base name (without ".png"); initImages registers images under that
// key, and drawing.Image(key) looks them up here.
const (
	ImgTitle        = "title"
	ImgBackground   = "background" // layout-sized, seamlessly-tiling scrolling backdrop
	ImgTank         = "tank"
	ImgEnemy        = "enemy" // grunt (default zako)
	ImgEnemySwarmer = "enemy_swarmer"
	ImgEnemyBrute   = "enemy_brute"
	ImgBoss1        = "boss1" // 3-min boss (Prototype Hauler)
	ImgBoss2        = "boss2" // 6-min boss (Siege Engine)
	ImgBoss3        = "boss3" // 10-min final boss (The Disconnector)
	ImgCandlestick  = "candlestick"
	ImgGem          = "gem"
	ImgNipper       = "nipper"
	ImgHeart        = "heart" // HP pickup dropped (rarely) by candlesticks
	ImgProjectile   = "projectile"

	// Turret tiles (24x24).
	ImgTile           = "tile" // plain base tile every weapon/junk is drawn on top of
	ImgTileGenerator  = "tile_generator"
	ImgTileCapacitor  = "tile_capacitor"
	ImgTileRepairUnit = "tile_repair"
	ImgTileArmor      = "tile_armor"

	// Junk devices get one image per type, keyed by core.JunkImageKey(name)
	// (e.g. "junk_unusual_banana"). Flat junk is a 24x24 tile overlay; tall junk
	// (e.g. "junk_five_storied_pagoda") is a 24x72 always-upright fixture. Placeholder
	// art is produced by tools/genjunkimg; drop in real PNGs by overwriting the
	// matching files. The scene resolves these keys via core.JunkImageKey, so they
	// have no constants here.

	// Weapon tiles, one per WeaponKind.
	ImgTileWeaponCannon  = "tile_weapon_cannon"
	ImgTileWeaponShotgun = "tile_weapon_shotgun"
	ImgTileWeaponSniper  = "tile_weapon_sniper"
	ImgTileWeaponLaser   = "tile_weapon_laser"
	ImgTileWeaponGatling = "tile_weapon_gatling"
	ImgTileWeaponGrenade = "tile_weapon_grenade"
	ImgTileWeaponCIWS    = "tile_weapon_ciws"
	ImgTileWeaponMissile = "tile_weapon_missile"
)
