package data

import "github.com/noppikinatta/ebitenginegamejam2026/core"

// weaponParams returns the balance preset for every weapon kind. Edit these to
// tune combat feel without touching game logic. See core.WeaponParams for the
// formulae these numbers feed.
func weaponParams() map[core.WeaponKind]core.WeaponParams {
	return map[core.WeaponKind]core.WeaponParams{
		core.KindCannon: {
			BaseDamage:   5,
			BaseInterval: 45, MinInterval: 6,
			ProjSpeed: 6, BaseRange: 220,
			LevelMult: 1.2,
		},
		core.KindShotgun: {
			BaseDamage:   3,
			BaseInterval: 28, MinInterval: 8,
			ProjSpeed: 5, BaseRange: 150,
			LevelMult: 1.2,
		},
		core.KindSniper: {
			BaseDamage:   20,
			BaseInterval: 120, MinInterval: 20,
			ProjSpeed: 10, BaseRange: 400,
			LevelMult: 1.2,
		},
		core.KindLaser: {
			BaseDamage:   2,
			BaseInterval: 90, MinInterval: 15,
			BaseRange:        300,
			BeamBaseLength:   300,
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
