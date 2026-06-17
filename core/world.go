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

// speedBonusPerTilePurge is the px/tick speed gained per tile-purge.
const speedBonusPerTilePurge = 0.3

// startingNippers is how many tile cuts the player can make before needing to
// find more. Nippers are the limited resource that gates mid-combat cutting.
const startingNippers = 3

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

// rollChoices builds the list of available purge choices. If nothing can be
// purged (last weapon), applies a fallback heal instead of showing the menu.
func (w *World) rollChoices() {
	choices := w.buildDisconnectChoices()
	if len(choices) == 0 {
		// Nothing to purge: heal the player as a consolation reward.
		heal := w.Player.MaxHP * 0.1
		w.Player.HP = math.Min(w.Player.HP+heal, w.Player.MaxHP)
		w.pendingLevelUps--
		if w.pendingLevelUps > 0 {
			w.rollChoices()
		} else {
			w.State = StatePlaying
		}
		return
	}
	w.Choices = choices
}

// buildDisconnectChoices returns the set of tile-purge and weapon-purge choices
// available on the current turret. At least one weapon must survive each choice.
func (w *World) buildDisconnectChoices() []Upgrade {
	var choices []Upgrade
	for idx := range w.turret.Tiles() {
		if w.turret.IsGenerator(idx) {
			continue
		}
		idx := idx // capture for closure

		if w.turret.CanPurgeTile(idx, 1) {
			name, desc := tileLabel(w.turret, idx)
			choices = append(choices, Upgrade{
				Name: name,
				Desc: desc,
				Apply: func(world *World) {
					world.turret.PurgeTile(idx)
					world.Player.Speed += speedBonusPerTilePurge
					world.Player.Weapons = world.turret.ActiveWeapons()
				},
			})
		}
	}
	return choices
}

// tileLabel returns the display name and description for a tile-purge choice.
func tileLabel(t *Turret, idx hexmap.Index) (name, desc string) {
	tile := t.Tiles()[idx]
	compName := componentName(tile)

	// Count downstream tiles that would be cut along with this one.
	var downstream int
	for checkIdx, checkTile := range t.Tiles() {
		if checkTile.purged || t.IsGenerator(checkIdx) {
			continue
		}
		if idx == checkIdx {
			continue
		}
		// Temporarily purge to test reachability after cut.
		tile.purged = true
		dist := t.distancesFrom(t.generators[0])
		tile.purged = false
		if _, reachable := dist[checkIdx]; !reachable {
			downstream++
		}
	}

	if downstream > 0 {
		name = fmt.Sprintf("Cut %s %s", compName, idx)
		desc = fmt.Sprintf("Remove tile + %d downstream — gain +%.1f speed", downstream, speedBonusPerTilePurge)
	} else {
		name = fmt.Sprintf("Cut %s %s", compName, idx)
		desc = fmt.Sprintf("Remove tile — gain +%.1f speed", speedBonusPerTilePurge)
	}
	return
}

func componentName(tile *Tile) string {
	if tile == nil || tile.Component == nil {
		return "?"
	}
	return tile.Component.Name()
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
