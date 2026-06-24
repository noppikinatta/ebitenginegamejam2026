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
const (
	SpriteBalloon  = "proj_balloon"
	SpriteSteam    = "proj_steam"
	SpriteToast    = "proj_toast"
	SpriteNote     = "proj_note"
	SpriteDuck     = "proj_duck"
	SpriteFirework = "proj_firework"
)

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

// coffeeEmitter (the Coffee Maker junk) puffs out clouds of steam that drift
// gently up the screen with a sideways wobble.
var coffeeEmitter = EmitterSpec{
	Interval: 70,
	Aim:      EmitUp,
	Speed:    0.5,
	Life:     120,
	Radius:   6,
	Sprite:   SpriteSteam,
	Mover:    NewRiseMover(0.02, 0.04, 1.2),
}

// toasterEmitter pops a slice of toast straight up that arcs back down under
// gravity.
var toasterEmitter = EmitterSpec{
	Interval: 120,
	Aim:      EmitUp,
	Speed:    2.2,
	Life:     160,
	Radius:   7,
	Sprite:   SpriteToast,
	Mover:    NewGravityMover(0.04, 4),
}

// musicBoxEmitter drifts musical notes outward with a gentle sway (a riseMover
// with no lift, so the note keeps its outward drift while wobbling).
var musicBoxEmitter = EmitterSpec{
	Interval: 60,
	Aim:      EmitOutward,
	Speed:    0.5,
	Life:     200,
	Radius:   8,
	Sprite:   SpriteNote,
	Mover:    NewRiseMover(0, 0.015, 1.2),
}

// duckEmitter scatters short-lived rubber ducks in random directions that drop
// to the ground around the tank.
var duckEmitter = EmitterSpec{
	Interval: 100,
	Aim:      EmitRandom,
	Speed:    1.0,
	Life:     90,
	Radius:   8,
	Sprite:   SpriteDuck,
	Mover:    NewGravityMover(0.05, 4),
}

// fireworksEmitter launches a shell straight up that bursts into a cosmetic
// (0-damage) explosion when it expires.
var fireworksEmitter = EmitterSpec{
	Interval:      150,
	Aim:           EmitUp,
	Speed:         2.5,
	Life:          70,
	Radius:        6,
	Sprite:        SpriteFirework,
	Mover:         NewGravityMover(0.02, 4),
	ExplodeRadius: 40,
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
			Firework:      e.spec.ExplodeRadius > 0, // only firework shells burst
			PassThrough:   true,                     // cosmetic: never interacts with enemies
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
