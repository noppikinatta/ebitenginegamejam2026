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
// rather than snapping its velocity straight at the target.
type homingMover struct {
	turn     float64 // max steering force applied toward the target each tick
	maxSpeed float64 // cruising / cap speed (0 = keep current speed)
}

// NewHomingMover returns a mover that homes onto the nearest enemy. Each tick it
// nudges the velocity toward "head at maxSpeed straight at the target" by at most
// turn, so a smaller turn gives a wider, lazier arc. With no enemies the
// projectile coasts along its current velocity.
func NewHomingMover(turn, maxSpeed float64) ProjectileMover {
	return homingMover{turn: turn, maxSpeed: maxSpeed}
}

func (m homingMover) Steer(p *Projectile, w *World) {
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
