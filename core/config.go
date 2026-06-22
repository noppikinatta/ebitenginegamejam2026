package core

// Config aggregates every balance/tuning value the simulation consumes. core
// defines the SHAPE of the configuration but deliberately holds no concrete
// numbers: the values are supplied by an outer layer (the data package) and
// injected through NewWorld. The dependency therefore points inward
// (data -> core), never core -> data.
type Config struct {
	StartingNippers        int     // nippers the player begins every run with
	MaxTurretTiles         int     // soft cap on turret size; forces non-tile offers
	CandlestickInterval    int     // ticks between candlestick spawns
	HeartDropChance        float64 // chance a candlestick drops a heart (HP) instead of a nipper
	HeartHeal              float64 // HP restored when a heart pickup is collected
	XPToNextGrowth         float64 // XPToNext multiplier applied each level-up
	CapacitorFireRateBonus float64 // fire-rate multiplier added per connected Capacitor equipment

	Player Player       // starting-stat template (Pos/Weapons/Facing/Nippers set by NewWorld)
	Pickup PickupRanges // gem/nipper magnet+collect behaviour
	Spawn  SpawnSpec    // enemy/candlestick placement and timing

	// Enemy roster and director. Zako (trash) enemies come in several kinds
	// (EnemyKinds), chosen each spawn by the time-gated SpawnPhases weights and
	// emitted in packs. HPDoublingTicks scales every zako's HP with elapsed time.
	// Bosses spawn once each at their scheduled ticks; killing the Final boss
	// clears the run.
	EnemyKinds      map[EnemyKind]EnemyStats
	HPDoublingTicks float64
	SpawnPhases     []SpawnPhase
	Bosses          []BossSpec

	Doctor      DoctorSpec                  // level-up offer balance
	Candlestick Enemy                       // candlestick template (Pos/alive set on spawn)
	TurretGen   TurretGenConfig             // random starting-turret generation params
	Weapons     map[WeaponKind]WeaponParams // per-kind weapon stat curves
	PowerCurve  []PowerPoint                // tile-count → fire-rate multiplier breakpoints
}

// PowerPoint is one breakpoint of the power curve: at Tiles connected consumer
// tiles, weapons fire at Mult× their base rate. The curve is a list of these
// points sorted ascending by Tiles; PowerMultiplier interpolates linearly
// between adjacent points and clamps to the end points outside the range. Fewer
// tiles → higher Mult, so cutting tiles re-concentrates power into faster fire.
type PowerPoint struct {
	Tiles int
	Mult  float64
}

// WeaponParams contains the balance numbers for one weapon kind.
//
// Fire cadence uses an accumulator: each tick a weapon's progress advances by
// the turret fire-rate multiplier (capped to BaseInterval/MinInterval) and a
// shot fires when progress reaches BaseInterval. So the average interval is
// BaseInterval/fireMult, floored at MinInterval. Other values:
//
//	Damage   = BaseDamage × LevelMult^Level
//	Range    = BaseRange  (lock-on radius for aiming)
//	ProjLife = round(ProjMaxDist / ProjSpeed)  (projectile lifetime in ticks)
//
// Laser-only fields (BeamBase*) are zero for projectile weapons;
// ProjSpeed/ProjMaxDist/ProjRadius are zero for KindLaser.
type WeaponParams struct {
	BaseDamage   float64
	BaseInterval float64 // accumulator threshold; average ticks between shots at fireMult=1
	MinInterval  int     // floor on the effective interval (caps the fastest fire rate)
	ProjSpeed    float64
	ProjMaxDist  float64 // max travel distance; projectile dies after ProjMaxDist/ProjSpeed ticks
	ProjRadius   float64 // projectile collision radius
	BaseRange    float64
	// Per-shot pellets. Pellets>1 emits multiple projectiles per shot; SpreadRad
	// is the half-angle of the spread (0 = none); SpreadRandom picks each pellet's
	// angle randomly in ±SpreadRad (else evenly spaced). BurstGap>0 staggers the
	// pellets that many ticks apart (a stream) instead of firing them at once.
	Pellets      int
	SpreadRad    float64
	SpreadRandom bool
	BurstGap     int
	// Aim selects how the weapon points (lock-on / forward / outward).
	Aim AimMode
	// HoldWhenNoTarget keeps a full accumulator charged (instead of firing into
	// empty space) until an enemy enters range. Used by interception weapons
	// (CIWS); other weapons fire even with no target.
	HoldWhenNoTarget bool
	// Explosive projectiles: ExplodeRadius>0 makes a projectile deal ExplodeDamage
	// within ExplodeRadius when it expires. PassThrough makes it ignore contact so
	// it only detonates on expiry (the grenade); contact projectiles (missiles)
	// still hit enemies directly.
	ExplodeRadius float64
	ExplodeDamage float64
	PassThrough   bool
	// Mover steers each fired projectile (homing, drift). nil flies straight.
	Mover ProjectileMover
	// Projectile appearance. Sprite is the image key fired projectiles are drawn
	// with (empty falls back to the default bullet). ProjDrawW/ProjDrawH are the
	// draw footprint in px (independent of ProjRadius, which is collision only),
	// so non-square art like the missile (8×12) can be expressed. ProjFaceVelocity
	// rotates the sprite to point along its travel direction (for elongated art:
	// cannon shell, sniper dart, homing missile); round bullets leave it false.
	Sprite           string
	ProjDrawW        float64
	ProjDrawH        float64
	ProjFaceVelocity bool
	// Laser-only.
	BeamBaseLength   float64
	BeamBaseWidth    float64
	BeamBaseDuration float64
	// LevelMult is the Damage multiplier applied per doctor upgrade Level.
	LevelMult float64
}

// AimMode selects how a weapon chooses its firing direction.
type AimMode int

const (
	AimLockOn  AimMode = iota // aim at the nearest enemy in range, else forward
	AimForward                // always the tank's forward facing (never locks on)
	AimOutward                // always radially outward through the weapon's tile
)

// PickupRanges are shared by XP gems and nipper pickups.
type PickupRanges struct {
	PickupDist  float64 // collect on contact within this distance
	MagnetDist  float64 // start moving toward player within this distance
	MagnetSpeed float64 // px per tick toward the player
}

// SpawnSpec controls enemy and candlestick spawn placement. The spawn cadence
// and enemy mix are time-banded (see SpawnPhase).
type SpawnSpec struct {
	EnemyDist       float64 // fixed spawn distance from player
	CandleDist      float64 // minimum candlestick distance from player
	CandleDistRange float64 // random extra distance beyond CandleDist
}

// DoctorSpec controls the balance of the three level-up offer types:
//   - Nippers: spare tile cuts.
//   - Weapon upgrade: selected weapons gain +1 Level.
//   - Tile bundle: 1-MaxBundleTiles new tiles, each 50% weapon / 50% junk.
type DoctorSpec struct {
	NipperChance    float64 // probability of a nipper offer (evaluated first)
	UpgradeChance   float64 // cumulative: upgrade if r < UpgradeChance after nipper check
	NipperMin       int     // minimum nippers per nipper offer
	NipperMax       int     // maximum nippers per nipper offer
	MaxUpgrades     int     // max weapons upgraded per upgrade offer
	MaxBundleTiles  int     // max tiles added per tile-bundle offer
	CapacitorChance float64 // per bundle tile, probability it is a Capacitor (else weapon/junk)
}

// EnemyStats is the spawn template for one zako (trash) enemy kind. Only HP
// scales with time: HP = HPBase × 2^(tick / Config.HPDoublingTicks). The rest is
// constant per enemy. A spawn emits a cluster of PackMin..PackMax of the kind.
type EnemyStats struct {
	HPBase  float64
	Speed   float64
	Radius  float64
	Damage  float64
	XPValue float64
	PackMin int // smallest cluster spawned at once (>=1)
	PackMax int // largest cluster (>=PackMin)
}

// KindWeight is one entry in a spawn phase's weighted table.
type KindWeight struct {
	Kind   EnemyKind
	Weight int
}

// SpawnPhase is one time band: while the game tick is below UntilTick, enemies
// spawn every Interval ticks with the given per-kind weights. Phases are listed
// in ascending UntilTick order; the last one should use a very large UntilTick to
// cover the rest of the run. Weights are an ordered slice (not a map) so spawning
// stays deterministic. This is the single place to tune how the enemy mix and
// cadence change over time.
type SpawnPhase struct {
	UntilTick int
	Interval  int // ticks between spawns while in this band
	Weights   []KindWeight
}

// Boss sprite keys. The asset layer provides matching PNGs (boss1.png …) and the
// scene draws Enemy.Sprite directly, mirroring the cosmetic junk sprite keys.
const (
	BossSprite1 = "boss1"
	BossSprite2 = "boss2"
	BossSprite3 = "boss3"
)

// BossSpec schedules a single boss to spawn once at AtTick. Bosses do not use
// time-based HP scaling (HP is fixed here). Killing the boss whose Final flag is
// set clears the run.
type BossSpec struct {
	AtTick  int
	Name    string
	HP      float64
	Speed   float64
	Radius  float64
	Damage  float64
	XPValue float64
	Final   bool
	Sprite  string // image key for this boss; empty falls back to the default boss sprite
}
