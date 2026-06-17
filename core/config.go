package core

// Config aggregates every balance/tuning value the simulation consumes. core
// defines the SHAPE of the configuration but deliberately holds no concrete
// numbers: the values are supplied by an outer layer (the data package) and
// injected through NewWorld. The dependency therefore points inward
// (data -> core), never core -> data.
type Config struct {
	StartingNippers     int     // nippers the player begins every run with
	MaxTurretTiles      int     // soft cap on turret size; forces non-tile offers
	CandlestickInterval int     // ticks between candlestick spawns
	XPToNextGrowth      float64 // XPToNext multiplier applied each level-up

	Player       Player                       // starting-stat template (Pos/Weapons/Facing/Nippers set by NewWorld)
	Pickup       PickupRanges                 // gem/nipper magnet+collect behaviour
	Spawn        SpawnSpec                    // enemy/candlestick placement and timing
	Doctor       DoctorSpec                   // level-up offer balance
	EnemyScaling EnemyScaling                 // basic enemy stats and HP scaling
	Candlestick  Enemy                        // candlestick template (Pos/alive set on spawn)
	TurretGen    TurretGenConfig              // random starting-turret generation params
	Weapons      map[WeaponKind]WeaponParams  // per-kind weapon stat curves
}

// WeaponParams contains the balance numbers for one weapon kind.
//
// Formulae used by Weapon.StatsFromEnergy:
//
//	Damage       = (BaseDamage + e×EnergyDamage) × LevelMult^Level
//	FireInterval = max(MinInterval, int(BaseInterval − e×EnergyInterval))
//	Range        = BaseRange + e×EnergyRange
//
// Laser-only fields (BeamBase*, BeamEnergy*) are zero for projectile weapons;
// ProjSpeed is zero for KindLaser.
type WeaponParams struct {
	BaseDamage     float64
	EnergyDamage   float64
	BaseInterval   float64
	EnergyInterval float64
	MinInterval    int
	ProjSpeed      float64
	BaseRange      float64
	EnergyRange    float64
	// Laser-only.
	BeamBaseLength     float64
	BeamEnergyLength   float64
	BeamBaseWidth      float64
	BeamEnergyWidth    float64
	BeamBaseDuration   float64
	BeamEnergyDuration float64
	// LevelMult is the Damage multiplier applied per doctor upgrade Level.
	LevelMult float64
}

// PickupRanges are shared by XP gems and nipper pickups.
type PickupRanges struct {
	PickupDist  float64 // collect on contact within this distance
	MagnetDist  float64 // start moving toward player within this distance
	MagnetSpeed float64 // px per tick toward the player
}

// SpawnSpec controls enemy and candlestick spawn placement and timing.
type SpawnSpec struct {
	EnemyDist          float64 // fixed spawn distance from player
	EnemyBaseInterval  int     // ticks between spawns at tick 0
	EnemyMinInterval   int     // floor for spawn interval (faster cap)
	EnemyIntervalDecay int     // game ticks that reduce spawn interval by 1
	CandleDist         float64 // minimum candlestick distance from player
	CandleDistRange    float64 // random extra distance beyond CandleDist
}

// DoctorSpec controls the balance of the three level-up offer types:
//   - Nippers: spare tile cuts.
//   - Weapon upgrade: selected weapons gain +1 Level.
//   - Tile bundle: 1-MaxBundleTiles new tiles, each 50% weapon / 50% junk.
type DoctorSpec struct {
	NipperChance   float64 // probability of a nipper offer (evaluated first)
	UpgradeChance  float64 // cumulative: upgrade if r < UpgradeChance after nipper check
	NipperMin      int     // minimum nippers per nipper offer
	NipperMax      int     // maximum nippers per nipper offer
	MaxUpgrades    int     // max weapons upgraded per upgrade offer
	MaxBundleTiles int     // max tiles added per tile-bundle offer
}

// EnemyScaling holds the basic chasing enemy's stats. Only HP scales with time
// (HP = HPBase + tick×HPPerTick); the other fields are constant per enemy.
type EnemyScaling struct {
	HPBase    float64
	HPPerTick float64
	Speed     float64
	Radius    float64
	Damage    float64
	XPValue   float64
}
