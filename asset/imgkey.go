package asset

// Image keys for embedded PNGs in asset/img. Each constant MUST equal the
// file's base name (without ".png"); initImages registers images under that
// key, and drawing.Image(key) looks them up here.
const (
	ImgTitle        = "title"
	ImgTank         = "tank"
	ImgEnemy        = "enemy" // grunt (default zako)
	ImgEnemySwarmer = "enemy_swarmer"
	ImgEnemyBrute   = "enemy_brute"
	ImgBoss         = "boss"
	ImgCandlestick  = "candlestick"
	ImgGem          = "gem"
	ImgNipper       = "nipper"
	ImgProjectile   = "projectile"

	// Turret tiles (24x24).
	ImgTile          = "tile" // plain base tile every weapon/junk is drawn on top of
	ImgTileGenerator = "tile_generator"
	ImgTileJunk      = "tile_junk"
	ImgTileCapacitor = "tile_capacitor"

	// Tall junk fixtures, drawn as always-upright sprites (mount tile at bottom).
	ImgJunkTower = "junk_tower"

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
