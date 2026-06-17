package data

import "github.com/noppikinatta/ebitenginegamejam2026/core"

// weaponParams returns the balance preset for every weapon kind. Edit these to
// tune combat feel without touching game logic. See core.WeaponParams for the
// formulae these numbers feed.
func weaponParams() map[core.WeaponKind]core.WeaponParams {
	return map[core.WeaponKind]core.WeaponParams{
		core.KindCannon: {
			BaseDamage: 5, EnergyDamage: 3,
			BaseInterval: 45, EnergyInterval: 4, MinInterval: 6,
			ProjSpeed: 6, BaseRange: 220, EnergyRange: 20,
			LevelMult: 1.2,
		},
		core.KindShotgun: {
			BaseDamage: 3, EnergyDamage: 1.5,
			BaseInterval: 28, EnergyInterval: 2, MinInterval: 8,
			ProjSpeed: 5, BaseRange: 150, EnergyRange: 10,
			LevelMult: 1.2,
		},
		core.KindSniper: {
			BaseDamage: 20, EnergyDamage: 8,
			BaseInterval: 120, EnergyInterval: 7, MinInterval: 20,
			ProjSpeed: 10, BaseRange: 400, EnergyRange: 40,
			LevelMult: 1.2,
		},
		core.KindLaser: {
			BaseDamage: 2, EnergyDamage: 0.8,
			BaseInterval: 90, EnergyInterval: 5, MinInterval: 15,
			BaseRange: 300, EnergyRange: 25,
			BeamBaseLength: 300, BeamEnergyLength: 25,
			BeamBaseWidth: 6, BeamEnergyWidth: 0.5,
			BeamBaseDuration: 30, BeamEnergyDuration: 4,
			LevelMult: 1.2,
		},
	}
}
