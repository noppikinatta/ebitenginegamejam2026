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
	StateGameOver       // run ended
)

// World holds all gameplay state for a single run. It has no Ebiten dependency
// so the simulation can be unit-tested in isolation.
type World struct {
	Player      *Player
	Enemies     []*Enemy
	Projectiles []*Projectile
	Gems        []*Gem
	Pickups     []*Pickup // dropped nippers awaiting collection

	// Choices are the doctor offers shown while State==StateLevelUp.
	Choices []Upgrade

	State State
	Tick  int
	Kills int

	turret           *Turret
	pendingLevelUps  int
	candlestickTimer int
	spawnTimer       int
	rng              *rand.Rand
	cfg              Config
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
	return PowerMultiplier(w.cfg.PowerCurve, w.turret.ConsumerTileCount())
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
	if w.State != StatePlaying {
		return
	}

	w.Tick++
	w.updatePlayer(move)
	w.updateWeapons()
	w.updateBeams()
	w.updateProjectiles()
	w.updateEnemies()
	w.updateGems()
	w.updatePickups()
	w.spawnEnemies()
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

func (w *World) updateWeapons() {
	fireMult := w.FireRateMultiplier()
	for _, weapon := range w.Player.Weapons {
		if weapon.cooldown > 0 {
			weapon.cooldown--
			continue
		}

		stats := weapon.Stats(w.cfg.Weapons[weapon.Kind], fireMult)
		target := w.nearestEnemy(w.Player.Pos, stats.Range)
		if target == nil {
			continue
		}

		muzzle := w.Player.Pos.Add(MuzzleOffset(weapon.TileIdx, w.Player.FacingAngle))

		if weapon.Kind == KindLaser {
			weapon.beamTicksLeft = stats.BeamDuration
			weapon.cooldown = stats.FireInterval
			continue
		}

		// Projectiles aim from the turret tile muzzle toward the target.
		dir := target.Pos.Subtract(muzzle)
		d := dir.Abs()
		if d == 0 {
			continue
		}
		baseAngle := dir.Angle()

		for _, offset := range weapon.ProjectileOffsets() {
			vel := geom.PointFFromPolar(stats.ProjectileSpeed, baseAngle+offset)
			w.Projectiles = append(w.Projectiles, &Projectile{
				Pos:    muzzle,
				Vel:    vel,
				Damage: stats.Damage,
				Radius: 5,
				Life:   120,
				alive:  true,
			})
		}
		weapon.cooldown = stats.FireInterval
	}
}

// updateBeams applies DPS from active laser beams each tick.
// Beams track the nearest enemy each frame and penetrate all enemies in path.
func (w *World) updateBeams() {
	fireMult := w.FireRateMultiplier()
	for _, weapon := range w.Player.Weapons {
		if weapon.Kind != KindLaser || weapon.beamTicksLeft <= 0 {
			continue
		}
		weapon.beamTicksLeft--

		stats := weapon.Stats(w.cfg.Weapons[weapon.Kind], fireMult)
		muzzle := w.Player.Pos.Add(MuzzleOffset(weapon.TileIdx, w.Player.FacingAngle))
		target := w.nearestEnemy(muzzle, stats.Range)
		if target == nil {
			continue
		}

		dir := target.Pos.Subtract(muzzle)
		d := dir.Abs()
		if d == 0 {
			continue
		}
		unitDir := dir.Multiply(1 / d)
		end := muzzle.Add(unitDir.Multiply(stats.BeamLength))
		halfWidth := stats.BeamWidth / 2

		for _, e := range w.Enemies {
			if !e.alive {
				continue
			}
			if geom.PointSegmentDistance(e.Pos, muzzle, end) <= halfWidth+e.Radius {
				e.HP -= stats.Damage
				if e.HP <= 0 {
					w.killEnemy(e)
				}
			}
		}
	}
}

// ActiveBeams returns snapshots of all currently firing laser beams for drawing.
func (w *World) ActiveBeams() []BeamView {
	fireMult := w.FireRateMultiplier()
	var out []BeamView
	for _, weapon := range w.Player.Weapons {
		if weapon.Kind != KindLaser || weapon.beamTicksLeft <= 0 {
			continue
		}
		stats := weapon.Stats(w.cfg.Weapons[weapon.Kind], fireMult)
		muzzle := w.Player.Pos.Add(MuzzleOffset(weapon.TileIdx, w.Player.FacingAngle))
		target := w.nearestEnemy(muzzle, stats.Range)
		if target == nil {
			continue
		}
		dir := target.Pos.Subtract(muzzle)
		d := dir.Abs()
		if d == 0 {
			continue
		}
		out = append(out, BeamView{
			Origin: muzzle,
			Dir:    dir.Multiply(1 / d),
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
		p.Pos = p.Pos.Add(p.Vel)
		p.Life--
		if p.Life <= 0 {
			p.alive = false
			continue
		}
		for _, e := range w.Enemies {
			if !e.alive {
				continue
			}
			if p.Pos.Distance(e.Pos) <= p.Radius+e.Radius {
				e.HP -= p.Damage
				p.alive = false
				if e.HP <= 0 {
					w.killEnemy(e)
				}
				break
			}
		}
	}
}

func (w *World) killEnemy(e *Enemy) {
	e.alive = false
	w.Kills++
	if e.XPValue > 0 {
		w.Gems = append(w.Gems, &Gem{Pos: e.Pos, Value: e.XPValue, alive: true})
	}
	if e.DropsNipper {
		w.Pickups = append(w.Pickups, &Pickup{Pos: e.Pos, alive: true})
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
	w.Player.invuln = 30
	if w.Player.HP <= 0 {
		w.Player.HP = 0
		w.State = StateGameOver
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

// rollDoctorChoice produces a single doctor offer. There are three offer types:
//   - Nippers (~25%): 5-10 spare nippers to fund future cuts.
//   - Weapon upgrade (~37.5%): 1-3 existing weapons each gain +1 Level (+20% damage).
//   - Tile bundle (~37.5%): 1-3 new tiles added; each is 50% weapon, 50% junk.
//
// atCap forces non-tile offers (upgrade or nippers) so the turret doesn't exceed
// maxTurretTiles.
func (w *World) rollDoctorChoice(atCap bool) Upgrade {
	doc := doctorNames[w.rng.Intn(len(doctorNames))]
	r := w.rng.Float64()
	activeWeapons := w.turret.ActiveWeapons()
	dd := w.cfg.Doctor

	// ~25%: nippers. Also forced when atCap and no weapons exist to upgrade.
	if r < dd.NipperChance || (atCap && len(activeWeapons) == 0) {
		n := dd.NipperMin + w.rng.Intn(dd.NipperMax-dd.NipperMin+1)
		return Upgrade{
			Name: fmt.Sprintf("Dr. %s: spare nippers (+%d)", doc, n),
			Desc: "Plastic-model nippers to cut tiles mid-battle.",
			Apply: func(world *World) {
				world.Player.Nippers += n
			},
		}
	}

	// atCap or ~37.5% of remaining: weapon upgrade (1-3 existing weapons +1 Level).
	if atCap || (len(activeWeapons) > 0 && r < dd.UpgradeChance) {
		maxCount := len(activeWeapons)
		if maxCount > dd.MaxUpgrades {
			maxCount = dd.MaxUpgrades
		}
		count := 1 + w.rng.Intn(maxCount)
		perm := w.rng.Perm(len(activeWeapons))
		selected := make([]*Weapon, count)
		for i := 0; i < count; i++ {
			selected[i] = activeWeapons[perm[i]]
		}
		nameStr := ""
		for i, wp := range selected {
			if i > 0 {
				nameStr += ", "
			}
			nameStr += wp.Name
		}
		return Upgrade{
			Name: fmt.Sprintf("Dr. %s upgrades: %s", doc, nameStr),
			Desc: "Each upgraded weapon deals 20% more damage per level (multiplicative).",
			Apply: func(world *World) {
				for _, wp := range selected {
					wp.Level++
				}
			},
		}
	}

	// Tile bundle: 1-MaxBundleTiles tiles, each independently 50% weapon / 50% junk.
	tileCount := 1 + w.rng.Intn(dd.MaxBundleTiles)
	comps := make([]Component, tileCount)
	bundleDesc := ""
	for i := range comps {
		if i > 0 {
			bundleDesc += " + "
		}
		if w.rng.Float64() < 0.5 {
			kind := pickWeaponKind(w.rng)
			comps[i] = WeaponComponent{Weapon: NewWeapon(kind.String(), kind)}
			bundleDesc += kind.String()
		} else {
			name := junkDeviceNames[w.rng.Intn(len(junkDeviceNames))]
			comps[i] = Junk{DeviceName: name}
			bundleDesc += name
		}
	}
	return Upgrade{
		Name: fmt.Sprintf("Dr. %s: %s", doc, bundleDesc),
		Desc: fmt.Sprintf("Adds %d tile(s) — dilutes power for all, but may bring new weapons.", tileCount),
		Apply: func(world *World) {
			for _, comp := range comps {
				world.turret.AddTile(comp, world.rng)
			}
			world.Player.Weapons = world.turret.ActiveWeapons()
		},
	}
}

func (w *World) spawnEnemies() {
	if w.spawnTimer > 0 {
		w.spawnTimer--
		return
	}
	sp := w.cfg.Spawn
	interval := sp.EnemyBaseInterval - w.Tick/sp.EnemyIntervalDecay
	if interval < sp.EnemyMinInterval {
		interval = sp.EnemyMinInterval
	}
	w.spawnTimer = interval

	angle := w.rng.Float64() * 2 * math.Pi
	pos := w.Player.Pos.Add(geom.PointFFromPolar(sp.EnemyDist, angle))

	// HP scales linearly with time so enemies get tankier as the run goes on.
	sc := w.cfg.EnemyScaling
	w.Enemies = append(w.Enemies, &Enemy{
		Pos:     pos,
		HP:      sc.HPBase + float64(w.Tick)*sc.HPPerTick,
		Speed:   sc.Speed,
		Radius:  sc.Radius,
		Damage:  sc.Damage,
		XPValue: sc.XPValue,
		alive:   true,
	})
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
