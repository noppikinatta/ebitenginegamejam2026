package core

import (
	"math"
	"math/rand"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

// Logical playfield is unbounded; these are the render viewport dimensions the
// camera follows, used here only to decide where to spawn off-screen enemies.
const (
	ViewW = 1280
	ViewH = 720
)

// State is the high-level state of a run.
type State int

const (
	StatePlaying State = iota
	StateLevelUp // reserved for the Phase 2 level-up choice screen
	StateGameOver
)

// World holds all gameplay state for a single run. It has no dependency on
// Ebiten so the simulation can be unit-tested in isolation.
type World struct {
	Player      *Player
	Enemies     []*Enemy
	Projectiles []*Projectile
	Gems        []*Gem

	State State
	Tick  int
	Kills int

	spawnTimer int
	rng        *rand.Rand
}

// NewWorld builds a fresh run. seed makes enemy spawning deterministic for tests.
func NewWorld(seed int64) *World {
	p := &Player{
		Pos:      geom.PointF{X: 0, Y: 0},
		HP:       100,
		MaxHP:    100,
		Speed:    3,
		Radius:   16,
		Level:    1,
		XPToNext: 10,
		Weapons:  []*Weapon{NewWeapon("Cannon", 3)},
	}
	return &World{
		Player: p,
		State:  StatePlaying,
		rng:    rand.New(rand.NewSource(seed)),
	}
}

// Update advances the simulation by one tick. move is the desired movement
// direction (each axis in [-1,1]); it is normalised internally so diagonal
// movement is not faster.
func (w *World) Update(move geom.PointF) {
	if w.State != StatePlaying {
		return
	}

	w.Tick++
	w.updatePlayer(move)
	w.updateWeapons()
	w.updateProjectiles()
	w.updateEnemies()
	w.updateGems()
	w.spawnEnemies()
	w.compact()
}

func (w *World) updatePlayer(move geom.PointF) {
	if mag := move.Abs(); mag > 1 {
		move = move.Multiply(1 / mag)
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

		dir := target.Pos.Subtract(w.Player.Pos)
		d := dir.Abs()
		if d == 0 {
			continue
		}

		w.Projectiles = append(w.Projectiles, &Projectile{
			Pos:    w.Player.Pos,
			Vel:    dir.Multiply(stats.ProjectileSpeed / d),
			Damage: stats.Damage,
			Radius: 5,
			Life:   120,
			alive:  true,
		})
		weapon.cooldown = stats.FireInterval
	}
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
		w.levelUp()
	}
}

func (w *World) levelUp() {
	w.Player.Level++
	w.Player.XPToNext = math.Ceil(w.Player.XPToNext * 1.25)

	// Phase 1 placeholder reward until the Phase 2 level-up choice / Disconnect
	// screen: route a bit more energy to the first weapon and heal a little.
	if len(w.Player.Weapons) > 0 {
		w.Player.Weapons[0].Energy++
	}
	w.Player.HP = math.Min(w.Player.MaxHP, w.Player.HP+10)
}

func (w *World) spawnEnemies() {
	if w.spawnTimer > 0 {
		w.spawnTimer--
		return
	}

	// Spawn cadence accelerates over time (down to one every 18 ticks).
	interval := 60 - w.Tick/600
	if interval < 18 {
		interval = 18
	}
	w.spawnTimer = interval

	angle := w.rng.Float64() * 2 * math.Pi
	const spawnDist = 520.0 // just beyond the visible viewport around the player
	pos := w.Player.Pos.Add(geom.PointFFromPolar(spawnDist, angle))

	w.Enemies = append(w.Enemies, &Enemy{
		Pos:     pos,
		HP:      10 + float64(w.Tick)/120.0, // tougher as the run goes on
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

// filterAlive compacts a slice in place, keeping only elements for which
// alive returns true. The backing array is reused, so dead entries do not
// accumulate across ticks.
func filterAlive[T any](s []T, alive func(T) bool) []T {
	out := s[:0]
	for _, v := range s {
		if alive(v) {
			out = append(out, v)
		}
	}
	return out
}
