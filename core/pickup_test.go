package core

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

func TestKillEnemy_CandlestickDropsNipper(t *testing.T) {
	w := &World{Player: &Player{}, State: StatePlaying}
	e := &Enemy{Pos: geom.PointF{X: 10, Y: 10}, DropsNipper: true, alive: true}

	w.killEnemy(e)

	if len(w.Pickups) != 1 {
		t.Fatalf("candlestick should drop 1 pickup, got %d", len(w.Pickups))
	}
	if len(w.Gems) != 0 {
		t.Errorf("candlestick (XPValue 0) should drop no gem, got %d", len(w.Gems))
	}
}

func TestKillEnemy_NormalEnemyDropsGemNotNipper(t *testing.T) {
	w := &World{Player: &Player{}, State: StatePlaying}
	e := &Enemy{Pos: geom.PointF{}, XPValue: 3, alive: true}

	w.killEnemy(e)

	if len(w.Gems) != 1 {
		t.Errorf("normal enemy should drop a gem, got %d", len(w.Gems))
	}
	if len(w.Pickups) != 0 {
		t.Errorf("normal enemy should not drop nippers, got %d", len(w.Pickups))
	}
}

func TestUpdatePickups_CollectGrantsNipper(t *testing.T) {
	w := &World{Player: &Player{Pos: geom.PointF{}, Nippers: 0}}
	w.Pickups = []*Pickup{{Pos: geom.PointF{}, alive: true}}

	w.updatePickups()

	if w.Player.Nippers != 1 {
		t.Errorf("Nippers = %d, want 1 after collecting a drop", w.Player.Nippers)
	}
	if w.Pickups[0].alive {
		t.Errorf("collected pickup should be marked dead")
	}
}

func TestSpawnCandlestick_AppearsAndIsHarmless(t *testing.T) {
	w := NewWorld(testSeed)
	w.Update(geom.PointF{}) // first tick: candlestickTimer starts at 0, so it spawns

	var found *Enemy
	for _, e := range w.Enemies {
		if e.DropsNipper {
			found = e
			break
		}
	}
	if found == nil {
		t.Fatal("no candlestick spawned on the first tick")
	}
	if found.Speed != 0 {
		t.Errorf("candlestick should be stationary, Speed = %v", found.Speed)
	}
	if found.Damage != 0 {
		t.Errorf("candlestick should be harmless, Damage = %v", found.Damage)
	}
}
