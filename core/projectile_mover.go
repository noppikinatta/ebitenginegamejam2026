package core

import "math"

// ProjectileMover steers a projectile each tick, before the world integrates its
// position. It adjusts p.Vel (acceleration / homing / drift); lifetime, position
// integration and collisions stay in the world. A nil mover flies straight.
//
// Movers should be stateless (configuration only) so a single instance can be
// shared by every projectile a weapon fires; any per-projectile state lives on
// the Projectile. New behaviours (e.g. a wobbling, rising balloon for junk that
// reuses the projectile system) implement this interface and are attached via
// WeaponParams.Mover.
type ProjectileMover interface {
	Steer(p *Projectile, w *World)
}

// homingMover steers a projectile toward the nearest enemy with a bounded turn
// force each tick (a "seek" steering behaviour), curving its flight path in
// rather than snapping its velocity straight at the target. For its first
// straight ticks it does not steer at all, so the shell visibly boosts out along
// its launch heading (clearing the tank/turret) before it begins to seek.
type homingMover struct {
	turn     float64 // max steering force applied toward the target each tick
	maxSpeed float64 // cruising / cap speed (0 = keep current speed)
	straight int     // ticks the shell flies straight after launch before homing
}

// NewHomingMover returns a mover that homes onto the nearest enemy. For the first
// straight ticks the shell flies straight along its launch heading (a visible
// boost-out clear of the turret); after that, each tick it nudges the velocity
// toward "head at maxSpeed straight at the target" by at most turn, so a smaller
// turn gives a wider, lazier arc. With no enemies the projectile coasts along
// its current velocity.
func NewHomingMover(turn, maxSpeed float64, straight int) ProjectileMover {
	return homingMover{turn: turn, maxSpeed: maxSpeed, straight: straight}
}

// riseMover makes a projectile drift up the screen (toward world -Y) with a
// gentle horizontal sway — a wobbling, rising balloon. Used by cosmetic junk
// emitters; it ignores enemies entirely (movement only).
type riseMover struct {
	lift     float64 // upward acceleration per tick (toward -Y)
	wobble   float64 // horizontal sway acceleration amplitude
	maxSpeed float64 // speed cap (0 = uncapped)
}

// NewRiseMover returns a mover that floats a projectile upward with a sideways
// wobble. lift is the per-tick upward acceleration, wobble the sway amplitude,
// maxSpeed the cap (0 = none).
func NewRiseMover(lift, wobble, maxSpeed float64) ProjectileMover {
	return riseMover{lift: lift, wobble: wobble, maxSpeed: maxSpeed}
}

// gravityMover pulls a projectile down the screen (toward world +Y) with a
// constant per-tick acceleration, so anything launched outward or upward arcs
// back down like a thrown object. Used by cosmetic junk emitters (coffee spray,
// popped toast, dropped rubber ducks, firework shells); it ignores enemies.
type gravityMover struct {
	gravity  float64 // downward acceleration per tick (toward +Y)
	maxSpeed float64 // speed cap (0 = uncapped)
}

// NewGravityMover returns a mover that accelerates a projectile downward each
// tick. gravity is the per-tick downward acceleration, maxSpeed the cap (0 =
// none).
func NewGravityMover(gravity, maxSpeed float64) ProjectileMover {
	return gravityMover{gravity: gravity, maxSpeed: maxSpeed}
}

func (m gravityMover) Steer(p *Projectile, w *World) {
	p.Vel.Y += m.gravity
	if m.maxSpeed > 0 {
		if s := p.Vel.Abs(); s > m.maxSpeed {
			p.Vel = p.Vel.Multiply(m.maxSpeed / s)
		}
	}
}

func (m riseMover) Steer(p *Projectile, w *World) {
	p.Vel.Y -= m.lift
	// Phase the sway by the projectile's spawn X so balloons don't sway in unison.
	p.Vel.X += math.Sin(float64(w.Tick)*0.15+p.Pos.X) * m.wobble
	if m.maxSpeed > 0 {
		if s := p.Vel.Abs(); s > m.maxSpeed {
			p.Vel = p.Vel.Multiply(m.maxSpeed / s)
		}
	}
}

func (m homingMover) Steer(p *Projectile, w *World) {
	if p.age < m.straight {
		return // boost-out phase: fly straight along the launch heading first
	}
	target := w.nearestEnemy(p.Pos, math.MaxFloat64)
	if target == nil {
		return
	}
	dir := target.Pos.Subtract(p.Pos)
	d := dir.Abs()
	if d == 0 {
		return
	}
	speed := m.maxSpeed
	if speed <= 0 {
		speed = p.Vel.Abs()
	}
	// Seek: steer toward the desired velocity (top speed, straight at the
	// target), but limit the change to `turn` so the path bends instead of
	// snapping. Subtracting the current velocity also bleeds off sideways drift,
	// so the missile converges onto the target instead of orbiting it.
	desired := dir.Multiply(speed / d)
	steer := desired.Subtract(p.Vel)
	if mag := steer.Abs(); m.turn > 0 && mag > m.turn {
		steer = steer.Multiply(m.turn / mag)
	}
	p.Vel = p.Vel.Add(steer)
	if m.maxSpeed > 0 {
		if sp := p.Vel.Abs(); sp > m.maxSpeed {
			p.Vel = p.Vel.Multiply(m.maxSpeed / sp)
		}
	}
}
