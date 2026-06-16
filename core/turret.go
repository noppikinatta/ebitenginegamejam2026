package core

import (
	"sort"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// Component is the behavior of a single hex tile on the turret.
// Distribute is called during power solving with the total power received by
// this tile and the number of downstream neighbors that need power. It returns
// (selfConsumption, throughput): selfConsumption is how much this tile "uses"
// (e.g. for a weapon), throughput is the power forwarded to downstream tiles.
// The solver ensures selfConsumption + throughput <= received.
type Component interface {
	Distribute(received float64, downstreamCount int) (selfConsumption, throughput float64)
}

// Wire is a passive conductor: passes all power downstream, consumes nothing.
type Wire struct{}

func (Wire) Distribute(received float64, downstreamCount int) (float64, float64) {
	return 0, received
}

// Capacitor amplifies power by a fixed multiplier before forwarding downstream.
// It never stores power; the amplification is instantaneous.
type Capacitor struct {
	Multiplier float64
}

func (c Capacitor) Distribute(received float64, downstreamCount int) (float64, float64) {
	return 0, received * c.Multiplier
}

// ProportionalWeapon consumes a share of received power for itself and
// forwards the rest equally to downstream neighbors. The share consumed equals
// 1/(downstreamCount+1) of received power, keeping one part.
type ProportionalWeapon struct {
	Weapon *Weapon
}

func (pw ProportionalWeapon) Distribute(received float64, downstreamCount int) (float64, float64) {
	parts := float64(downstreamCount + 1)
	self := received / parts
	return self, received - self
}

// ThresholdWeapon requires at least MinPower to activate. If received >= MinPower
// it consumes MinPower and forwards the remainder; otherwise it does not activate
// and forwards nothing (the Weapon.Energy is set to 0 to signal inactivation).
type ThresholdWeapon struct {
	Weapon   *Weapon
	MinPower float64
}

func (tw ThresholdWeapon) Distribute(received float64, downstreamCount int) (float64, float64) {
	if received < tw.MinPower {
		return 0, 0
	}
	return tw.MinPower, received - tw.MinPower
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

// ---- Turret ----

// Turret is the hex-grid turret mounted on the tank. Generator tiles at given
// positions emit a fixed power level; power flows outward from each generator
// through connected tiles following the BFS distance-ring algorithm.
//
// Data model is generator-list-friendly for future multi-generator expansion,
// but the current solver only handles a single central generator.
type Turret struct {
	tiles      map[hexmap.Index]*Tile
	generators []hexmap.Index // in multi-gen future: independent distribution passes
	genPower   float64        // power output per generator tile
}

// NewTurret creates a Turret with the given generator positions and per-generator
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

// WeaponPower returns a map of hex index → power delivered to each weapon tile,
// computed by ComputePower. Only tiles with ProportionalWeapon or ThresholdWeapon
// components appear in the result; inactive/non-weapon tiles are omitted.
func (t *Turret) WeaponPower() map[hexmap.Index]float64 {
	power := t.ComputePower()
	out := make(map[hexmap.Index]float64)
	for idx, p := range power {
		tile := t.tiles[idx]
		if tile == nil || tile.purged {
			continue
		}
		switch tile.Component.(type) {
		case ProportionalWeapon, ThresholdWeapon:
			out[idx] = p
		}
	}
	return out
}

// ComputePower runs the BFS power-distribution solver.
//
// Algorithm (single-generator case; generalises to multi-gen with independent
// passes sharing no tile ownership):
//
//  1. Start with the generator tile receiving genPower.
//  2. Process tiles in increasing hex-distance rings from the generator.
//  3. A tile receives power only from strictly-closer neighbors (prevents back-flow).
//  4. Multiple strictly-closer parents sum their contributions into the child.
//  5. Each tile's Component.Distribute splits received power into selfConsumption
//     and throughput; throughput is divided equally among downstream neighbors
//     (those exactly one ring further away that are active tiles).
//  6. If a tile has no active downstream neighbors, all throughput is absorbed
//     (it becomes the edge of the grid; nothing is wasted by caller design).
func (t *Turret) ComputePower() map[hexmap.Index]float64 {
	if len(t.generators) == 0 {
		return nil
	}
	// For now, single-generator scope.
	gen := t.generators[0]
	return t.computeFromGenerator(gen, t.genPower)
}

func (t *Turret) computeFromGenerator(gen hexmap.Index, startPower float64) map[hexmap.Index]float64 {
	// receivedPower accumulates contributions to each tile from closer-ring parents.
	receivedPower := map[hexmap.Index]float64{}
	receivedPower[gen] = startPower

	// Build distance map from generator for all active tiles.
	dist := t.distancesFrom(gen)

	// Order all active-tile indices by distance so we process each ring in order.
	type idxDist struct {
		idx hexmap.Index
		d   int
	}
	ordered := make([]idxDist, 0, len(dist))
	for idx, d := range dist {
		ordered = append(ordered, idxDist{idx, d})
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].d < ordered[j].d
	})

	result := map[hexmap.Index]float64{}

	for _, id := range ordered {
		idx := id.idx
		received, ok := receivedPower[idx]
		if !ok {
			// No power reached this tile (upstream was cut or zero).
			result[idx] = 0
			continue
		}

		tile := t.tiles[idx]
		if tile == nil || tile.purged || tile.Component == nil {
			// Empty/purged slot: no distribution.
			result[idx] = 0
			continue
		}

		// Find active downstream neighbors (exactly dist+1 away and present in grid).
		downstream := t.activeDownstream(idx, id.d, dist)

		selfConsumption, throughput := tile.Component.Distribute(received, len(downstream))
		result[idx] = selfConsumption

		if len(downstream) == 0 || throughput <= 0 {
			continue
		}
		share := throughput / float64(len(downstream))
		for _, dnIdx := range downstream {
			receivedPower[dnIdx] += share
		}
	}

	return result
}

// distancesFrom returns the hex distance from origin for all active (non-purged,
// non-nil-component) tiles in the turret, reachable through active tile paths.
// The generator tile itself is always included (distance 0).
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

// activeDownstream returns active neighbor indices whose distance from the
// generator is exactly thisDist+1.
func (t *Turret) activeDownstream(idx hexmap.Index, thisDist int, dist map[hexmap.Index]int) []hexmap.Index {
	var result []hexmap.Index
	var neighbors []hexmap.Index
	neighbors = idx.AppendAround(neighbors)
	for _, nb := range neighbors {
		d, present := dist[nb]
		if !present {
			continue
		}
		if d != thisDist+1 {
			continue
		}
		nbTile := t.tiles[nb]
		if nbTile == nil || nbTile.purged || nbTile.Component == nil {
			continue
		}
		result = append(result, nb)
	}
	return result
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

// CanPurgeTile returns true if purging the tile at idx (marking it gone and
// cutting its downstream) would still leave at least minWeapons active weapons.
// Returns false for generator tiles, absent tiles, or already-purged tiles.
func (t *Turret) CanPurgeTile(idx hexmap.Index, minWeapons int) bool {
	tile, ok := t.tiles[idx]
	if !ok || tile.purged || t.IsGenerator(idx) {
		return false
	}
	orig := tile.purged
	tile.purged = true
	n := len(t.ActiveWeapons())
	tile.purged = orig
	return n >= minWeapons
}

// CanPurgeWeapon returns true if replacing the weapon component at idx with a
// Wire would leave at least minWeapons active weapons. Returns false for
// non-weapon, absent, or purged tiles.
func (t *Turret) CanPurgeWeapon(idx hexmap.Index, minWeapons int) bool {
	tile, ok := t.tiles[idx]
	if !ok || tile.purged {
		return false
	}
	switch tile.Component.(type) {
	case ProportionalWeapon, ThresholdWeapon:
	default:
		return false
	}
	orig := tile.Component
	tile.Component = Wire{}
	n := len(t.ActiveWeapons())
	tile.Component = orig
	return n >= minWeapons
}

// PurgeTile removes the tile at idx entirely. The tile stops conducting power;
// all downstream tiles lose their power supply. Returns false if the tile does
// not exist or is already purged.
func (t *Turret) PurgeTile(idx hexmap.Index) bool {
	tile, ok := t.tiles[idx]
	if !ok || tile.purged {
		return false
	}
	tile.purged = true
	return true
}

// PurgeWeapon replaces the component at idx with a Wire, so the tile continues
// to conduct power downstream but the weapon is removed. Returns false if the
// tile does not exist, is purged, or has no weapon component.
func (t *Turret) PurgeWeapon(idx hexmap.Index) bool {
	tile, ok := t.tiles[idx]
	if !ok || tile.purged {
		return false
	}
	switch tile.Component.(type) {
	case ProportionalWeapon, ThresholdWeapon:
		tile.Component = Wire{}
		return true
	}
	return false
}

// ActiveWeapons returns all weapon instances on active, powered tiles, with their
// received power applied to Weapon.Energy. Call this each tick (or after purging)
// to get the current armed weapon list.
func (t *Turret) ActiveWeapons() []*Weapon {
	power := t.WeaponPower()
	var weapons []*Weapon
	for idx, p := range power {
		tile := t.tiles[idx]
		if tile == nil {
			continue
		}
		var w *Weapon
		switch c := tile.Component.(type) {
		case ProportionalWeapon:
			w = c.Weapon
		case ThresholdWeapon:
			if p >= c.MinPower {
				w = c.Weapon
			}
		}
		if w != nil {
			w.Energy = p
			weapons = append(weapons, w)
		}
	}
	return weapons
}
