package core_test

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/data"
	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// This file is a black-box (package core_test) companion to the white-box unit
// tests in package core. It exercises core strictly through its exported API —
// the surface the scene layer actually depends on — using the real production
// balance from data.NewConfig.

const bbSeed = 0xC0FFEE

func newGameWorld() *core.World {
	return core.NewWorld(bbSeed, data.NewConfig())
}

func TestBlackBox_NewWorld(t *testing.T) {
	w := newGameWorld()
	cfg := data.NewConfig()

	if w.State != core.StatePlaying {
		t.Errorf("fresh world should be StatePlaying, got %v", w.State)
	}
	if w.Player == nil {
		t.Fatal("Player should not be nil")
	}
	if w.Player.HP <= 0 || w.Player.HP != w.Player.MaxHP {
		t.Errorf("player should start at full health, got HP=%v MaxHP=%v", w.Player.HP, w.Player.MaxHP)
	}
	if w.Player.Nippers != cfg.StartingNippers {
		t.Errorf("player should start with %d nippers, got %d", cfg.StartingNippers, w.Player.Nippers)
	}
	if w.Turret() == nil {
		t.Fatal("Turret() should not be nil")
	}
	if len(w.Turret().ActiveWeapons()) == 0 {
		t.Error("a fresh turret should have at least one active weapon")
	}
}

func TestBlackBox_UpdateAdvancesAndMoves(t *testing.T) {
	w := newGameWorld()
	start := w.Player.Pos

	// Drive straight right for several ticks via the public Update API.
	for i := 0; i < 10; i++ {
		w.Update(geom.PointF{X: 1, Y: 0})
	}

	if w.Tick == 0 {
		t.Error("Tick should advance after Update")
	}
	if w.Player.Pos.X <= start.X {
		t.Errorf("player should have moved right, start X=%v now X=%v", start.X, w.Player.Pos.X)
	}
}

func TestBlackBox_UpdateFiresWeapons(t *testing.T) {
	w := newGameWorld()

	// Most weapons fire even without a target, so over enough ticks the public
	// state must show evidence of combat (projectiles, beams or sounds).
	fired := false
	for i := 0; i < 3000 && !fired; i++ {
		w.Update(geom.PointF{})
		if len(w.Projectiles) > 0 || len(w.ActiveBeams()) > 0 || len(w.SoundEvents) > 0 {
			fired = true
		}
	}
	if !fired {
		t.Error("expected weapons to fire (projectiles/beams/sounds) within 3000 ticks")
	}
}

func TestBlackBox_CutTile(t *testing.T) {
	w := newGameWorld()

	// Find a non-generator, active tile to cut.
	var target hexmap.Index
	found := false
	for idx, tile := range w.Turret().Tiles() {
		if w.Turret().IsGenerator(idx) {
			continue
		}
		if tile.IsActive() {
			target = idx
			found = true
			break
		}
	}
	if !found {
		t.Skip("no cuttable consumer tile in generated turret")
	}

	beforeNippers := w.Player.Nippers
	beforeTiles := w.Turret().TileCount()

	if !w.CutTile(target) {
		t.Fatal("CutTile should succeed on an active consumer tile")
	}
	if w.Player.Nippers != beforeNippers-1 {
		t.Errorf("CutTile should consume one nipper, before=%d after=%d", beforeNippers, w.Player.Nippers)
	}
	if w.Turret().TileCount() >= beforeTiles {
		t.Errorf("TileCount should drop after a cut, before=%d after=%d", beforeTiles, w.Turret().TileCount())
	}
}

func TestBlackBox_CutTileGuards(t *testing.T) {
	w := newGameWorld()

	// A coordinate well outside the turret cannot be cut.
	if w.CutTile(hexmap.IdxXY(100, 100)) {
		t.Error("CutTile on a non-existent tile should fail")
	}

	// With no nippers, cutting is refused even for a valid tile.
	w.Player.Nippers = 0
	for idx := range w.Turret().Tiles() {
		if !w.Turret().IsGenerator(idx) {
			if w.CutTile(idx) {
				t.Error("CutTile should fail when the player has no nippers")
			}
			break
		}
	}
}

func TestBlackBox_ChooseUpgradeNoopWhilePlaying(t *testing.T) {
	w := newGameWorld()
	// Outside StateLevelUp, ChooseUpgrade must be a safe no-op.
	w.ChooseUpgrade(0)
	if w.State != core.StatePlaying {
		t.Errorf("ChooseUpgrade should not change state while playing, got %v", w.State)
	}
}

func TestBlackBox_ComponentNames(t *testing.T) {
	cases := []struct {
		comp core.Component
		want string
	}{
		{core.Wire{}, "Wire"},
		{core.Junk{}, "Junk"},
		{core.Junk{DeviceName: "Toaster"}, "Toaster"},
		{core.Capacitor{}, "Capacitor"},
		{core.WeaponComponent{Weapon: core.NewWeapon("Cannon", core.KindCannon)}, "Cannon"},
	}
	for _, c := range cases {
		if got := c.comp.Name(); got != c.want {
			t.Errorf("Name() = %q, want %q", got, c.want)
		}
	}
}

func TestBlackBox_ComponentMods(t *testing.T) {
	// Only the Capacitor contributes a modifier; the rest are inert.
	if got := (core.Capacitor{DamageBonus: 0.1}).Mods().DamageBonus; got != 0.1 {
		t.Errorf("Capacitor DamageBonus = %v, want 0.1", got)
	}
	inert := []core.Component{core.Wire{}, core.Junk{}, core.WeaponComponent{Weapon: core.NewWeapon("C", core.KindCannon)}}
	for _, c := range inert {
		if c.Mods() != (core.Modifier{}) {
			t.Errorf("%s should have zero Mods, got %+v", c.Name(), c.Mods())
		}
	}
}

func TestBlackBox_TurretGeneratorsAndTiles(t *testing.T) {
	w := newGameWorld()
	tr := w.Turret()

	gens := tr.Generators()
	if len(gens) == 0 {
		t.Fatal("turret should have at least one generator")
	}
	for _, g := range gens {
		if !tr.IsGenerator(g) {
			t.Errorf("Generators() returned %v but IsGenerator reports false", g)
		}
	}

	// A freshly generated turret should have active (un-purged) tiles.
	active := 0
	for _, tile := range tr.Tiles() {
		if tile.IsActive() {
			active++
		}
	}
	if active == 0 {
		t.Error("expected at least one active tile in a fresh turret")
	}
}

func TestBlackBox_FireRateMultBounds(t *testing.T) {
	w := newGameWorld()

	min, max := w.FireRateMultBounds()
	if min <= 0 {
		t.Errorf("min fire-rate multiplier should be positive, got %v", min)
	}
	if min > max {
		t.Errorf("min (%v) should be <= max (%v)", min, max)
	}

	// The fire-rate multiplier comes straight from the power curve, so it must
	// sit within the curve bounds.
	if cur := w.FireRateMultiplier(); cur < min {
		t.Errorf("current multiplier %v should be >= curve floor %v", cur, min)
	}
}

func TestBlackBox_FireRateMultBoundsEmptyCurve(t *testing.T) {
	cfg := data.NewConfig()
	cfg.PowerCurve = nil // no curve -> neutral, fixed bounds
	w := core.NewWorld(bbSeed, cfg)

	min, max := w.FireRateMultBounds()
	if min != 1 || max != 1 {
		t.Errorf("empty curve should yield bounds (1,1), got (%v,%v)", min, max)
	}
}

func TestBlackBox_WeaponIsBeamActive(t *testing.T) {
	w := newGameWorld()
	// Before any weapon fires a beam, IsBeamActive must report false for every
	// active weapon.
	for _, wp := range w.Turret().ActiveWeapons() {
		if wp.IsBeamActive() {
			t.Errorf("weapon %q should not have an active beam on a fresh turret", wp.Name)
		}
	}
}

func TestBlackBox_PowerMultiplier(t *testing.T) {
	curve := []core.PowerPoint{
		{Tiles: 10, Mult: 4.0},
		{Tiles: 30, Mult: 1.0},
	}
	cases := []struct {
		name  string
		tiles int
		want  float64
	}{
		{"below range clamps to first", 5, 4.0},
		{"at first point", 10, 4.0},
		{"midpoint interpolates", 20, 2.5},
		{"above range clamps to last", 99, 1.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := core.PowerMultiplier(curve, c.tiles); got != c.want {
				t.Errorf("PowerMultiplier(%d) = %v, want %v", c.tiles, got, c.want)
			}
		})
	}

	// An empty curve falls back to a neutral multiplier.
	if got := core.PowerMultiplier(nil, 12); got != 1 {
		t.Errorf("empty curve should yield 1, got %v", got)
	}
}

func TestBlackBox_JunkDeviceFuncs(t *testing.T) {
	names := core.JunkDeviceNames()
	if len(names) == 0 {
		t.Fatal("JunkDeviceNames should not be empty")
	}
	for _, n := range names {
		if key := core.JunkImageKey(n); key == "" {
			t.Errorf("JunkImageKey(%q) should not be empty", n)
		}
	}

	// "Five-storied Pagoda" is a tall fixture; an unknown name is not.
	if !core.JunkDeviceTall("Five-storied Pagoda") {
		t.Error("Five-storied Pagoda should be a tall device")
	}
	if core.JunkDeviceTall("definitely not a device") {
		t.Error("unknown device should not be tall")
	}

	// Slugification: lowercase, non-alphanumerics collapse to single underscore.
	if got := core.JunkImageKey("Wi-Fi Antenna"); got != "junk_wi_fi_antenna" {
		t.Errorf("JunkImageKey slug = %q, want junk_wi_fi_antenna", got)
	}
}

func TestBlackBox_FireSound(t *testing.T) {
	if got := core.FireSound(core.KindCannon); got != core.SndFireCannon {
		t.Errorf("FireSound(Cannon) = %v, want SndFireCannon", got)
	}
	if got := core.FireSound(core.KindShotgun); got != core.SndFireShotgun {
		t.Errorf("FireSound(Shotgun) = %v, want SndFireShotgun", got)
	}
}

// fakeSink records dispatched sounds so DispatchSounds can be tested without the
// Ebiten audio backend.
type fakeSink struct{ played []core.SoundEvent }

func (f *fakeSink) PlaySound(e core.SoundEvent) { f.played = append(f.played, e) }

func TestBlackBox_DispatchSounds(t *testing.T) {
	sink := &fakeSink{}
	// Duplicate events within a tick collapse to one play each.
	core.DispatchSounds([]core.SoundEvent{
		core.SndFireCannon, core.SndFireCannon, core.SndExplosion,
	}, sink)

	if len(sink.played) != 2 {
		t.Fatalf("expected 2 distinct sounds, got %d (%v)", len(sink.played), sink.played)
	}
}

func TestBlackBox_ActiveBossAndBeamsInitiallyEmpty(t *testing.T) {
	w := newGameWorld()
	if w.ActiveBoss() != nil {
		t.Error("no boss should be active at tick 0")
	}
	if len(w.ActiveBeams()) != 0 {
		t.Error("no beams should be active before any weapon fires")
	}
}
