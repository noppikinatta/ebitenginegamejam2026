package core

// MetaStat identifies one persistent (meta-progression) upgrade track. Coins
// earned across runs buy levels in these; the resulting bonuses are folded into
// a fresh run's Config by the data layer (see data.ApplyMeta), so core stays
// free of the balance numbers.
type MetaStat int

const (
	MetaHP     MetaStat = iota // +MaxHP (and starting HP)
	MetaArmor                  // +flat damage reduction per hit
	MetaRegen                  // +HP healed per repair cycle
	MetaSpeed                  // +move speed
	MetaAttack                 // +global weapon damage multiplier
	metaStatCount
)

// MetaStats lists every upgrade track in display/iteration order. The image
// generator, the data balance table and the workshop UI all iterate this slice
// so they stay in sync.
var MetaStats = []MetaStat{MetaHP, MetaArmor, MetaRegen, MetaSpeed, MetaAttack}

// MetaStatKey returns the stable slug for a stat, used to build image keys
// ("meta_<key>") and localisation keys ("meta-<key>-name" etc.).
func MetaStatKey(s MetaStat) string {
	switch s {
	case MetaHP:
		return "hp"
	case MetaArmor:
		return "armor"
	case MetaRegen:
		return "regen"
	case MetaSpeed:
		return "speed"
	case MetaAttack:
		return "attack"
	default:
		return "unknown"
	}
}

// MetaStatImageKey is the asset image key for a stat's 24x24 icon.
func MetaStatImageKey(s MetaStat) string { return "meta_" + MetaStatKey(s) }

// MetaState is the persistent player progression for one session: spendable
// coins plus a purchased level per upgrade track. It is plain data (no Ebiten or
// I/O) so it can live in a shared holder and be unit-tested.
type MetaState struct {
	Coins int
	Lv    [metaStatCount]int
}

// Level returns the purchased level of one stat.
func (m MetaState) Level(s MetaStat) int { return m.Lv[s] }

// EarnedCoins is the coin reward for a finished run: one coin per junk tile
// still mounted on the turret at the end, plus one (so a junk-free turret still
// pays out a single coin).
func EarnedCoins(junk int) int {
	return junk + 1
}
