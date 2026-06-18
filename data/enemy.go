package data

import (
	"math"

	"github.com/noppikinatta/ebitenginegamejam2026/core"
)

// Minute markers in ticks (60 TPS) for the spawn director and boss schedule.
const (
	min3  = 3 * 60 * 60  // 10800
	min6  = 6 * 60 * 60  // 21600
	min10 = 10 * 60 * 60 // 36000
)

// enemyKinds returns the zako (trash) enemy spawn templates. HP scales with time
// via Config.HPDoublingTicks; the rest are constant. Tune during playtesting.
func enemyKinds() map[core.EnemyKind]core.EnemyStats {
	return map[core.EnemyKind]core.EnemyStats{
		// Grunt: the balanced staple, one at a time.
		core.EnemyGrunt: {
			HPBase: 10, Speed: 1.2, Radius: 16, Damage: 8, XPValue: 3,
			PackMin: 1, PackMax: 1,
		},
		// Swarmer: fast and fragile, arrives in packs to pressure positioning.
		core.EnemySwarmer: {
			HPBase: 5, Speed: 2.1, Radius: 11, Damage: 4, XPValue: 1,
			PackMin: 3, PackMax: 6,
		},
		// Brute: slow wall of HP that hits hard; forces the player to kite.
		core.EnemyBrute: {
			HPBase: 60, Speed: 0.7, Radius: 26, Damage: 18, XPValue: 8,
			PackMin: 1, PackMax: 1,
		},
	}
}

// spawnPhases returns the time-gated spawn weights. Early on it's mostly grunts
// with a few swarmers; brutes join after the first boss; the mix thickens for
// the final stretch. The last phase covers the rest of the run.
func spawnPhases() []core.SpawnPhase {
	return []core.SpawnPhase{
		{UntilTick: min3, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 7},
			{Kind: core.EnemySwarmer, Weight: 3},
		}},
		{UntilTick: min6, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 5},
			{Kind: core.EnemySwarmer, Weight: 4},
			{Kind: core.EnemyBrute, Weight: 1},
		}},
		{UntilTick: math.MaxInt, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 4},
			{Kind: core.EnemySwarmer, Weight: 4},
			{Kind: core.EnemyBrute, Weight: 2},
		}},
	}
}

// bosses returns the three scheduled bosses. The 3- and 6-minute bosses are
// checkpoints; defeating the 10-minute boss (Final) clears the run. HP is fixed
// (no time scaling). Tune during playtesting.
func bosses() []core.BossSpec {
	return []core.BossSpec{
		{AtTick: min3, Name: "Prototype Hauler", HP: 1200, Speed: 0.9, Radius: 40, Damage: 20, XPValue: 50},
		{AtTick: min6, Name: "Siege Engine", HP: 3000, Speed: 0.85, Radius: 46, Damage: 26, XPValue: 100},
		{AtTick: min10, Name: "The Disconnector", HP: 8000, Speed: 0.8, Radius: 54, Damage: 32, XPValue: 200, Final: true},
	}
}

// candlestick returns the template for the stationary harmless destructible that
// drops a nipper pickup when broken. Pos and alive are set on spawn;
// Speed/Damage/XPValue stay zero (stationary, harmless, no XP).
func candlestick() core.Enemy {
	return core.Enemy{
		HP:          40,
		Radius:      16, // matches the 32x32 candlestick sprite
		DropsNipper: true,
	}
}

// defaultSpawn returns the enemy/candlestick placement and timing parameters.
func defaultSpawn() core.SpawnSpec {
	return core.SpawnSpec{
		EnemyDist:          520,
		EnemyBaseInterval:  60,
		EnemyMinInterval:   18,
		EnemyIntervalDecay: 600,
		CandleDist:         220,
		CandleDistRange:    220,
	}
}
