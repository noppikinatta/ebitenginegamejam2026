package core

import (
	"math"
	"math/rand"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

func damageWorld() *World {
	return &World{
		Player: &Player{Pos: geom.PointF{X: 5, Y: 7}, FacingAngle: -math.Pi / 2, HP: 100, MaxHP: 100},
		State:  StatePlaying,
		rng:    rand.New(rand.NewSource(1)),
		cfg:    testConfig(),
	}
}

// TestDamage_ExplosionEmitsWhitePerEnemy: explode records one enemy (white)
// damage event at each hit enemy's position.
func TestDamage_ExplosionEmitsWhitePerEnemy(t *testing.T) {
	w := damageWorld()
	e := &Enemy{Pos: geom.PointF{X: 20, Y: 0}, HP: 100, Radius: 8, alive: true}
	w.Enemies = []*Enemy{e}

	w.explode(geom.PointF{X: 20, Y: 0}, 32, 10)

	if len(w.DamageEvents) != 1 {
		t.Fatalf("want 1 damage event, got %d", len(w.DamageEvents))
	}
	d := w.DamageEvents[0]
	if d.ToPlayer {
		t.Error("enemy hit should not be ToPlayer")
	}
	if d.Amount != 10 {
		t.Errorf("amount = %v, want 10", d.Amount)
	}
	if d.Pos != e.Pos {
		t.Errorf("pos = %v, want enemy pos %v", d.Pos, e.Pos)
	}
}

// TestDamage_ZeroDamageEmitsNothing: a cosmetic 0-damage blast (e.g. the
// firework junk) records no damage number.
func TestDamage_ZeroDamageEmitsNothing(t *testing.T) {
	w := damageWorld()
	w.Enemies = []*Enemy{{Pos: geom.PointF{}, HP: 100, Radius: 8, alive: true}}

	w.explode(geom.PointF{}, 32, 0)

	if len(w.DamageEvents) != 0 {
		t.Errorf("0-damage blast emitted %d events, want 0", len(w.DamageEvents))
	}
}

// TestDamage_PlayerHitEmitsRed: damagePlayer records a red (ToPlayer) event at
// the tank's position.
func TestDamage_PlayerHitEmitsRed(t *testing.T) {
	w := damageWorld()

	w.damagePlayer(7)

	if len(w.DamageEvents) != 1 {
		t.Fatalf("want 1 damage event, got %d", len(w.DamageEvents))
	}
	d := w.DamageEvents[0]
	if !d.ToPlayer {
		t.Error("player hit should be ToPlayer")
	}
	if d.Amount != 7 || d.Pos != w.Player.Pos {
		t.Errorf("event = %+v, want amount 7 at player pos %v", d, w.Player.Pos)
	}
}

// TestDamage_ClearedEachTick: damage events from one tick do not leak into the
// next.
func TestDamage_ClearedEachTick(t *testing.T) {
	w := damageWorld()
	w.DamageEvents = append(w.DamageEvents, DamageEvent{Amount: 99}) // stale event

	w.Update(geom.PointF{})

	if len(w.DamageEvents) != 0 {
		t.Errorf("stale damage events survived into a new tick: %d", len(w.DamageEvents))
	}
}
