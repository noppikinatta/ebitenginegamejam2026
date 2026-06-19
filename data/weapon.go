package data

import "github.com/noppikinatta/ebitenginegamejam2026/core"

// weaponParams returns the balance preset for every weapon kind. Edit these to
// tune combat feel without touching game logic. See core.WeaponParams for the
// formulae these numbers feed.
func weaponParams() map[core.WeaponKind]core.WeaponParams {
	return map[core.WeaponKind]core.WeaponParams{
		core.KindCannon: {
			BaseDamage:   20,
			BaseInterval: 720, MinInterval: 6, // 12s between shots at fireMult=1
			ProjSpeed: 6, ProjMaxDist: 260, ProjRadius: 6, BaseRange: 200,
			LevelMult: 1.2,
		},
		core.KindShotgun: {
			BaseDamage:   8,                   // ×4 pellets per shot
			BaseInterval: 720, MinInterval: 8, // 12s
			ProjSpeed: 5, ProjMaxDist: 150, ProjRadius: 2, BaseRange: 100,
			Pellets: 4, SpreadRad: 0.3, // simultaneous fixed 4-pellet spread
			LevelMult: 1.2,
		},
		core.KindSniper: {
			BaseDamage:   10,
			BaseInterval: 960, MinInterval: 20, // 16s
			ProjSpeed: 10, ProjMaxDist: 640, ProjRadius: 2, BaseRange: 360,
			LevelMult: 1.2,
		},
		core.KindLaser: {
			BaseDamage:   1,                     // per tick; ×BeamBaseDuration(30) = 30 over a burst
			BaseInterval: 1440, MinInterval: 15, // 24s
			BaseRange:        200,
			BeamBaseLength:   200,
			BeamBaseWidth:    6,
			BeamBaseDuration: 30,
			LevelMult:        1.2,
		},
		core.KindGatling: {
			BaseDamage:   2,
			BaseInterval: 480, MinInterval: 6, // 8s between bursts
			ProjSpeed: 5, ProjMaxDist: 240, ProjRadius: 2,
			Pellets: 10, SpreadRad: 0.2, SpreadRandom: true, BurstGap: 3, // staggered random stream
			Aim:       core.AimForward, // never locks on; always fires forward
			LevelMult: 1.2,
		},
		core.KindGrenade: {
			BaseDamage:   0,                     // no contact damage; all damage is the explosion
			BaseInterval: 1800, MinInterval: 30, // 30s
			ProjSpeed: 2, ProjMaxDist: 120, // lobbed slowly; expires after 60 ticks
			Aim:           core.AimOutward,                          // never locks on; always fires outward
			ExplodeRadius: 64, ExplodeDamage: 15, PassThrough: true, // flies through; detonates where it lands
			LevelMult: 1.2,
		},
		core.KindCIWS: {
			BaseDamage:   2,
			BaseInterval: 480, MinInterval: 6, // 8s between bursts
			ProjSpeed: 5, ProjMaxDist: 120, ProjRadius: 2, BaseRange: 80, // very short range point defence
			Pellets: 10, SpreadRad: 0.1, SpreadRandom: true, BurstGap: 2, // tight rapid burst
			HoldWhenNoTarget: true, // stays charged until something enters range
			LevelMult:        1.2,
		},
		core.KindMissile: {
			BaseDamage:   8,                    // contact damage
			BaseInterval: 960, MinInterval: 20, // 16s
			ProjSpeed: 2, ProjMaxDist: 240, ProjRadius: 6, BaseRange: 240, // slow shell, long lock range
			ExplodeRadius: 48, ExplodeDamage: 10, // smaller blast than the grenade, only if it expires unhit
			Mover:     core.NewHomingMover(0.3, 6), // homes onto the nearest enemy (turn force 0.3, cruise speed 6)
			LevelMult: 1.2,
		},
	}
}

// powerCurve maps the connected consumer tile count to a fire-rate multiplier.
// Fewer tiles concentrate power into faster fire; more tiles dilute it. Points
// must be sorted ascending by Tiles; outside the range the end values clamp.
// These are placeholder breakpoints — tune them during playtesting.
func powerCurve() []core.PowerPoint {
	return []core.PowerPoint{
		{Tiles: 10, Mult: 4.0},
		{Tiles: 32, Mult: 1.0},
		{Tiles: 40, Mult: 0.5},
	}
}
