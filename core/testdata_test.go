package core

import (
	"math"
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
		HeartDropChance:        0.1,
		HeartHeal:              30,
		RepairInterval:         300,
		RepairHealAmount:       1,
		ArmorReduction:         1,
		XPToNextGrowth:         1.25,
		CapacitorFireRateBonus: 0.1,
		Player:                 Player{HP: 100, MaxHP: 100, Speed: 3, Radius: 36, Level: 1, XPToNext: 10},
		Pickup:                 PickupRanges{PickupDist: 28, MagnetDist: 90, MagnetSpeed: 4},
		Spawn:                  SpawnSpec{EnemyDist: 520, CandleDist: 220, CandleDistRange: 220},
		Doctor:                 DoctorSpec{NipperWeight: 0.25, WeaponAddWeight: 0.25, WeaponUpgradeWeight: 0.10, JunkWeight: 0.40, NipperMin: 5, NipperMax: 10, MaxItems: 3},
		EnemyKinds:             testEnemyKinds(),
		HPDoublingTicks:        18000,
		SpawnPhases:            testSpawnPhases(),
		Bosses:                 testBosses(),
		Candlestick:            Enemy{HP: 40, Radius: 16, DropsNipper: true},
		TurretGen:              testTurretGenConfig(),
		Weapons:                testWeapons(),
		PowerCurve:             testPowerCurve(),
	}
}

// testEnemyKinds / testSpawnPhases / testBosses mirror the data package so core
// tests exercise the real spawn director and boss schedule.
func testEnemyKinds() map[EnemyKind]EnemyStats {
	return map[EnemyKind]EnemyStats{
		EnemyGrunt:   {HPBase: 10, Speed: 1.2, Radius: 16, Damage: 8, XPValue: 3, PackMin: 1, PackMax: 1},
		EnemySwarmer: {HPBase: 5, Speed: 2.1, Radius: 11, Damage: 4, XPValue: 1, PackMin: 3, PackMax: 6},
		EnemyBrute:   {HPBase: 60, Speed: 0.7, Radius: 26, Damage: 18, XPValue: 8, PackMin: 1, PackMax: 1},
	}
}

func testSpawnPhases() []SpawnPhase {
	const min1_5, min3, min6, min8 = 90 * 60, 3 * 3600, 6 * 3600, 8 * 3600
	return []SpawnPhase{
		{UntilTick: min1_5, Interval: 70, Weights: []KindWeight{{EnemyGrunt, 8}, {EnemySwarmer, 2}}},
		{UntilTick: min3, Interval: 54, Weights: []KindWeight{{EnemyGrunt, 6}, {EnemySwarmer, 4}}},
		{UntilTick: min6, Interval: 44, Weights: []KindWeight{{EnemyGrunt, 5}, {EnemySwarmer, 4}, {EnemyBrute, 1}}},
		{UntilTick: min8, Interval: 34, Weights: []KindWeight{{EnemyGrunt, 4}, {EnemySwarmer, 5}, {EnemyBrute, 2}}},
		{UntilTick: math.MaxInt, Interval: 26, Weights: []KindWeight{{EnemyGrunt, 3}, {EnemySwarmer, 5}, {EnemyBrute, 3}}},
	}
}

func testBosses() []BossSpec {
	return []BossSpec{
		{AtTick: 3 * 3600, Name: "Prototype Hauler", HP: 1200, Speed: 0.9, Radius: 40, Damage: 20, XPValue: 50},
		{AtTick: 6 * 3600, Name: "Siege Engine", HP: 3000, Speed: 0.85, Radius: 46, Damage: 26, XPValue: 100},
		{AtTick: 10 * 3600, Name: "The Disconnector", HP: 8000, Speed: 0.8, Radius: 54, Damage: 32, XPValue: 200, Final: true},
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
	m := make(map[WeaponKind]WeaponParams, 8)
	for _, k := range []WeaponKind{KindCannon, KindShotgun, KindSniper, KindLaser, KindGatling, KindGrenade, KindCIWS, KindMissile} {
		m[k] = testParams(k)
	}
	return m
}

// testParams returns the balance preset for a weapon kind, matching the
// data package presets.
func testParams(kind WeaponKind) WeaponParams {
	switch kind {
	case KindShotgun:
		return WeaponParams{BaseDamage: 8, BaseInterval: 720, MinInterval: 8, ProjSpeed: 5, ProjMaxDist: 150, ProjRadius: 2, BaseRange: 100, Pellets: 4, SpreadRad: 0.3, Sprite: SpriteShotgun, ProjDrawW: 6, ProjDrawH: 6, LevelMult: 1.2}
	case KindSniper:
		return WeaponParams{BaseDamage: 10, BaseInterval: 960, MinInterval: 20, ProjSpeed: 10, ProjMaxDist: 640, ProjRadius: 2, BaseRange: 360, Target: TargetFarthest, Sprite: SpriteSniper, ProjDrawW: 4, ProjDrawH: 16, ProjFaceVelocity: true, LevelMult: 1.2}
	case KindLaser:
		return WeaponParams{BaseDamage: 1, BaseInterval: 1440, MinInterval: 15, BaseRange: 200, Target: TargetFarthest, BeamBaseLength: 200, BeamBaseWidth: 6, BeamBaseDuration: 30, LevelMult: 1.2}
	case KindGatling:
		return WeaponParams{BaseDamage: 2, BaseInterval: 480, MinInterval: 6, ProjSpeed: 5, ProjMaxDist: 240, ProjRadius: 2, Pellets: 10, SpreadRad: 0.2, SpreadRandom: true, BurstGap: 3, Aim: AimForward, Sprite: SpriteGatling, ProjDrawW: 6, ProjDrawH: 6, LevelMult: 1.2}
	case KindGrenade:
		return WeaponParams{BaseDamage: 0, BaseInterval: 1800, MinInterval: 30, ProjSpeed: 2, ProjMaxDist: 120, Aim: AimOutward, ExplodeRadius: 64, ExplodeDamage: 15, PassThrough: true, Sprite: SpriteGrenade, ProjDrawW: 14, ProjDrawH: 14, LevelMult: 1.2}
	case KindCIWS:
		return WeaponParams{BaseDamage: 2, BaseInterval: 480, MinInterval: 6, ProjSpeed: 5, ProjMaxDist: 120, ProjRadius: 2, BaseRange: 80, Pellets: 10, SpreadRad: 0.1, SpreadRandom: true, BurstGap: 2, HoldWhenNoTarget: true, Sprite: SpriteCIWS, ProjDrawW: 6, ProjDrawH: 6, LevelMult: 1.2}
	case KindMissile:
		return WeaponParams{BaseDamage: 8, BaseInterval: 960, MinInterval: 20, ProjSpeed: 2, ProjMaxDist: 240, ProjRadius: 6, BaseRange: 240, ExplodeRadius: 48, ExplodeDamage: 10, Mover: NewHomingMover(0.3, 6, 15), Sprite: SpriteMissile, ProjDrawW: 8, ProjDrawH: 12, ProjFaceVelocity: true, LevelMult: 1.2}
	default: // KindCannon
		return WeaponParams{BaseDamage: 20, BaseInterval: 720, MinInterval: 6, ProjSpeed: 6, ProjMaxDist: 260, ProjRadius: 6, BaseRange: 200, Sprite: SpriteCannon, ProjDrawW: 8, ProjDrawH: 14, ProjFaceVelocity: true, LevelMult: 1.2}
	}
}

func testTurretGenConfig() TurretGenConfig {
	return TurretGenConfig{
		WeaponCount: 3,
		JunkCount:   29,
		BranchProb:  0.35,
		Generators:  []GeneratorConfig{{Index: hexmap.IdxXY(0, 0), Power: 100}},
	}
}

// DefaultTurretGenConfig is the test-only entry point the turret-generation
// tests use. rng is unused (the single generator sits at a fixed origin) but
// kept so existing call sites compile unchanged.
func DefaultTurretGenConfig(rng *rand.Rand) TurretGenConfig {
	_ = rng
	return testTurretGenConfig()
}
