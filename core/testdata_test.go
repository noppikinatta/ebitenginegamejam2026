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
		StartingNippers:        3,
		MaxTurretTiles:         40,
		CandlestickInterval:    600,
		XPToNextGrowth:         1.25,
		CapacitorFireRateBonus: 0.1,
		Player:                 Player{HP: 100, MaxHP: 100, Speed: 3, Radius: 36, Level: 1, XPToNext: 10},
		Pickup:                 PickupRanges{PickupDist: 28, MagnetDist: 90, MagnetSpeed: 4},
		Spawn:                  SpawnSpec{EnemyDist: 520, EnemyBaseInterval: 60, EnemyMinInterval: 18, EnemyIntervalDecay: 600, CandleDist: 220, CandleDistRange: 220},
		Doctor:                 DoctorSpec{NipperChance: 0.25, UpgradeChance: 0.625, NipperMin: 5, NipperMax: 10, MaxUpgrades: 3, MaxBundleTiles: 3, CapacitorChance: 0.15},
		EnemyScaling:           EnemyScaling{HPBase: 10, HPDoublingTicks: 18000, Speed: 1.2, Radius: 16, Damage: 8, XPValue: 3},
		Candlestick:            Enemy{HP: 40, Radius: 16, DropsNipper: true},
		TurretGen:              testTurretGenConfig(),
		Weapons:                testWeapons(),
		PowerCurve:             testPowerCurve(),
	}
}

// testPowerCurve mirrors data.powerCurve() so core tests share the production
// tile-count → fire-rate breakpoints.
func testPowerCurve() []PowerPoint {
	return []PowerPoint{
		{Tiles: 10, Mult: 4.0},
		{Tiles: 32, Mult: 1.0},
		{Tiles: 40, Mult: 0.5},
	}
}

func testWeapons() map[WeaponKind]WeaponParams {
	m := make(map[WeaponKind]WeaponParams, 6)
	for _, k := range []WeaponKind{KindCannon, KindShotgun, KindSniper, KindLaser, KindGatling, KindGrenade} {
		m[k] = testParams(k)
	}
	return m
}

// testParams returns the balance preset for a weapon kind, matching the
// data package presets.
func testParams(kind WeaponKind) WeaponParams {
	switch kind {
	case KindShotgun:
		return WeaponParams{BaseDamage: 8, BaseInterval: 720, MinInterval: 8, ProjSpeed: 5, ProjMaxDist: 150, ProjRadius: 2, BaseRange: 100, Pellets: 4, SpreadRad: 0.3, LevelMult: 1.2}
	case KindSniper:
		return WeaponParams{BaseDamage: 10, BaseInterval: 960, MinInterval: 20, ProjSpeed: 10, ProjMaxDist: 640, ProjRadius: 2, BaseRange: 360, LevelMult: 1.2}
	case KindLaser:
		return WeaponParams{BaseDamage: 1, BaseInterval: 1440, MinInterval: 15, BaseRange: 200, BeamBaseLength: 200, BeamBaseWidth: 6, BeamBaseDuration: 30, LevelMult: 1.2}
	case KindGatling:
		return WeaponParams{BaseDamage: 2, BaseInterval: 480, MinInterval: 6, ProjSpeed: 5, ProjMaxDist: 240, ProjRadius: 2, Pellets: 10, SpreadRad: 0.2, SpreadRandom: true, BurstGap: 3, Aim: AimForward, LevelMult: 1.2}
	case KindGrenade:
		return WeaponParams{BaseDamage: 0, BaseInterval: 1800, MinInterval: 30, ProjSpeed: 2, ProjMaxDist: 120, Aim: AimOutward, ExplodeRadius: 64, ExplodeDamage: 15, LevelMult: 1.2}
	default: // KindCannon
		return WeaponParams{BaseDamage: 20, BaseInterval: 720, MinInterval: 6, ProjSpeed: 6, ProjMaxDist: 260, ProjRadius: 6, BaseRange: 200, LevelMult: 1.2}
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
