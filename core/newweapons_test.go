package core

import (
	"math"
	"math/rand"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// buildWeaponWorld makes a minimal StatePlaying world with a single weapon of
// the given kind on the given tile (no turret, so FireRateMultiplier == 1).
func buildWeaponWorld(kind WeaponKind, tile hexmap.Index) (*World, *Weapon) {
	w := &World{
		Player: &Player{Pos: geom.PointF{}, FacingAngle: -math.Pi / 2, HP: 100, MaxHP: 100},
		State:  StatePlaying,
		rng:    rand.New(rand.NewSource(1)),
		cfg:    testConfig(),
	}
	wp := NewWeapon(kind.String(), kind)
	wp.TileIdx = tile
	w.Player.Weapons = []*Weapon{wp}
	return w, wp
}

func angleClose(a, b, tol float64) bool {
	d := math.Atan2(math.Sin(a-b), math.Cos(a-b))
	return math.Abs(d) <= tol
}

// TestGatling_ForwardStaggeredBurst: a triggered gatling emits all its pellets
// over time, all aimed forward within the spread, with no enemy present (no lock).
func TestGatling_ForwardStaggeredBurst(t *testing.T) {
	w, wp := buildWeaponWorld(KindGatling, hexmap.IdxXY(1, 0))
	p := w.cfg.Weapons[KindGatling] // Pellets 10, SpreadRad 0.2, BurstGap 3

	wp.fireProgress = p.BaseInterval // make it trigger on the next tick
	for i := 0; i < 60; i++ {
		w.updateWeapons()
	}

	if len(w.Projectiles) != p.Pellets {
		t.Fatalf("gatling emitted %d projectiles, want %d", len(w.Projectiles), p.Pellets)
	}
	forward := w.Player.FacingAngle
	for i, pr := range w.Projectiles {
		if !angleClose(pr.Vel.Angle(), forward, p.SpreadRad+1e-9) {
			t.Errorf("pellet %d angle %.3f not within %.2f of forward %.3f", i, pr.Vel.Angle(), p.SpreadRad, forward)
		}
	}
}

// TestGrenade_ExplodesOnExpiryAndPassesThrough: the grenade ignores contact (an
// enemy sitting on the muzzle is unharmed in flight) and deals area damage where
// it expires.
func TestGrenade_ExplodesOnExpiryAndPassesThrough(t *testing.T) {
	tile := hexmap.IdxXY(1, 0)
	w, wp := buildWeaponWorld(KindGrenade, tile)
	p := w.cfg.Weapons[KindGrenade] // ProjSpeed 2, ProjMaxDist 120 -> life 60, ExplodeRadius 64, ExplodeDamage 15

	muzzle := w.Player.Pos.Add(MuzzleOffset(tile, w.Player.FacingAngle))
	unit := geom.PointFFromPolar(1, MuzzleOffset(tile, w.Player.FacingAngle).Angle())
	landing := muzzle.Add(unit.Multiply(p.ProjMaxDist)) // muzzle + 120 px outward

	// Enemy sitting on the muzzle: the grenade spawns on it but must not detonate
	// on contact, and it is far from the blast, so it stays at full HP.
	passthrough := &Enemy{Pos: muzzle, HP: 100, Radius: 16, alive: true}
	// Enemy at the landing point: caught in the explosion.
	blastTarget := &Enemy{Pos: landing, HP: 100, Radius: 8, alive: true}
	w.Enemies = []*Enemy{passthrough, blastTarget}

	wp.fireProgress = p.BaseInterval
	w.updateWeapons() // spawns one grenade
	if len(w.Projectiles) != 1 {
		t.Fatalf("want 1 grenade projectile, got %d", len(w.Projectiles))
	}

	for i := 0; i < 60; i++ { // fly until it expires and explodes
		w.updateProjectiles()
	}

	if passthrough.HP != 100 {
		t.Errorf("muzzle enemy HP = %.0f, want 100 (grenade must pass through, no contact damage)", passthrough.HP)
	}
	if blastTarget.HP != 85 {
		t.Errorf("blast enemy HP = %.0f, want 85 (100 - 15 explosion)", blastTarget.HP)
	}
}
