package core

import (
	"math/rand"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// GeneratorConfig describes one generator tile to place on the turret.
type GeneratorConfig struct {
	Index hexmap.Index
	Power float64
}

// TurretGenConfig controls random turret generation.
type TurretGenConfig struct {
	// MaxTiles is the target total number of tiles (including generator).
	MaxTiles int

	// BranchProb is the probability [0,1] that a frontier tile spawns an
	// additional branch instead of continuing linearly. Higher values produce
	// more forking, lower values produce longer linear arms.
	BranchProb float64

	// WeaponDensity is the probability [0,1] that an eligible leaf or mid-arm
	// tile becomes a weapon (ProportionalWeapon) rather than a plain Wire.
	WeaponDensity float64

	// CapacitorDensity is the probability [0,1] that a non-weapon, non-generator
	// tile becomes a Capacitor instead of a Wire.
	CapacitorDensity float64

	// CapacitorMultiplier is the power multiplier applied by each Capacitor tile.
	CapacitorMultiplier float64

	// Generators lists the generator positions and their output power.
	// For the initial single-generator version, this should have exactly one entry.
	Generators []GeneratorConfig
}

// DefaultTurretGenConfig returns a starting config tuned for the initial game.
func DefaultTurretGenConfig(rng *rand.Rand) TurretGenConfig {
	return TurretGenConfig{
		MaxTiles:            22,
		BranchProb:          0.35,
		WeaponDensity:       0.45,
		CapacitorDensity:    0.10,
		CapacitorMultiplier: 1.5,
		Generators: []GeneratorConfig{
			{Index: hexmap.IdxXY(0, 0), Power: 100},
		},
	}
}

// GenerateTurret builds a randomized branch-like Turret using a frontier-growth
// algorithm seeded by rng. The shape is intentionally NOT a filled disc: it grows
// one tile at a time along the frontier, with a configurable branching probability
// that controls how tree-like vs. linear the result is.
func GenerateTurret(cfg TurretGenConfig, rng *rand.Rand) *Turret {
	tiles := make(map[hexmap.Index]*Tile)

	// Place generator tiles first.
	genPositions := make([]hexmap.Index, 0, len(cfg.Generators))
	for _, gc := range cfg.Generators {
		tiles[gc.Index] = &Tile{Component: Wire{}} // generator is treated as a wire (pass-through)
		genPositions = append(genPositions, gc.Index)
	}

	// Use the first generator's power for the Turret.
	// (Multi-gen extension: pass all powers; for now use cfg.Generators[0].Power)
	genPower := 100.0
	if len(cfg.Generators) > 0 {
		genPower = cfg.Generators[0].Power
	}

	// Frontier: list of (parent, candidate) pairs available to grow.
	type candidateEdge struct {
		parent    hexmap.Index
		candidate hexmap.Index
	}
	var frontier []candidateEdge

	addCandidates := func(parent hexmap.Index) {
		var neighbors []hexmap.Index
		neighbors = parent.AppendAround(neighbors)
		for _, nb := range neighbors {
			if _, exists := tiles[nb]; exists {
				continue
			}
			frontier = append(frontier, candidateEdge{parent: parent, candidate: nb})
		}
	}

	for _, gc := range cfg.Generators {
		addCandidates(gc.Index)
	}

	for len(tiles) < cfg.MaxTiles && len(frontier) > 0 {
		// Pick a random frontier entry.
		i := rng.Intn(len(frontier))
		edge := frontier[i]
		// Remove chosen (swap with last).
		frontier[i] = frontier[len(frontier)-1]
		frontier = frontier[:len(frontier)-1]

		// Skip if the candidate was already placed by another branch.
		if _, exists := tiles[edge.candidate]; exists {
			continue
		}

		// Place the tile with a randomly chosen component.
		idx := edge.candidate
		tiles[idx] = pickComponent(cfg, rng)

		// Optionally branch (add multiple new frontier entries) or grow linearly.
		addCandidates(idx)

		// BranchProb controls how much we prune the frontier after adding new
		// candidates. Low BranchProb → collapse multiple new candidates to one,
		// producing long linear arms. High BranchProb → keep all, producing forks.
		if rng.Float64() > cfg.BranchProb {
			// Collect entries from the new tile vs. the rest first (no aliasing).
			var fromIdx []candidateEdge
			var rest []candidateEdge
			for _, e := range frontier {
				if e.parent == idx {
					fromIdx = append(fromIdx, e)
				} else {
					rest = append(rest, e)
				}
			}
			// Retain at most one candidate from this parent.
			frontier = rest
			if len(fromIdx) > 0 {
				frontier = append(frontier, fromIdx[rng.Intn(len(fromIdx))])
			}
		}
	}

	return NewTurret(tiles, genPositions, genPower)
}

// pickComponent returns a tile whose Component is chosen probabilistically.
func pickComponent(cfg TurretGenConfig, rng *rand.Rand) *Tile {
	r := rng.Float64()
	if r < cfg.WeaponDensity {
		kind := pickWeaponKind(rng)
		w := NewWeapon(kind.String(), 0, kind)
		return &Tile{Component: ProportionalWeapon{Weapon: w}}
	}
	r -= cfg.WeaponDensity
	if r < cfg.CapacitorDensity {
		return &Tile{Component: Capacitor{Multiplier: cfg.CapacitorMultiplier}}
	}
	return &Tile{Component: Wire{}}
}

func pickWeaponKind(rng *rand.Rand) WeaponKind {
	switch rng.Intn(4) {
	case 0:
		return KindShotgun
	case 1:
		return KindSniper
	case 2:
		return KindLaser
	default:
		return KindCannon
	}
}
