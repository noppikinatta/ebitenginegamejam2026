package data

// TurretGenSpec controls the shape and weapon density of the randomly generated
// starting turret. Adjust these to change how complex/powerful the initial
// loadout feels.
type TurretGenSpec struct {
	MaxTiles      int
	BranchProb    float64 // higher = more forking, lower = longer arms
	WeaponDensity float64 // probability each non-generator tile is a weapon
	JunkDensity   float64 // probability a non-weapon tile is useless junk
	GenPower      float64 // generator output (shared equally across all tiles)
}

// DefaultTurretGen returns the standard starting turret generation parameters.
func DefaultTurretGen() TurretGenSpec {
	return TurretGenSpec{
		MaxTiles:      22,
		BranchProb:    0.35,
		WeaponDensity: 0.45,
		JunkDensity:   0.15,
		GenPower:      100,
	}
}

// DoctorSpec controls the balance of the three level-up offer types:
//   - Nippers: instantly collectible tile cuts.
//   - Weapon upgrade: selected weapons gain +1 Level (+20% Damage each).
//   - Tile bundle: 1-MaxBundleTiles new tiles, each 50% weapon / 50% junk.
type DoctorSpec struct {
	NipperChance   float64 // probability of a nipper offer (evaluated first)
	UpgradeChance  float64 // cumulative: upgrade if r < UpgradeChance after nipper check
	NipperMin      int     // minimum nippers per nipper offer
	NipperMax      int     // maximum nippers per nipper offer
	MaxUpgrades    int     // max weapons upgraded per upgrade offer
	MaxBundleTiles int     // max tiles added per tile-bundle offer
}

// DefaultDoctor returns the standard level-up offer balance.
func DefaultDoctor() DoctorSpec {
	return DoctorSpec{
		NipperChance:   0.25,
		UpgradeChance:  0.625,
		NipperMin:      5,
		NipperMax:      10,
		MaxUpgrades:    3,
		MaxBundleTiles: 3,
	}
}
