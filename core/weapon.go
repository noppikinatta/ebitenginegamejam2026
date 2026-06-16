package core

// WeaponStats are the concrete combat numbers a weapon fires with. They are
// always derived from the weapon's current energy via StatsFromEnergy, so the
// upcoming wiring-tree ("Disconnect") mechanic can change a weapon's behaviour
// purely by changing how much energy is routed to it — without touching the
// firing logic in World.
type WeaponStats struct {
	Damage          float64
	FireInterval    int     // ticks between shots (lower = faster)
	ProjectileSpeed float64 // px per tick
	Range           float64 // px; only enemies within range are targeted
}

// Weapon is a single auto-firing armament bolted to the tank.
type Weapon struct {
	Name     string
	Energy   float64 // energy currently routed to this weapon
	cooldown int     // ticks remaining until the next shot
}

func NewWeapon(name string, energy float64) *Weapon {
	return &Weapon{Name: name, Energy: energy}
}

// StatsFromEnergy maps routed energy onto concrete combat numbers. This is the
// single integration seam for the wiring-tree/Disconnect feature: more energy
// means more damage, faster fire and longer range.
func (w *Weapon) StatsFromEnergy() WeaponStats {
	e := w.Energy
	if e < 0 {
		e = 0
	}

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
