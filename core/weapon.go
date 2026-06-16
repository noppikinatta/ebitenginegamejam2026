package core

import "github.com/noppikinatta/ebitenginegamejam2026/hexmap"

// WeaponKind determines the firing pattern and stat scaling of a weapon.
type WeaponKind int

const (
	KindCannon  WeaponKind = iota // balanced auto-fire
	KindShotgun                   // 3-projectile spread, short range
	KindSniper                    // single high-damage shot, very long range
)

func (k WeaponKind) String() string {
	switch k {
	case KindShotgun:
		return "Shotgun"
	case KindSniper:
		return "Sniper"
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
	ProjectileSpeed float64 // px per tick
	Range           float64 // only enemies within this distance are targeted
}

// Weapon is a single auto-firing armament mounted on a turret tile.
type Weapon struct {
	Name     string
	Kind     WeaponKind
	Energy   float64      // assigned by the Turret power solver; do not set directly
	TileIdx  hexmap.Index // the turret tile this weapon sits on; set by ActiveWeapons
	cooldown int
}

func NewWeapon(name string, energy float64, kind WeaponKind) *Weapon {
	return &Weapon{Name: name, Energy: energy, Kind: kind}
}

// StatsFromEnergy maps routed energy onto combat numbers. Stat curves vary by
// weapon kind so each type has a distinct identity even at the same energy level.
func (w *Weapon) StatsFromEnergy() WeaponStats {
	e := w.Energy
	if e < 0 {
		e = 0
	}
	switch w.Kind {
	case KindShotgun:
		interval := int(28 - e*2)
		if interval < 8 {
			interval = 8
		}
		return WeaponStats{
			Damage:          3 + e*1.5,
			FireInterval:    interval,
			ProjectileSpeed: 5,
			Range:           150 + e*10,
		}
	case KindSniper:
		interval := int(120 - e*7)
		if interval < 20 {
			interval = 20
		}
		return WeaponStats{
			Damage:          20 + e*8,
			FireInterval:    interval,
			ProjectileSpeed: 10,
			Range:           400 + e*40,
		}
	default: // KindCannon
		interval := int(45 - e*4)
		if interval < 6 {
			interval = 6
		}
		return WeaponStats{
			Damage:          5 + e*3,
			FireInterval:    interval,
			ProjectileSpeed: 6,
			Range:           220 + e*20,
		}
	}
}

// ProjectileOffsets returns angular offsets (radians) relative to the target
// direction for each projectile fired per shot. Shotgun fires a 3-way spread.
func (w *Weapon) ProjectileOffsets() []float64 {
	if w.Kind == KindShotgun {
		return []float64{-0.25, 0, 0.25}
	}
	return []float64{0}
}
