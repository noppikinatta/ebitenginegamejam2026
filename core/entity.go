package core

import "github.com/noppikinatta/ebitenginegamejam2026/geom"

// Player is the tank controlled by the player.
type Player struct {
	Pos         geom.PointF
	HP          float64
	MaxHP       float64
	Speed       float64 // movement-speed coefficient; effective px/tick = Speed × turret power multiplier (see World.PlayerSpeed)
	Radius      float64
	Weapons     []*Weapon
	Level       int
	XP          float64
	XPToNext    float64
	FacingAngle float64 // radians; direction the tank/turret faces. -pi/2 = straight up = forward (default)
	Nippers     int     // plastic-model nippers: consumed to cut a turret tile mid-combat
	invuln      int     // i-frame ticks remaining after taking contact damage
	repairTimer int     // ticks since the last repair-unit heal cycle
}

// EnemyKind identifies a zako (trash) enemy spawn template, used both to pick
// stats at spawn and to choose a sprite when drawing.
type EnemyKind int

const (
	EnemyGrunt   EnemyKind = iota // balanced chaser
	EnemySwarmer                  // fast, fragile, spawns in packs
	EnemyBrute                    // slow, tanky, big, hits hard
)

// Enemy chases the player and deals contact damage. A candlestick is a special
// stationary enemy (Speed 0, no contact damage) that drops a nipper when broken.
// Bosses are large scheduled enemies; killing the Final boss clears the run.
type Enemy struct {
	Pos         geom.PointF
	Kind        EnemyKind
	HP          float64
	MaxHP       float64 // spawn HP, for boss health bars
	Speed       float64
	Radius      float64
	Damage      float64
	XPValue     float64
	DropsNipper bool   // candlestick: spawns a nipper pickup on death
	IsBoss      bool   // scheduled boss: drawn large, shows a health bar
	Final       bool   // final boss: killing it clears the run
	Name        string // boss display name
	Sprite      string // explicit sprite-key override (bosses set their own art); empty falls back to the per-kind sprite
	alive       bool
}

// Projectile is fired by a weapon toward an enemy.
type Projectile struct {
	Pos    geom.PointF
	Vel    geom.PointF
	Damage float64
	Radius float64
	Life   int // ticks remaining before it expires
	// ExplodeRadius>0 makes the projectile deal ExplodeDamage to every enemy
	// within ExplodeRadius of its position when it expires.
	ExplodeRadius float64
	ExplodeDamage float64
	// PassThrough projectiles ignore contact with enemies (they only matter on
	// expiry, e.g. the grenade shell that detonates where it lands). Contact
	// projectiles (including missiles) die on the first enemy they touch.
	PassThrough bool
	// Mover steers the projectile each tick (homing, drifting). nil flies straight.
	Mover ProjectileMover
	// Sprite is the image key this projectile is drawn with; empty uses the
	// default bullet sprite. Junk emitters and weapons set it to their own art.
	Sprite string
	// DrawW/DrawH are the sprite's draw footprint in px (0 lets the scene pick a
	// default). FaceVelocity rotates the sprite to point along Vel (elongated art
	// like the cannon shell, sniper dart and homing missile).
	DrawW        float64
	DrawH        float64
	FaceVelocity bool
	alive        bool
}

// Explosion is a short-lived visual effect queued where an explosive shell
// detonates. It has no gameplay effect (the area damage is applied at spawn
// time); the scene draws it as a fading circle. Life counts down each tick, and
// alpha = Life/MaxLife. It is queued in World because the projectile that
// spawned it is gone by the time it should be drawn.
type Explosion struct {
	Pos     geom.PointF
	Radius  float64
	Life    int // ticks remaining
	MaxLife int // initial Life, for alpha = Life/MaxLife
}

// Gem drops from a dead enemy and grants XP when collected.
type Gem struct {
	Pos   geom.PointF
	Value float64
	alive bool
}

// PickupKind distinguishes what a dropped pickup grants when collected.
type PickupKind int

const (
	PickupNipper PickupKind = iota // grants one nipper (default)
	PickupHeart                    // restores HP
)

// Pickup is a dropped item a candlestick leaves behind: a nipper, or (rarely) a
// heart that restores HP. Kind selects which; the zero value is a nipper.
type Pickup struct {
	Pos   geom.PointF
	Kind  PickupKind
	alive bool
}

// BeamView is a read-only snapshot of an active laser beam for the scene layer to draw.
type BeamView struct {
	Origin geom.PointF // world-space muzzle position
	Dir    geom.PointF // unit vector toward the target
	Length float64
	Width  float64
}
