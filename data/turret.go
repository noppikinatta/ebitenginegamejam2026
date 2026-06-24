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

// defaultDoctor returns the standard level-up offer balance. The four weights
// set how often each item kind appears (normalised, so only their ratios
// matter); tune these to shift the overall feel of level-ups.
func defaultDoctor() core.DoctorSpec {
	return core.DoctorSpec{
		NipperWeight:        0.10, // spare tile cuts
		WeaponAddWeight:     0.25, // new weapon / equipment tile
		WeaponUpgradeWeight: 0.15, // level up an existing weapon
		JunkWeight:          0.50, // useless junk tile (the doctors' specialty)
		NipperMin:           1,
		NipperMax:           3,
		MaxItems:            3, // an offer carries 1..3 items
	}
}
