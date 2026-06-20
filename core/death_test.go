package core

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

// TestDeath_KillEmitsEvent: killing an enemy records one death event carrying
// its sprite-selection fields and fall position.
func TestDeath_KillEmitsEvent(t *testing.T) {
	w := damageWorld()
	e := &Enemy{
		Pos:         geom.PointF{X: 12, Y: -8},
		Kind:        EnemyBrute,
		Radius:      20,
		DropsNipper: true,
		alive:       true,
	}
	w.Enemies = []*Enemy{e}

	w.killEnemy(e)

	if len(w.DeathEvents) != 1 {
		t.Fatalf("want 1 death event, got %d", len(w.DeathEvents))
	}
	d := w.DeathEvents[0]
	if d.Pos != e.Pos || d.Kind != EnemyBrute || d.Radius != 20 || !d.DropsNipper {
		t.Errorf("death event = %+v, want pos %v kind Brute radius 20 dropsNipper", d, e.Pos)
	}
}

// TestDeath_ClearedEachTick: death events from one tick do not leak into the
// next.
func TestDeath_ClearedEachTick(t *testing.T) {
	w := damageWorld()
	w.DeathEvents = append(w.DeathEvents, DeathEvent{Radius: 99}) // stale event

	w.Update(geom.PointF{})

	if len(w.DeathEvents) != 0 {
		t.Errorf("stale death events survived into a new tick: %d", len(w.DeathEvents))
	}
}
