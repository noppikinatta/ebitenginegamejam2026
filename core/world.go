package core

import (
	"fmt"
	"math"
	"math/rand"

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
	weapons := turret.ActiveWeapons()

	p := cfg.Player // copy the starting-stat template
	p.Pos = geom.PointF{}
	p.Weapons = weapons
	p.FacingAngle = -math.Pi / 2
	p.Nippers = cfg.StartingNippers
	return &World{
		Player: &p,
		State:  StatePlaying,
		turret: turret,
		rng:    rng,
		cfg:    cfg,
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
	w.Player.Pos = w.Player.Pos.Add(move.Multiply(w.Player.Speed))
	if w.Player.invuln > 0 {
		w.Player.invuln--
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

func (w *World) updateWeapons() {
	fireMult := w.FireRateMultiplier()
	for _, weapon := range w.Player.Weapons {
		params := w.cfg.Weapons[weapon.Kind]
		stats := weapon.Stats(params)

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
		if target := w.nearestEnemy(w.Player.Pos, params.BaseRange); target != nil {
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

		stats := weapon.Stats(w.cfg.Weapons[weapon.Kind])
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

// beamDir returns the unit direction a laser burst points: toward the nearest
// enemy within range (centred on the player), or the burst's captured forward
// angle when no enemy is locked.
func (w *World) beamDir(weapon *Weapon, muzzle geom.PointF, rng float64) geom.PointF {
	if target := w.nearestEnemy(w.Player.Pos, rng); target != nil {
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

func (w *World) nearestEnemy(from geom.PointF, maxRange float64) *Enemy {
	var best *Enemy
	bestD := maxRange
	for _, e := range w.Enemies {
		if !e.alive {
			continue
		}
		if d := e.Pos.Distance(from); d <= bestD {
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
		p.Pos = p.Pos.Add(p.Vel)
		p.Life--
		if p.Life <= 0 {
			p.alive = false
			if p.ExplodeRadius > 0 {
				w.explode(p.Pos, p.ExplodeRadius, p.ExplodeDamage)
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
		w.Pickups = append(w.Pickups, &Pickup{Pos: e.Pos, alive: true})
	}
	if e.Final && w.State == StatePlaying {
		w.State = StateCleared // defeating the final boss wins the run
	}
}

func (w *World) updateEnemies() {
	for _, e := range w.Enemies {
		if !e.alive {
			continue
		}
		dir := w.Player.Pos.Subtract(e.Pos)
		d := dir.Abs()
		if d > 0 {
			e.Pos = e.Pos.Add(dir.Multiply(e.Speed / d))
		}
		if e.Damage > 0 && d <= e.Radius+w.Player.Radius && w.Player.invuln == 0 {
			w.damagePlayer(e.Damage)
		}
	}
}

func (w *World) damagePlayer(dmg float64) {
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
		switch {
		case d <= pr.PickupDist:
			g.alive = false
			w.addXP(g.Value)
		case d <= pr.MagnetDist && d > 0:
			dir := w.Player.Pos.Subtract(g.Pos)
			g.Pos = g.Pos.Add(dir.Multiply(pr.MagnetSpeed / d))
		}
	}
}

// updatePickups magnets and collects dropped nippers, granting one per pickup.
func (w *World) updatePickups() {
	pr := w.cfg.Pickup
	for _, p := range w.Pickups {
		if !p.alive {
			continue
		}
		d := p.Pos.Distance(w.Player.Pos)
		switch {
		case d <= pr.PickupDist:
			p.alive = false
			w.Player.Nippers++
		case d <= pr.MagnetDist && d > 0:
			dir := w.Player.Pos.Subtract(p.Pos)
			p.Pos = p.Pos.Add(dir.Multiply(pr.MagnetSpeed / d))
		}
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

// doctorNames are the eccentric doctors who keep bolting things onto the tank.
var doctorNames = []string{
	"Volt", "Sprocket", "Fizz", "Cogsworth", "Ohm",
	"Wattson", "Pixel", "Gizmo", "Tinker", "Bolt",
}

// rollChoices builds three "doctor" offers. Each level-up grows the turret:
// the chosen doctor bolts a new tile (weapon or useless junk) onto a random
// spot — or, occasionally, hands over spare nippers instead.
func (w *World) rollChoices() {
	atCap := w.turret.TileCount() >= w.cfg.MaxTurretTiles
	choices := make([]Upgrade, 0, 3)
	for i := 0; i < 3; i++ {
		choices = append(choices, w.rollDoctorChoice(atCap))
	}
	w.Choices = choices
}

// rollDoctorChoice produces a single doctor proposal. A proposal is either:
//   - a Nippers offer (~NipperChance, also forced when capped with no weapons to
//     upgrade): spare nippers to fund future cuts; or
//   - a "build" offer: 1-MaxBundleTiles items, each independently a weapon
//     upgrade or a new tile (weapon / junk / capacitor). Adds and upgrades are
//     mixed freely within the one proposal.
//
// atCap forces every build item to be an upgrade (no new tiles) so the turret
// doesn't exceed maxTurretTiles.
func (w *World) rollDoctorChoice(atCap bool) Upgrade {
	doc := doctorNames[w.rng.Intn(len(doctorNames))]
	r := w.rng.Float64()
	activeWeapons := w.turret.ActiveWeapons()
	dd := w.cfg.Doctor

	// Nippers proposal. Also the only option when capped with nothing to upgrade.
	if r < dd.NipperChance || (atCap && len(activeWeapons) == 0) {
		n := dd.NipperMin + w.rng.Intn(dd.NipperMax-dd.NipperMin+1)
		return Upgrade{
			Doctor: doc,
			Items:  []OfferItem{{Kind: OfferNippers, Amount: n, Text: fmt.Sprintf("+%d Nippers", n)}},
			Apply:  func(world *World) { world.Player.Nippers += n },
		}
	}

	// Build proposal: a mix of upgrades and tile additions.
	itemCount := 1 + w.rng.Intn(dd.MaxBundleTiles)
	// Shuffled pool of existing weapons so upgrade items target distinct weapons.
	perm := w.rng.Perm(len(activeWeapons))
	upIdx := 0
	if atCap && itemCount > len(activeWeapons) {
		itemCount = len(activeWeapons) // every item must be an upgrade
	}

	pUpgrade := upgradeShare(dd)
	items := make([]OfferItem, 0, itemCount)
	var upgrades []*Weapon // existing weapons to level on Apply
	var comps []Component  // new tiles to add on Apply

	for k := 0; k < itemCount; k++ {
		canUpgrade := upIdx < len(activeWeapons)
		if canUpgrade && (atCap || w.rng.Float64() < pUpgrade) {
			wp := activeWeapons[perm[upIdx]]
			upIdx++
			upgrades = append(upgrades, wp)
			items = append(items, OfferItem{Kind: OfferUpgrade, Weapon: wp.Kind, Text: wp.Name})
			continue
		}
		if atCap {
			continue // out of distinct weapons and can't add tiles
		}
		// Otherwise add a new tile: capacitor / weapon / junk.
		switch {
		case w.rng.Float64() < dd.CapacitorChance:
			comps = append(comps, Capacitor{FireRateBonus: w.cfg.CapacitorFireRateBonus})
			items = append(items, OfferItem{Kind: OfferAddCapacitor, Text: "Capacitor"})
		case w.rng.Float64() < 0.5:
			kind := pickWeaponKind(w.rng)
			comps = append(comps, WeaponComponent{Weapon: NewWeapon(kind.String(), kind)})
			items = append(items, OfferItem{Kind: OfferAddWeapon, Weapon: kind, Text: kind.String()})
		default:
			j := randomJunk(w.rng)
			comps = append(comps, j)
			items = append(items, OfferItem{Kind: OfferAddJunk, Text: j.DeviceName})
		}
	}

	return Upgrade{
		Doctor: doc,
		Items:  items,
		Apply: func(world *World) {
			for _, wp := range upgrades {
				wp.Level++
			}
			for _, comp := range comps {
				world.turret.AddTile(comp, world.rng)
			}
			world.Player.Weapons = world.turret.ActiveWeapons()
		},
	}
}

// upgradeShare is the probability a build item is an upgrade (vs a new tile),
// derived from the doctor probabilities: the upgrade mass over the non-nipper
// mass. Falls back to 0.5 when the masses are degenerate.
func upgradeShare(dd DoctorSpec) float64 {
	upMass := dd.UpgradeChance - dd.NipperChance
	tileMass := 1 - dd.UpgradeChance
	if upMass < 0 {
		upMass = 0
	}
	if tileMass < 0 {
		tileMass = 0
	}
	if total := upMass + tileMass; total > 0 {
		return upMass / total
	}
	return 0.5
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
	return &Enemy{
		Pos: pos, Kind: kind, HP: hp, MaxHP: hp,
		Speed: s.Speed, Radius: s.Radius, Damage: s.Damage, XPValue: s.XPValue,
		alive: true,
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
		w.Enemies = append(w.Enemies, &Enemy{
			Pos: pos, HP: b.HP, MaxHP: b.HP,
			Speed: b.Speed, Radius: b.Radius, Damage: b.Damage, XPValue: b.XPValue,
			IsBoss: true, Final: b.Final, Name: b.Name, alive: true,
		})
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
