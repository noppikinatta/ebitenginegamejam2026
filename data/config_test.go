package data_test

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/data"
)

// allWeaponKinds is the full WeaponKind enumeration. NewConfig must supply
// params for every one, or core.NewWorld would build a turret with weapons it
// cannot fire.
var allWeaponKinds = []struct {
	kind core.WeaponKind
	name string
}{
	{core.KindCannon, "Cannon"},
	{core.KindShotgun, "Shotgun"},
	{core.KindSniper, "Sniper"},
	{core.KindLaser, "Laser"},
	{core.KindGatling, "Gatling"},
	{core.KindGrenade, "Grenade"},
	{core.KindCIWS, "CIWS"},
	{core.KindMissile, "Missile"},
}

// allEnemyKinds is the full zako EnemyKind enumeration that spawn weights may
// reference.
var allEnemyKinds = []struct {
	kind core.EnemyKind
	name string
}{
	{core.EnemyGrunt, "Grunt"},
	{core.EnemySwarmer, "Swarmer"},
	{core.EnemyBrute, "Brute"},
}

func TestNewConfig_Scalars(t *testing.T) {
	c := data.NewConfig()

	if c.StartingNippers <= 0 {
		t.Errorf("StartingNippers should be positive, got %d", c.StartingNippers)
	}
	if c.MaxTurretTiles <= 0 {
		t.Errorf("MaxTurretTiles should be positive, got %d", c.MaxTurretTiles)
	}
	if c.CandlestickInterval <= 0 {
		t.Errorf("CandlestickInterval should be positive, got %d", c.CandlestickInterval)
	}
	if c.XPToNextGrowth < 1 {
		t.Errorf("XPToNextGrowth should be >= 1 (XP requirement must not shrink), got %v", c.XPToNextGrowth)
	}
	if c.HPDoublingTicks <= 0 {
		t.Errorf("HPDoublingTicks should be positive, got %v", c.HPDoublingTicks)
	}
	if c.CapacitorDamageBonus <= 0 {
		t.Errorf("CapacitorDamageBonus should be positive, got %v", c.CapacitorDamageBonus)
	}
}

func TestNewConfig_Player(t *testing.T) {
	p := data.NewConfig().Player

	if p.HP <= 0 || p.MaxHP <= 0 {
		t.Errorf("player HP/MaxHP should be positive, got HP=%v MaxHP=%v", p.HP, p.MaxHP)
	}
	if p.HP != p.MaxHP {
		t.Errorf("a fresh player should start at full health, got HP=%v MaxHP=%v", p.HP, p.MaxHP)
	}
	if p.Speed <= 0 {
		t.Errorf("player Speed should be positive, got %v", p.Speed)
	}
	if p.Radius <= 0 {
		t.Errorf("player Radius should be positive, got %v", p.Radius)
	}
	if p.Level < 1 {
		t.Errorf("player should start at level >= 1, got %d", p.Level)
	}
	if p.XPToNext <= 0 {
		t.Errorf("player XPToNext should be positive, got %v", p.XPToNext)
	}
}

func TestNewConfig_PickupRanges(t *testing.T) {
	pr := data.NewConfig().Pickup

	if pr.PickupDist <= 0 {
		t.Errorf("PickupDist should be positive, got %v", pr.PickupDist)
	}
	if pr.MagnetSpeed <= 0 {
		t.Errorf("MagnetSpeed should be positive, got %v", pr.MagnetSpeed)
	}
	// The magnet must reach farther than the pickup radius, otherwise gems are
	// collected before they ever start drifting toward the player.
	if pr.MagnetDist < pr.PickupDist {
		t.Errorf("MagnetDist (%v) should be >= PickupDist (%v)", pr.MagnetDist, pr.PickupDist)
	}
}

func TestNewConfig_EnemyKinds(t *testing.T) {
	kinds := data.NewConfig().EnemyKinds

	for _, ek := range allEnemyKinds {
		stats, ok := kinds[ek.kind]
		if !ok {
			t.Errorf("missing stats for enemy kind %s", ek.name)
			continue
		}
		if stats.HPBase <= 0 {
			t.Errorf("%s: HPBase should be positive, got %v", ek.name, stats.HPBase)
		}
		if stats.Speed <= 0 {
			t.Errorf("%s: Speed should be positive, got %v", ek.name, stats.Speed)
		}
		if stats.Radius <= 0 {
			t.Errorf("%s: Radius should be positive, got %v", ek.name, stats.Radius)
		}
		if stats.PackMin < 1 {
			t.Errorf("%s: PackMin should be >= 1, got %d", ek.name, stats.PackMin)
		}
		if stats.PackMax < stats.PackMin {
			t.Errorf("%s: PackMax (%d) should be >= PackMin (%d)", ek.name, stats.PackMax, stats.PackMin)
		}
	}
}

func TestNewConfig_SpawnPhases(t *testing.T) {
	c := data.NewConfig()
	phases := c.SpawnPhases

	if len(phases) == 0 {
		t.Fatal("SpawnPhases must not be empty")
	}

	prevUntil := 0
	for i, ph := range phases {
		// currentPhase picks the first band whose UntilTick exceeds the current
		// tick, so the boundaries must strictly increase for every band to be
		// reachable.
		if ph.UntilTick <= prevUntil {
			t.Errorf("phase %d: UntilTick (%d) must be > previous (%d)", i, ph.UntilTick, prevUntil)
		}
		prevUntil = ph.UntilTick

		if ph.Interval <= 0 {
			t.Errorf("phase %d: Interval should be positive, got %d", i, ph.Interval)
		}
		if len(ph.Weights) == 0 {
			t.Errorf("phase %d: must define at least one weight", i)
		}
		for _, w := range ph.Weights {
			if w.Weight <= 0 {
				t.Errorf("phase %d: weight for kind %d should be positive, got %d", i, w.Kind, w.Weight)
			}
			// Cross-check: every weighted kind must have stats, otherwise the
			// spawn director would pick a kind it cannot build.
			if _, ok := c.EnemyKinds[w.Kind]; !ok {
				t.Errorf("phase %d: weight references enemy kind %d with no stats", i, w.Kind)
			}
		}
	}
}

func TestNewConfig_Bosses(t *testing.T) {
	bosses := data.NewConfig().Bosses

	if len(bosses) == 0 {
		t.Fatal("Bosses must not be empty")
	}

	prevAt := -1
	finals := 0
	for i, b := range bosses {
		// spawnBosses walks the slice in order and stops at the first boss whose
		// AtTick is still in the future, so AtTick must strictly increase.
		if b.AtTick <= prevAt {
			t.Errorf("boss %d (%s): AtTick (%d) must be > previous (%d)", i, b.Name, b.AtTick, prevAt)
		}
		prevAt = b.AtTick

		if b.Name == "" {
			t.Errorf("boss %d: Name should not be empty", i)
		}
		if b.HP <= 0 {
			t.Errorf("boss %d (%s): HP should be positive, got %v", i, b.Name, b.HP)
		}
		if b.Speed <= 0 {
			t.Errorf("boss %d (%s): Speed should be positive, got %v", i, b.Name, b.Speed)
		}
		if b.Final {
			finals++
		}
	}

	if finals != 1 {
		t.Errorf("exactly one boss must be Final (clears the run), got %d", finals)
	}
	// The Final boss must be the last one scheduled; nothing should spawn after
	// the run-ending fight.
	if !bosses[len(bosses)-1].Final {
		t.Errorf("the last scheduled boss must be the Final boss")
	}
}

func TestNewConfig_Candlestick(t *testing.T) {
	cs := data.NewConfig().Candlestick

	if cs.HP <= 0 {
		t.Errorf("candlestick HP should be positive, got %v", cs.HP)
	}
	if cs.Radius <= 0 {
		t.Errorf("candlestick Radius should be positive, got %v", cs.Radius)
	}
	if !cs.DropsNipper {
		t.Error("candlestick should drop a nipper when broken")
	}
	// Stationary and harmless by design.
	if cs.Speed != 0 {
		t.Errorf("candlestick should be stationary, got Speed=%v", cs.Speed)
	}
	if cs.Damage != 0 {
		t.Errorf("candlestick should be harmless, got Damage=%v", cs.Damage)
	}
}

func TestNewConfig_Spawn(t *testing.T) {
	s := data.NewConfig().Spawn

	if s.EnemyDist <= 0 {
		t.Errorf("EnemyDist should be positive, got %v", s.EnemyDist)
	}
	if s.CandleDist <= 0 {
		t.Errorf("CandleDist should be positive, got %v", s.CandleDist)
	}
	if s.CandleDistRange < 0 {
		t.Errorf("CandleDistRange should not be negative, got %v", s.CandleDistRange)
	}
}

func TestNewConfig_TurretGen(t *testing.T) {
	tg := data.NewConfig().TurretGen

	if tg.WeaponCount <= 0 {
		t.Errorf("WeaponCount should be positive so the run is playable, got %d", tg.WeaponCount)
	}
	if tg.JunkCount < 0 {
		t.Errorf("JunkCount should not be negative, got %d", tg.JunkCount)
	}
	if tg.BranchProb < 0 || tg.BranchProb > 1 {
		t.Errorf("BranchProb should be a probability in [0,1], got %v", tg.BranchProb)
	}
	if len(tg.Generators) == 0 {
		t.Error("TurretGen must define at least one generator")
	}
	for i, g := range tg.Generators {
		if g.Power <= 0 {
			t.Errorf("generator %d: Power should be positive, got %v", i, g.Power)
		}
	}
}

func TestNewConfig_Doctor(t *testing.T) {
	d := data.NewConfig().Doctor

	weights := map[string]float64{
		"NipperWeight":        d.NipperWeight,
		"WeaponAddWeight":     d.WeaponAddWeight,
		"WeaponUpgradeWeight": d.WeaponUpgradeWeight,
		"JunkWeight":          d.JunkWeight,
	}
	total := 0.0
	for name, wgt := range weights {
		if wgt < 0 {
			t.Errorf("%s should not be negative, got %v", name, wgt)
		}
		total += wgt
	}
	if total <= 0 {
		t.Errorf("offer weights should sum to a positive total, got %v", total)
	}
	if d.NipperMin > d.NipperMax {
		t.Errorf("NipperMin (%d) should be <= NipperMax (%d)", d.NipperMin, d.NipperMax)
	}
	if d.MaxItems <= 0 {
		t.Errorf("MaxItems should be positive, got %d", d.MaxItems)
	}
}

func TestNewConfig_Weapons(t *testing.T) {
	weapons := data.NewConfig().Weapons

	for _, wk := range allWeaponKinds {
		p, ok := weapons[wk.kind]
		if !ok {
			t.Errorf("missing params for weapon kind %s", wk.name)
			continue
		}
		if p.BaseInterval <= 0 {
			t.Errorf("%s: BaseInterval should be positive, got %v", wk.name, p.BaseInterval)
		}
		if p.MinInterval <= 0 {
			t.Errorf("%s: MinInterval should be positive, got %d", wk.name, p.MinInterval)
		}
		// Fire-rate scaling clamps the interval down to MinInterval, so it must
		// not exceed the base cadence.
		if float64(p.MinInterval) > p.BaseInterval {
			t.Errorf("%s: MinInterval (%d) should be <= BaseInterval (%v)", wk.name, p.MinInterval, p.BaseInterval)
		}
		if p.LevelMult <= 0 {
			t.Errorf("%s: LevelMult should be positive, got %v", wk.name, p.LevelMult)
		}
	}
}

func TestNewConfig_PowerCurve(t *testing.T) {
	curve := data.NewConfig().PowerCurve

	if len(curve) == 0 {
		t.Fatal("PowerCurve must not be empty")
	}
	// PowerMultiplier interpolates between points and clamps at the ends, which
	// requires the points to be sorted ascending by Tiles.
	prevTiles := -1
	for i, pt := range curve {
		if pt.Tiles <= prevTiles {
			t.Errorf("point %d: Tiles (%d) must be > previous (%d) (ascending order required)", i, pt.Tiles, prevTiles)
		}
		prevTiles = pt.Tiles
		if pt.Mult <= 0 {
			t.Errorf("point %d: Mult should be positive, got %v", i, pt.Mult)
		}
	}
}

// TestNewConfig_BuildsWorld is an integration smoke test: the canonical config
// must actually drive core.NewWorld without panicking and produce a live,
// playing world with a turret and at least one weapon.
func TestNewConfig_BuildsWorld(t *testing.T) {
	w := core.NewWorld(12345, data.NewConfig())
	if w == nil {
		t.Fatal("NewWorld returned nil")
	}
	if w.State != core.StatePlaying {
		t.Errorf("fresh world should be StatePlaying, got %v", w.State)
	}
	if len(w.Turret().ActiveWeapons()) == 0 {
		t.Error("a fresh turret should have at least one active weapon")
	}
}
