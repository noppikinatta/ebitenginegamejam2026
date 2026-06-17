package data

// WeaponParams contains the balance numbers for one weapon kind.
//
// Formulae used by core.Weapon.StatsFromEnergy:
//
//	Damage       = (BaseDamage + e×EnergyDamage) × WeaponLevelMult^Level
//	FireInterval = max(MinInterval, int(BaseInterval − e×EnergyInterval))
//	Range        = BaseRange + e×EnergyRange
//
// Laser-only fields (BeamBase*, BeamEnergy*) are zero for projectile weapons.
type WeaponParams struct {
	BaseDamage     float64
	EnergyDamage   float64
	BaseInterval   float64
	EnergyInterval float64
	MinInterval    int
	ProjSpeed      float64 // projectile speed px/tick; 0 for KindLaser
	BaseRange      float64
	EnergyRange    float64
	// Laser-only.
	BeamBaseLength     float64
	BeamEnergyLength   float64
	BeamBaseWidth      float64
	BeamEnergyWidth    float64
	BeamBaseDuration   float64
	BeamEnergyDuration float64
}

// Weapon balance presets — one per WeaponKind (canonical order: Cannon, Shotgun,
// Sniper, Laser). Edit these to tune combat feel without touching game logic.
var (
	Cannon = WeaponParams{
		BaseDamage: 5, EnergyDamage: 3,
		BaseInterval: 45, EnergyInterval: 4, MinInterval: 6,
		ProjSpeed: 6, BaseRange: 220, EnergyRange: 20,
	}
	Shotgun = WeaponParams{
		BaseDamage: 3, EnergyDamage: 1.5,
		BaseInterval: 28, EnergyInterval: 2, MinInterval: 8,
		ProjSpeed: 5, BaseRange: 150, EnergyRange: 10,
	}
	Sniper = WeaponParams{
		BaseDamage: 20, EnergyDamage: 8,
		BaseInterval: 120, EnergyInterval: 7, MinInterval: 20,
		ProjSpeed: 10, BaseRange: 400, EnergyRange: 40,
	}
	Laser = WeaponParams{
		BaseDamage: 2, EnergyDamage: 0.8,
		BaseInterval: 90, EnergyInterval: 5, MinInterval: 15,
		BaseRange: 300, EnergyRange: 25,
		BeamBaseLength: 300, BeamEnergyLength: 25,
		BeamBaseWidth: 6, BeamEnergyWidth: 0.5,
		BeamBaseDuration: 30, BeamEnergyDuration: 4,
	}
)
