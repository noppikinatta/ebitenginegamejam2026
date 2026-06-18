package core

import (
	"math"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// WeaponKind determines the firing pattern and stat scaling of a weapon.
type WeaponKind int

const (
	KindCannon  WeaponKind = iota // balanced auto-fire
	KindShotgun                   // 3-projectile spread, short range
	KindSniper                    // single high-damage shot, very long range
	KindLaser                     // sustained beam, DPS, penetrates all enemies in path
)

func (k WeaponKind) String() string {
	switch k {
	case KindShotgun:
		return "Shotgun"
	case KindSniper:
		return "Sniper"
	case KindLaser:
		return "Laser"
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
	beamTicksLeft int          // KindLaser: ticks remaining in current beam burst
	beamAngle     float64      // KindLaser: world angle the current burst points when no enemy is in range
}

func NewWeapon(name string, kind WeaponKind) *Weapon {
	return &Weapon{Name: name, Kind: kind}
}

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
		if p.ProjSpeed > 0 {
			stats.ProjLife = int(math.Round(p.ProjMaxDist / p.ProjSpeed))
		}
	}
	if w.Level > 0 {
		stats.Damage *= math.Pow(p.LevelMult, float64(w.Level))
	}
	return stats
}

// ProjectileOffsets returns angular offsets (radians) relative to the target
// direction for each projectile fired per shot. Shotgun fires a 3-way spread.
// Returns nil for KindLaser (beams are not projectiles).
func (w *Weapon) ProjectileOffsets() []float64 {
	switch w.Kind {
	case KindShotgun:
		return []float64{-0.25, 0, 0.25}
	case KindLaser:
		return nil
	default:
		return []float64{0}
	}
}
