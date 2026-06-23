package data

import (
	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// defaultTurretGen returns the standard starting-turret generation parameters,
// including the single central generator. Adjust these to change how
// complex/powerful the initial loadout feels.
func defaultTurretGen() core.TurretGenConfig {
	return core.TurretGenConfig{
		WeaponCount: 3,
		JunkCount:   29,
		BranchProb:  0.35,
		Generators: []core.GeneratorConfig{
			{Index: hexmap.IdxXY(0, 0), Power: 100},
		},
	}
}

// defaultDoctor returns the standard level-up offer balance.
func defaultDoctor() core.DoctorSpec {
	return core.DoctorSpec{
		NipperChance:     0.25,
		UpgradeChance:    0.625,
		NipperMin:        5,
		NipperMax:        10,
		MaxUpgrades:      3,
		MaxBundleTiles:   3,
		CapacitorChance:  0.15, // per bundle tile, chance it is a Capacitor instead of weapon/junk
		RepairUnitChance: 0.1,  // per bundle tile, chance it is a Repair Unit
		ArmorChance:      0.1,  // per bundle tile, chance it is an Armor tile
	}
}
