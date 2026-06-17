package geom_test

import (
	"math"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

func TestPointSegmentDistance_PointOnSegment(t *testing.T) {
	a := geom.PointF{X: 0, Y: 0}
	b := geom.PointF{X: 10, Y: 0}
	p := geom.PointF{X: 5, Y: 3}
	got := geom.PointSegmentDistance(p, a, b)
	if math.Abs(got-3) > 1e-9 {
		t.Fatalf("want 3, got %v", got)
	}
}

func TestPointSegmentDistance_PointBeforeStart(t *testing.T) {
	a := geom.PointF{X: 0, Y: 0}
	b := geom.PointF{X: 10, Y: 0}
	p := geom.PointF{X: -3, Y: 4}
	got := geom.PointSegmentDistance(p, a, b)
	want := 5.0 // distance to a=(0,0)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestPointSegmentDistance_PointBeyondEnd(t *testing.T) {
	a := geom.PointF{X: 0, Y: 0}
	b := geom.PointF{X: 10, Y: 0}
	p := geom.PointF{X: 13, Y: 4}
	got := geom.PointSegmentDistance(p, a, b)
	want := 5.0 // distance to b=(10,0)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestPointSegmentDistance_ZeroLengthSegment(t *testing.T) {
	a := geom.PointF{X: 3, Y: 4}
	b := geom.PointF{X: 3, Y: 4}
	p := geom.PointF{X: 0, Y: 0}
	got := geom.PointSegmentDistance(p, a, b)
	want := 5.0
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("want %v, got %v", want, got)
	}
}
