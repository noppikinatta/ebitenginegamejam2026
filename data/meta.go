package data

import "github.com/noppikinatta/ebitenginegamejam2026/core"

// MetaUpgradeSpec defines one persistent-upgrade track's balance: how strong each
// level is, the level cap, and the (linear) coin-cost ramp. The cost to go from
// level L to L+1 is CostBase + CostStep*L, so even a 99-level track stays
// affordable and never overflows. This is a jam entry — the curves are tuned so
// a determined player can max out and steamroll a clear; tweak freely here.
type MetaUpgradeSpec struct {
	MaxLevel int
	Bonus    float64 // added to the stat per level (units depend on the stat; see ApplyMeta)
	CostBase int
	CostStep int
}

// metaSpecs is the balance table for every meta stat. Bonuses are intentionally
// generous and the cap is high (99) so upgrades pile up absurdly:
//   - HP:     +20 MaxHP/level   (max +1980)
//   - Armor:  +1 flat reduction (max -99 per hit; min 1 always lands)
//   - Regen:  +1 HP/repair cycle (max +99 per ~5 s)
//   - Speed:  +0.15 px/tick      (max +14.85 on a base of 3)
//   - Attack: +5% damage         (max +495% → ~6× damage)
var metaSpecs = map[core.MetaStat]MetaUpgradeSpec{
	core.MetaHP:     {MaxLevel: 99, Bonus: 20, CostBase: 8, CostStep: 4},
	core.MetaArmor:  {MaxLevel: 99, Bonus: 1, CostBase: 12, CostStep: 6},
	core.MetaRegen:  {MaxLevel: 99, Bonus: 1, CostBase: 10, CostStep: 5},
	core.MetaSpeed:  {MaxLevel: 99, Bonus: 0.15, CostBase: 15, CostStep: 8},
	core.MetaAttack: {MaxLevel: 99, Bonus: 0.05, CostBase: 12, CostStep: 6},
}

// MetaSpec returns the balance spec for a stat.
func MetaSpec(s core.MetaStat) MetaUpgradeSpec { return metaSpecs[s] }

// MetaCost is the coin cost to buy the next level of a stat from its current
// level. It returns 0 once the stat is at (or above) its cap; callers should
// check MetaMaxed first to distinguish "free" from "maxed".
func MetaCost(s core.MetaStat, level int) int {
	spec := metaSpecs[s]
	if level >= spec.MaxLevel {
		return 0
	}
	return spec.CostBase + spec.CostStep*level
}

// MetaMaxed reports whether a stat has reached its level cap.
func MetaMaxed(s core.MetaStat, level int) bool {
	return level >= metaSpecs[s].MaxLevel
}

// BuyMeta attempts to buy the next level of a stat, deducting its cost. It
// returns the updated state (a value copy; MetaState holds the levels in an
// array) and whether the purchase happened — false when maxed or short on coins.
func BuyMeta(m core.MetaState, s core.MetaStat) (core.MetaState, bool) {
	if MetaMaxed(s, m.Level(s)) {
		return m, false
	}
	cost := MetaCost(s, m.Level(s))
	if m.Coins < cost {
		return m, false
	}
	m.Coins -= cost
	m.Lv[s]++
	return m, true
}

// ApplyMeta folds a player's purchased meta levels into a fresh run Config:
// boosting starting HP/MaxHP and speed, and setting the base armor, base repair
// and global damage multiplier the simulation reads. The input Config is taken
// by value and the adjusted copy returned, leaving NewConfig() untouched.
func ApplyMeta(cfg core.Config, m core.MetaState) core.Config {
	cfg.Player.MaxHP += metaSpecs[core.MetaHP].Bonus * float64(m.Level(core.MetaHP))
	cfg.Player.HP = cfg.Player.MaxHP
	cfg.Player.Speed += metaSpecs[core.MetaSpeed].Bonus * float64(m.Level(core.MetaSpeed))
	cfg.BaseArmor += metaSpecs[core.MetaArmor].Bonus * float64(m.Level(core.MetaArmor))
	cfg.BaseHPRegen += metaSpecs[core.MetaRegen].Bonus * float64(m.Level(core.MetaRegen))
	cfg.DamageMult = 1 + metaSpecs[core.MetaAttack].Bonus*float64(m.Level(core.MetaAttack))
	return cfg
}
