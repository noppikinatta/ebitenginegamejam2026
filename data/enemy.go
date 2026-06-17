package data

import "github.com/noppikinatta/ebitenginegamejam2026/core"

// basicEnemyScaling returns the standard chasing enemy's stats. HP scales with
// the game tick (HP = HPBase + tick×HPPerTick); core applies the formula.
func basicEnemyScaling() core.EnemyScaling {
	return core.EnemyScaling{
		HPBase:    10,
		HPPerTick: 1.0 / 120.0,
		Speed:     1.2,
		Radius:    16, // matches the 32x32 enemy sprite
		Damage:    8,
		XPValue:   3,
	}
}

// candlestick returns the template for the stationary harmless destructible
// that drops a nipper pickup when broken. Pos and alive are set on spawn;
// Speed/Damage/XPValue stay at their zero values (stationary, harmless, no XP).
func candlestick() core.Enemy {
	return core.Enemy{
		HP:          40,
		Radius:      16, // matches the 32x32 candlestick sprite
		DropsNipper: true,
	}
}

// defaultSpawn returns the standard enemy and candlestick spawn parameters.
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
