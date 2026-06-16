package core

import (
	"testing"
)

func TestStatsFromEnergy_TableDriven(t *testing.T) {
	tests := []struct {
		name              string
		energy            float64
		wantDamageMin     float64 // Damage must be >= this
		wantRangeMin      float64 // Range must be >= this
		wantIntervalMax   int     // FireInterval must be <= this
		wantIntervalFloor int     // FireInterval must be >= this (clamp floor)
	}{
		{
			name:              "zero energy",
			energy:            0,
			wantDamageMin:     5,   // 5 + 0*3
			wantRangeMin:      220, // 220 + 0*20
			wantIntervalMax:   45,  // 45 - 0*4
			wantIntervalFloor: 6,
		},
		{
			name:              "small energy (3)",
			energy:            3,
			wantDamageMin:     14,  // 5 + 3*3
			wantRangeMin:      280, // 220 + 3*20
			wantIntervalMax:   33,  // 45 - 3*4
			wantIntervalFloor: 6,
		},
		{
			name:              "medium energy (9)",
			energy:            9,
			wantDamageMin:     32,  // 5 + 9*3
			wantRangeMin:      400, // 220 + 9*20
			wantIntervalMax:   9,   // 45 - 9*4 = 9
			wantIntervalFloor: 6,
		},
		{
			name:              "high energy hits fire-interval floor",
			energy:            20,
			wantDamageMin:     65,  // 5 + 20*3
			wantRangeMin:      620, // 220 + 20*20
			wantIntervalMax:   6,   // clamped to 6
			wantIntervalFloor: 6,
		},
		{
			name:              "very high energy still clamped",
			energy:            100,
			wantDamageMin:     305,  // 5 + 100*3
			wantRangeMin:      2220, // 220 + 100*20
			wantIntervalMax:   6,    // clamped to 6
			wantIntervalFloor: 6,
		},
	}

	// Also verify that damage and range strictly increase with energy.
	energyLevels := []float64{0, 1, 2, 5, 10, 20}
	for i := 1; i < len(energyLevels); i++ {
		lo := NewWeapon("w", energyLevels[i-1], KindCannon).StatsFromEnergy()
		hi := NewWeapon("w", energyLevels[i], KindCannon).StatsFromEnergy()
		if hi.Damage <= lo.Damage {
			t.Errorf("Damage should increase: energy %.0f => %.2f, energy %.0f => %.2f",
				energyLevels[i-1], lo.Damage, energyLevels[i], hi.Damage)
		}
		if hi.Range <= lo.Range {
			t.Errorf("Range should increase: energy %.0f => %.2f, energy %.0f => %.2f",
				energyLevels[i-1], lo.Range, energyLevels[i], hi.Range)
		}
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWeapon("Cannon", tc.energy, KindCannon)
			stats := w.StatsFromEnergy()

			if stats.Damage < tc.wantDamageMin {
				t.Errorf("Damage = %.2f, want >= %.2f", stats.Damage, tc.wantDamageMin)
			}
			if stats.Range < tc.wantRangeMin {
				t.Errorf("Range = %.2f, want >= %.2f", stats.Range, tc.wantRangeMin)
			}
			if stats.FireInterval > tc.wantIntervalMax {
				t.Errorf("FireInterval = %d, want <= %d", stats.FireInterval, tc.wantIntervalMax)
			}
			if stats.FireInterval < tc.wantIntervalFloor {
				t.Errorf("FireInterval = %d, want >= %d (floor)", stats.FireInterval, tc.wantIntervalFloor)
			}
		})
	}
}

func TestStatsFromEnergy_NegativeEnergyTreatedAsZero(t *testing.T) {
	wNeg := NewWeapon("Cannon", -5, KindCannon)
	wZero := NewWeapon("Cannon", 0, KindCannon)

	sNeg := wNeg.StatsFromEnergy()
	sZero := wZero.StatsFromEnergy()

	if sNeg.Damage != sZero.Damage {
		t.Errorf("negative energy: Damage %.2f != zero energy Damage %.2f", sNeg.Damage, sZero.Damage)
	}
	if sNeg.FireInterval != sZero.FireInterval {
		t.Errorf("negative energy: FireInterval %d != zero energy FireInterval %d", sNeg.FireInterval, sZero.FireInterval)
	}
	if sNeg.Range != sZero.Range {
		t.Errorf("negative energy: Range %.2f != zero energy Range %.2f", sNeg.Range, sZero.Range)
	}
}

func TestStatsFromEnergy_FireIntervalFloorAt6(t *testing.T) {
	// energy=10 would give interval = 45 - 10*4 = 5, which should be clamped to 6.
	w := NewWeapon("Cannon", 10, KindCannon)
	stats := w.StatsFromEnergy()
	if stats.FireInterval != 6 {
		t.Errorf("FireInterval = %d, want 6 (floor clamp)", stats.FireInterval)
	}

	// energy=9 gives 45 - 36 = 9, which is above the floor.
	w9 := NewWeapon("Cannon", 9, KindCannon)
	stats9 := w9.StatsFromEnergy()
	if stats9.FireInterval != 9 {
		t.Errorf("FireInterval = %d, want 9", stats9.FireInterval)
	}
}
