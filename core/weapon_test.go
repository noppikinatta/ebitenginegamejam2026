package core

import (
	"math"
	"testing"
)

// TestStats_FireIntervalScalesWithMultiplier: the fire interval is the base
// interval divided by the fire-rate multiplier (rounded), clamped to MinInterval.
// Damage and range do not depend on the multiplier.
func TestStats_FireIntervalScalesWithMultiplier(t *testing.T) {
	p := testParams(KindCannon) // BaseInterval 45, MinInterval 6, BaseDamage 5, BaseRange 220

	tests := []struct {
		name         string
		fireMult     float64
		wantInterval int
	}{
		{"1x is base", 1, 45},
		{"3x divides", 3, 15},     // 45/3 = 15
		{"4x rounds", 4, 11},      // 45/4 = 11.25 -> 11
		{"high mult clamps", 10, 6}, // 45/10 = 4.5 -> 5, clamped to MinInterval 6
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWeapon("Cannon", KindCannon)
			stats := w.Stats(p, tc.fireMult)
			if stats.FireInterval != tc.wantInterval {
				t.Errorf("FireInterval = %d, want %d", stats.FireInterval, tc.wantInterval)
			}
			if stats.Damage != p.BaseDamage {
				t.Errorf("Damage = %.2f, want %.2f (mult must not affect damage)", stats.Damage, p.BaseDamage)
			}
			if stats.Range != p.BaseRange {
				t.Errorf("Range = %.2f, want %.2f (mult must not affect range)", stats.Range, p.BaseRange)
			}
		})
	}
}

// TestStats_NonPositiveMultiplierTreatedAsOne guards against divide-by-zero or
// negative multipliers producing nonsense intervals.
func TestStats_NonPositiveMultiplierTreatedAsOne(t *testing.T) {
	p := testParams(KindCannon)
	w := NewWeapon("Cannon", KindCannon)
	if got := w.Stats(p, 0).FireInterval; got != 45 {
		t.Errorf("zero mult: FireInterval = %d, want 45 (treated as 1x)", got)
	}
	if got := w.Stats(p, -2).FireInterval; got != 45 {
		t.Errorf("negative mult: FireInterval = %d, want 45 (treated as 1x)", got)
	}
}

// TestStats_LevelScalesDamageOnly: doctor upgrade Level multiplies damage by
// LevelMult^Level and leaves the fire interval untouched.
func TestStats_LevelScalesDamageOnly(t *testing.T) {
	p := testParams(KindCannon) // BaseDamage 5, LevelMult 1.2
	w := NewWeapon("Cannon", KindCannon)
	w.Level = 2

	stats := w.Stats(p, 1)
	want := 5 * math.Pow(1.2, 2)
	if math.Abs(stats.Damage-want) > eps {
		t.Errorf("Damage = %.4f, want %.4f", stats.Damage, want)
	}
	if stats.FireInterval != 45 {
		t.Errorf("FireInterval = %d, want 45 (level must not affect interval)", stats.FireInterval)
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
