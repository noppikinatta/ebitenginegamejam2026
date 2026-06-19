package core

import "testing"

// itemTally counts add vs upgrade lines in a proposal.
func itemTally(c Upgrade) (adds, upgrades int) {
	for _, it := range c.Items {
		switch it.Kind {
		case OfferUpgrade:
			upgrades++
		case OfferAddWeapon, OfferAddJunk, OfferAddCapacitor:
			adds++
		}
	}
	return
}

// TestRollDoctorChoice_MixesAddAndUpgrade: a single build proposal can contain
// both an add and an upgrade line (the two are no longer separate offer types).
func TestRollDoctorChoice_MixesAddAndUpgrade(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	if len(w.turret.ActiveWeapons()) == 0 {
		t.Fatal("test turret has no weapons to upgrade")
	}

	mixed := false
	for i := 0; i < 300 && !mixed; i++ {
		adds, upgrades := itemTally(w.rollDoctorChoice(false))
		mixed = adds > 0 && upgrades > 0
	}
	if !mixed {
		t.Errorf("no proposal mixed an add and an upgrade in 300 rolls")
	}
}

// TestRollDoctorChoice_ApplyMatchesItems: applying a proposal adds exactly one
// tile per add line and one weapon level per upgrade line.
func TestRollDoctorChoice_ApplyMatchesItems(t *testing.T) {
	for i := 0; i < 40; i++ {
		w := NewWorld(testSeed+int64(i), testConfig()) // fresh turret each time
		c := w.rollDoctorChoice(false)
		adds, upgrades := itemTally(c)
		if adds == 0 && upgrades == 0 {
			continue // nipper proposal
		}
		beforeTiles := w.turret.TileCount()
		beforeLevels := totalWeaponLevel(w)
		c.Apply(w)
		if got := w.turret.TileCount() - beforeTiles; got != adds {
			t.Fatalf("seed %d: tiles added = %d, want %d", testSeed+int64(i), got, adds)
		}
		if got := totalWeaponLevel(w) - beforeLevels; got != upgrades {
			t.Fatalf("seed %d: levels gained = %d, want %d", testSeed+int64(i), got, upgrades)
		}
	}
}

// TestRollDoctorChoice_AtCapAllUpgrades: at the tile cap, a build proposal adds
// no tiles — every non-nipper line is an upgrade.
func TestRollDoctorChoice_AtCapAllUpgrades(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	for w.turret.TileCount() < w.cfg.MaxTurretTiles {
		if _, ok := w.turret.AddTile(Junk{DeviceName: "Toaster"}, w.rng); !ok {
			t.Fatal("AddTile ran out of room before the cap")
		}
	}
	for i := 0; i < 50; i++ {
		c := w.rollDoctorChoice(true)
		if adds, _ := itemTally(c); adds != 0 {
			t.Fatalf("at cap a proposal offered %d add line(s)", adds)
		}
	}
}
