package core

import (
	"math"
	"math/rand"
	"sort"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// TurretTileSize is the world-space spacing of turret tiles (= screen px at the
// game's 1:1 camera). The scene combat renderer must use the same value so that
// drawn tiles line up with the muzzle positions weapons fire from.
const TurretTileSize = 14.0

// tileLocalOffset returns a tile's unrotated offset from the turret centre in
// world units, using the vertical brick layout where local "up" (-Y) = forward.
func tileLocalOffset(idx hexmap.Index) geom.PointF {
	q := float64(idx.X())
	r := float64(idx.Y())
	return geom.PointF{X: q * TurretTileSize * 0.866, Y: (r + q*0.5) * TurretTileSize}
}

// MuzzleOffset returns the world-space offset of a turret tile from the tank
// centre, rotated so the turret's local "up" aligns with facingAngle. Add this
// to Player.Pos to get the world position a weapon on that tile fires from.
func MuzzleOffset(idx hexmap.Index, facingAngle float64) geom.PointF {
	off := tileLocalOffset(idx)
	if off.Abs() == 0 {
		return off
	}
	theta := facingAngle + math.Pi/2
	return geom.PointFFromPolar(off.Abs(), off.Angle()+theta)
}

// Component is what occupies a single hex tile on the turret. Power is shared
// flatly across all connected tiles, so a component's only job is to identify
// itself and (for weapons) carry the mounted Weapon.
type Component interface {
	Name() string
}

// Wire is a passive conductor: it carries power but does nothing with it.
type Wire struct{}

func (Wire) Name() string { return "Wire" }

// WeaponComponent mounts a weapon on a tile. The weapon's Energy is set to the
// tile's share of generator power by ActiveWeapons.
type WeaponComponent struct {
	Weapon *Weapon
}

func (w WeaponComponent) Name() string { return w.Weapon.Name }

// Junk is a useless device a doctor bolted on (espresso machine, balloon
// launcher, rubber duck...). It conducts power like a wire but does nothing,
// diluting the per-tile power share. The flavour is the point.
type Junk struct {
	DeviceName string
}

func (j Junk) Name() string {
	if j.DeviceName == "" {
		return "Junk"
	}
	return j.DeviceName
}

// ---- Tile ----

// Tile is a single hex cell on the turret grid.
type Tile struct {
	Component Component // nil means the tile slot is empty (no component placed)
	purged    bool      // true when tile-purged: tile is gone, carries no power
}

// IsActive reports whether the tile carries power (not purged).
func (t *Tile) IsActive() bool {
	return !t.purged && t.Component != nil
}

// IsPurged reports whether the tile has been purged from the grid.
func (t *Tile) IsPurged() bool {
	return t.purged
}

// ---- Turret ----

// Turret is the hex-grid turret mounted on the tank. A generator tile emits a
// fixed power level which is shared evenly across every connected (reachable
// through active tiles) non-generator tile. More tiles → less power each, so a
// bloated turret makes every weapon weaker.
//
// Data model is generator-list-friendly for future multi-generator expansion,
// but the current version uses a single central generator.
type Turret struct {
	tiles      map[hexmap.Index]*Tile
	generators []hexmap.Index
	genPower   float64 // total power output of the generator
}

// NewTurret creates a Turret with the given generator positions and generator
// power output. Tiles map is shared (caller should not mutate it externally).
func NewTurret(tiles map[hexmap.Index]*Tile, generators []hexmap.Index, genPower float64) *Turret {
	return &Turret{tiles: tiles, generators: generators, genPower: genPower}
}

// Tiles returns the internal tiles map (read-only use expected).
func (t *Turret) Tiles() map[hexmap.Index]*Tile {
	return t.tiles
}

// Generators returns the generator positions.
func (t *Turret) Generators() []hexmap.Index {
	return t.generators
}

// connectedConsumers returns the set of tiles reachable from the generator
// (including the generator) and the count of non-generator consumer tiles among
// them. Returns (nil, 0) if there is no generator.
func (t *Turret) connectedConsumers() (reachable map[hexmap.Index]int, n int) {
	if len(t.generators) == 0 {
		return nil, 0
	}
	reachable = t.distancesFrom(t.generators[0])
	for idx := range reachable {
		if t.IsGenerator(idx) {
			continue
		}
		n++
	}
	return reachable, n
}

// PowerPerTile returns the power each connected non-generator tile receives:
// the generator's output divided evenly among all connected consumer tiles.
// Returns 0 when there are no consumer tiles.
func (t *Turret) PowerPerTile() float64 {
	_, n := t.connectedConsumers()
	if n == 0 {
		return 0
	}
	return t.genPower / float64(n)
}

// ComputePower returns the power delivered to each connected tile. Every
// connected non-generator tile gets PowerPerTile; the generator gets 0.
// Disconnected (unreachable) and purged tiles are omitted.
func (t *Turret) ComputePower() map[hexmap.Index]float64 {
	reachable, n := t.connectedConsumers()
	result := map[hexmap.Index]float64{}
	if n == 0 {
		return result
	}
	per := t.genPower / float64(n)
	for idx := range reachable {
		if t.IsGenerator(idx) {
			result[idx] = 0
			continue
		}
		result[idx] = per
	}
	return result
}

// WeaponPower returns a map of hex index → power delivered to each weapon tile.
// Only active weapon tiles appear in the result.
func (t *Turret) WeaponPower() map[hexmap.Index]float64 {
	power := t.ComputePower()
	out := make(map[hexmap.Index]float64)
	for idx, p := range power {
		tile := t.tiles[idx]
		if tile == nil || tile.purged {
			continue
		}
		if _, ok := tile.Component.(WeaponComponent); ok {
			out[idx] = p
		}
	}
	return out
}

// distancesFrom returns the hex distance from origin for all active (non-purged,
// non-nil-component) tiles reachable through active tile paths. The origin tile
// itself is always included (distance 0).
func (t *Turret) distancesFrom(origin hexmap.Index) map[hexmap.Index]int {
	dist := map[hexmap.Index]int{origin: 0}
	queue := []hexmap.Index{origin}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curD := dist[cur]
		var neighbors []hexmap.Index
		neighbors = cur.AppendAround(neighbors)
		for _, nb := range neighbors {
			if _, visited := dist[nb]; visited {
				continue
			}
			nbTile := t.tiles[nb]
			if nbTile == nil || nbTile.purged || nbTile.Component == nil {
				continue
			}
			dist[nb] = curD + 1
			queue = append(queue, nb)
		}
	}
	return dist
}

// TileCount returns the number of active (non-purged) tiles, including the
// generator. Used to gate turret growth.
func (t *Turret) TileCount() int {
	n := 0
	for _, tile := range t.tiles {
		if !tile.purged {
			n++
		}
	}
	return n
}

// AddTile places comp on a random empty cell adjacent to an existing active
// tile, growing the turret outward. Purged cells are reusable. Returns the
// chosen index and true, or false if there is no adjacent room.
func (t *Turret) AddTile(comp Component, rng *rand.Rand) (hexmap.Index, bool) {
	seen := map[hexmap.Index]bool{}
	var candidates []hexmap.Index
	for idx, tile := range t.tiles {
		if tile.purged || tile.Component == nil {
			continue
		}
		var nbs []hexmap.Index
		nbs = idx.AppendAround(nbs)
		for _, nb := range nbs {
			if existing, ok := t.tiles[nb]; ok && !existing.purged {
				continue // occupied by an active tile
			}
			if seen[nb] {
				continue
			}
			seen[nb] = true
			candidates = append(candidates, nb)
		}
	}
	if len(candidates) == 0 {
		return hexmap.Index{}, false
	}
	// Sort for determinism before the random pick.
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].X() != candidates[j].X() {
			return candidates[i].X() < candidates[j].X()
		}
		return candidates[i].Y() < candidates[j].Y()
	})
	chosen := candidates[rng.Intn(len(candidates))]
	t.tiles[chosen] = &Tile{Component: comp}
	return chosen, true
}

// IsGenerator reports whether idx is a generator position.
func (t *Turret) IsGenerator(idx hexmap.Index) bool {
	for _, g := range t.generators {
		if g == idx {
			return true
		}
	}
	return false
}

// PurgeTile removes the tile at idx entirely. The tile stops conducting power;
// any tiles that become unreachable from the generator as a result are also
// marked purged (cascade), so the visible turret matches its connectivity.
// Returns false if the tile does not exist, is a generator, or is already purged.
func (t *Turret) PurgeTile(idx hexmap.Index) bool {
	tile, ok := t.tiles[idx]
	if !ok || tile.purged || t.IsGenerator(idx) {
		return false
	}
	tile.purged = true
	t.propagatePurge()
	return true
}

// propagatePurge marks every tile that can no longer be reached from the
// generator (through active tiles) as purged, cascading the effect of a
// tile-purge to its now-orphaned downstream tiles.
func (t *Turret) propagatePurge() {
	if len(t.generators) == 0 {
		return
	}
	reachable := t.distancesFrom(t.generators[0])
	for idx, tile := range t.tiles {
		if tile.purged {
			continue
		}
		if _, ok := reachable[idx]; !ok {
			tile.purged = true
		}
	}
}

// ActiveWeapons returns all weapon instances on active, powered tiles, with each
// weapon's Energy set to its tile's flat power share. Call this each tick (or
// after purging/adding tiles) to get the current armed weapon list.
func (t *Turret) ActiveWeapons() []*Weapon {
	power := t.WeaponPower()
	var weapons []*Weapon
	for idx, p := range power {
		tile := t.tiles[idx]
		if tile == nil {
			continue
		}
		if c, ok := tile.Component.(WeaponComponent); ok {
			c.Weapon.Energy = p
			c.Weapon.TileIdx = idx
			weapons = append(weapons, c.Weapon)
		}
	}
	return weapons
}
