package asset

// Image keys for embedded PNGs in asset/img. Each constant MUST equal the
// file's base name (without ".png"); initImages registers images under that
// key, and drawing.Image(key) looks them up here.
const (
	ImgTitle       = "title"
	ImgTank        = "tank"
	ImgEnemy       = "enemy"
	ImgCandlestick = "candlestick"
	ImgGem         = "gem"
	ImgNipper      = "nipper"
	ImgProjectile  = "projectile"

	// Turret tiles (24x24).
	ImgTileWire      = "tile_wire"
	ImgTileGenerator = "tile_generator"
	ImgTileJunk      = "tile_junk"
	ImgTileCapacitor = "tile_capacitor"

	// Weapon tiles, one per WeaponKind.
	ImgTileWeaponCannon  = "tile_weapon_cannon"
	ImgTileWeaponShotgun = "tile_weapon_shotgun"
	ImgTileWeaponSniper  = "tile_weapon_sniper"
	ImgTileWeaponLaser   = "tile_weapon_laser"
)
