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

// TestExplosion_QueuedAndDecays: explode() queues a visual effect that ages each
// tick and is removed by compact when its Life hits zero.
func TestExplosion_QueuedAndDecays(t *testing.T) {
	w, _ := buildWeaponWorld(KindGrenade, hexmap.IdxXY(1, 0))

	w.explode(geom.PointF{X: 10, Y: 20}, 64, 15)
	if len(w.Explosions) != 1 {
		t.Fatalf("explode queued %d effects, want 1", len(w.Explosions))
	}
	e := w.Explosions[0]
	if e.Radius != 64 || e.Life != e.MaxLife || e.Life <= 0 {
		t.Fatalf("bad explosion: %+v", e)
	}

	for i := 0; i < e.MaxLife; i++ {
		w.updateExplosions()
	}
	if e.Life != 0 {
		t.Errorf("Life = %d after MaxLife ticks, want 0", e.Life)
	}
	w.compact()
	if len(w.Explosions) != 0 {
		t.Errorf("expired explosion not removed by compact: %d remain", len(w.Explosions))
	}
}

// TestCIWS_HoldsThenBurstsAtTarget: with nothing in range the CIWS keeps its
// charge (fires nothing); once an enemy enters range it unleashes the full burst.
func TestCIWS_HoldsThenBurstsAtTarget(t *testing.T) {
	w, wp := buildWeaponWorld(KindCIWS, hexmap.IdxXY(1, 0))
	p := w.cfg.Weapons[KindCIWS]

	wp.fireProgress = p.BaseInterval // ready to fire
	for i := 0; i < 40; i++ {
		w.updateWeapons()
	}
	if len(w.Projectiles) != 0 {
		t.Fatalf("CIWS fired %d projectiles with no target, want 0 (should hold)", len(w.Projectiles))
	}
	if wp.fireProgress != p.BaseInterval {
		t.Errorf("held fireProgress = %.1f, want clamped to %.1f", wp.fireProgress, p.BaseInterval)
	}

	// Enemy enters the short lock range: the burst should now fire.
	w.Enemies = []*Enemy{{Pos: geom.PointF{X: 40, Y: 0}, HP: 1000, Radius: 8, alive: true}}
	for i := 0; i < 40; i++ {
		w.updateWeapons()
	}
	if len(w.Projectiles) != p.Pellets {
		t.Fatalf("CIWS burst emitted %d projectiles, want %d", len(w.Projectiles), p.Pellets)
	}
}

// TestHomingMover_CurvesTowardEnemy: a projectile launched away from the enemy
// is accelerated toward it each tick (homing) and eventually scores a contact
// hit rather than flying straight past.
func TestHomingMover_CurvesTowardEnemy(t *testing.T) {
	w, _ := buildWeaponWorld(KindMissile, hexmap.IdxXY(1, 0))
	enemy := &Enemy{Pos: geom.PointF{X: 100, Y: 0}, HP: 1000, Radius: 10, alive: true}
	w.Enemies = []*Enemy{enemy}

	// Heading straight up; the enemy is to the right, so a straight shot misses.
	m := &Projectile{
		Pos: geom.PointF{X: 0, Y: 0}, Vel: geom.PointF{X: 0, Y: -3},
		Damage: 8, Radius: 6, Life: 400,
		Mover: NewHomingMover(0.3, 6),
		alive: true,
	}
	w.Projectiles = []*Projectile{m}

	w.updateProjectiles() // one steer step toward the enemy (+X)
	if m.Vel.X <= 0 {
		t.Fatalf("homing did not accelerate toward enemy: Vel.X = %.3f, want > 0", m.Vel.X)
	}

	for i := 0; i < 400 && enemy.HP == 1000; i++ {
		w.updateProjectiles()
	}
	if enemy.HP != 992 {
		t.Errorf("homing missile failed to land a contact hit; enemy HP = %.0f, want 992", enemy.HP)
	}
}

// TestMissile_ExplodesOnExpiry: a contact (non-PassThrough) explosive shell that
// flies past everything still detonates on expiry, dealing area damage (not the
// 8 contact damage) to a nearby enemy it never touched.
func TestMissile_ExplodesOnExpiry(t *testing.T) {
	w, _ := buildWeaponWorld(KindMissile, hexmap.IdxXY(1, 0))
	enemy := &Enemy{Pos: geom.PointF{X: 10, Y: 20}, HP: 1000, Radius: 4, alive: true}
	w.Enemies = []*Enemy{enemy}

	// Flies along the X axis; the enemy is 20px off it — out of contact range
	// (6+4) the whole way, but inside the 48px blast when it expires at (10,0).
	m := &Projectile{
		Pos: geom.PointF{X: 0, Y: 0}, Vel: geom.PointF{X: 1, Y: 0},
		Damage: 8, Radius: 6, Life: 10,
		ExplodeRadius: 48, ExplodeDamage: 10,
		alive: true,
	}
	w.Projectiles = []*Projectile{m}

	for i := 0; i < 10; i++ {
		w.updateProjectiles()
	}
	if enemy.HP != 990 {
		t.Errorf("enemy HP = %.0f, want 990 (10 explosion, not 8 contact)", enemy.HP)
	}
	if len(w.Explosions) == 0 {
		t.Error("missile expiry did not queue an explosion effect")
	}
}
