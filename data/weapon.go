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
			BaseDamage:   8,                   // ×4 pellets per shot (see Weapon.ProjectileOffsets)
			BaseInterval: 720, MinInterval: 8, // 12s
			ProjSpeed: 5, ProjMaxDist: 150, ProjRadius: 2, BaseRange: 100,
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
