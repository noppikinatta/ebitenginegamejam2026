package core

import (
	"math"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// ---- helpers ----

func makeTile(c Component) *Tile { return &Tile{Component: c} }
func wireT() *Tile               { return makeTile(Wire{}) }
func capT(m float64) *Tile       { return makeTile(Capacitor{Multiplier: m}) }
func propT(energy float64) *Tile {
	return makeTile(ProportionalWeapon{Weapon: NewWeapon("test", energy, KindCannon)})
}
func threshT(min float64) *Tile {
	return makeTile(ThresholdWeapon{Weapon: NewWeapon("test", 0, KindCannon), MinPower: min})
}

// buildTurret is a convenience: tiles is a map of index→*Tile; generator is at
// hexmap.IdxXY(0,0) and emits genPower.
func buildTurret(tiles map[hexmap.Index]*Tile, genPower float64) *Turret {
	gen := hexmap.IdxXY(0, 0)
	return NewTurret(tiles, []hexmap.Index{gen}, genPower)
}

const eps = 1e-9

func approx(a, b float64) bool {
	return math.Abs(a-b) < eps
}

// ---- Component unit tests ----

func TestWire_PassesAll(t *testing.T) {
	self, through := Wire{}.Distribute(10, 2)
	if !approx(self, 0) || !approx(through, 10) {
		t.Errorf("Wire: got self=%v through=%v, want 0, 10", self, through)
	}
}

func TestCapacitor_Amplifies(t *testing.T) {
	self, through := Capacitor{Multiplier: 2}.Distribute(5, 1)
	if !approx(self, 0) || !approx(through, 10) {
		t.Errorf("Capacitor: got self=%v through=%v, want 0, 10", self, through)
	}
}

func TestProportionalWeapon_KeepsOneShare(t *testing.T) {
	pw := ProportionalWeapon{Weapon: NewWeapon("w", 0, KindCannon)}
	cases := []struct {
		received     float64
		downstream   int
		wantSelf     float64
		wantThrough  float64
	}{
		{12, 0, 12, 0},  // no downstream: keeps everything
		{12, 2, 4, 8},   // 3 shares total: keeps 1, forwards 2
		{12, 5, 2, 10},  // 6 shares: keeps 1, forwards 5
	}
	for _, tc := range cases {
		self, through := pw.Distribute(tc.received, tc.downstream)
		if !approx(self, tc.wantSelf) || !approx(through, tc.wantThrough) {
			t.Errorf("received=%v downstream=%d: got self=%v through=%v, want %v/%v",
				tc.received, tc.downstream, self, through, tc.wantSelf, tc.wantThrough)
		}
	}
}

func TestThresholdWeapon_ActivatesAboveMin(t *testing.T) {
	tw := ThresholdWeapon{Weapon: NewWeapon("w", 0, KindCannon), MinPower: 5}

	// Below threshold: no self-consumption, no throughput.
	self, through := tw.Distribute(4.9, 0)
	if !approx(self, 0) || !approx(through, 0) {
		t.Errorf("below threshold: got self=%v through=%v, want 0,0", self, through)
	}

	// Exactly at threshold: consumes MinPower, forwards remainder.
	self, through = tw.Distribute(5, 1)
	if !approx(self, 5) || !approx(through, 0) {
		t.Errorf("at threshold: got self=%v through=%v, want 5,0", self, through)
	}

	// Above threshold: consumes MinPower, forwards surplus.
	self, through = tw.Distribute(8, 1)
	if !approx(self, 5) || !approx(through, 3) {
		t.Errorf("above threshold: got self=%v through=%v, want 5,3", self, through)
	}
}

// ---- Turret / power solver tests ----

// TestComputePower_GeneratorOnly: a turret with only the generator tile should
// deliver genPower to itself and nothing to any downstream.
func TestComputePower_GeneratorOnly(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	tiles := map[hexmap.Index]*Tile{gen: wireT()}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	power := tr.ComputePower()
	if !approx(power[gen], 0) {
		// Generator tile is a Wire: selfConsumption=0, throughput=100.
		// No downstream active tiles, so throughput is absorbed at edge.
		t.Errorf("generator wire self=%v, want 0", power[gen])
	}
}

// TestComputePower_LinearChain: generator → wire → proportionalWeapon
// Power should flow: 100 → wire passes 100 → weapon keeps 1/(0+1)=100.
func TestComputePower_LinearChain(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	mid := hexmap.IdxXY(1, 0)    // Direction03 neighbor
	weapon := hexmap.IdxXY(2, 0) // one further

	w := NewWeapon("cannon", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		mid:    wireT(),
		weapon: makeTile(ProportionalWeapon{Weapon: w}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	power := tr.ComputePower()

	// wire at gen: self=0, throughput=100 to mid.
	// wire at mid: self=0, throughput=100 to weapon.
	// prop weapon at leaf (no downstream): self=100, throughput=0.
	if !approx(power[weapon], 100) {
		t.Errorf("weapon power=%v, want 100", power[weapon])
	}
}

// TestComputePower_TwoEqualBranches: generator → two weapon tiles equally.
func TestComputePower_TwoEqualBranches(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	left := hexmap.IdxXY(1, 0)  // Direction03
	right := hexmap.IdxXY(0, 1) // Direction05

	wL := NewWeapon("L", 0, KindCannon)
	wR := NewWeapon("R", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:   wireT(),
		left:  makeTile(ProportionalWeapon{Weapon: wL}),
		right: makeTile(ProportionalWeapon{Weapon: wR}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	power := tr.ComputePower()

	// wire at gen: forwards 100 equally to left and right → 50 each.
	// each PropWeapon is a leaf: keeps all 50.
	if !approx(power[left], 50) || !approx(power[right], 50) {
		t.Errorf("branch power: left=%v right=%v, want 50,50", power[left], power[right])
	}
}

// TestComputePower_PurgedTileBlocksDownstream: if the mid tile is purged,
// the downstream weapon receives no power.
func TestComputePower_PurgedTileBlocksDownstream(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	mid := hexmap.IdxXY(1, 0)
	weapon := hexmap.IdxXY(2, 0)

	w := NewWeapon("w", 0, KindCannon)
	midTile := wireT()
	midTile.purged = true
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		mid:    midTile,
		weapon: makeTile(ProportionalWeapon{Weapon: w}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	power := tr.ComputePower()

	if v, ok := power[weapon]; ok && !approx(v, 0) {
		t.Errorf("weapon behind purged tile: power=%v, want 0 or absent", v)
	}
}

// TestComputePower_Capacitor: generator → capacitor(x2) → weapon.
// Weapon should receive 200 (100 * 2).
func TestComputePower_Capacitor(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	cap1 := hexmap.IdxXY(1, 0)
	weapon := hexmap.IdxXY(2, 0)

	w := NewWeapon("w", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		cap1:   capT(2.0),
		weapon: makeTile(ProportionalWeapon{Weapon: w}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	power := tr.ComputePower()

	if !approx(power[weapon], 200) {
		t.Errorf("weapon after capacitor: power=%v, want 200", power[weapon])
	}
}

// TestComputePower_ThresholdNotMet: weapon requires 50 but only 40 supplied.
func TestComputePower_ThresholdNotMet(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	weapon := hexmap.IdxXY(1, 0)

	w := NewWeapon("w", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		weapon: makeTile(ThresholdWeapon{Weapon: w, MinPower: 50}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 40)
	power := tr.ComputePower()

	// ThresholdWeapon below min: self=0, through=0.
	// WeaponPower should not include this tile (or show 0).
	wp := tr.WeaponPower()
	if v, ok := wp[weapon]; ok && !approx(v, 0) {
		t.Errorf("below-threshold weapon in WeaponPower=%v, want absent/0", v)
	}
	_ = power // power entry may be 0 via self-consumption
}

// TestComputePower_ThresholdMet: weapon requires 50, receives 80; surplus goes downstream.
func TestComputePower_ThresholdMet(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	weapon := hexmap.IdxXY(1, 0)
	wire2 := hexmap.IdxXY(2, 0)

	w := NewWeapon("w", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		weapon: makeTile(ThresholdWeapon{Weapon: w, MinPower: 50}),
		wire2:  wireT(),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 80)
	power := tr.ComputePower()

	// ThresholdWeapon: self=50, through=30 to wire2.
	if !approx(power[weapon], 50) {
		t.Errorf("threshold weapon self power=%v, want 50", power[weapon])
	}
	// wire2 receives 30, self=0.
	_ = power[wire2]
}

// TestComputePower_DAGMerge: two parents at distance 1 both feed a single child
// at distance 2. The child should receive the sum of both contributions.
//
//	gen(0,0) → p1(1,0) \
//	                     → child(1,1)
//	gen(0,0) → p2(0,1) /
//
// (1,1) is adjacent to both (1,0) via Direction05 and (0,1) via Direction03,
// and is at hex-distance 2 from gen.
func TestComputePower_DAGMerge(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	p1 := hexmap.IdxXY(1, 0) // Direction03 from gen
	p2 := hexmap.IdxXY(0, 1) // Direction05 from gen
	child := hexmap.IdxXY(1, 1)
	w := NewWeapon("w", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:   wireT(),
		p1:    wireT(),
		p2:    wireT(),
		child: makeTile(ProportionalWeapon{Weapon: w}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	power := tr.ComputePower()

	// gen(wire): through=100, split equally to p1 and p2 → 50 each.
	// p1(wire): through=50, downstream: child only → forwards 50 to child.
	// p2(wire): through=50, downstream: child only → forwards 50 to child.
	// child receives 50+50=100. PropWeapon with no downstream → self=100.
	if !approx(power[child], 100) {
		t.Errorf("DAG merge child power=%v, want 100", power[child])
	}
}

// TestPurgeTile_BlocksDownstream: PurgeTile sets purged=true; ComputePower
// no longer routes through it.
func TestPurgeTile_BlocksDownstream(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	mid := hexmap.IdxXY(1, 0)
	weapon := hexmap.IdxXY(2, 0)

	w := NewWeapon("w", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		mid:    wireT(),
		weapon: makeTile(ProportionalWeapon{Weapon: w}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)

	// Verify power flows before purge.
	before := tr.ComputePower()
	if !approx(before[weapon], 100) {
		t.Fatalf("pre-purge weapon power=%v, want 100", before[weapon])
	}

	ok := tr.PurgeTile(mid)
	if !ok {
		t.Fatal("PurgeTile returned false unexpectedly")
	}

	after := tr.ComputePower()
	if v := after[weapon]; v != 0 && !approx(v, 0) {
		t.Errorf("post-purge weapon power=%v, want 0", v)
	}
}

// TestPurgeTile_CascadesToOrphans: purging a tile mid-chain also marks the now
// unreachable downstream tiles as purged (cascade), not just unpowered.
func TestPurgeTile_CascadesToOrphans(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	mid := hexmap.IdxXY(1, 0)
	weapon := hexmap.IdxXY(2, 0)
	tail := hexmap.IdxXY(3, 0)

	w := NewWeapon("w", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		mid:    wireT(),
		weapon: makeTile(ProportionalWeapon{Weapon: w}),
		tail:   wireT(),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)

	if !tr.PurgeTile(mid) {
		t.Fatal("PurgeTile(mid) returned false unexpectedly")
	}

	// mid was purged directly; weapon and tail are now unreachable from gen and
	// must be cascade-purged.
	for _, idx := range []hexmap.Index{mid, weapon, tail} {
		if !tr.Tiles()[idx].IsPurged() {
			t.Errorf("tile %v should be purged after cascade", idx)
		}
	}
	// The generator must remain unpurged.
	if tr.Tiles()[gen].IsPurged() {
		t.Errorf("generator tile was purged by cascade")
	}
}

// TestPurgeTile_DoesNotCascadeAcrossAlternatePath: if an orphaned-looking tile
// still has another route to the generator, it stays active.
func TestPurgeTile_DoesNotCascadeAcrossAlternatePath(t *testing.T) {
	// Diamond: gen feeds p1 and p2; both feed child. Purging p1 leaves child
	// reachable via p2, so child must NOT be cascade-purged.
	gen := hexmap.IdxXY(0, 0)
	p1 := hexmap.IdxXY(1, 0)
	p2 := hexmap.IdxXY(0, 1)
	child := hexmap.IdxXY(1, 1)

	w := NewWeapon("w", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:   wireT(),
		p1:    wireT(),
		p2:    wireT(),
		child: makeTile(ProportionalWeapon{Weapon: w}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)

	if !tr.PurgeTile(p1) {
		t.Fatal("PurgeTile(p1) returned false unexpectedly")
	}

	if !tr.Tiles()[p1].IsPurged() {
		t.Errorf("p1 should be purged")
	}
	if tr.Tiles()[child].IsPurged() {
		t.Errorf("child has an alternate path via p2 and must not be cascade-purged")
	}
	if tr.Tiles()[p2].IsPurged() {
		t.Errorf("p2 should remain active")
	}
}

// TestPurgeWeapon_TileRemainsActive: PurgeWeapon replaces the component with Wire
// so downstream tiles still receive power.
func TestPurgeWeapon_TileRemainsActive(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	midWeapon := hexmap.IdxXY(1, 0)
	downstream := hexmap.IdxXY(2, 0)

	wMid := NewWeapon("mid", 0, KindCannon)
	wDown := NewWeapon("down", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:        wireT(),
		midWeapon:  makeTile(ProportionalWeapon{Weapon: wMid}),
		downstream: makeTile(ProportionalWeapon{Weapon: wDown}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)

	// Before purge: mid (ProportionalWeapon with 1 downstream) keeps 50, sends 50 down.
	before := tr.ComputePower()
	if !approx(before[midWeapon], 50) || !approx(before[downstream], 50) {
		t.Fatalf("pre-purge mid=%v down=%v, want 50,50", before[midWeapon], before[downstream])
	}

	ok := tr.PurgeWeapon(midWeapon)
	if !ok {
		t.Fatal("PurgeWeapon returned false unexpectedly")
	}

	after := tr.ComputePower()
	// mid is now Wire: self=0, all 100 forwarded to downstream.
	if !approx(after[midWeapon], 0) {
		t.Errorf("post-weapon-purge mid self=%v, want 0", after[midWeapon])
	}
	if !approx(after[downstream], 100) {
		t.Errorf("post-weapon-purge downstream power=%v, want 100", after[downstream])
	}
}

// TestActiveWeapons_SetsEnergy: ActiveWeapons returns weapons with Energy set
// to their computed power.
func TestActiveWeapons_SetsEnergy(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	left := hexmap.IdxXY(1, 0)
	right := hexmap.IdxXY(0, 1)

	wL := NewWeapon("L", 0, KindCannon)
	wR := NewWeapon("R", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:   wireT(),
		left:  makeTile(ProportionalWeapon{Weapon: wL}),
		right: makeTile(ProportionalWeapon{Weapon: wR}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	weapons := tr.ActiveWeapons()

	if len(weapons) != 2 {
		t.Fatalf("got %d active weapons, want 2", len(weapons))
	}
	for _, w := range weapons {
		if !approx(w.Energy, 50) {
			t.Errorf("weapon %q energy=%v, want 50", w.Name, w.Energy)
		}
	}
}

// TestActiveWeapons_ExcludesInactiveThreshold: a threshold weapon below its
// minimum should not appear in ActiveWeapons.
func TestActiveWeapons_ExcludesInactiveThreshold(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	weapon := hexmap.IdxXY(1, 0)

	w := NewWeapon("w", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		weapon: makeTile(ThresholdWeapon{Weapon: w, MinPower: 200}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	weapons := tr.ActiveWeapons()

	if len(weapons) != 0 {
		t.Errorf("expected 0 active weapons below threshold, got %d", len(weapons))
	}
}
