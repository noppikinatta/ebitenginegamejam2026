package core

import (
	"math/rand"
	"strings"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// GeneratorConfig describes one generator tile to place on the turret.
type GeneratorConfig struct {
	Index hexmap.Index
	Power float64
}

// TurretGenConfig controls random turret generation.
type TurretGenConfig struct {
	// WeaponCount is the exact number of weapon tiles placed on the turret.
	WeaponCount int

	// JunkCount is the exact number of junk-device tiles placed on the turret.
	// The total tile count (including the generator, which also holds a component)
	// is WeaponCount + JunkCount; no bare Wire tiles are generated.
	JunkCount int

	// BranchProb is the probability [0,1] that a frontier tile spawns an
	// additional branch instead of continuing linearly. Higher values produce
	// more forking, lower values produce longer linear arms.
	BranchProb float64

	// Generators lists the generator positions and their output power.
	// For the initial single-generator version, this should have exactly one entry.
	Generators []GeneratorConfig
}

// GenerateTurret builds a randomized branch-like Turret using a frontier-growth
// algorithm seeded by rng. The shape is intentionally NOT a filled disc: it grows
// one tile at a time along the frontier, with a configurable branching probability
// that controls how tree-like vs. linear the result is.
func GenerateTurret(cfg TurretGenConfig, rng *rand.Rand) *Turret {
	tiles := make(map[hexmap.Index]*Tile)

	// Build the fixed loadout: a shuffled multiset of exactly WeaponCount weapons
	// and JunkCount junk devices. Composition is deterministic (no bare Wire tiles
	// are produced); only the placement and which weapon/junk lands on which tile
	// is randomized. The bag also bounds growth, so the turret ends up with
	// exactly len(bag) tiles.
	bag := buildLoadout(cfg, rng)
	next := 0
	popComponent := func() Component {
		c := bag[next]
		next++
		return c
	}

	// Place generator tiles first. The generator is the uncuttable connectivity
	// root (the anchor the cut-cascade hangs from), but it also holds a real
	// component (weapon / junk) like any tile, so the central slot isn't wasted on
	// an empty power tile. It is still excluded from the consumer count, so a
	// central weapon is effectively a "free" main gun.
	genPositions := make([]hexmap.Index, 0, len(cfg.Generators))
	for _, gc := range cfg.Generators {
		if next < len(bag) {
			tiles[gc.Index] = &Tile{Component: popComponent()}
		} else {
			tiles[gc.Index] = &Tile{Component: Wire{}}
		}
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

	for next < len(bag) && len(frontier) > 0 {
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

		// Place the tile with the next component from the loadout bag.
		idx := edge.candidate
		tiles[idx] = &Tile{Component: popComponent()}

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

// junkSpec defines one absurd device a doctor might bolt on. Name is the English
// display string (also the localisation slug source); Tall marks junk drawn as a
// tall, always-upright fixture rather than a flat tile. Emitter, when set, makes
// the junk periodically spit out a cosmetic projectile; nil junk is inert.
type junkSpec struct {
	Name    string
	Tall    bool
	Emitter *EmitterSpec
}

// junkSpecs is the pool of junk devices. The first group is purely cosmetic
// novelties; the second group is the "emits something" junk — only those wired
// with an Emitter actually fire (the rest are inert until their behaviour lands).
var junkSpecs = []junkSpec{
	// Inert novelties.
	{Name: "Unusual Banana"},
	{Name: "Electric Fan"},
	{Name: "Calculator"},
	{Name: "Wi-Fi Antenna"},
	{Name: "Five-storied Pagoda", Tall: true},
	{Name: "Fax Machine"},
	{Name: "Lava Lamp"},
	{Name: "Oil Heater"},
	{Name: "Rice Cooker"},
	{Name: "Modern Art Fountain"},
	{Name: "Concept"},
	{Name: "NFT Nuclear Missile"},
	{Name: "Horns"},
	{Name: "High-End GPU"},
	// Devices that emit something. Each is wired to a cosmetic emitter.
	{Name: "Balloon Service Unit", Emitter: &balloonEmitter},
	{Name: "Coffee Maker", Emitter: &coffeeEmitter},
	{Name: "Toaster", Emitter: &toasterEmitter},
	{Name: "Music Box", Emitter: &musicBoxEmitter},
	{Name: "Rubber Duck Dispenser", Emitter: &duckEmitter},
	{Name: "Fireworks", Emitter: &fireworksEmitter},
}

// junkBySpec finds the spec for a device name (nil if not in the pool).
func junkBySpec(name string) *junkSpec {
	for i := range junkSpecs {
		if junkSpecs[i].Name == name {
			return &junkSpecs[i]
		}
	}
	return nil
}

// JunkDeviceNames returns every junk device name in the pool, in declaration
// order. Exposed so tooling (e.g. placeholder image generation) can enumerate
// the junk that needs an image without reaching into core internals.
func JunkDeviceNames() []string {
	names := make([]string, len(junkSpecs))
	for i := range junkSpecs {
		names[i] = junkSpecs[i].Name
	}
	return names
}

// JunkDeviceTall reports whether the named device renders as a tall fixture
// (and therefore needs a taller-than-tile image).
func JunkDeviceTall(name string) bool {
	if s := junkBySpec(name); s != nil {
		return s.Tall
	}
	return false
}

// JunkImageKey is the per-device image key (and PNG base name) for a junk
// device, so every junk type gets its own art. It slugifies the display name:
// lowercase, runs of non-alphanumerics collapse to a single underscore, e.g.
// "Wi-Fi Antenna" -> "junk_wi_fi_antenna". The placeholder-image generator and
// the scene renderer share this one mapping so files line up with lookups.
func JunkImageKey(deviceName string) string {
	var b strings.Builder
	b.WriteString("junk_")
	pendingUnderscore := false
	wroteBody := false
	for _, r := range strings.ToLower(deviceName) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingUnderscore && wroteBody {
				b.WriteByte('_')
			}
			b.WriteRune(r)
			wroteBody = true
			pendingUnderscore = false
		} else {
			pendingUnderscore = true
		}
	}
	return b.String()
}

// junkFromSpec builds a Junk from a spec, attaching a fresh emitter (with its
// own firing accumulator) when the spec defines one.
func junkFromSpec(s junkSpec) Junk {
	j := Junk{DeviceName: s.Name, Tall: s.Tall}
	if s.Emitter != nil {
		spec := *s.Emitter
		j.emitter = &junkEmitter{spec: spec}
	}
	return j
}

// newJunk builds a Junk for the given device name (inert if unknown).
func newJunk(name string) Junk {
	if s := junkBySpec(name); s != nil {
		return junkFromSpec(*s)
	}
	return Junk{DeviceName: name}
}

// randomJunk builds a Junk with a random device from the pool.
func randomJunk(rng *rand.Rand) Junk {
	return junkFromSpec(junkSpecs[rng.Intn(len(junkSpecs))])
}

// buildLoadout returns the shuffled multiset of components to place on the
// turret: exactly WeaponCount weapons (random kinds) followed by JunkCount junk
// devices, then shuffled so the weapons and junk are interleaved across the
// generated shape. No Wire tiles are included.
func buildLoadout(cfg TurretGenConfig, rng *rand.Rand) []Component {
	bag := make([]Component, 0, cfg.WeaponCount+cfg.JunkCount)
	for i := 0; i < cfg.WeaponCount; i++ {
		kind := pickWeaponKind(rng)
		bag = append(bag, WeaponComponent{Weapon: NewWeapon(kind.String(), kind)})
	}
	for i := 0; i < cfg.JunkCount; i++ {
		bag = append(bag, randomJunk(rng))
	}
	rng.Shuffle(len(bag), func(i, j int) { bag[i], bag[j] = bag[j], bag[i] })
	return bag
}

func pickWeaponKind(rng *rand.Rand) WeaponKind {
	switch rng.Intn(8) {
	case 0:
		return KindShotgun
	case 1:
		return KindSniper
	case 2:
		return KindLaser
	case 3:
		return KindGatling
	case 4:
		return KindGrenade
	case 5:
		return KindCIWS
	case 6:
		return KindMissile
	default:
		return KindCannon
	}
}
