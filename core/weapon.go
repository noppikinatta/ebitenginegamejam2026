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
	TileIdx       hexmap.Index // the turret tile this weapon sits on; set by ActiveWeapons
	Level         int          // doctor upgrade level; each +1 multiplies Damage by WeaponParams.LevelMult
	cooldown      int
	beamTicksLeft int // KindLaser: ticks remaining in current beam burst
}

func NewWeapon(name string, kind WeaponKind) *Weapon {
	return &Weapon{Name: name, Kind: kind}
}

// IsBeamActive reports whether this weapon is currently emitting a laser beam.
func (w *Weapon) IsBeamActive() bool { return w.beamTicksLeft > 0 }

// Stats maps the weapon's balance params onto concrete combat numbers. The
// turret's power is expressed as a single fire-rate multiplier (derived from the
// turret tile count, the same for every weapon) and affects ONLY the fire
// interval: FireInterval = round(BaseInterval / fireMult), clamped to
// MinInterval. Damage, range and beam geometry are fixed by the params;
// each weapon Level multiplies Damage by p.LevelMult. A higher fireMult means
// a shorter interval, i.e. more frequent fire.
func (w *Weapon) Stats(p WeaponParams, fireMult float64) WeaponStats {
	if fireMult <= 0 {
		fireMult = 1
	}
	interval := int(math.Round(p.BaseInterval / fireMult))
	if interval < p.MinInterval {
		interval = p.MinInterval
	}
	stats := WeaponStats{
		Damage:       p.BaseDamage,
		FireInterval: interval,
		Range:        p.BaseRange,
	}
	if w.Kind == KindLaser {
		stats.BeamLength = p.BeamBaseLength
		stats.BeamWidth = p.BeamBaseWidth
		stats.BeamDuration = int(p.BeamBaseDuration)
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
