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

// WeaponStats are the concrete combat numbers derived from a weapon's energy.
// Everything flows through StatsFromEnergy so the wiring-tree (Disconnect)
// mechanic can tune weapon behaviour purely by changing energy routing.
type WeaponStats struct {
	Damage          float64
	FireInterval    int     // ticks between shots (lower = faster)
	ProjectileSpeed float64 // px per tick; unused by KindLaser
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
	Energy        float64      // assigned by the Turret power solver; do not set directly
	TileIdx       hexmap.Index // the turret tile this weapon sits on; set by ActiveWeapons
	Level         int          // doctor upgrade level; each +1 multiplies Damage by WeaponParams.LevelMult
	cooldown      int
	beamTicksLeft int // KindLaser: ticks remaining in current beam burst
}

func NewWeapon(name string, energy float64, kind WeaponKind) *Weapon {
	return &Weapon{Name: name, Energy: energy, Kind: kind}
}

// IsBeamActive reports whether this weapon is currently emitting a laser beam.
func (w *Weapon) IsBeamActive() bool { return w.beamTicksLeft > 0 }

// StatsFromEnergy maps routed energy onto combat numbers using the weapon's
// balance params (supplied by the caller from the injected Config, keyed by
// Kind). The energy→stats curve is the only coupling point between the power
// solver and combat behaviour. Each weapon Level multiplies Damage by
// p.LevelMult.
func (w *Weapon) StatsFromEnergy(p WeaponParams) WeaponStats {
	e := w.Energy
	if e < 0 {
		e = 0
	}
	interval := int(p.BaseInterval - e*p.EnergyInterval)
	if interval < p.MinInterval {
		interval = p.MinInterval
	}
	stats := WeaponStats{
		Damage:       p.BaseDamage + e*p.EnergyDamage,
		FireInterval: interval,
		Range:        p.BaseRange + e*p.EnergyRange,
	}
	if w.Kind == KindLaser {
		stats.BeamLength = p.BeamBaseLength + e*p.BeamEnergyLength
		stats.BeamWidth = p.BeamBaseWidth + e*p.BeamEnergyWidth
		stats.BeamDuration = int(p.BeamBaseDuration + e*p.BeamEnergyDuration)
	} else {
		stats.ProjectileSpeed = p.ProjSpeed
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
