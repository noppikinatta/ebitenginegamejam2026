package core

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// State is the high-level state of a run.
type State int

const (
	StatePlaying  State = iota
	StateLevelUp        // awaiting the player's purge choice
	StateGameOver       // run ended in defeat (player died)
	StateCleared        // run won (final boss defeated)
)

// World holds all gameplay state for a single run. It has no Ebiten dependency
// so the simulation can be unit-tested in isolation.
type World struct {
	Player      *Player
	Enemies     []*Enemy
	Projectiles []*Projectile
	Gems        []*Gem
	Pickups     []*Pickup    // dropped nippers awaiting collection
	Explosions  []*Explosion // active explosion visual effects

	// SoundEvents are the sound effects triggered during the current tick.
	// The scene layer drains this after Update; it is reset each tick.
	SoundEvents []SoundEvent

	// DamageEvents are the hits landed during the current tick (for floating
	// damage numbers). Drained by the scene after Update; reset each tick.
	DamageEvents []DamageEvent

	// DeathEvents are the enemies that died this tick (for the scene's fade-out
	// effect). Drained by the scene after Update; reset each tick.
	DeathEvents []DeathEvent

	// Choices are the doctor offers shown while State==StateLevelUp.
	Choices []Upgrade

	State State
	Tick  int
	Kills int

	turret           *Turret
	pendingLevelUps  int
	candlestickTimer int
	magnetTimer      int
	spawnTimer       int
	bossesSpawned    int // index of the next boss in cfg.Bosses to spawn
	rng              *rand.Rand
	cfg              Config
}

// ActiveBoss returns the first alive boss enemy, or nil if none is on the field.
// The scene uses it to draw a boss health bar.
func (w *World) ActiveBoss() *Enemy {
	for _, e := range w.Enemies {
		if e.alive && e.IsBoss {
			return e
		}
	}
	return nil
}

// ActiveBosses returns every alive boss on the field, in slice (spawn) order, so
// the HUD can stack one health bar per boss when a mid-boss is still alive as the
// next boss spawns (otherwise a single shared bar would overlap).
func (w *World) ActiveBosses() []*Enemy {
	var bosses []*Enemy
	for _, e := range w.Enemies {
		if e.alive && e.IsBoss {
			bosses = append(bosses, e)
		}
	}
	return bosses
}

// Turret exposes the turret grid read-only so the scene layer can draw it.
func (w *World) Turret() *Turret { return w.turret }

// FireRateMultiplier is the turret-wide power multiplier applied to every
// weapon's fire interval (interval = baseInterval / multiplier). It is derived
// from the connected consumer tile count via the configured power curve, so
// cutting tiles raises it. Used by the simulation and by the HUD/gauge.
func (w *World) FireRateMultiplier() float64 {
	if w.turret == nil {
		return 1
	}
	base := PowerMultiplier(w.cfg.PowerCurve, w.turret.ConsumerTileCount())
	return base + w.turret.Modifiers().FireRateAdd
}

// PlayerSpeed is the tank's effective movement speed in px/tick: the player's
// speed coefficient scaled by the same turret-wide power multiplier that drives
// the weapons. A turret bloated with tiles dilutes the power, so the tank also
// crawls; cutting tiles back re-concentrates power and speeds it up again.
func (w *World) PlayerSpeed() float64 {
	return w.Player.Speed * (0.5 + w.FireRateMultiplier()/2)
}

// FireRateMultBounds returns the minimum and maximum fire-rate multiplier the
// power curve can produce, so the HUD can normalise the multiplier into a [0,1]
// gauge fill. Falls back to (1, 1) for an empty curve.
func (w *World) FireRateMultBounds() (min, max float64) {
	if len(w.cfg.PowerCurve) == 0 {
		return 1, 1
	}
	min = w.cfg.PowerCurve[0].Mult
	max = min
	for _, p := range w.cfg.PowerCurve {
		if p.Mult < min {
			min = p.Mult
		}
		if p.Mult > max {
			max = p.Mult
		}
	}
	return min, max
}

// NewWorld builds a fresh run. seed makes enemy spawning and turret generation
// deterministic for tests; pass time.Now().UnixNano() for real gameplay. cfg
// supplies every balance number (see Config); the data package provides the
// canonical values via data.NewConfig().
func NewWorld(seed int64, cfg Config) *World {
	rng := rand.New(rand.NewSource(seed))
	turret := GenerateTurret(cfg.TurretGen, rng)

	p := cfg.Player // copy the starting-stat template
	p.Pos = geom.PointF{}
	p.Weapons = turret.ActiveWeapons()
	p.FacingAngle = -math.Pi / 2
	p.Nippers = cfg.StartingNippers
	w := &World{
		Player: &p,
		State:  StatePlaying,
		turret: turret,
		rng:    rng,
		cfg:    cfg,
	}
	w.primeNewWeapons()
	return w
}

// primeNewWeapons seeds each not-yet-primed weapon's fire accumulator with a
// random 0-70% of its interval, so a freshly mounted weapon starts out of phase
// with the others (attacks feel scattered) instead of every weapon firing on
// the same tick. Call it wherever weapons are (re)assigned to the player; the
// primed flag keeps it idempotent, so cutting/adding tiles never re-randomizes
// weapons that are already mounted.
func (w *World) primeNewWeapons() {
	if w.rng == nil { // manually-built test worlds skip the random seeding
		return
	}
	// Collect the not-yet-primed weapons and seed them in a stable order (by tile
	// index). ActiveWeapons returns weapons in map-iteration order, which varies
	// per run; drawing from the shared rng in that order would make two same-seed
	// worlds diverge. A deterministic order keeps runs reproducible.
	var fresh []*Weapon
	for _, weapon := range w.Player.Weapons {
		if !weapon.primed {
			fresh = append(fresh, weapon)
		}
	}
	sort.Slice(fresh, func(i, j int) bool {
		a, b := fresh[i].TileIdx, fresh[j].TileIdx
		if a.X() != b.X() {
			return a.X() < b.X()
		}
		return a.Y() < b.Y()
	})
	for _, weapon := range fresh {
		weapon.fireProgress = w.rng.Float64() * 0.7 * w.cfg.Weapons[weapon.Kind].BaseInterval
		weapon.primed = true
	}
}

// Update advances the simulation by one tick. move is the desired movement
// direction (each axis in [-1,1]); it is normalised so diagonal is not faster.
// While State != StatePlaying the world is frozen; use ChooseUpgrade to resume.
func (w *World) Update(move geom.PointF) {
	// Clear last tick's sound events before the guard so the scene never
	// replays stale effects while the world is frozen (level-up, game over).
	w.SoundEvents = w.SoundEvents[:0]
	w.DamageEvents = w.DamageEvents[:0]
	w.DeathEvents = w.DeathEvents[:0]

	if w.State != StatePlaying {
		return
	}

	w.Tick++
	w.updatePlayer(move)
	w.repairPlayer()
	w.updateWeapons()
	w.updateJunkEmitters()
	w.updateBeams()
	w.updateExplosions() // age existing effects before new ones may spawn this tick
	w.updateProjectiles()
	w.updateEnemies()
	w.updateGems()
	w.updatePickups()
	w.spawnEnemies()
	w.spawnBosses()
	w.spawnCandlestick()
	w.spawnAutoMagnet()
	w.compact()
}

// ChooseUpgrade applies the i-th purge choice and resumes play (or presents
// the next queued choice if multiple levels were earned at once).
// No-op unless State==StateLevelUp and i is a valid index.
func (w *World) ChooseUpgrade(i int) {
	if w.State != StateLevelUp || i < 0 || i >= len(w.Choices) {
		return
	}
	w.Choices[i].Apply(w)
	w.pendingLevelUps--
	if w.pendingLevelUps > 0 {
		w.rollChoices()
		return
	}
	w.Choices = nil
	w.State = StatePlaying
}

// CutTile cuts the turret tile at idx during combat, spending one nipper.
// The tile (and any tiles it orphans) is purged and the active weapon list is
// rebuilt so the remaining tiles reconcentrate power. Returns false (no nipper
// spent) if the player has no nippers, the tile is the generator, or the tile
// is absent/already purged.
func (w *World) CutTile(idx hexmap.Index) bool {
	if w.State != StatePlaying || w.Player.Nippers <= 0 {
		return false
	}
	if !w.turret.PurgeTile(idx) {
		return false
	}
	w.Player.Nippers--
	w.Player.Weapons = w.turret.ActiveWeapons()
	w.primeNewWeapons()
	return true
}

// ---- internal update steps ----

func (w *World) updatePlayer(move geom.PointF) {
	if mag := move.Abs(); mag > 1 {
		move = move.Multiply(1 / mag)
	}
	if move.Abs() > 0 {
		w.Player.FacingAngle = move.Angle()
	}
	w.Player.Pos = w.Player.Pos.Add(move.Multiply(w.PlayerSpeed()))
	if w.Player.invuln > 0 {
		w.Player.invuln--
	}
}

// repairPlayer heals the tank once every RepairInterval ticks, capped at MaxHP,
// by the meta base regen plus the connected Repair Units' total HPRegen. With no
// base regen and no repair units the heal is 0, so this is a no-op.
func (w *World) repairPlayer() {
	if w.cfg.RepairInterval <= 0 {
		return
	}
	w.Player.repairTimer++
	if w.Player.repairTimer < w.cfg.RepairInterval {
		return
	}
	w.Player.repairTimer = 0
	regen := w.cfg.BaseHPRegen
	if w.turret != nil {
		regen += w.turret.Modifiers().HPRegen
	}
	w.Player.HP += regen
	if w.Player.HP > w.Player.MaxHP {
		w.Player.HP = w.Player.MaxHP
	}
}

// aimSmooth is the per-tick fraction the rendered barrel angle eases toward the
// weapon's live aim, so barrels track targets without jittering on target swaps.
const aimSmooth = 0.3

// stepAngle eases a toward b by fraction t along the shortest angular path.
func stepAngle(a, b, t float64) float64 {
	return a + math.Atan2(math.Sin(b-a), math.Cos(b-a))*t
}

// wrapAngle normalizes an angle to (-π, π].
func wrapAngle(a float64) float64 {
	return math.Atan2(math.Sin(a), math.Cos(a))
}

// weaponStats returns a weapon's combat stats with the global meta damage
// multiplier applied (DamageMult defaults to 0/1 so unset configs are
// unchanged). Damage and ExplodeDamage scale; geometry and cadence do not.
func (w *World) weaponStats(weapon *Weapon) WeaponStats {
	stats := weapon.Stats(w.cfg.Weapons[weapon.Kind])
	if m := w.cfg.DamageMult; m > 0 && m != 1 {
		stats.Damage *= m
		stats.ExplodeDamage *= m
	}
	return stats
}

func (w *World) updateWeapons() {
	fireMult := w.FireRateMultiplier()
	for _, weapon := range w.Player.Weapons {
		params := w.cfg.Weapons[weapon.Kind]
		stats := w.weaponStats(weapon)

		// Smooth the rendered barrel angle toward where the weapon currently aims,
		// so lock-on barrels visibly track their target (drawing only).
		weapon.aimRender = stepAngle(weapon.aimRender, w.weaponAim(weapon, params), aimSmooth)

		// Fire step: advance the accumulator; trigger a shot at BaseInterval.
		weapon.fireProgress += fireIncrement(params, fireMult)
		if weapon.fireProgress >= params.BaseInterval {
			// Interception weapons (CIWS) hold a full charge until a target
			// appears instead of firing into empty space; others fire regardless
			// (forward/outward) so the player feels the cadence.
			if params.HoldWhenNoTarget && w.nearestEnemy(w.Player.Pos, stats.Range) == nil {
				weapon.fireProgress = params.BaseInterval
			} else {
				weapon.fireProgress -= params.BaseInterval
				w.emit(FireSound(weapon.Kind)) // one fire SE per shot (not per pellet), per weapon kind
				if weapon.Kind == KindLaser {
					weapon.beamTicksLeft = stats.BeamDuration
					weapon.beamAngle = w.weaponAim(weapon, params)
				} else {
					weapon.pelletsLeft = pelletCount(params) // queue this shot's pellets
					weapon.pelletTimer = 0
				}
			}
		}

		// Emit any due pellets (handles both simultaneous shots and staggered bursts).
		if weapon.Kind != KindLaser {
			w.emitPellets(weapon, params, stats)
		}
	}
}

// emitPellets fires the projectiles owed for the weapon's current shot. With
// BurstGap==0 all remaining pellets fire at once; otherwise one fires every
// BurstGap ticks (a stream).
func (w *World) emitPellets(weapon *Weapon, params WeaponParams, stats WeaponStats) {
	if weapon.pelletsLeft <= 0 {
		return
	}
	if weapon.pelletTimer > 0 {
		weapon.pelletTimer--
		return
	}
	n := weapon.pelletsLeft
	if params.BurstGap > 0 {
		n = 1
	}
	total := pelletCount(params)
	muzzle := w.Player.Pos.Add(MuzzleOffset(weapon.TileIdx, w.Player.FacingAngle))
	aim := w.weaponAim(weapon, params)
	for k := 0; k < n; k++ {
		i := total - weapon.pelletsLeft + k // pellet index within the shot
		offset := pelletOffset(params, i, total, w.rng)
		vel := geom.PointFFromPolar(stats.ProjectileSpeed, aim+offset)
		w.Projectiles = append(w.Projectiles, &Projectile{
			Pos:           muzzle,
			Vel:           vel,
			Damage:        stats.Damage,
			Radius:        stats.ProjRadius,
			Life:          stats.ProjLife,
			ExplodeRadius: stats.ExplodeRadius,
			ExplodeDamage: stats.ExplodeDamage,
			PassThrough:   params.PassThrough,
			Mover:         params.Mover,
			Sprite:        params.Sprite,
			DrawW:         params.ProjDrawW,
			DrawH:         params.ProjDrawH,
			FaceVelocity:  params.ProjFaceVelocity,
			alive:         true,
		})
	}
	weapon.pelletsLeft -= n
	if weapon.pelletsLeft > 0 {
		weapon.pelletTimer = params.BurstGap
	}
}

// weaponAim returns the world angle a weapon fires along, per its AimMode:
// toward the nearest in-range enemy (else forward) for lock-on, the tank's
// forward facing for forward weapons, or radially outward through its tile.
func (w *World) weaponAim(weapon *Weapon, params WeaponParams) float64 {
	switch params.Aim {
	case AimForward:
		return w.Player.FacingAngle
	case AimOutward:
		if off := MuzzleOffset(weapon.TileIdx, w.Player.FacingAngle); off.Abs() > 0 {
			return off.Angle()
		}
		return w.Player.FacingAngle
	default: // AimLockOn
		muzzle := w.Player.Pos.Add(MuzzleOffset(weapon.TileIdx, w.Player.FacingAngle))
		if target := w.targetEnemy(w.Player.Pos, params.BaseRange, params.Target); target != nil {
			if dir := target.Pos.Subtract(muzzle); dir.Abs() > 0 {
				// Store the aim relative to the tank's facing so that, when the
				// target leaves range, the barrel freezes pointing the same way
				// relative to the tank (turning with it) instead of snapping forward.
				weapon.aimOffset = wrapAngle(dir.Angle() - w.Player.FacingAngle)
			}
		}
		return w.Player.FacingAngle + weapon.aimOffset
	}
}

// updateBeams applies DPS from active laser beams each tick. A beam tracks the
// nearest enemy in range, or fires along its captured forward angle when none is
// locked, and penetrates all enemies in its path.
func (w *World) updateBeams() {
	for _, weapon := range w.Player.Weapons {
		if weapon.Kind != KindLaser || weapon.beamTicksLeft <= 0 {
			continue
		}
		weapon.beamTicksLeft--

		stats := w.weaponStats(weapon)
		muzzle := w.Player.Pos.Add(MuzzleOffset(weapon.TileIdx, w.Player.FacingAngle))
		unitDir := w.beamDir(weapon, muzzle, stats.Range)
		end := muzzle.Add(unitDir.Multiply(stats.BeamLength))
		halfWidth := stats.BeamWidth / 2

		for _, e := range w.Enemies {
			if !e.alive {
				continue
			}
			if geom.PointSegmentDistance(e.Pos, muzzle, end) <= halfWidth+e.Radius {
				e.HP -= stats.Damage
				w.emitDamage(e.Pos, stats.Damage, false)
				if e.HP <= 0 {
					w.killEnemy(e)
				}
			}
		}
	}
}

// beamDir returns the unit direction a laser burst points: toward its target
// enemy within range (centred on the player; nearest or farthest per the
// weapon's TargetMode), or the burst's captured forward angle when none is in range.
func (w *World) beamDir(weapon *Weapon, muzzle geom.PointF, rng float64) geom.PointF {
	if target := w.targetEnemy(w.Player.Pos, rng, w.cfg.Weapons[weapon.Kind].Target); target != nil {
		dir := target.Pos.Subtract(muzzle)
		if d := dir.Abs(); d > 0 {
			return dir.Multiply(1 / d)
		}
	}
	return geom.PointFFromPolar(1, weapon.beamAngle)
}

// ActiveBeams returns snapshots of all currently firing laser beams for drawing.
func (w *World) ActiveBeams() []BeamView {
	var out []BeamView
	for _, weapon := range w.Player.Weapons {
		if weapon.Kind != KindLaser || weapon.beamTicksLeft <= 0 {
			continue
		}
		stats := weapon.Stats(w.cfg.Weapons[weapon.Kind])
		muzzle := w.Player.Pos.Add(MuzzleOffset(weapon.TileIdx, w.Player.FacingAngle))
		out = append(out, BeamView{
			Origin: muzzle,
			Dir:    w.beamDir(weapon, muzzle, stats.Range),
			Length: stats.BeamLength,
			Width:  stats.BeamWidth,
		})
	}
	return out
}

// nearestEnemy returns the closest living enemy within maxRange (nil if none).
func (w *World) nearestEnemy(from geom.PointF, maxRange float64) *Enemy {
	return w.targetEnemy(from, maxRange, TargetNearest)
}

// targetEnemy returns the living enemy within maxRange chosen per mode: the
// closest (TargetNearest) or the farthest (TargetFarthest). nil if none in range.
func (w *World) targetEnemy(from geom.PointF, maxRange float64, mode TargetMode) *Enemy {
	var best *Enemy
	bestD := -1.0
	for _, e := range w.Enemies {
		if !e.alive {
			continue
		}
		d := e.Pos.Distance(from)
		if d > maxRange {
			continue
		}
		if best == nil || (mode == TargetFarthest && d > bestD) || (mode == TargetNearest && d < bestD) {
			bestD = d
			best = e
		}
	}
	return best
}

func (w *World) updateProjectiles() {
	for _, p := range w.Projectiles {
		if !p.alive {
			continue
		}
		if p.Mover != nil {
			p.Mover.Steer(p, w) // homing / drift: adjust velocity before moving
		}
		p.age++
		p.Pos = p.Pos.Add(p.Vel)
		p.Life--
		if p.Life <= 0 {
			p.alive = false
			if p.ExplodeRadius > 0 {
				if p.Firework {
					w.explodeFirework(p.Pos, p.ExplodeRadius)
				} else {
					w.explode(p.Pos, p.ExplodeRadius, p.ExplodeDamage)
				}
			}
			continue
		}
		// Pass-through shells (e.g. the grenade) ignore contact and only matter
		// on expiry; contact projectiles fall through to the hit test below.
		if p.PassThrough {
			continue
		}
		for _, e := range w.Enemies {
			if !e.alive {
				continue
			}
			if p.Pos.Distance(e.Pos) <= p.Radius+e.Radius {
				e.HP -= p.Damage
				w.emitDamage(e.Pos, p.Damage, false)
				p.alive = false
				if p.ExplodeRadius > 0 {
					w.explode(p.Pos, p.ExplodeRadius, p.ExplodeDamage)
				}
				if e.HP <= 0 {
					w.killEnemy(e)
				}
				break
			}
		}
	}
}

// explosionLife is how many ticks an explosion effect stays visible (fading out).
const explosionLife = 24

// explode deals dmg to every alive enemy within radius of center (area damage)
// and queues a visual explosion effect at that spot.
func (w *World) explode(center geom.PointF, radius, dmg float64) {
	w.Explosions = append(w.Explosions, &Explosion{Pos: center, Radius: radius, Life: explosionLife, MaxLife: explosionLife})
	w.emit(SndExplosion)
	for _, e := range w.Enemies {
		if !e.alive {
			continue
		}
		if e.Pos.Distance(center) <= radius+e.Radius {
			e.HP -= dmg
			w.emitDamage(e.Pos, dmg, false)
			if e.HP <= 0 {
				w.killEnemy(e)
			}
		}
	}
}

// explodeFirework queues a purely cosmetic spark burst for firework junk: no
// area damage, no enemy scan, and a random hue so each shell bursts in its own
// color. It deliberately skips the weapon explosion SFX so a harmless firework
// neither looks nor sounds like real ordnance.
func (w *World) explodeFirework(center geom.PointF, radius float64) {
	w.Explosions = append(w.Explosions, &Explosion{
		Pos:      center,
		Radius:   radius,
		Life:     explosionLife,
		MaxLife:  explosionLife,
		Firework: true,
		Hue:      w.rng.Float64(),
	})
}

// updateExplosions ages active explosion effects; compact() drops expired ones.
func (w *World) updateExplosions() {
	for _, e := range w.Explosions {
		if e.Life > 0 {
			e.Life--
		}
	}
}

func (w *World) killEnemy(e *Enemy) {
	e.alive = false
	w.emitDeath(e)
	w.Kills++
	if e.XPValue > 0 {
		w.Gems = append(w.Gems, &Gem{Pos: e.Pos, Value: e.XPValue, alive: true})
	}
	if e.DropsNipper {
		kind := PickupNipper
		// Rarely a heart (HP) drops instead of a nipper. rng is nil in manually
		// built test worlds, which then always drop a nipper.
		if w.rng != nil && w.rng.Float64() < w.cfg.HeartDropChance {
			kind = PickupHeart
		}
		w.Pickups = append(w.Pickups, &Pickup{Pos: e.Pos, Kind: kind, alive: true})
	}
	// Mid-bosses (any non-final boss) drop a magnet pickup; collecting it pulls
	// every gem and pickup on the field to the player at once.
	if e.IsBoss && !e.Final {
		w.Pickups = append(w.Pickups, &Pickup{Pos: e.Pos, Kind: PickupMagnet, alive: true})
	}
	if e.Final && w.State == StatePlaying {
		w.State = StateCleared // defeating the final boss wins the run
	}
}

// despawnDistance is the per-axis (X or Y) gap from the player at which a
// regular enemy is considered shaken off: it is removed so we stop spending
// work chasing something the player has long outrun. Bosses are not removed;
// instead they are reeled back to this distance so they keep pursuing.
const despawnDistance = 1000

// clampAbs returns v clamped to the range [-limit, limit] (limit assumed >= 0).
func clampAbs(v, limit float64) float64 {
	if v > limit {
		return limit
	}
	if v < -limit {
		return -limit
	}
	return v
}

func (w *World) updateEnemies() {
	for _, e := range w.Enemies {
		if !e.alive {
			continue
		}
		// Drop (or, for bosses, reel in) enemies the player has outrun by more
		// than despawnDistance on either axis. Candlesticks are stationary
		// pickups, so leave them be.
		if !e.DropsNipper {
			dx := e.Pos.X - w.Player.Pos.X
			dy := e.Pos.Y - w.Player.Pos.Y
			if dx > despawnDistance || dx < -despawnDistance ||
				dy > despawnDistance || dy < -despawnDistance {
				if e.IsBoss {
					e.Pos.X = w.Player.Pos.X + clampAbs(dx, despawnDistance)
					e.Pos.Y = w.Player.Pos.Y + clampAbs(dy, despawnDistance)
				} else {
					e.alive = false
					continue
				}
			}
		}
		dir := w.Player.Pos.Subtract(e.Pos)
		d := dir.Abs()
		if d > 0 {
			if e.Turn > 0 {
				// Bounded-turn chase (seek steering): nudge the current velocity
				// toward "head straight at the player at Speed" by at most Turn this
				// tick, so the enemy banks into a curve instead of snapping its
				// heading. Smaller Turn = wider, lazier arcs; once Turn is large
				// enough to cover the full turn in one tick it behaves like instant
				// follow. Turn == 0 takes the instant branch below.
				desired := dir.Multiply(e.Speed / d)
				steer := desired.Subtract(e.Vel)
				if mag := steer.Abs(); mag > e.Turn {
					steer = steer.Multiply(e.Turn / mag)
				}
				e.Vel = e.Vel.Add(steer)
				if s := e.Vel.Abs(); s > e.Speed {
					e.Vel = e.Vel.Multiply(e.Speed / s) // cap at Speed
				}
				e.Pos = e.Pos.Add(e.Vel)
			} else {
				// Instant follow: re-aim straight at the player every tick.
				e.Pos = e.Pos.Add(dir.Multiply(e.Speed / d))
			}
		}
		if e.Damage > 0 && d <= e.Radius+w.Player.Radius && w.Player.invuln == 0 {
			w.damagePlayer(e.Damage)
		}
	}
}

func (w *World) damagePlayer(dmg float64) {
	// Armor subtracts a flat amount, but at least 1 damage always lands. It is the
	// sum of the meta base armor and the connected Armor tiles (turret is nil in
	// some manually-built test worlds, which then only have the base armor).
	armor := w.cfg.BaseArmor
	if w.turret != nil {
		armor += w.turret.Modifiers().Armor
	}
	if armor > 0 {
		dmg -= armor
		if dmg < 1 {
			dmg = 1
		}
	}
	w.Player.HP -= dmg
	w.emitDamage(w.Player.Pos, dmg, true)
	w.Player.invuln = 30
	w.emit(SndPlayerHit)
	if w.Player.HP <= 0 {
		w.Player.HP = 0
		if w.State == StatePlaying { // don't override a win decided earlier this tick
			w.State = StateGameOver
		}
	}
}

func (w *World) updateGems() {
	pr := w.cfg.Pickup
	for _, g := range w.Gems {
		if !g.alive {
			continue
		}
		d := g.Pos.Distance(w.Player.Pos)
		if d <= pr.MagnetDist {
			g.tracking = true // entered magnet range: home from now on, even if the player outruns it
		}
		switch {
		case d <= pr.PickupDist:
			g.alive = false
			w.addXP(g.Value)
		case g.tracking && d > 0:
			dir := w.Player.Pos.Subtract(g.Pos)
			g.Pos = g.Pos.Add(dir.Multiply(pr.MagnetSpeed / d))
		}
	}
}

// updatePickups magnets and collects dropped pickups: a nipper grants one
// nipper, a heart restores HeartHeal HP (capped at MaxHP).
func (w *World) updatePickups() {
	pr := w.cfg.Pickup
	for _, p := range w.Pickups {
		if !p.alive {
			continue
		}
		d := p.Pos.Distance(w.Player.Pos)
		if d <= pr.MagnetDist {
			p.tracking = true // entered magnet range: home from now on, even if the player outruns it
		}
		switch {
		case d <= pr.PickupDist:
			p.alive = false
			w.collectPickup(p)
		case p.tracking && d > 0:
			dir := w.Player.Pos.Subtract(p.Pos)
			p.Pos = p.Pos.Add(dir.Multiply(pr.MagnetSpeed / d))
		}
	}
}

// collectPickup applies a collected pickup's effect to the player.
func (w *World) collectPickup(p *Pickup) {
	switch p.Kind {
	case PickupHeart:
		w.Player.HP += w.cfg.HeartHeal
		if w.Player.HP > w.Player.MaxHP {
			w.Player.HP = w.Player.MaxHP
		}
	case PickupMagnet:
		w.magnetizeAll()
	default:
		w.Player.Nippers++
	}
}

// magnetizeAll latches every gem and pickup into permanent homing, so they all
// fly to the player at once. Triggered by collecting a magnet pickup (dropped by
// mid-bosses).
func (w *World) magnetizeAll() {
	for _, g := range w.Gems {
		g.tracking = true
	}
	for _, p := range w.Pickups {
		p.tracking = true
	}
}

func (w *World) addXP(v float64) {
	w.Player.XP += v
	for w.Player.XP >= w.Player.XPToNext {
		w.Player.XP -= w.Player.XPToNext
		w.Player.Level++
		w.Player.XPToNext = math.Ceil(w.Player.XPToNext * w.cfg.XPToNextGrowth)
		w.pendingLevelUps++
	}
	if w.pendingLevelUps > 0 && w.State == StatePlaying {
		w.State = StateLevelUp
		w.rollChoices()
	}
}

// doctorTitles are the honorifics of the eccentric doctors who keep bolting
// things onto the tank. A doctor's full name is a title plus a random uppercase
// letter (e.g. "Doctor X"); the scene formats the two via a per-language
// template so the letter can sit before or after the title.
var doctorTitles = []string{
	"Doctor", "Professor", "Doktor", "Instructor", "Master",
}

// rollChoices builds three "doctor" offers for the level-up screen. Each offer
// is rolled independently; see rollDoctorChoice for the per-item breakdown.
func (w *World) rollChoices() {
	choices := make([]Upgrade, 0, 3)
	for i := 0; i < 3; i++ {
		choices = append(choices, w.rollDoctorChoice())
	}
	w.Choices = choices
}

// rollDoctorChoice produces a single doctor proposal: 1..MaxItems items, each
// independently one of four kinds chosen by DoctorSpec weight —
//   - WeaponAdd: bolt on a new weapon or equipment tile (rollAddable);
//   - WeaponUpgrade: level up a random equipped weapon (falls back to WeaponAdd
//     when nothing is equipped yet);
//   - Junk: bolt on a useless junk tile;
//   - Nippers: hand over spare tile cuts.
//
// Tile-adding items (WeaponAdd / Junk) fall back to a Nippers line once the
// turret — counting tiles already queued by this same offer — has reached
// MaxTurretTiles, so an offer never grows it past the cap.
func (w *World) rollDoctorChoice() Upgrade {
	title := doctorTitles[w.rng.Intn(len(doctorTitles))]
	alphabet := string(rune('A' + w.rng.Intn(26)))
	dd := w.cfg.Doctor
	activeWeapons := w.turret.ActiveWeapons()

	itemCount := 1 + w.rng.Intn(dd.MaxItems)
	items := make([]OfferItem, 0, itemCount)
	var nippers int        // total nippers granted on Apply
	var upgrades []*Weapon // existing weapons to level on Apply
	var comps []Component  // new tiles to add on Apply

	for k := 0; k < itemCount; k++ {
		kind := w.pickOfferKind(dd)

		// An upgrade has nothing to target before any weapon is equipped.
		if kind == OfferUpgrade && len(activeWeapons) == 0 {
			kind = OfferAddWeapon
		}
		// Tile-adding items respect the cap (counting this offer's own queue).
		if kind == OfferAddWeapon || kind == OfferAddJunk {
			if w.turret.TileCount()+len(comps) >= w.cfg.MaxTurretTiles {
				kind = OfferNippers
			}
		}

		switch kind {
		case OfferNippers:
			n := dd.NipperMin + w.rng.Intn(dd.NipperMax-dd.NipperMin+1)
			nippers += n
			items = append(items, OfferItem{Kind: OfferNippers, Amount: n, Text: fmt.Sprintf("+%d Nippers", n)})
		case OfferUpgrade:
			wp := activeWeapons[w.rng.Intn(len(activeWeapons))]
			upgrades = append(upgrades, wp)
			items = append(items, OfferItem{Kind: OfferUpgrade, Weapon: wp.Kind, Text: wp.Name})
		case OfferAddJunk:
			j := randomJunk(w.rng)
			comps = append(comps, j)
			items = append(items, OfferItem{Kind: OfferAddJunk, Text: j.DeviceName})
		default: // OfferAddWeapon: a new weapon or equipment tile
			comp, item := w.rollAddable()
			comps = append(comps, comp)
			items = append(items, item)
		}
	}

	return Upgrade{
		Doctor:         title,
		DoctorAlphabet: alphabet,
		Items:          items,
		Apply: func(world *World) {
			world.Player.Nippers += nippers
			for _, wp := range upgrades {
				wp.Level++
			}
			if len(comps) > 0 {
				for _, comp := range comps {
					world.turret.AddTile(comp, world.rng)
				}
				world.Player.Weapons = world.turret.ActiveWeapons()
				world.primeNewWeapons()
			}
		},
	}
}

// pickOfferKind selects one offer kind by DoctorSpec weight. The four weights
// are normalised by their total (only their ratios matter); a non-positive
// total degenerates to a junk tile.
func (w *World) pickOfferKind(dd DoctorSpec) OfferKind {
	total := dd.NipperWeight + dd.WeaponAddWeight + dd.WeaponUpgradeWeight + dd.JunkWeight
	if total <= 0 {
		return OfferAddJunk
	}
	r := w.rng.Float64() * total
	if r < dd.NipperWeight {
		return OfferNippers
	}
	r -= dd.NipperWeight
	if r < dd.WeaponAddWeight {
		return OfferAddWeapon
	}
	r -= dd.WeaponAddWeight
	if r < dd.WeaponUpgradeWeight {
		return OfferUpgrade
	}
	return OfferAddJunk
}

// rollAddable picks a new "useful" tile uniformly from the weapon kinds plus the
// three equipment tiles (capacitor / repair unit / armor), returning the
// component to bolt on and the matching offer line. Weapons and equipment share
// one pool because both are useful additions (unlike inert junk).
func (w *World) rollAddable() (Component, OfferItem) {
	const equipCount = 3
	i := w.rng.Intn(weaponKindCount + equipCount)
	if i < weaponKindCount {
		kind := WeaponKind(i)
		return WeaponComponent{Weapon: NewWeapon(kind.String(), kind)},
			OfferItem{Kind: OfferAddWeapon, Weapon: kind, Text: kind.String()}
	}
	switch i - weaponKindCount {
	case 0:
		return Capacitor{FireRateBonus: w.cfg.CapacitorFireRateBonus},
			OfferItem{Kind: OfferAddCapacitor, Text: "Capacitor"}
	case 1:
		return RepairUnit{HealAmount: w.cfg.RepairHealAmount},
			OfferItem{Kind: OfferAddRepairUnit, Text: "Repair Unit"}
	default:
		return Armor{Reduction: w.cfg.ArmorReduction},
			OfferItem{Kind: OfferAddArmor, Text: "Armor"}
	}
}

// defaultSpawnInterval is used only if the active phase has no Interval set.
const defaultSpawnInterval = 60

func (w *World) spawnEnemies() {
	if w.spawnTimer > 0 {
		w.spawnTimer--
		return
	}
	ph := w.currentPhase()
	interval := defaultSpawnInterval
	if ph != nil && ph.Interval > 0 {
		interval = ph.Interval
	}
	w.spawnTimer = interval

	kind := EnemyGrunt
	if ph != nil {
		kind = w.pickKind(ph.Weights)
	}
	w.spawnPackOf(kind)
}

// spawnPackOf spawns a cluster (pack) of one zako kind around a single bearing,
// so swarmers arrive as a group while grunts/brutes come one at a time. Pack
// size is PackMin..PackMax.
func (w *World) spawnPackOf(kind EnemyKind) {
	stats := w.cfg.EnemyKinds[kind]
	n := stats.PackMin
	if n < 1 {
		n = 1
	}
	if stats.PackMax > n {
		n += w.rng.Intn(stats.PackMax - n + 1)
	}
	base := w.rng.Float64() * 2 * math.Pi
	for i := 0; i < n; i++ {
		a := base + (w.rng.Float64()-0.5)*0.6 // small spread so the pack isn't a single point
		pos := w.Player.Pos.Add(geom.PointFFromPolar(w.cfg.Spawn.EnemyDist, a))
		w.Enemies = append(w.Enemies, w.makeEnemy(kind, stats, pos))
	}
}

// makeEnemy builds a zako enemy of the given kind, applying time-based HP
// scaling: HP = HPBase × 2^(tick / HPDoublingTicks).
func (w *World) makeEnemy(kind EnemyKind, s EnemyStats, pos geom.PointF) *Enemy {
	hp := s.HPBase
	if w.cfg.HPDoublingTicks > 0 {
		hp = s.HPBase * math.Pow(2, float64(w.Tick)/w.cfg.HPDoublingTicks)
	}
	e := &Enemy{
		Pos: pos, Kind: kind, HP: hp, MaxHP: hp,
		Speed: s.Speed, Turn: s.Turn, Radius: s.Radius, Damage: s.Damage, XPValue: s.XPValue,
		alive: true,
	}
	w.initEnemyVel(e)
	return e
}

// initEnemyVel seeds a turning enemy's velocity so it starts moving at full
// speed straight toward the player; Turn then only limits how fast it can change
// heading from there. No-op for instant-follow enemies (Turn == 0), which ignore
// Vel entirely.
func (w *World) initEnemyVel(e *Enemy) {
	if e.Turn <= 0 {
		return
	}
	dir := w.Player.Pos.Subtract(e.Pos)
	if d := dir.Abs(); d > 0 {
		e.Vel = dir.Multiply(e.Speed / d)
	}
}

// pickSpawnKind chooses an enemy kind using the weights of the current spawn
// phase. Returns EnemyGrunt if there is no phase.
func (w *World) pickSpawnKind() EnemyKind {
	ph := w.currentPhase()
	if ph == nil {
		return EnemyGrunt
	}
	return w.pickKind(ph.Weights)
}

// pickKind does a weighted random pick over an ordered weight slice. Iteration
// order is fixed (slice, not map) so the choice is deterministic for a given RNG
// state.
func (w *World) pickKind(weights []KindWeight) EnemyKind {
	total := 0
	for _, kw := range weights {
		if kw.Weight > 0 {
			total += kw.Weight
		}
	}
	if total <= 0 {
		return EnemyGrunt
	}
	r := w.rng.Intn(total)
	for _, kw := range weights {
		if kw.Weight <= 0 {
			continue
		}
		if r < kw.Weight {
			return kw.Kind
		}
		r -= kw.Weight
	}
	return EnemyGrunt
}

// currentPhase returns the spawn band for the current tick: the first phase whose
// UntilTick hasn't been passed, else the last phase, else nil.
func (w *World) currentPhase() *SpawnPhase {
	for i := range w.cfg.SpawnPhases {
		if w.Tick < w.cfg.SpawnPhases[i].UntilTick {
			return &w.cfg.SpawnPhases[i]
		}
	}
	if n := len(w.cfg.SpawnPhases); n > 0 {
		return &w.cfg.SpawnPhases[n-1]
	}
	return nil
}

// spawnBosses spawns each scheduled boss once, when its AtTick is reached.
func (w *World) spawnBosses() {
	for w.bossesSpawned < len(w.cfg.Bosses) {
		b := w.cfg.Bosses[w.bossesSpawned]
		if w.Tick < b.AtTick {
			break
		}
		w.bossesSpawned++
		angle := w.rng.Float64() * 2 * math.Pi
		pos := w.Player.Pos.Add(geom.PointFFromPolar(w.cfg.Spawn.EnemyDist, angle))
		boss := &Enemy{
			Pos: pos, HP: b.HP, MaxHP: b.HP,
			Speed: b.Speed, Turn: b.Turn, Radius: b.Radius, Damage: b.Damage, XPValue: b.XPValue,
			IsBoss: true, Final: b.Final, Name: b.Name, Sprite: b.Sprite, alive: true,
		}
		w.initEnemyVel(boss)
		w.Enemies = append(w.Enemies, boss)
	}
}

// spawnCandlestick periodically drops a stationary, harmless candlestick the
// player can break for a nipper — at the cost of leaving a safe position.
func (w *World) spawnCandlestick() {
	if w.candlestickTimer > 0 {
		w.candlestickTimer--
		return
	}
	w.candlestickTimer = w.cfg.CandlestickInterval

	sp := w.cfg.Spawn
	angle := w.rng.Float64() * 2 * math.Pi
	dist := sp.CandleDist + w.rng.Float64()*sp.CandleDistRange
	pos := w.Player.Pos.Add(geom.PointFFromPolar(dist, angle))

	e := w.cfg.Candlestick // copy the candlestick template
	e.Pos = pos
	e.alive = true
	w.Enemies = append(w.Enemies, &e)
}

// spawnAutoMagnet keeps the on-field item population from snowballing. On the
// same cadence as candlesticks it may drop a magnet pickup — the more gems and
// pickups litter the field, the likelier it spawns — so collecting it sweeps
// everything to the player at once and frees the per-item update/draw work.
// Magnets the player has long outrun are reeled back within reach (the same
// far-out clamp bosses use) so they stay collectible instead of stranding.
func (w *World) spawnAutoMagnet() {
	// Reel any far-flung magnets back toward the player, exactly as bosses are.
	magnets := 0
	for _, p := range w.Pickups {
		if !p.alive || p.Kind != PickupMagnet {
			continue
		}
		magnets++
		dx := p.Pos.X - w.Player.Pos.X
		dy := p.Pos.Y - w.Player.Pos.Y
		if dx > despawnDistance || dx < -despawnDistance ||
			dy > despawnDistance || dy < -despawnDistance {
			p.Pos.X = w.Player.Pos.X + clampAbs(dx, despawnDistance)
			p.Pos.Y = w.Player.Pos.Y + clampAbs(dy, despawnDistance)
		}
	}

	if w.magnetTimer > 0 {
		w.magnetTimer--
		return
	}
	w.magnetTimer = w.cfg.CandlestickInterval

	if magnets >= 3 {
		return // already enough magnets pending; don't pile on
	}

	// More on-field items (gems + nipper/heart pickups) → higher spawn chance.
	items := len(w.Gems)
	for _, p := range w.Pickups {
		if p.alive && p.Kind != PickupMagnet {
			items++
		}
	}
	if w.rng.Float64() >= float64(items)/100 {
		return
	}

	sp := w.cfg.Spawn
	angle := w.rng.Float64() * 2 * math.Pi
	pos := w.Player.Pos.Add(geom.PointFFromPolar(sp.EnemyDist, angle))
	w.Pickups = append(w.Pickups, &Pickup{Pos: pos, Kind: PickupMagnet, alive: true})
}

func (w *World) compact() {
	w.Enemies = filterAlive(w.Enemies, func(e *Enemy) bool { return e.alive })
	w.Projectiles = filterAlive(w.Projectiles, func(p *Projectile) bool { return p.alive })
	w.Gems = filterAlive(w.Gems, func(g *Gem) bool { return g.alive })
	w.Pickups = filterAlive(w.Pickups, func(p *Pickup) bool { return p.alive })
	w.Explosions = filterAlive(w.Explosions, func(e *Explosion) bool { return e.Life > 0 })
}

func filterAlive[T any](s []T, alive func(T) bool) []T {
	out := s[:0]
	for _, v := range s {
		if alive(v) {
			out = append(out, v)
		}
	}
	return out
}
