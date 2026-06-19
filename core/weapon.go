package core

import (
	"math"
	"math/rand"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// WeaponKind determines the firing pattern and stat scaling of a weapon.
type WeaponKind int

const (
	KindCannon  WeaponKind = iota // balanced auto-fire
	KindShotgun                   // 4-pellet spread, short range
	KindSniper                    // single high-damage shot, very long range
	KindLaser                     // sustained beam, DPS, penetrates all enemies in path
	KindGatling                   // forward-only staggered burst of random-spread pellets
	KindGrenade                   // outward-only lobbed shell that explodes (AoE) on expiry
	KindCIWS                      // short-range point defence: holds fire until a target is in range, then a burst
	KindMissile                   // long-range homing shell; contact damage plus a small explosion on expiry
)

func (k WeaponKind) String() string {
	switch k {
	case KindShotgun:
		return "Shotgun"
	case KindSniper:
		return "Sniper"
	case KindLaser:
		return "Laser"
	case KindGatling:
		return "Gatling"
	case KindGrenade:
		return "Grenade"
	case KindCIWS:
		return "CIWS"
	case KindMissile:
		return "Missile"
	default:
		return "Cannon"
	}
}

// WeaponStats are the concrete combat numbers a weapon fires with. They flow
// through Weapon.Stats, where the turret-wide fire-rate multiplier (the
// Disconnect mechanic's power expression) modulates the fire interval.
type WeaponStats struct {
	Damage          float64
	ProjectileSpeed float64 // px per tick; unused by KindLaser
	ProjLife        int     // ticks a projectile lives before expiring; unused by KindLaser
	ProjRadius      float64 // projectile collision radius; unused by KindLaser
	ExplodeRadius   float64 // >0: projectile explodes on expiry, dealing ExplodeDamage in this radius
	ExplodeDamage   float64
	Range           float64 // only enemies within this distance are targeted
	// Laser-only fields (zero for projectile weapons).
	BeamLength   float64
	BeamWidth    float64
	BeamDuration int // ticks the beam stays active per activation
}

// Weapon is a single auto-firing armament mounted on a turret tile.
type Weapon struct {
	Name          string
	Kind          WeaponKind
	TileIdx       hexmap.Index // the turret tile this weapon sits on; set by ActiveWeapons
	Level         int          // doctor upgrade level; each +1 multiplies Damage by WeaponParams.LevelMult
	fireProgress  float64      // accumulator: advances by fireIncrement each tick, fires at BaseInterval
	pelletsLeft   int          // projectiles still to emit in the current shot (for staggered bursts)
	pelletTimer   int          // ticks until the next staggered pellet
	beamTicksLeft int          // KindLaser: ticks remaining in current beam burst
	beamAngle     float64      // KindLaser: world angle the current burst points when no enemy is in range
	aimRender     float64      // smoothed world aim angle, for drawing the barrel pointing where it fires
	aimOffset     float64      // AimLockOn: last aim angle relative to the tank's facing; held when no target so the barrel freezes relative to the tank
}

func NewWeapon(name string, kind WeaponKind) *Weapon {
	// Rest the barrel pointing "up" (forward) so it doesn't swing in from angle 0
	// on the first tick.
	return &Weapon{Name: name, Kind: kind, aimRender: -math.Pi / 2}
}

// RenderAngle is the smoothed world-space angle the weapon's barrel should be
// drawn pointing along (toward its target for lock-on weapons, forward/outward
// otherwise). Updated each tick by the simulation; used by the scene only.
func (w *Weapon) RenderAngle() float64 { return w.aimRender }

// IsBeamActive reports whether this weapon is currently emitting a laser beam.
func (w *Weapon) IsBeamActive() bool { return w.beamTicksLeft > 0 }

// fireIncrement is how far a weapon's fire accumulator advances per tick at the
// given turret fire-rate multiplier. It equals the multiplier, capped so the
// effective interval can never drop below MinInterval (i.e. inc ≤
// BaseInterval/MinInterval). Non-positive multipliers yield 0 (no firing).
func fireIncrement(p WeaponParams, fireMult float64) float64 {
	if fireMult <= 0 {
		return 0
	}
	inc := fireMult
	if p.MinInterval > 0 {
		if maxInc := p.BaseInterval / float64(p.MinInterval); inc > maxInc {
			inc = maxInc
		}
	}
	return inc
}

// Stats maps the weapon's balance params onto concrete combat numbers. Fire
// cadence is NOT here — it is driven by the fire accumulator (see fireIncrement
// and World.updateWeapons), which reads BaseInterval/MinInterval and the turret
// fire-rate multiplier directly. Damage = BaseDamage × LevelMult^Level; range,
// projectile and beam geometry are fixed by the params.
func (w *Weapon) Stats(p WeaponParams) WeaponStats {
	stats := WeaponStats{
		Damage: p.BaseDamage,
		Range:  p.BaseRange,
	}
	if w.Kind == KindLaser {
		stats.BeamLength = p.BeamBaseLength
		stats.BeamWidth = p.BeamBaseWidth
		stats.BeamDuration = int(p.BeamBaseDuration)
	} else {
		stats.ProjectileSpeed = p.ProjSpeed
		stats.ProjRadius = p.ProjRadius
		stats.ExplodeRadius = p.ExplodeRadius
		stats.ExplodeDamage = p.ExplodeDamage
		if p.ProjSpeed > 0 {
			stats.ProjLife = int(math.Round(p.ProjMaxDist / p.ProjSpeed))
		}
	}
	if w.Level > 0 {
		mult := math.Pow(p.LevelMult, float64(w.Level))
		stats.Damage *= mult
		stats.ExplodeDamage *= mult
	}
	return stats
}

// pelletCount returns how many projectiles a single shot emits (≥1).
func pelletCount(p WeaponParams) int {
	if p.Pellets > 1 {
		return p.Pellets
	}
	return 1
}

// pelletOffset returns the angular offset (radians) for pellet i of n in a shot.
// With SpreadRandom the offset is uniform random in ±SpreadRad (rng required);
// otherwise pellets are spread evenly across ±SpreadRad by index.
func pelletOffset(p WeaponParams, i, n int, rng *rand.Rand) float64 {
	if p.SpreadRad <= 0 {
		return 0
	}
	if p.SpreadRandom {
		if rng == nil {
			return 0
		}
		return (rng.Float64()*2 - 1) * p.SpreadRad
	}
	if n <= 1 {
		return 0
	}
	return -p.SpreadRad + 2*p.SpreadRad*float64(i)/float64(n-1)
}
