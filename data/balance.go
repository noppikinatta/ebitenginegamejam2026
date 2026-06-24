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
		HeartDropChance:        0.1,  // 10% of candlestick drops are a heart (HP) instead of a nipper
		HeartHeal:              30,   // HP restored when a heart is collected
		RepairInterval:         300,  // ticks between Repair Unit heal cycles (~5 s at 60 TPS)
		RepairHealAmount:       1,    // HP healed per connected Repair Unit each cycle
		ArmorReduction:         1,    // damage subtracted per connected Armor (min 1 still lands)
		XPToNextGrowth:         1.25, // XPToNext multiplier applied each level-up
		CapacitorFireRateBonus: 0.1,  // each connected Capacitor adds +0.1 to the fire-rate multiplier
		DamageMult:             1,    // baseline weapon damage multiplier (meta upgrades raise it via ApplyMeta)

		Player:          defaultPlayer(),
		Pickup:          defaultPickupRanges(),
		Spawn:           defaultSpawn(),
		Doctor:          defaultDoctor(),
		EnemyKinds:      enemyKinds(),
		HPDoublingTicks: 18000, // zako HP doubles every 5 min at 60 TPS (×2 @5min, ×4 @10min)
		SpawnPhases:     spawnPhases(),
		Bosses:          bosses(),
		Candlestick:     candlestick(),
		TurretGen:       defaultTurretGen(),
		Weapons:         weaponParams(),
		PowerCurve:      powerCurve(),
	}
}

// defaultPlayer returns balanced starting stats for the player tank. Pos,
// Weapons, FacingAngle and Nippers are filled in by core.NewWorld.
func defaultPlayer() core.Player {
	return core.Player{
		HP: 100, MaxHP: 100,
		// Speed is a coefficient: effective px/tick = Speed × turret power multiplier.
		Speed: 1.5, Radius: 36, // Radius matches the tall tank sprite for collision
		Level:    1,
		XPToNext: 10,
	}
}

// defaultPickupRanges returns the standard gem/nipper pickup behaviour.
func defaultPickupRanges() core.PickupRanges {
	return core.PickupRanges{PickupDist: 28, MagnetDist: 90, MagnetSpeed: 4}
}
