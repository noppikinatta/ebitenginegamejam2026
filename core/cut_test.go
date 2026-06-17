package core

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// newCutTestWorld builds a StatePlaying world with a generator and two weapon
// tiles in a line: gen(0,0) → a(1,0) → b(2,0).
func newCutTestWorld(nippers int) (w *World, a, b hexmap.Index) {
	gen := hexmap.IdxXY(0, 0)
	a = hexmap.IdxXY(1, 0)
	b = hexmap.IdxXY(2, 0)
	tiles := map[hexmap.Index]*Tile{
		gen: wireT(),
		a:   weaponT(),
		b:   weaponT(),
	}
	turret := NewTurret(tiles, []hexmap.Index{gen}, 100)
	w = &World{
		Player: &Player{HP: 100, MaxHP: 100, Nippers: nippers, Weapons: turret.ActiveWeapons()},
		State:  StatePlaying,
		turret: turret,
	}
	return w, a, b
}

func TestNewWorld_StartsWithNippers(t *testing.T) {
	w := NewWorld(testSeed)
	if w.Player.Nippers != startingNippers {
		t.Errorf("starting nippers = %d, want %d", w.Player.Nippers, startingNippers)
	}
}

func TestCutTile_SpendsNipperAndPurges(t *testing.T) {
	w, a, b := newCutTestWorld(3)

	if !w.CutTile(a) {
		t.Fatal("CutTile(a) returned false")
	}
	if w.Player.Nippers != 2 {
		t.Errorf("nippers = %d, want 2", w.Player.Nippers)
	}
	if !w.turret.Tiles()[a].IsPurged() {
		t.Errorf("tile a should be purged")
	}
	// b was downstream of a, so it cascade-purges too.
	if !w.turret.Tiles()[b].IsPurged() {
		t.Errorf("tile b (downstream of a) should cascade-purge")
	}
}

func TestCutTile_ReconcentratesRemainingPower(t *testing.T) {
	// gen → a(1,0); gen → c(0,1). Cutting c leaves a as the sole consumer,
	// doubling its power.
	gen := hexmap.IdxXY(0, 0)
	a := hexmap.IdxXY(1, 0)
	c := hexmap.IdxXY(0, 1)
	tiles := map[hexmap.Index]*Tile{
		gen: wireT(),
		a:   weaponT(),
		c:   weaponT(),
	}
	turret := NewTurret(tiles, []hexmap.Index{gen}, 100)
	w := &World{
		Player: &Player{HP: 100, MaxHP: 100, Nippers: 1, Weapons: turret.ActiveWeapons()},
		State:  StatePlaying,
		turret: turret,
	}

	if !w.CutTile(c) {
		t.Fatal("CutTile(c) returned false")
	}
	if len(w.Player.Weapons) != 1 {
		t.Fatalf("weapons after cut = %d, want 1", len(w.Player.Weapons))
	}
	if got := w.Player.Weapons[0].Energy; got != 100 {
		t.Errorf("remaining weapon energy = %v, want 100 (reconcentrated)", got)
	}
}

func TestCutTile_NoNippersFails(t *testing.T) {
	w, a, _ := newCutTestWorld(0)

	if w.CutTile(a) {
		t.Error("CutTile should fail with zero nippers")
	}
	if w.turret.Tiles()[a].IsPurged() {
		t.Error("tile should not be purged when cut fails")
	}
	if w.Player.Nippers != 0 {
		t.Errorf("nippers = %d, want 0", w.Player.Nippers)
	}
}

func TestCutTile_RejectsGenerator(t *testing.T) {
	w, _, _ := newCutTestWorld(3)
	gen := hexmap.IdxXY(0, 0)

	if w.CutTile(gen) {
		t.Error("CutTile(generator) should fail")
	}
	if w.Player.Nippers != 3 {
		t.Errorf("nippers = %d, want 3 (no spend on failed cut)", w.Player.Nippers)
	}
}

func TestCutTile_FailsWhenNotPlaying(t *testing.T) {
	w, a, _ := newCutTestWorld(3)
	w.State = StateLevelUp

	if w.CutTile(a) {
		t.Error("CutTile should fail when not in StatePlaying")
	}
	if w.Player.Nippers != 3 {
		t.Errorf("nippers = %d, want 3", w.Player.Nippers)
	}
}
