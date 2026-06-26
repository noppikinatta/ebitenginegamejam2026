package core

import (
	"math/rand"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// TestCapacitor_ContributesDamageModifier: a connected Capacitor adds its
// DamageBonus to the turret's aggregate modifier, and multiple stack.
func TestCapacitor_ContributesDamageModifier(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	a := hexmap.IdxXY(1, 0)
	b := hexmap.IdxXY(0, 1)
	tiles := map[hexmap.Index]*Tile{
		gen: wireT(),
		a:   makeTile(Capacitor{DamageBonus: 0.1}),
		b:   makeTile(Capacitor{DamageBonus: 0.1}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)

	if got := tr.Modifiers().DamageBonus; !approx(got, 0.2) {
		t.Errorf("two capacitors: DamageBonus = %v, want 0.2", got)
	}
}

// TestCapacitor_RaisesWeaponDamage: the bonus scales every weapon's damage by
// 1+DamageBonus, multiplicatively on top of the meta DamageMult.
func TestCapacitor_RaisesWeaponDamage(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	wpn := hexmap.IdxXY(1, 0)
	weapon := NewWeapon("test", KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen: wireT(),
		wpn: makeTile(WeaponComponent{Weapon: weapon}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	w := &World{turret: tr, cfg: testConfig()}

	// No capacitors yet: raw weapon damage (testConfig leaves DamageMult unset).
	base := w.weaponStats(weapon).Damage

	// Bolt on a capacitor: every weapon's damage rises by 10%.
	if _, ok := tr.AddTile(Capacitor{DamageBonus: 0.1}, rand.New(rand.NewSource(1))); !ok {
		t.Fatal("AddTile(Capacitor) returned false")
	}
	if got := w.weaponStats(weapon).Damage; !approx(got, base*1.1) {
		t.Errorf("damage with one capacitor = %v, want %v", got, base*1.1)
	}

	// The meta DamageMult multiplies on top: capacitor and meta upgrades stack
	// multiplicatively, not additively.
	w.cfg.DamageMult = 2
	if got := w.weaponStats(weapon).Damage; !approx(got, base*1.1*2) {
		t.Errorf("damage with capacitor + meta = %v, want %v", got, base*1.1*2)
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
		capIdx: makeTile(Capacitor{DamageBonus: 0.1}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	if got := tr.Modifiers().DamageBonus; !approx(got, 0.1) {
		t.Fatalf("before cut: DamageBonus = %v, want 0.1", got)
	}

	// Cutting mid orphans the capacitor downstream, so the bonus disappears.
	if !tr.PurgeTile(mid) {
		t.Fatal("PurgeTile(mid) returned false")
	}
	if got := tr.Modifiers().DamageBonus; !approx(got, 0) {
		t.Errorf("after cut: DamageBonus = %v, want 0 (capacitor disconnected)", got)
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
	if got := tr.Modifiers().DamageBonus; !approx(got, 0) {
		t.Fatalf("before add: DamageBonus = %v, want 0", got)
	}

	if _, ok := tr.AddTile(Capacitor{DamageBonus: 0.1}, rand.New(rand.NewSource(1))); !ok {
		t.Fatal("AddTile(Capacitor) returned false")
	}
	if got := tr.Modifiers().DamageBonus; !approx(got, 0.1) {
		t.Errorf("after add: DamageBonus = %v, want 0.1", got)
	}
}
