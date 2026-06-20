package core

import (
	"math"
	"math/rand"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// buildEmitterWorld makes a StatePlaying world whose turret has comp on a tile
// adjacent to (and so connected to) the central generator.
func buildEmitterWorld(comp Component) *World {
	gen := hexmap.IdxXY(0, 0)
	mid := hexmap.IdxXY(1, 0)
	tiles := map[hexmap.Index]*Tile{
		gen: makeTile(Wire{}),
		mid: makeTile(comp),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	return &World{
		Player: &Player{Pos: geom.PointF{}, FacingAngle: -math.Pi / 2, HP: 100, MaxHP: 100},
		State:  StatePlaying,
		rng:    rand.New(rand.NewSource(1)),
		cfg:    testConfig(),
		turret: tr,
	}
}

// TestJunkEmitter_FiresOnInterval: a balloon junk emits one cosmetic projectile
// once its interval elapses, and not before.
func TestJunkEmitter_FiresOnInterval(t *testing.T) {
	w := buildEmitterWorld(newJunk("Balloon Service Unit"))
	interval := balloonEmitter.Interval

	for i := 0; i < interval-1; i++ {
		w.updateJunkEmitters()
	}
	if len(w.Projectiles) != 0 {
		t.Fatalf("emitted before interval: %d projectiles after %d ticks", len(w.Projectiles), interval-1)
	}

	w.updateJunkEmitters() // interval-th tick fires
	if len(w.Projectiles) != 1 {
		t.Fatalf("want 1 projectile after %d ticks, got %d", interval, len(w.Projectiles))
	}

	p := w.Projectiles[0]
	if p.Damage != 0 || p.ExplodeDamage != 0 || !p.PassThrough {
		t.Errorf("junk projectile must be cosmetic: damage=%v explode=%v passthrough=%v", p.Damage, p.ExplodeDamage, p.PassThrough)
	}
	if p.Sprite != SpriteBalloon {
		t.Errorf("sprite = %q, want %q", p.Sprite, SpriteBalloon)
	}
}

// TestJunkEmitter_AllEmittersFireCosmetic: every junk wired with an emitter
// spits out exactly one cosmetic (0-damage, pass-through) projectile carrying
// the spec's sprite once its interval elapses.
func TestJunkEmitter_AllEmittersFireCosmetic(t *testing.T) {
	cases := []struct {
		device string
		spec   EmitterSpec
	}{
		{"Balloon Service Unit", balloonEmitter},
		{"Coffee Maker", coffeeEmitter},
		{"Toaster", toasterEmitter},
		{"Music Box", musicBoxEmitter},
		{"Rubber Duck Dispenser", duckEmitter},
		{"Fireworks", fireworksEmitter},
	}
	for _, tc := range cases {
		t.Run(tc.device, func(t *testing.T) {
			w := buildEmitterWorld(newJunk(tc.device))
			for i := 0; i < tc.spec.Interval-1; i++ {
				w.updateJunkEmitters()
			}
			if len(w.Projectiles) != 0 {
				t.Fatalf("emitted before interval %d: %d projectiles", tc.spec.Interval, len(w.Projectiles))
			}
			w.updateJunkEmitters() // interval-th tick fires
			if len(w.Projectiles) != 1 {
				t.Fatalf("want 1 projectile after %d ticks, got %d", tc.spec.Interval, len(w.Projectiles))
			}
			p := w.Projectiles[0]
			if p.Damage != 0 || p.ExplodeDamage != 0 || !p.PassThrough {
				t.Errorf("junk projectile must be cosmetic: damage=%v explode=%v passthrough=%v", p.Damage, p.ExplodeDamage, p.PassThrough)
			}
			if p.Sprite != tc.spec.Sprite {
				t.Errorf("sprite = %q, want %q", p.Sprite, tc.spec.Sprite)
			}
		})
	}
}

// TestGravityMover_PullsDown: the gravity mover accelerates a projectile toward
// the bottom of the screen (positive Y).
func TestGravityMover_PullsDown(t *testing.T) {
	w := &World{}
	p := &Projectile{}
	NewGravityMover(0.05, 0).Steer(p, w)
	if p.Vel.Y <= 0 {
		t.Errorf("gravity mover should push down (positive Y), got Vel.Y=%v", p.Vel.Y)
	}
}

// TestJunkEmitter_InertNoFire: junk without an emitter never spawns projectiles.
func TestJunkEmitter_InertNoFire(t *testing.T) {
	w := buildEmitterWorld(newJunk("Calculator"))
	for i := 0; i < 300; i++ {
		w.updateJunkEmitters()
	}
	if len(w.Projectiles) != 0 {
		t.Errorf("inert junk emitted %d projectiles", len(w.Projectiles))
	}
}

// TestRiseMover_FloatsUp: the rise mover accelerates a projectile toward the top
// of the screen (negative Y).
func TestRiseMover_FloatsUp(t *testing.T) {
	w := &World{}
	p := &Projectile{}
	NewRiseMover(0.05, 0, 0).Steer(p, w)
	if p.Vel.Y >= 0 {
		t.Errorf("rise mover should push up (negative Y), got Vel.Y=%v", p.Vel.Y)
	}
}
