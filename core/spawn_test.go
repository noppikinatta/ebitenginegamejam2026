package core

import (
	"math"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

// TestPickSpawnKind_PhaseGated: brutes never spawn in the first phase (weight 0)
// but do appear once the run reaches a later phase.
func TestPickSpawnKind_PhaseGated(t *testing.T) {
	w := NewWorld(1, testConfig())

	w.Tick = 0
	for i := 0; i < 500; i++ {
		if w.pickSpawnKind() == EnemyBrute {
			t.Fatalf("brute spawned during the opening phase (should be weight 0)")
		}
	}

	w.Tick = 7 * 3600 // final phase: brutes have weight
	sawBrute := false
	for i := 0; i < 1000; i++ {
		if w.pickSpawnKind() == EnemyBrute {
			sawBrute = true
			break
		}
	}
	if !sawBrute {
		t.Errorf("expected brutes to spawn in the late phase")
	}
}

// TestMakeEnemy_HPScalesWithTime: HP doubles after one HPDoublingTicks interval.
func TestMakeEnemy_HPScalesWithTime(t *testing.T) {
	w := NewWorld(1, testConfig())
	s := w.cfg.EnemyKinds[EnemyGrunt]

	w.Tick = 0
	if e := w.makeEnemy(EnemyGrunt, s, geom.PointF{}); math.Abs(e.HP-s.HPBase) > 1e-9 || e.MaxHP != e.HP {
		t.Errorf("tick 0: HP=%.2f MaxHP=%.2f, want %.2f", e.HP, e.MaxHP, s.HPBase)
	}
	w.Tick = int(w.cfg.HPDoublingTicks)
	if e := w.makeEnemy(EnemyGrunt, s, geom.PointF{}); math.Abs(e.HP-2*s.HPBase) > 1e-9 {
		t.Errorf("after one doubling interval: HP=%.2f, want %.2f", e.HP, 2*s.HPBase)
	}
}

// TestSpawnEnemies_PackClusters: a swarmer spawn drops a whole pack at once,
// while a grunt spawn drops exactly one.
func TestSpawnEnemies_PackClusters(t *testing.T) {
	w := NewWorld(1, testConfig())

	swarm := w.cfg.EnemyKinds[EnemySwarmer]
	w.Enemies = nil
	w.spawnTimer = 0
	w.spawnPackOf(EnemySwarmer)
	if len(w.Enemies) < swarm.PackMin || len(w.Enemies) > swarm.PackMax {
		t.Errorf("swarmer pack size = %d, want %d..%d", len(w.Enemies), swarm.PackMin, swarm.PackMax)
	}

	w.Enemies = nil
	w.spawnPackOf(EnemyGrunt)
	if len(w.Enemies) != 1 {
		t.Errorf("grunt pack size = %d, want 1", len(w.Enemies))
	}
}

// TestSpawnBosses_TimingAndFinalClear: bosses appear at their scheduled ticks,
// only the 10-minute boss is Final, and killing it clears the run.
func TestSpawnBosses_TimingAndFinalClear(t *testing.T) {
	w := NewWorld(1, testConfig())

	w.Tick = 3*3600 - 1
	w.spawnBosses()
	if b := w.ActiveBoss(); b != nil {
		t.Fatalf("boss spawned before its scheduled tick: %q", b.Name)
	}

	w.Tick = 3 * 3600
	w.spawnBosses()
	b := w.ActiveBoss()
	if b == nil || b.Final {
		t.Fatalf("expected a non-final boss at 3 min, got %+v", b)
	}

	w.Tick = 10 * 3600
	w.spawnBosses()
	var final *Enemy
	for _, e := range w.Enemies {
		if e.IsBoss && e.Final {
			final = e
		}
	}
	if final == nil {
		t.Fatal("final boss not spawned at 10 min")
	}

	w.killEnemy(final)
	if w.State != StateCleared {
		t.Errorf("State = %v after killing the final boss, want StateCleared", w.State)
	}
}

// TestClearIsSticky: once cleared, a player death the same tick must not flip the
// run to game over.
func TestClearIsSticky(t *testing.T) {
	w := NewWorld(1, testConfig())
	w.State = StateCleared
	w.Player.invuln = 0
	w.damagePlayer(99999)
	if w.State != StateCleared {
		t.Errorf("player death overrode the clear: State = %v", w.State)
	}
}
