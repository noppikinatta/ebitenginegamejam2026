package data_test

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/data"
)

// TestMetaCostProgressive checks the cost ramp is CostBase * CostStep^level
// (clamped to 100000) and that a maxed stat reports cost 0 / maxed.
func TestMetaCostProgressive(t *testing.T) {
	spec := data.MetaSpec(core.MetaHP)
	for lv := 0; lv < spec.MaxLevel; lv++ {
		want := spec.CostBase
		for range lv {
			want *= spec.CostStep
			if want > 100000 {
				want = 100000
				break
			}
		}
		if got := data.MetaCost(core.MetaHP, lv); got != want {
			t.Errorf("MetaCost(HP, %d) = %d, want %d", lv, got, want)
		}
	}
	if !data.MetaMaxed(core.MetaHP, spec.MaxLevel) {
		t.Errorf("level %d should be maxed", spec.MaxLevel)
	}
	if got := data.MetaCost(core.MetaHP, spec.MaxLevel); got != 0 {
		t.Errorf("MetaCost at cap = %d, want 0", got)
	}
}

// TestBuyMeta covers a successful purchase, an insufficient-coins purchase and a
// maxed-out purchase.
func TestBuyMeta(t *testing.T) {
	cost0 := data.MetaCost(core.MetaHP, 0)

	m := core.MetaState{Coins: cost0}
	m, ok := data.BuyMeta(m, core.MetaHP)
	if !ok || m.Level(core.MetaHP) != 1 || m.Coins != 0 {
		t.Fatalf("buy: ok=%v lv=%d coins=%d, want true/1/0", ok, m.Level(core.MetaHP), m.Coins)
	}

	// Now broke: next buy must fail and leave the state untouched.
	before := m
	m, ok = data.BuyMeta(m, core.MetaHP)
	if ok || m != before {
		t.Errorf("buy with no coins: ok=%v state changed=%v", ok, m != before)
	}

	// Maxed track cannot be bought even with coins.
	maxed := core.MetaState{Coins: 1 << 30}
	maxed.Lv[core.MetaArmor] = data.MetaSpec(core.MetaArmor).MaxLevel
	if _, ok := data.BuyMeta(maxed, core.MetaArmor); ok {
		t.Errorf("buying a maxed stat should fail")
	}
}

// TestApplyMeta verifies each level translates into the right Config bonus.
func TestApplyMeta(t *testing.T) {
	base := data.NewConfig()
	m := core.MetaState{}
	m.Lv[core.MetaHP] = 2
	m.Lv[core.MetaArmor] = 3
	m.Lv[core.MetaRegen] = 4
	m.Lv[core.MetaSpeed] = 5
	m.Lv[core.MetaAttack] = 6

	cfg := data.ApplyMeta(data.NewConfig(), m)

	wantHP := base.Player.MaxHP + data.MetaSpec(core.MetaHP).Bonus*2
	if cfg.Player.MaxHP != wantHP || cfg.Player.HP != wantHP {
		t.Errorf("HP: MaxHP=%.2f HP=%.2f, want %.2f", cfg.Player.MaxHP, cfg.Player.HP, wantHP)
	}
	if want := base.Player.Speed + data.MetaSpec(core.MetaSpeed).Bonus*5; cfg.Player.Speed != want {
		t.Errorf("Speed=%.3f, want %.3f", cfg.Player.Speed, want)
	}
	if want := data.MetaSpec(core.MetaArmor).Bonus * 3; cfg.BaseArmor != want {
		t.Errorf("BaseArmor=%.2f, want %.2f", cfg.BaseArmor, want)
	}
	if want := data.MetaSpec(core.MetaRegen).Bonus * 4; cfg.BaseHPRegen != want {
		t.Errorf("BaseHPRegen=%.2f, want %.2f", cfg.BaseHPRegen, want)
	}
	if want := 1 + data.MetaSpec(core.MetaAttack).Bonus*6; cfg.DamageMult != want {
		t.Errorf("DamageMult=%.3f, want %.3f", cfg.DamageMult, want)
	}

	// Zero state must leave the baseline config unchanged.
	zero := data.ApplyMeta(data.NewConfig(), core.MetaState{})
	if zero.Player.MaxHP != base.Player.MaxHP || zero.DamageMult != 1 || zero.BaseArmor != 0 {
		t.Errorf("zero meta changed the config: %+v", zero)
	}
}
