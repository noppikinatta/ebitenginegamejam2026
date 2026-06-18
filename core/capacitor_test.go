package core

import (
	"math/rand"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// TestCapacitor_ContributesFireRateModifier: a connected Capacitor adds its
// FireRateBonus to the turret's aggregate modifier, and multiple stack.
func TestCapacitor_ContributesFireRateModifier(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	a := hexmap.IdxXY(1, 0)
	b := hexmap.IdxXY(0, 1)
	tiles := map[hexmap.Index]*Tile{
		gen: wireT(),
		a:   makeTile(Capacitor{FireRateBonus: 0.1}),
		b:   makeTile(Capacitor{FireRateBonus: 0.1}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)

	if got := tr.Modifiers().FireRateAdd; !approx(got, 0.2) {
		t.Errorf("two capacitors: FireRateAdd = %v, want 0.2", got)
	}
}

// TestCapacitor_RaisesFireRateMultiplier: the bonus flows through to the
// turret-wide fire-rate multiplier on top of the curve value.
func TestCapacitor_RaisesFireRateMultiplier(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	a := hexmap.IdxXY(1, 0)
	tiles := map[hexmap.Index]*Tile{
		gen: wireT(),
		a:   makeTile(Capacitor{FireRateBonus: 0.1}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	w := &World{turret: tr, cfg: testConfig()}

	// 1 consumer tile → below the curve's first point → 4.0, plus the 0.1 bonus.
	if got := w.FireRateMultiplier(); !approx(got, 4.1) {
		t.Errorf("FireRateMultiplier with capacitor = %v, want 4.1", got)
	}
}

// TestCapacitor_BonusRemovedOnDisconnect: purging the wire upstream of a
// capacitor cascade-removes it, and the bonus is recomputed away.
func TestCapacitor_BonusRemovedOnDisconnect(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	mid := hexmap.IdxXY(1, 0)
	capIdx := hexmap.IdxXY(2, 0)
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		mid:    wireT(),
		capIdx: makeTile(Capacitor{FireRateBonus: 0.1}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	if got := tr.Modifiers().FireRateAdd; !approx(got, 0.1) {
		t.Fatalf("before cut: FireRateAdd = %v, want 0.1", got)
	}

	// Cutting mid orphans the capacitor downstream, so the bonus disappears.
	if !tr.PurgeTile(mid) {
		t.Fatal("PurgeTile(mid) returned false")
	}
	if got := tr.Modifiers().FireRateAdd; !approx(got, 0) {
		t.Errorf("after cut: FireRateAdd = %v, want 0 (capacitor disconnected)", got)
	}
}

// TestCapacitor_BonusAddedOnAddTile: AddTile recomputes the modifier so a newly
// bolted-on capacitor takes effect immediately.
func TestCapacitor_BonusAddedOnAddTile(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	a := hexmap.IdxXY(1, 0)
	tiles := map[hexmap.Index]*Tile{
		gen: wireT(),
		a:   weaponT(),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	if got := tr.Modifiers().FireRateAdd; !approx(got, 0) {
		t.Fatalf("before add: FireRateAdd = %v, want 0", got)
	}

	if _, ok := tr.AddTile(Capacitor{FireRateBonus: 0.1}, rand.New(rand.NewSource(1))); !ok {
		t.Fatal("AddTile(Capacitor) returned false")
	}
	if got := tr.Modifiers().FireRateAdd; !approx(got, 0.1) {
		t.Errorf("after add: FireRateAdd = %v, want 0.1", got)
	}
}
