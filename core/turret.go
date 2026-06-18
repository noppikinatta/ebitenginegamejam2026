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
const TurretTileSize = 24.0

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

// ConsumerTileCount returns the number of connected non-generator tiles (the
// tiles that draw power). This is the value fed into the power curve to derive
// the fire-rate multiplier: cutting tiles lowers it, raising the multiplier.
func (t *Turret) ConsumerTileCount() int {
	_, n := t.connectedConsumers()
	return n
}

// PowerMultiplier interpolates the fire-rate multiplier for a given connected
// tile count over the curve. The curve is a list of PowerPoint sorted ascending
// by Tiles; below the first point the first Mult is used, above the last the
// last Mult, and in between it is linear. An empty curve yields 1.
func PowerMultiplier(curve []PowerPoint, tiles int) float64 {
	if len(curve) == 0 {
		return 1
	}
	if tiles <= curve[0].Tiles {
		return curve[0].Mult
	}
	last := curve[len(curve)-1]
	if tiles >= last.Tiles {
		return last.Mult
	}
	for i := 1; i < len(curve); i++ {
		a, b := curve[i-1], curve[i]
		if tiles <= b.Tiles {
			span := float64(b.Tiles - a.Tiles)
			if span <= 0 {
				return b.Mult
			}
			f := float64(tiles-a.Tiles) / span
			return a.Mult + (b.Mult-a.Mult)*f
		}
	}
	return last.Mult
}

// ComputePower returns a connectivity map: every connected non-generator tile
// gets a positive value, the generator gets 0, and disconnected (unreachable)
// or purged tiles are omitted. The actual value is no longer used for combat
// (power is a turret-wide multiplier); callers only check presence / > 0 to know
// whether a tile is powered, e.g. for render dimming.
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

// CutPreview returns the set of tiles that PurgeTile(idx) would remove: idx
// itself plus every tile that would become unreachable from the generator as a
// result (the cascade). It does NOT mutate the turret, so the scene can show a
// cut preview on hover. Returns nil if idx cannot be cut (missing, already
// purged, or a generator).
func (t *Turret) CutPreview(idx hexmap.Index) map[hexmap.Index]bool {
	tile, ok := t.tiles[idx]
	if !ok || tile.purged || t.IsGenerator(idx) {
		return nil
	}
	reachable := t.reachableExcluding(idx)
	result := map[hexmap.Index]bool{idx: true}
	for i, tl := range t.tiles {
		if tl.purged || tl.Component == nil || t.IsGenerator(i) || i == idx {
			continue
		}
		if !reachable[i] {
			result[i] = true
		}
	}
	return result
}

// reachableExcluding returns the set of active tiles reachable from the first
// generator through active tiles, treating excluded as if it were purged.
func (t *Turret) reachableExcluding(excluded hexmap.Index) map[hexmap.Index]bool {
	visited := map[hexmap.Index]bool{}
	if len(t.generators) == 0 {
		return visited
	}
	origin := t.generators[0]
	if origin == excluded {
		return visited
	}
	visited[origin] = true
	queue := []hexmap.Index{origin}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		var nbs []hexmap.Index
		nbs = cur.AppendAround(nbs)
		for _, nb := range nbs {
			if visited[nb] || nb == excluded {
				continue
			}
			nbTile := t.tiles[nb]
			if nbTile == nil || nbTile.purged || nbTile.Component == nil {
				continue
			}
			visited[nb] = true
			queue = append(queue, nb)
		}
	}
	return visited
}

// ActiveWeapons returns all weapon instances on connected (powered) tiles, with
// each weapon's TileIdx set to its turret position. Call this each tick (or
// after purging/adding tiles) to get the current armed weapon list. Power no
// longer scales per weapon; it is expressed as a turret-wide fire-rate
// multiplier (see PowerMultiplier) applied in Weapon.Stats.
func (t *Turret) ActiveWeapons() []*Weapon {
	power := t.WeaponPower()
	var weapons []*Weapon
	for idx := range power {
		tile := t.tiles[idx]
		if tile == nil {
			continue
		}
		if c, ok := tile.Component.(WeaponComponent); ok {
			c.Weapon.TileIdx = idx
			weapons = append(weapons, c.Weapon)
		}
	}
	return weapons
}
