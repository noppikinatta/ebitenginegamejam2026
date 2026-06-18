// Package data is the single source of truth for the game's balance numbers.
//
// It imports core and supplies a fully populated core.Config; core itself
// defines the config/spec types but holds no concrete numbers. The dependency
// therefore points data -> core (config/infrastructure layer depends on the
// domain), and the scene layer injects data.NewConfig() into core.NewWorld.
//
// To tune gameplay, edit the values here — in a longer-lived product this is
// where loading from a file or database would live.
package data

import "github.com/noppikinatta/ebitenginegamejam2026/core"

// NewConfig returns the canonical balance configuration for a fresh run.
func NewConfig() core.Config {
	return core.Config{
		StartingNippers:        3,    // nippers the player begins every run with
		MaxTurretTiles:         40,   // soft cap on turret size; forces non-tile offers
		CandlestickInterval:    600,  // ticks between candlestick spawns (~10 s at 60 TPS)
		XPToNextGrowth:         1.25, // XPToNext multiplier applied each level-up
		CapacitorFireRateBonus: 0.1,  // each connected Capacitor adds +0.1 to the fire-rate multiplier

		Player:       defaultPlayer(),
		Pickup:       defaultPickupRanges(),
		Spawn:        defaultSpawn(),
		Doctor:       defaultDoctor(),
		EnemyScaling: basicEnemyScaling(),
		Candlestick:  candlestick(),
		TurretGen:    defaultTurretGen(),
		Weapons:      weaponParams(),
		PowerCurve:   powerCurve(),
	}
}

// defaultPlayer returns balanced starting stats for the player tank. Pos,
// Weapons, FacingAngle and Nippers are filled in by core.NewWorld.
func defaultPlayer() core.Player {
	return core.Player{
		HP: 100, MaxHP: 100,
		Speed: 3, Radius: 36, // collision matches the tall tank sprite
		Level:    1,
		XPToNext: 10,
	}
}

// defaultPickupRanges returns the standard gem/nipper pickup behaviour.
func defaultPickupRanges() core.PickupRanges {
	return core.PickupRanges{PickupDist: 28, MagnetDist: 90, MagnetSpeed: 4}
}
