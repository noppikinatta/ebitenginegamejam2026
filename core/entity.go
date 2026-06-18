package core

import "github.com/noppikinatta/ebitenginegamejam2026/geom"

// Player is the tank controlled by the player.
type Player struct {
	Pos         geom.PointF
	HP          float64
	MaxHP       float64
	Speed       float64 // px per tick
	Radius      float64
	Weapons     []*Weapon
	Level       int
	XP          float64
	XPToNext    float64
	FacingAngle float64 // radians; direction the tank/turret faces. -pi/2 = straight up = forward (default)
	Nippers     int     // plastic-model nippers: consumed to cut a turret tile mid-combat
	invuln      int     // i-frame ticks remaining after taking contact damage
}

// Enemy chases the player and deals contact damage. A candlestick is a special
// stationary enemy (Speed 0, no contact damage) that drops a nipper when broken.
type Enemy struct {
	Pos         geom.PointF
	HP          float64
	Speed       float64
	Radius      float64
	Damage      float64
	XPValue     float64
	DropsNipper bool // candlestick: spawns a nipper pickup on death
	alive       bool
}

// Projectile is fired by a weapon toward an enemy.
type Projectile struct {
	Pos    geom.PointF
	Vel    geom.PointF
	Damage float64
	Radius float64
	Life   int // ticks remaining before it expires
	// ExplodeRadius>0 marks an explosive shell: it ignores contact and, on expiry,
	// deals ExplodeDamage to every enemy within ExplodeRadius of its position.
	ExplodeRadius float64
	ExplodeDamage float64
	alive         bool
}

// Gem drops from a dead enemy and grants XP when collected.
type Gem struct {
	Pos   geom.PointF
	Value float64
	alive bool
}

// Pickup is a dropped nipper that grants one nipper when collected.
type Pickup struct {
	Pos   geom.PointF
	alive bool
}

// BeamView is a read-only snapshot of an active laser beam for the scene layer to draw.
type BeamView struct {
	Origin geom.PointF // world-space muzzle position
	Dir    geom.PointF // unit vector toward the target
	Length float64
	Width  float64
}
