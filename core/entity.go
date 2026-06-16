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
	invuln      int     // i-frame ticks remaining after taking contact damage
}

// Enemy chases the player and deals contact damage.
type Enemy struct {
	Pos     geom.PointF
	HP      float64
	Speed   float64
	Radius  float64
	Damage  float64
	XPValue float64
	alive   bool
}

// Projectile is fired by a weapon toward an enemy.
type Projectile struct {
	Pos    geom.PointF
	Vel    geom.PointF
	Damage float64
	Radius float64
	Life   int // ticks remaining before it expires
	alive  bool
}

// Gem drops from a dead enemy and grants XP when collected.
type Gem struct {
	Pos   geom.PointF
	Value float64
	alive bool
}
