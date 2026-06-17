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

// startingNippers is how many tile cuts the player can make before needing to
// find more. Nippers are the limited resource that gates mid-combat cutting.
const startingNippers = 3

// maxTurretTiles caps turret growth so the miniature stays drawable and the
// cursor navigable. At the cap, doctors offer nippers instead of new tiles.
const maxTurretTiles = 40

// World holds all gameplay state for a single run. It has no Ebiten dependency
// so the simulation can be unit-tested in isolation.
type World struct {
	Player      *Player
	Enemies     []*Enemy
	Projectiles []*Projectile
	Gems        []*Gem

	// Choices are the purge options shown while State==StateLevelUp.
	Choices []Upgrade

	State State
	Tick  int
	Kills int

	turret          *Turret
	pendingLevelUps int
	spawnTimer      int
	rng             *rand.Rand
}

// Turret exposes the turret grid read-only so the scene layer can draw it.
func (w *World) Turret() *Turret { return w.turret }

// NewWorld builds a fresh run. seed makes enemy spawning and turret generation
// deterministic for tests; pass time.Now().UnixNano() for real gameplay.
func NewWorld(seed int64) *World {
	rng := rand.New(rand.NewSource(seed))
	cfg := DefaultTurretGenConfig(rng)
	turret := GenerateTurret(cfg, rng)
	weapons := turret.ActiveWeapons()

	p := &Player{
		Pos:         geom.PointF{X: 0, Y: 0},
		HP:          100,
		MaxHP:       100,
		Speed:       3,
		Radius:      16,
		Level:       1,
		XPToNext:    10,
		Weapons:     weapons,
		FacingAngle: -math.Pi / 2,
		Nippers:     startingNippers,
	}
	return &World{
		Player: p,
		State:  StatePlaying,
		turret: turret,
		rng:    rng,
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
	w.spawnEnemies()
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
	for _, weapon := range w.Player.Weapons {
		if weapon.cooldown > 0 {
			weapon.cooldown--
			continue
		}

		stats := weapon.StatsFromEnergy()
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
	for _, weapon := range w.Player.Weapons {
		if weapon.Kind != KindLaser || weapon.beamTicksLeft <= 0 {
			continue
		}
		weapon.beamTicksLeft--

		stats := weapon.StatsFromEnergy()
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
	var out []BeamView
	for _, weapon := range w.Player.Weapons {
		if weapon.Kind != KindLaser || weapon.beamTicksLeft <= 0 {
			continue
		}
		stats := weapon.StatsFromEnergy()
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
	w.Gems = append(w.Gems, &Gem{Pos: e.Pos, Value: e.XPValue, alive: true})
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
		if d <= e.Radius+w.Player.Radius && w.Player.invuln == 0 {
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
	const pickupRange = 28.0
	const magnetRange = 90.0
	const magnetSpeed = 4.0

	for _, g := range w.Gems {
		if !g.alive {
			continue
		}
		d := g.Pos.Distance(w.Player.Pos)
		switch {
		case d <= pickupRange:
			g.alive = false
			w.addXP(g.Value)
		case d <= magnetRange && d > 0:
			dir := w.Player.Pos.Subtract(g.Pos)
			g.Pos = g.Pos.Add(dir.Multiply(magnetSpeed / d))
		}
	}
}

func (w *World) addXP(v float64) {
	w.Player.XP += v
	for w.Player.XP >= w.Player.XPToNext {
		w.Player.XP -= w.Player.XPToNext
		w.Player.Level++
		w.Player.XPToNext = math.Ceil(w.Player.XPToNext * 1.25)
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
	atCap := w.turret.TileCount() >= maxTurretTiles
	choices := make([]Upgrade, 0, 3)
	for i := 0; i < 3; i++ {
		choices = append(choices, w.rollDoctorChoice(atCap))
	}
	w.Choices = choices
}

// rollDoctorChoice produces a single doctor offer. atCap forces nipper offers
// when the turret has hit its size limit (no more room to bolt tiles on).
func (w *World) rollDoctorChoice(atCap bool) Upgrade {
	doc := doctorNames[w.rng.Intn(len(doctorNames))]
	r := w.rng.Float64()

	// At cap, or occasionally otherwise, a doctor hands over spare nippers.
	if atCap || r < 0.12 {
		n := 2 + w.rng.Intn(2) // 2 or 3
		return Upgrade{
			Name: fmt.Sprintf("Dr. %s: spare nippers (+%d)", doc, n),
			Desc: "Plastic-model nippers to cut tiles mid-battle. They break fast.",
			Apply: func(world *World) {
				world.Player.Nippers += n
			},
		}
	}

	// Some doctors proudly install a useless gadget (dilutes power — the catch).
	if r < 0.37 {
		name := junkDeviceNames[w.rng.Intn(len(junkDeviceNames))]
		comp := Junk{DeviceName: name}
		return Upgrade{
			Name: fmt.Sprintf("Dr. %s installs a %s", doc, name),
			Desc: "A useless gadget. Adds a tile and dilutes every weapon's power.",
			Apply: func(world *World) {
				world.turret.AddTile(comp, world.rng)
				world.Player.Weapons = world.turret.ActiveWeapons()
			},
		}
	}

	// Otherwise, a new weapon tile.
	kind := pickWeaponKind(w.rng)
	comp := WeaponComponent{Weapon: NewWeapon(kind.String(), 0, kind)}
	return Upgrade{
		Name: fmt.Sprintf("Dr. %s bolts on a %s", doc, kind.String()),
		Desc: "Adds a weapon tile at a random spot — but splits power further.",
		Apply: func(world *World) {
			world.turret.AddTile(comp, world.rng)
			world.Player.Weapons = world.turret.ActiveWeapons()
		},
	}
}

func (w *World) spawnEnemies() {
	if w.spawnTimer > 0 {
		w.spawnTimer--
		return
	}
	interval := 60 - w.Tick/600
	if interval < 18 {
		interval = 18
	}
	w.spawnTimer = interval

	angle := w.rng.Float64() * 2 * math.Pi
	const spawnDist = 520.0
	pos := w.Player.Pos.Add(geom.PointFFromPolar(spawnDist, angle))

	w.Enemies = append(w.Enemies, &Enemy{
		Pos:     pos,
		HP:      10 + float64(w.Tick)/120.0,
		Speed:   1.2,
		Radius:  12,
		Damage:  8,
		XPValue: 3,
		alive:   true,
	})
}

func (w *World) compact() {
	w.Enemies = filterAlive(w.Enemies, func(e *Enemy) bool { return e.alive })
	w.Projectiles = filterAlive(w.Projectiles, func(p *Projectile) bool { return p.alive })
	w.Gems = filterAlive(w.Gems, func(g *Gem) bool { return g.alive })
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
