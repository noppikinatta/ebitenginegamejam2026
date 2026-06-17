package data

// EnemySpec contains construction parameters for one enemy variant.
// All zero-value fields (Speed, Damage, XPValue) mean "none" for special enemies.
type EnemySpec struct {
	HP, Speed, Radius, Damage, XPValue float64
	DropsNipper                         bool
}

// BasicEnemy returns the spec for the standard chasing enemy at the given
// game tick. HP scales linearly so enemies get tankier as time passes.
func BasicEnemy(tick int) EnemySpec {
	return EnemySpec{
		HP:      10 + float64(tick)/120.0,
		Speed:   1.2,
		Radius:  12,
		Damage:  8,
		XPValue: 3,
	}
}

// Candlestick returns the spec for the stationary harmless destructible that
// drops a nipper pickup when broken.
func Candlestick() EnemySpec {
	return EnemySpec{
		HP:          40,
		Radius:      13,
		DropsNipper: true,
		// Speed=0, Damage=0, XPValue=0 — set by Go zero value
	}
}

// SpawnSpec controls enemy and candlestick spawn placement and timing.
type SpawnSpec struct {
	EnemyDist          float64 // fixed spawn distance from player
	EnemyBaseInterval  int     // ticks between spawns at tick 0
	EnemyMinInterval   int     // floor for spawn interval (faster cap)
	EnemyIntervalDecay int     // game ticks that reduce spawn interval by 1
	CandleDist         float64 // minimum candlestick distance from player
	CandleDistRange    float64 // random extra distance beyond CandleDist
}

// DefaultSpawn returns the standard enemy and candlestick spawn parameters.
func DefaultSpawn() SpawnSpec {
	return SpawnSpec{
		EnemyDist:          520,
		EnemyBaseInterval:  60,
		EnemyMinInterval:   18,
		EnemyIntervalDecay: 600,
		CandleDist:         220,
		CandleDistRange:    220,
	}
}
