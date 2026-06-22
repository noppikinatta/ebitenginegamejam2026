package core

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

// targetTestWorld places three enemies on the +X axis at 50/150/300 from the
// origin so distance ordering is unambiguous.
func targetTestWorld() (*World, *Enemy, *Enemy, *Enemy) {
	near := &Enemy{Pos: geom.PointF{X: 50}, alive: true}
	mid := &Enemy{Pos: geom.PointF{X: 150}, alive: true}
	far := &Enemy{Pos: geom.PointF{X: 300}, alive: true}
	w := &World{Enemies: []*Enemy{near, mid, far}}
	return w, near, mid, far
}

func TestTargetEnemy_NearestPicksClosest(t *testing.T) {
	w, near, _, _ := targetTestWorld()
	if got := w.targetEnemy(geom.PointF{}, 400, TargetNearest); got != near {
		t.Errorf("TargetNearest picked %v, want the closest enemy", got)
	}
}

func TestTargetEnemy_FarthestPicksFurthestInRange(t *testing.T) {
	w, _, _, far := targetTestWorld()
	if got := w.targetEnemy(geom.PointF{}, 400, TargetFarthest); got != far {
		t.Errorf("TargetFarthest picked %v, want the farthest enemy", got)
	}
}

func TestTargetEnemy_FarthestRespectsRange(t *testing.T) {
	w, _, mid, _ := targetTestWorld()
	// Range 200 excludes the enemy at 300, so the farthest in range is the mid one.
	if got := w.targetEnemy(geom.PointF{}, 200, TargetFarthest); got != mid {
		t.Errorf("TargetFarthest within range 200 picked %v, want the mid enemy", got)
	}
}

func TestTargetEnemy_NoneInRange(t *testing.T) {
	w, _, _, _ := targetTestWorld()
	if got := w.targetEnemy(geom.PointF{}, 10, TargetFarthest); got != nil {
		t.Errorf("targetEnemy with nothing in range = %v, want nil", got)
	}
}

func TestTargetEnemy_SkipsDeadEnemies(t *testing.T) {
	w, _, _, far := targetTestWorld()
	far.alive = false
	// With the farthest dead, the next-farthest in range (mid at 150) wins.
	got := w.targetEnemy(geom.PointF{}, 400, TargetFarthest)
	if got == nil || got.Pos.X != 150 {
		t.Errorf("TargetFarthest skipping dead picked %v, want the mid enemy at 150", got)
	}
}
