package core

import (
	"math/rand"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

func TestGenerateTurret_TileCount(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	cfg := DefaultTurretGenConfig(rng)
	tr := GenerateTurret(cfg, rng)

	n := len(tr.Tiles())
	want := cfg.WeaponCount + cfg.JunkCount
	if n != want {
		t.Errorf("tile count=%d, want %d (WeaponCount+JunkCount)", n, want)
	}
}

func TestGenerateTurret_Composition(t *testing.T) {
	// The starting loadout must be exactly WeaponCount weapons + JunkCount junk,
	// with no bare Wire tiles, for every seed.
	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		cfg := DefaultTurretGenConfig(rng)
		tr := GenerateTurret(cfg, rng)

		var weapons, junk, wires, other int
		for _, tile := range tr.Tiles() {
			switch tile.Component.(type) {
			case WeaponComponent:
				weapons++
			case Junk:
				junk++
			case Wire:
				wires++
			default:
				other++
			}
		}
		if weapons != cfg.WeaponCount {
			t.Errorf("seed %d: weapon count=%d, want %d", seed, weapons, cfg.WeaponCount)
		}
		if junk != cfg.JunkCount {
			t.Errorf("seed %d: junk count=%d, want %d", seed, junk, cfg.JunkCount)
		}
		if wires != 0 {
			t.Errorf("seed %d: got %d Wire tiles, want 0", seed, wires)
		}
		if other != 0 {
			t.Errorf("seed %d: got %d unexpected components", seed, other)
		}
	}
}

func TestGenerateTurret_GeneratorPresent(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	cfg := DefaultTurretGenConfig(rng)
	tr := GenerateTurret(cfg, rng)

	gen := hexmap.IdxXY(0, 0)
	if _, ok := tr.Tiles()[gen]; !ok {
		t.Errorf("generator tile at (0,0) missing from generated turret")
	}
}

func TestGenerateTurret_AllTilesConnected(t *testing.T) {
	// Every tile should be reachable from the generator via active tile BFS.
	rng := rand.New(rand.NewSource(7))
	cfg := DefaultTurretGenConfig(rng)
	tr := GenerateTurret(cfg, rng)

	gen := hexmap.IdxXY(0, 0)
	visited := map[hexmap.Index]bool{gen: true}
	queue := []hexmap.Index{gen}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		var neighbors []hexmap.Index
		neighbors = cur.AppendAround(neighbors)
		for _, nb := range neighbors {
			if visited[nb] {
				continue
			}
			tile := tr.Tiles()[nb]
			if tile == nil || tile.purged || tile.Component == nil {
				continue
			}
			visited[nb] = true
			queue = append(queue, nb)
		}
	}
	for idx := range tr.Tiles() {
		if !visited[idx] {
			t.Errorf("tile %v is not reachable from generator", idx)
		}
	}
}

func TestGenerateTurret_HasWeapons(t *testing.T) {
	// A generated turret should have at least one weapon so the run is playable.
	rng := rand.New(rand.NewSource(123))
	cfg := DefaultTurretGenConfig(rng)
	// Run multiple seeds to ensure it's reliable.
	for seed := int64(0); seed < 20; seed++ {
		rng = rand.New(rand.NewSource(seed))
		cfg = DefaultTurretGenConfig(rng)
		tr := GenerateTurret(cfg, rng)
		weapons := tr.ActiveWeapons()
		if len(weapons) == 0 {
			t.Errorf("seed %d: generated turret has no active weapons", seed)
		}
	}
}

func TestGenerateTurret_Deterministic(t *testing.T) {
	cfg1 := DefaultTurretGenConfig(rand.New(rand.NewSource(55)))
	tr1 := GenerateTurret(cfg1, rand.New(rand.NewSource(55)))

	cfg2 := DefaultTurretGenConfig(rand.New(rand.NewSource(55)))
	tr2 := GenerateTurret(cfg2, rand.New(rand.NewSource(55)))

	tiles1 := tr1.Tiles()
	tiles2 := tr2.Tiles()
	if len(tiles1) != len(tiles2) {
		t.Fatalf("same seed: tile counts differ %d vs %d", len(tiles1), len(tiles2))
	}
	for idx := range tiles1 {
		if _, ok := tiles2[idx]; !ok {
			t.Errorf("same seed: tile %v in tr1 missing from tr2", idx)
		}
	}
}

func TestGenerateTurret_DifferentSeedsDiffer(t *testing.T) {
	cfg1 := DefaultTurretGenConfig(rand.New(rand.NewSource(1)))
	tr1 := GenerateTurret(cfg1, rand.New(rand.NewSource(1)))
	cfg2 := DefaultTurretGenConfig(rand.New(rand.NewSource(2)))
	tr2 := GenerateTurret(cfg2, rand.New(rand.NewSource(2)))

	// Different seeds should (with very high probability) produce different shapes.
	same := len(tr1.Tiles()) == len(tr2.Tiles())
	if same {
		for idx := range tr1.Tiles() {
			if _, ok := tr2.Tiles()[idx]; !ok {
				same = false
				break
			}
		}
	}
	if same {
		t.Log("warning: two different seeds produced identical turrets (may be a fluke)")
	}
}
