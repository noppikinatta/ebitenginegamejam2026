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

// TestProjectile_AppearanceFields: fired bullets carry their weapon's sprite key,
// draw footprint and face-velocity flag through to the Projectile, so the scene
// can draw per-weapon art (and rotate the elongated cannon/sniper/missile).
func TestProjectile_AppearanceFields(t *testing.T) {
	cases := []struct {
		kind         WeaponKind
		sprite       string
		w, h         float64
		faceVelocity bool
	}{
		{KindCannon, SpriteCannon, 8, 14, true},
		{KindSniper, SpriteSniper, 4, 16, true},
		{KindMissile, SpriteMissile, 8, 12, true},
		{KindShotgun, SpriteShotgun, 6, 6, false},
		{KindGatling, SpriteGatling, 6, 6, false},
		{KindCIWS, SpriteCIWS, 6, 6, false},
	}
	for _, c := range cases {
		w, wp := buildWeaponWorld(c.kind, hexmap.IdxXY(1, 0))
		p := w.cfg.Weapons[c.kind]
		// CIWS holds fire without a target; give every weapon something to lock on.
		w.Enemies = append(w.Enemies, &Enemy{Pos: geom.PointF{X: 40, Y: 0}, HP: 1000, Radius: 16, alive: true})
		wp.fireProgress = p.BaseInterval // trigger on the next tick
		for i := 0; i < 30 && len(w.Projectiles) == 0; i++ {
			w.updateWeapons()
		}
		if len(w.Projectiles) == 0 {
			t.Fatalf("%v: no projectile fired", c.kind)
		}
		pr := w.Projectiles[0]
		if pr.Sprite != c.sprite {
			t.Errorf("%v: Sprite=%q, want %q", c.kind, pr.Sprite, c.sprite)
		}
		if pr.DrawW != c.w || pr.DrawH != c.h {
			t.Errorf("%v: DrawW/DrawH=%g/%g, want %g/%g", c.kind, pr.DrawW, pr.DrawH, c.w, c.h)
		}
		if pr.FaceVelocity != c.faceVelocity {
			t.Errorf("%v: FaceVelocity=%v, want %v", c.kind, pr.FaceVelocity, c.faceVelocity)
		}
	}
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

// TestRenderAngle_LockOnTracksForwardDoesnt: a lock-on weapon's barrel eases
// toward its target, while a forward weapon's barrel stays pointing forward.
func TestRenderAngle_LockOnTracksForwardDoesnt(t *testing.T) {
	// Lock-on cannon: barrel should swing from forward (-pi/2) toward the enemy
	// sitting at world angle 0 (to the right of the muzzle).
	w, cannon := buildWeaponWorld(KindCannon, hexmap.IdxXY(0, 0))
	w.Enemies = []*Enemy{{Pos: geom.PointF{X: 100, Y: 0}, HP: 1000, Radius: 10, alive: true}}
	if !angleClose(cannon.RenderAngle(), -math.Pi/2, 1e-9) {
		t.Fatalf("initial RenderAngle %.3f, want forward -pi/2", cannon.RenderAngle())
	}
	for i := 0; i < 80; i++ {
		w.updateWeapons()
	}
	if !angleClose(cannon.RenderAngle(), 0, 0.05) {
		t.Errorf("lock-on barrel angle %.3f did not track the enemy at angle 0", cannon.RenderAngle())
	}

	// Forward gatling: ignores the enemy and stays pointing forward.
	w2, gat := buildWeaponWorld(KindGatling, hexmap.IdxXY(1, 0))
	w2.Enemies = []*Enemy{{Pos: geom.PointF{X: 100, Y: 0}, HP: 1000, Radius: 10, alive: true}}
	for i := 0; i < 80; i++ {
		w2.updateWeapons()
	}
	if !angleClose(gat.RenderAngle(), -math.Pi/2, 0.02) {
		t.Errorf("forward barrel angle %.3f, want forward -pi/2", gat.RenderAngle())
	}
}

// TestLockOn_HoldsAimRelativeToTankWhenTargetLost: once a lock-on weapon has
// aimed at an enemy, losing that enemy must freeze the barrel pointing the same
// way relative to the tank (so it turns with the tank), not snap back to forward.
func TestLockOn_HoldsAimRelativeToTankWhenTargetLost(t *testing.T) {
	w, cannon := buildWeaponWorld(KindCannon, hexmap.IdxXY(0, 0))
	p := w.cfg.Weapons[KindCannon]

	// Enemy to the right (world angle 0): tank faces up (-pi/2), so the aim is
	// pi/2 to the right of forward.
	enemy := &Enemy{Pos: geom.PointF{X: 100, Y: 0}, HP: 1000, Radius: 10, alive: true}
	w.Enemies = []*Enemy{enemy}
	if got := w.weaponAim(cannon, p); !angleClose(got, 0, 1e-9) {
		t.Fatalf("aim with enemy = %.3f, want 0 (toward enemy)", got)
	}

	// Enemy leaves: aim must hold at world angle 0 (forward + stored offset),
	// not revert to the tank's forward facing.
	w.Enemies = nil
	if got := w.weaponAim(cannon, p); !angleClose(got, 0, 1e-9) {
		t.Errorf("aim after target lost = %.3f, want held at 0", got)
	}

	// Rotate the tank: the barrel offset is relative, so the held aim rotates too.
	w.Player.FacingAngle = 0 // now facing right
	if got := w.weaponAim(cannon, p); !angleClose(got, math.Pi/2, 1e-9) {
		t.Errorf("held aim after tank turn = %.3f, want pi/2 (offset preserved)", got)
	}

	// A shot fired with no target goes out along the held direction, not forward.
	cannon.fireProgress = p.BaseInterval
	w.updateWeapons()
	if len(w.Projectiles) == 0 {
		t.Fatalf("expected a projectile to be fired")
	}
	if got := w.Projectiles[0].Vel.Angle(); !angleClose(got, math.Pi/2, 1e-9) {
		t.Errorf("projectile angle = %.3f, want held aim pi/2", got)
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
