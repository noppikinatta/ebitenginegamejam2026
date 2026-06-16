package core

import (
	"math"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

// buildLaserWorld creates a World with a single KindLaser weapon at the origin,
// facing up (FacingAngle = -π/2), with the given enemies pre-placed.
func buildLaserWorld(enemies []*Enemy) *World {
	w := &World{
		Player: &Player{
			Pos:         geom.PointF{},
			FacingAngle: -math.Pi / 2,
			Speed:       3,
			Radius:      16,
			HP:          100,
			MaxHP:       100,
			Level:       1,
			XPToNext:    10,
		},
		State: StatePlaying,
		rng:   nil,
	}
	weapon := NewWeapon("TestLaser", 5, KindLaser)
	// beamTicksLeft > 0 makes the beam active immediately (bypass cooldown).
	weapon.beamTicksLeft = 60
	w.Player.Weapons = []*Weapon{weapon}
	w.Enemies = enemies
	return w
}

func TestBeamDPS_HitsEnemyInPath(t *testing.T) {
	// Player at origin, facing up (-π/2). Beam fires upward (negative Y).
	// Enemy directly above at (0, -100) is in the path.
	e := &Enemy{Pos: geom.PointF{X: 0, Y: -100}, HP: 100, Radius: 12, alive: true}
	w := buildLaserWorld([]*Enemy{e})
	weapon := w.Player.Weapons[0]

	stats := weapon.StatsFromEnergy()
	hpBefore := e.HP

	w.updateBeams()

	if e.HP >= hpBefore {
		t.Fatalf("enemy HP should decrease: was %v, now %v", hpBefore, e.HP)
	}
	if math.Abs((hpBefore-e.HP)-stats.Damage) > eps {
		t.Fatalf("expected %.4f damage per tick, got %.4f", stats.Damage, hpBefore-e.HP)
	}
}

func TestBeamDPS_SkipsEnemyOutsidePath(t *testing.T) {
	// The beam tracks the nearest enemy (target), so we need two enemies:
	// one in the beam path (target) and one to the side (bystander).
	target := &Enemy{Pos: geom.PointF{X: 0, Y: -80}, HP: 100, Radius: 12, alive: true}
	bystander := &Enemy{Pos: geom.PointF{X: 200, Y: 0}, HP: 100, Radius: 12, alive: true}
	w := buildLaserWorld([]*Enemy{target, bystander})

	w.updateBeams()

	// target (directly above) must be hit; bystander (far right) must not be.
	if target.HP >= 100 {
		t.Fatalf("target in beam path should be hit")
	}
	if bystander.HP != 100 {
		t.Fatalf("bystander outside beam path should not be hit; HP = %v", bystander.HP)
	}
}

func TestBeamDPS_PenetratesMultipleEnemies(t *testing.T) {
	// Two enemies in the beam's path; both should take damage each tick.
	e1 := &Enemy{Pos: geom.PointF{X: 0, Y: -80}, HP: 100, Radius: 12, alive: true}
	e2 := &Enemy{Pos: geom.PointF{X: 0, Y: -160}, HP: 100, Radius: 12, alive: true}
	w := buildLaserWorld([]*Enemy{e1, e2})

	w.updateBeams()

	if e1.HP >= 100 {
		t.Fatalf("enemy 1 in path should be hit")
	}
	if e2.HP >= 100 {
		t.Fatalf("enemy 2 in path should also be hit (beam penetrates)")
	}
}

func TestBeamDPS_BeamTicksDecrement(t *testing.T) {
	e := &Enemy{Pos: geom.PointF{X: 0, Y: -80}, HP: 1000, Radius: 12, alive: true}
	w := buildLaserWorld([]*Enemy{e})
	weapon := w.Player.Weapons[0]

	initial := weapon.beamTicksLeft
	w.updateBeams()
	if weapon.beamTicksLeft != initial-1 {
		t.Fatalf("beamTicksLeft should decrement: was %d, now %d", initial, weapon.beamTicksLeft)
	}
}

func TestBeamDPS_InactiveWhenTicksZero(t *testing.T) {
	e := &Enemy{Pos: geom.PointF{X: 0, Y: -80}, HP: 100, Radius: 12, alive: true}
	w := buildLaserWorld([]*Enemy{e})
	w.Player.Weapons[0].beamTicksLeft = 0 // force inactive

	w.updateBeams()

	if e.HP != 100 {
		t.Fatalf("beam should be inactive when beamTicksLeft==0")
	}
}

func TestActiveBeams_ReturnedWhenActive(t *testing.T) {
	e := &Enemy{Pos: geom.PointF{X: 0, Y: -80}, HP: 100, Radius: 12, alive: true}
	w := buildLaserWorld([]*Enemy{e})

	beams := w.ActiveBeams()
	if len(beams) != 1 {
		t.Fatalf("want 1 beam, got %d", len(beams))
	}
	b := beams[0]
	// Dir should be approximately unit length.
	if math.Abs(b.Dir.Abs()-1) > 1e-6 {
		t.Fatalf("Dir is not unit vector: |Dir| = %v", b.Dir.Abs())
	}
}

func TestActiveBeams_EmptyWhenInactive(t *testing.T) {
	e := &Enemy{Pos: geom.PointF{X: 0, Y: -80}, HP: 100, Radius: 12, alive: true}
	w := buildLaserWorld([]*Enemy{e})
	w.Player.Weapons[0].beamTicksLeft = 0

	if beams := w.ActiveBeams(); len(beams) != 0 {
		t.Fatalf("want 0 beams when inactive, got %d", len(beams))
	}
}
