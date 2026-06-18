package core

import (
	"math"
	"testing"
)

// TestFireIncrement_ScalesAndClamps: the accumulator advances by the fire-rate
// multiplier each tick, capped to BaseInterval/MinInterval so the effective
// interval never drops below MinInterval. Non-positive multipliers yield 0.
func TestFireIncrement_ScalesAndClamps(t *testing.T) {
	p := testParams(KindCannon) // BaseInterval 45, MinInterval 6 -> maxInc 7.5

	tests := []struct {
		name     string
		fireMult float64
		want     float64
	}{
		{"1x", 1, 1},
		{"2.5x", 2.5, 2.5},
		{"caps at BaseInterval/MinInterval", 10, 7.5}, // 45/6
		{"zero -> 0", 0, 0},
		{"negative -> 0", -2, 0},
	}
	for _, tc := range tests {
		if got := fireIncrement(p, tc.fireMult); math.Abs(got-tc.want) > eps {
			t.Errorf("%s: fireIncrement = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestStats_DamageAndRange: damage and range come straight from the params (no
// fire-rate coupling); the fire cadence lives in the accumulator, not Stats.
func TestStats_DamageAndRange(t *testing.T) {
	p := testParams(KindCannon)
	stats := NewWeapon("Cannon", KindCannon).Stats(p)
	if stats.Damage != p.BaseDamage {
		t.Errorf("Damage = %.2f, want %.2f", stats.Damage, p.BaseDamage)
	}
	if stats.Range != p.BaseRange {
		t.Errorf("Range = %.2f, want %.2f", stats.Range, p.BaseRange)
	}
}

// TestStats_ProjectileLifeFromMaxDist: projectile lifetime is derived from the
// data-driven max travel distance (ProjLife = round(ProjMaxDist/ProjSpeed)), and
// the collision radius passes through.
func TestStats_ProjectileLifeFromMaxDist(t *testing.T) {
	p := testParams(KindCannon) // ProjSpeed 6, ProjMaxDist 260, ProjRadius 5
	stats := NewWeapon("Cannon", KindCannon).Stats(p)
	if stats.ProjLife != 43 { // round(260/6) = 43
		t.Errorf("ProjLife = %d, want 43", stats.ProjLife)
	}
	if stats.ProjRadius != 5 {
		t.Errorf("ProjRadius = %v, want 5", stats.ProjRadius)
	}
}

// TestStats_LevelScalesDamage: doctor upgrade Level multiplies damage by
// LevelMult^Level.
func TestStats_LevelScalesDamage(t *testing.T) {
	p := testParams(KindCannon) // BaseDamage 5, LevelMult 1.2
	w := NewWeapon("Cannon", KindCannon)
	w.Level = 2

	stats := w.Stats(p)
	want := 5 * math.Pow(1.2, 2)
	if math.Abs(stats.Damage-want) > eps {
		t.Errorf("Damage = %.4f, want %.4f", stats.Damage, want)
	}
}

// TestPowerMultiplier_Interpolation covers clamping below/above the curve and
// linear interpolation between breakpoints.
func TestPowerMultiplier_Interpolation(t *testing.T) {
	curve := testPowerCurve() // {10,4.0} {32,1.0} {40,0.5}

	tests := []struct {
		tiles int
		want  float64
	}{
		{5, 4.0},   // below first point clamps to first Mult
		{10, 4.0},  // exactly first point
		{21, 2.5},  // 4 + (1-4)*((21-10)/(32-10)) = 2.5
		{32, 1.0},  // exactly middle point
		{36, 0.75}, // 1 + (0.5-1)*((36-32)/(40-32)) = 0.75
		{40, 0.5},  // exactly last point
		{50, 0.5},  // above last point clamps to last Mult
	}
	for _, tc := range tests {
		if got := PowerMultiplier(curve, tc.tiles); math.Abs(got-tc.want) > eps {
			t.Errorf("PowerMultiplier(%d) = %.4f, want %.4f", tc.tiles, got, tc.want)
		}
	}

	if got := PowerMultiplier(nil, 20); got != 1 {
		t.Errorf("empty curve: PowerMultiplier = %.4f, want 1", got)
	}
}
