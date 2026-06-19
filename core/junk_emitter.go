package core

import (
	"math"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// EmitAim selects the direction a junk emitter fires its projectile.
type EmitAim int

const (
	EmitUp      EmitAim = iota // toward the top of the screen (world -Y)
	EmitOutward                // radially outward through the junk's tile
	EmitRandom                 // a random direction
)

// Projectile sprite keys for junk emitters. Core stays Ebiten-free, so these are
// plain strings; the asset layer provides matching images and the scene draws
// Projectile.Sprite directly.
const SpriteBalloon = "proj_balloon"

// EmitterSpec is the firing configuration of a junk device that periodically
// spits out a projectile. The projectiles are intentionally useless: zero
// damage and pass-through (they never touch enemies), fired at a fixed cadence
// that does NOT scale with turret power. The junk still just dilutes power; the
// emission is pure flavour.
type EmitterSpec struct {
	Interval int     // fixed ticks between emissions
	Aim      EmitAim // direction the projectile is launched
	Speed    float64 // initial projectile speed
	Life     int     // projectile lifetime in ticks
	Radius   float64 // projectile draw/collision radius (collision is moot: pass-through)
	Sprite   string  // image key for the projectile
	Mover    ProjectileMover
	// ExplodeRadius>0 queues a cosmetic (0-damage) burst when the projectile
	// expires — for firework-style junk.
	ExplodeRadius float64
}

// junkEmitter is the per-tile runtime state of an emitting junk: its spec plus a
// firing accumulator. A pointer is stored on the Junk component so value copies
// share the same timer (mirroring WeaponComponent.Weapon).
type junkEmitter struct {
	spec  EmitterSpec
	timer int
}

// balloonEmitter floats wobbling balloons up the screen every ~1.5s.
var balloonEmitter = EmitterSpec{
	Interval: 90,
	Aim:      EmitUp,
	Speed:    0.6,
	Life:     180,
	Radius:   8,
	Sprite:   SpriteBalloon,
	Mover:    NewRiseMover(0.03, 0.02, 1.6),
}

// updateJunkEmitters advances every connected emitting junk and spawns its
// cosmetic projectiles. Called once per tick from World.Update.
func (w *World) updateJunkEmitters() {
	if w.turret == nil {
		return
	}
	facing := w.Player.FacingAngle
	for _, et := range w.turret.ActiveEmitters() {
		e := et.emitter
		e.timer++
		if e.timer < e.spec.Interval {
			continue
		}
		e.timer = 0

		muzzle := w.Player.Pos.Add(MuzzleOffset(et.idx, facing))
		angle := w.emitAngle(e.spec.Aim, et.idx, facing)
		w.Projectiles = append(w.Projectiles, &Projectile{
			Pos:           muzzle,
			Vel:           geom.PointFFromPolar(e.spec.Speed, angle),
			Radius:        e.spec.Radius,
			Life:          e.spec.Life,
			ExplodeRadius: e.spec.ExplodeRadius,
			PassThrough:   true, // cosmetic: never interacts with enemies
			Mover:         e.spec.Mover,
			Sprite:        e.spec.Sprite,
			alive:         true,
		})
	}
}

// emitAngle resolves the launch angle for an emitter's aim mode.
func (w *World) emitAngle(aim EmitAim, idx hexmap.Index, facing float64) float64 {
	switch aim {
	case EmitOutward:
		if off := MuzzleOffset(idx, facing); off.Abs() > 0 {
			return off.Angle()
		}
		return facing
	case EmitRandom:
		return w.rng.Float64() * 2 * math.Pi
	default: // EmitUp
		return -math.Pi / 2 // world up (toward the top of the screen)
	}
}
