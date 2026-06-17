package core

import (
	"math/rand"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// startingNippers mirrors the data package's StartingNippers for tests that
// assert on the initial nipper count without importing data.
const startingNippers = 3

// testConfig returns a Config whose numbers match data.NewConfig(), so the
// numeric expectations baked into the core tests stay valid. Production core
// holds no balance numbers; these fixtures live in test code only.
func testConfig() Config {
	return Config{
		StartingNippers:     3,
		MaxTurretTiles:      40,
		CandlestickInterval: 600,
		XPToNextGrowth:      1.25,
		Player:              Player{HP: 100, MaxHP: 100, Speed: 3, Radius: 36, Level: 1, XPToNext: 10},
		Pickup:              PickupRanges{PickupDist: 28, MagnetDist: 90, MagnetSpeed: 4},
		Spawn:               SpawnSpec{EnemyDist: 520, EnemyBaseInterval: 60, EnemyMinInterval: 18, EnemyIntervalDecay: 600, CandleDist: 220, CandleDistRange: 220},
		Doctor:              DoctorSpec{NipperChance: 0.25, UpgradeChance: 0.625, NipperMin: 5, NipperMax: 10, MaxUpgrades: 3, MaxBundleTiles: 3},
		EnemyScaling:        EnemyScaling{HPBase: 10, HPPerTick: 1.0 / 120.0, Speed: 1.2, Radius: 16, Damage: 8, XPValue: 3},
		Candlestick:         Enemy{HP: 40, Radius: 16, DropsNipper: true},
		TurretGen:           testTurretGenConfig(),
		Weapons:             testWeapons(),
	}
}

func testWeapons() map[WeaponKind]WeaponParams {
	m := make(map[WeaponKind]WeaponParams, 4)
	for _, k := range []WeaponKind{KindCannon, KindShotgun, KindSniper, KindLaser} {
		m[k] = testParams(k)
	}
	return m
}

// testParams returns the balance preset for a weapon kind, matching the
// data package presets.
func testParams(kind WeaponKind) WeaponParams {
	switch kind {
	case KindShotgun:
		return WeaponParams{BaseDamage: 3, EnergyDamage: 1.5, BaseInterval: 28, EnergyInterval: 2, MinInterval: 8, ProjSpeed: 5, BaseRange: 150, EnergyRange: 10, LevelMult: 1.2}
	case KindSniper:
		return WeaponParams{BaseDamage: 20, EnergyDamage: 8, BaseInterval: 120, EnergyInterval: 7, MinInterval: 20, ProjSpeed: 10, BaseRange: 400, EnergyRange: 40, LevelMult: 1.2}
	case KindLaser:
		return WeaponParams{BaseDamage: 2, EnergyDamage: 0.8, BaseInterval: 90, EnergyInterval: 5, MinInterval: 15, BaseRange: 300, EnergyRange: 25, BeamBaseLength: 300, BeamEnergyLength: 25, BeamBaseWidth: 6, BeamEnergyWidth: 0.5, BeamBaseDuration: 30, BeamEnergyDuration: 4, LevelMult: 1.2}
	default: // KindCannon
		return WeaponParams{BaseDamage: 5, EnergyDamage: 3, BaseInterval: 45, EnergyInterval: 4, MinInterval: 6, ProjSpeed: 6, BaseRange: 220, EnergyRange: 20, LevelMult: 1.2}
	}
}

func testTurretGenConfig() TurretGenConfig {
	return TurretGenConfig{
		MaxTiles:      22,
		BranchProb:    0.35,
		WeaponDensity: 0.45,
		JunkDensity:   0.15,
		Generators:    []GeneratorConfig{{Index: hexmap.IdxXY(0, 0), Power: 100}},
	}
}

// DefaultTurretGenConfig is the test-only entry point the turret-generation
// tests use. rng is unused (the single generator sits at a fixed origin) but
// kept so existing call sites compile unchanged.
func DefaultTurretGenConfig(rng *rand.Rand) TurretGenConfig {
	_ = rng
	return testTurretGenConfig()
}
