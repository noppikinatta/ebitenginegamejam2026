package geom_test

import (
	"image"
	"math"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

const eps = 1e-9

func closeF(a, b float64) bool { return math.Abs(a-b) <= eps }

func TestPointF_Add(t *testing.T) {
	got := geom.PointF{X: 1, Y: 2}.Add(geom.PointF{X: 3, Y: -5})
	if got != (geom.PointF{X: 4, Y: -3}) {
		t.Fatalf("want {4 -3}, got %v", got)
	}
}

func TestPointF_Subtract(t *testing.T) {
	got := geom.PointF{X: 1, Y: 2}.Subtract(geom.PointF{X: 3, Y: -5})
	if got != (geom.PointF{X: -2, Y: 7}) {
		t.Fatalf("want {-2 7}, got %v", got)
	}
}

func TestPointF_Multiply(t *testing.T) {
	got := geom.PointF{X: 2, Y: -3}.Multiply(2.5)
	if got != (geom.PointF{X: 5, Y: -7.5}) {
		t.Fatalf("want {5 -7.5}, got %v", got)
	}
}

func TestPointF_Abs(t *testing.T) {
	if got := (geom.PointF{X: 3, Y: 4}).Abs(); !closeF(got, 5) {
		t.Fatalf("want 5, got %v", got)
	}
	if got := (geom.PointF{}).Abs(); !closeF(got, 0) {
		t.Fatalf("want 0, got %v", got)
	}
}

func TestPointF_InnerProduct(t *testing.T) {
	got := geom.PointF{X: 2, Y: 3}.InnerProduct(geom.PointF{X: 4, Y: -1})
	if !closeF(got, 5) {
		t.Fatalf("want 5, got %v", got)
	}
	// Perpendicular vectors have a zero dot product.
	if got := (geom.PointF{X: 1, Y: 0}).InnerProduct(geom.PointF{X: 0, Y: 1}); !closeF(got, 0) {
		t.Fatalf("want 0, got %v", got)
	}
}

func TestPointF_Angle(t *testing.T) {
	cases := []struct {
		name string
		p    geom.PointF
		want float64
	}{
		{"positive-x", geom.PointF{X: 1, Y: 0}, 0},
		{"positive-y", geom.PointF{X: 0, Y: 1}, math.Pi / 2},
		{"negative-x", geom.PointF{X: -1, Y: 0}, math.Pi},
		{"negative-y", geom.PointF{X: 0, Y: -1}, -math.Pi / 2},
		{"diagonal", geom.PointF{X: 1, Y: 1}, math.Pi / 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.p.Angle(); !closeF(got, c.want) {
				t.Fatalf("want %v, got %v", c.want, got)
			}
		})
	}
}

func TestPointF_Distance(t *testing.T) {
	a := geom.PointF{X: 1, Y: 2}
	b := geom.PointF{X: 4, Y: 6}
	if got := a.Distance(b); !closeF(got, 5) {
		t.Fatalf("want 5, got %v", got)
	}
	// Distance is symmetric and zero to itself.
	if got := b.Distance(a); !closeF(got, 5) {
		t.Fatalf("want symmetric 5, got %v", got)
	}
	if got := a.Distance(a); !closeF(got, 0) {
		t.Fatalf("want 0, got %v", got)
	}
}

func TestPointFFromPolar(t *testing.T) {
	cases := []struct {
		name       string
		abs, angle float64
		want       geom.PointF
	}{
		{"zero-angle", 5, 0, geom.PointF{X: 5, Y: 0}},
		{"quarter-turn", 2, math.Pi / 2, geom.PointF{X: 0, Y: 2}},
		{"half-turn", 3, math.Pi, geom.PointF{X: -3, Y: 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := geom.PointFFromPolar(c.abs, c.angle)
			if !closeF(got.X, c.want.X) || !closeF(got.Y, c.want.Y) {
				t.Fatalf("want %v, got %v", c.want, got)
			}
		})
	}
}

func TestPointFFromPolar_RoundTrip(t *testing.T) {
	// A vector built from polar coordinates must report back the same
	// magnitude and angle.
	abs, angle := 7.0, 1.2
	p := geom.PointFFromPolar(abs, angle)
	if got := p.Abs(); !closeF(got, abs) {
		t.Fatalf("abs: want %v, got %v", abs, got)
	}
	if got := p.Angle(); !closeF(got, angle) {
		t.Fatalf("angle: want %v, got %v", angle, got)
	}
}

func TestPointFFromPoint(t *testing.T) {
	got := geom.PointFFromPoint(image.Point{X: -3, Y: 7})
	if got != (geom.PointF{X: -3, Y: 7}) {
		t.Fatalf("want {-3 7}, got %v", got)
	}
}

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
