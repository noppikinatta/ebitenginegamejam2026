// Package data holds all balance constants and spec types for the game.
// It has no imports from other packages in this module so that core can
// import it without creating a circular dependency.
//
// To tune gameplay numbers, edit this package — it is the single source of
// truth for every magic number that drives combat, levelling, and spawning.
package data

// World-level constants.
const (
	StartingNippers     = 3    // nippers the player begins every run with
	MaxTurretTiles      = 40   // soft cap on turret size; forces non-tile offers
	CandlestickInterval = 600  // ticks between candlestick spawns (~10 s at 60 TPS)
	XPToNextGrowth      = 1.25 // XPToNext multiplier applied each level-up
	WeaponLevelMult     = 1.2  // Damage multiplier per weapon Level increment
)

// PlayerSpec contains the player's starting stats for a fresh run.
type PlayerSpec struct {
	HP, MaxHP, Speed, Radius float64
	Level                    int
	XPToNext                 float64
}

// DefaultPlayer returns balanced starting stats for the player tank.
func DefaultPlayer() PlayerSpec {
	return PlayerSpec{
		HP: 100, MaxHP: 100,
		Speed: 3, Radius: 16,
		Level:    1,
		XPToNext: 10,
	}
}

// PickupRanges are shared by XP gems and nipper pickups.
type PickupRanges struct {
	PickupDist  float64 // collect on contact within this distance
	MagnetDist  float64 // start moving toward player within this distance
	MagnetSpeed float64 // px per tick toward the player
}

// DefaultPickupRanges returns the standard gem/nipper pickup behaviour.
func DefaultPickupRanges() PickupRanges {
	return PickupRanges{PickupDist: 28, MagnetDist: 90, MagnetSpeed: 4}
}
