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
//
// Turn caps how fast an enemy can re-aim toward the player each tick (a steering
// force in px/tick²): 0 = instant follow (re-aim every tick, the original
// behaviour); a small positive value makes the enemy bank into wide curves so it
// can be out-maneuvered. As a rule of thumb, Turn >= 2×Speed already turns about
// as tightly as instant follow, so meaningful arcs come from Turn well below
// Speed (e.g. 0.05–0.3 here).
func enemyKinds() map[core.EnemyKind]core.EnemyStats {
	return map[core.EnemyKind]core.EnemyStats{
		// Grunt: the balanced staple, one at a time.
		core.EnemyGrunt: {
			HPBase: 10, Speed: 1.2, Turn: 0, Radius: 16, Damage: 8, XPValue: 3,
			PackMin: 1, PackMax: 1,
		},
		// Swarmer: fast and fragile, arrives in packs to pressure positioning.
		core.EnemySwarmer: {
			HPBase: 5, Speed: 2.1, Turn: 0, Radius: 11, Damage: 4, XPValue: 1,
			PackMin: 3, PackMax: 6,
		},
		// Brute: slow wall of HP that hits hard; forces the player to kite.
		core.EnemyBrute: {
			HPBase: 60, Speed: 0.7, Turn: 0, Radius: 26, Damage: 18, XPValue: 8,
			PackMin: 1, PackMax: 1,
		},
	}
}

// spawnPhases defines the time bands: each sets the spawn Interval (cadence) and
// the per-kind weights for its stretch of the run. Edit these to retune how
// enemies appear over time — the bands escalate by shortening the interval and
// shifting the mix toward swarmers and brutes. The last band (math.MaxInt)
// covers the rest of the run.
func spawnPhases() []core.SpawnPhase {
	min := func(m float64) int {
		return int(m * 60 * 60)
	}

	return []core.SpawnPhase{
		{UntilTick: min(0.5), Interval: 300, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 10},
		}},
		{UntilTick: min(1), Interval: 300, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 9},
			{Kind: core.EnemySwarmer, Weight: 1},
		}},
		{UntilTick: min(2), Interval: 250, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 7},
			{Kind: core.EnemySwarmer, Weight: 3},
		}},
		{UntilTick: min(3), Interval: 200, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 6},
			{Kind: core.EnemySwarmer, Weight: 4},
		}},
		{UntilTick: min(4), Interval: 300, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 6},
			{Kind: core.EnemySwarmer, Weight: 4},
		}},
		{UntilTick: min(5), Interval: 150, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 8},
			{Kind: core.EnemySwarmer, Weight: 2},
		}},
		{UntilTick: min(6), Interval: 200, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 7},
			{Kind: core.EnemySwarmer, Weight: 2},
			{Kind: core.EnemyBrute, Weight: 1},
		}},
		{UntilTick: min(7), Interval: 400, Weights: []core.KindWeight{
			{Kind: core.EnemyBrute, Weight: 10},
		}},
		{UntilTick: min(8), Interval: 30, Weights: []core.KindWeight{
			{Kind: core.EnemySwarmer, Weight: 9},
			{Kind: core.EnemyBrute, Weight: 1},
		}},
		{UntilTick: min(9), Interval: 100, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 7},
			{Kind: core.EnemyBrute, Weight: 3},
		}},
		{UntilTick: math.MaxInt, Interval: 50, Weights: []core.KindWeight{
			{Kind: core.EnemyGrunt, Weight: 3},
			{Kind: core.EnemySwarmer, Weight: 3},
			{Kind: core.EnemyBrute, Weight: 4},
		}},
	}
}

// bosses returns the three scheduled bosses. The 3- and 6-minute bosses are
// checkpoints; defeating the 10-minute boss (Final) clears the run. HP is fixed
// (no time scaling). Tune during playtesting.
func bosses() []core.BossSpec {
	return []core.BossSpec{
		// Turn caps per-tick re-aim (see enemyKinds); 0 keeps bosses on instant follow.
		{AtTick: min3, Name: "Prototype Hauler", HP: 1200, Speed: 0.9, Turn: 0, Radius: 40, Damage: 20, XPValue: 50, Sprite: core.BossSprite1},
		{AtTick: min6, Name: "Siege Engine", HP: 3000, Speed: 0.85, Turn: 0, Radius: 46, Damage: 26, XPValue: 100, Sprite: core.BossSprite2},
		{AtTick: min10, Name: "The Disconnector", HP: 8000, Speed: 0.8, Turn: 0, Radius: 54, Damage: 32, XPValue: 200, Final: true, Sprite: core.BossSprite3},
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

// defaultSpawn returns the enemy/candlestick placement parameters. Spawn cadence
// and enemy mix are time-banded in spawnPhases.
func defaultSpawn() core.SpawnSpec {
	return core.SpawnSpec{
		EnemyDist:       520,
		CandleDist:      220,
		CandleDistRange: 220,
	}
}
