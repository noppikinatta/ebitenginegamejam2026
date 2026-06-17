package core

import (
	"math"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// ---- helpers ----

func makeTile(c Component) *Tile { return &Tile{Component: c} }
func wireT() *Tile               { return makeTile(Wire{}) }
func weaponT() *Tile {
	return makeTile(WeaponComponent{Weapon: NewWeapon("test", 0, KindCannon)})
}
func junkT() *Tile { return makeTile(Junk{DeviceName: "Rubber Duck"}) }

const eps = 1e-9

func approx(a, b float64) bool {
	return math.Abs(a-b) < eps
}

// ---- flat power solver tests ----

// TestComputePower_GeneratorOnly: a turret with only the generator has no
// consumer tiles, so ComputePower delivers nothing.
func TestComputePower_GeneratorOnly(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	tiles := map[hexmap.Index]*Tile{gen: wireT()}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	if pt := tr.PowerPerTile(); !approx(pt, 0) {
		t.Errorf("PowerPerTile with no consumers = %v, want 0", pt)
	}
}

// TestComputePower_FlatDistribution: power is split evenly among all connected
// non-generator tiles regardless of topology. gen → wire → weapon = 2 consumers.
func TestComputePower_FlatDistribution(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	mid := hexmap.IdxXY(1, 0)
	weapon := hexmap.IdxXY(2, 0)

	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		mid:    wireT(),
		weapon: weaponT(),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)

	// 2 consumer tiles (mid wire + weapon) → 50 each.
	if pt := tr.PowerPerTile(); !approx(pt, 50) {
		t.Errorf("PowerPerTile = %v, want 50", pt)
	}
	power := tr.ComputePower()
	if !approx(power[weapon], 50) {
		t.Errorf("weapon power = %v, want 50", power[weapon])
	}
	if !approx(power[mid], 50) {
		t.Errorf("mid wire power = %v, want 50", power[mid])
	}
	if !approx(power[gen], 0) {
		t.Errorf("generator power = %v, want 0", power[gen])
	}
}

// TestComputePower_TopologyIndependent: a linear chain and a fork with the same
// number of consumer tiles deliver the same per-tile power (topology no longer
// matters, unlike the old distance-ring solver).
func TestComputePower_TopologyIndependent(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	left := hexmap.IdxXY(1, 0)
	right := hexmap.IdxXY(0, 1)

	tiles := map[hexmap.Index]*Tile{
		gen:   wireT(),
		left:  weaponT(),
		right: weaponT(),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	power := tr.ComputePower()

	// 2 consumers → 50 each.
	if !approx(power[left], 50) || !approx(power[right], 50) {
		t.Errorf("fork power: left=%v right=%v, want 50,50", power[left], power[right])
	}
}

// TestComputePower_JunkDilutes: adding a useless junk tile reduces every other
// tile's power share.
func TestComputePower_JunkDilutes(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	weapon := hexmap.IdxXY(1, 0)
	junk := hexmap.IdxXY(2, 0)

	// Without junk: 1 consumer → 100.
	noJunk := NewTurret(map[hexmap.Index]*Tile{
		gen:    wireT(),
		weapon: weaponT(),
	}, []hexmap.Index{gen}, 100)
	if !approx(noJunk.ComputePower()[weapon], 100) {
		t.Fatalf("single weapon should get full 100, got %v", noJunk.ComputePower()[weapon])
	}

	// With junk: 2 consumers → 50 each. The junk dilutes the weapon.
	withJunk := NewTurret(map[hexmap.Index]*Tile{
		gen:    wireT(),
		weapon: weaponT(),
		junk:   junkT(),
	}, []hexmap.Index{gen}, 100)
	if !approx(withJunk.ComputePower()[weapon], 50) {
		t.Errorf("weapon power with junk = %v, want 50 (diluted)", withJunk.ComputePower()[weapon])
	}
}

// TestComputePower_DisconnectedTileGetsNothing: a tile not reachable from the
// generator (purged mid-chain) receives no power and is not a consumer.
func TestComputePower_PurgedTileBlocksDownstream(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	mid := hexmap.IdxXY(1, 0)
	weapon := hexmap.IdxXY(2, 0)

	midTile := wireT()
	midTile.purged = true
	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		mid:    midTile,
		weapon: weaponT(),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	power := tr.ComputePower()

	if v, ok := power[weapon]; ok && !approx(v, 0) {
		t.Errorf("weapon behind purged tile: power=%v, want 0 or absent", v)
	}
}

// TestPurgeReconcentratesPower: cutting a tile increases the remaining tiles'
// power share — the core risk/reward of the disconnect mechanic.
func TestPurgeReconcentratesPower(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	weapon := hexmap.IdxXY(1, 0)
	junk := hexmap.IdxXY(0, 1)

	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		weapon: weaponT(),
		junk:   junkT(),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)

	before := tr.ComputePower()[weapon]
	if !approx(before, 50) {
		t.Fatalf("pre-cut weapon power = %v, want 50", before)
	}

	if !tr.PurgeTile(junk) {
		t.Fatal("PurgeTile(junk) returned false unexpectedly")
	}

	after := tr.ComputePower()[weapon]
	if !approx(after, 100) {
		t.Errorf("post-cut weapon power = %v, want 100 (reconcentrated)", after)
	}
}

// TestPurgeTile_BlocksDownstream: PurgeTile sets purged=true; ComputePower no
// longer routes through it.
func TestPurgeTile_BlocksDownstream(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	mid := hexmap.IdxXY(1, 0)
	weapon := hexmap.IdxXY(2, 0)

	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		mid:    wireT(),
		weapon: weaponT(),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)

	// Verify power flows before purge (2 consumers → 50 each).
	before := tr.ComputePower()
	if !approx(before[weapon], 50) {
		t.Fatalf("pre-purge weapon power=%v, want 50", before[weapon])
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

	tiles := map[hexmap.Index]*Tile{
		gen:    wireT(),
		mid:    wireT(),
		weapon: weaponT(),
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

	tiles := map[hexmap.Index]*Tile{
		gen:   wireT(),
		p1:    wireT(),
		p2:    wireT(),
		child: weaponT(),
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

// TestPurgeTile_RejectsGenerator: the generator tile cannot be purged.
func TestPurgeTile_RejectsGenerator(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	weapon := hexmap.IdxXY(1, 0)
	tr := NewTurret(map[hexmap.Index]*Tile{
		gen:    wireT(),
		weapon: weaponT(),
	}, []hexmap.Index{gen}, 100)

	if tr.PurgeTile(gen) {
		t.Errorf("PurgeTile(generator) should return false")
	}
	if tr.Tiles()[gen].IsPurged() {
		t.Errorf("generator must not be purged")
	}
}

// TestActiveWeapons_SetsEnergy: ActiveWeapons returns weapons with Energy set to
// their flat power share.
func TestActiveWeapons_SetsEnergy(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	left := hexmap.IdxXY(1, 0)
	right := hexmap.IdxXY(0, 1)

	tiles := map[hexmap.Index]*Tile{
		gen:   wireT(),
		left:  makeTile(WeaponComponent{Weapon: NewWeapon("L", 0, KindCannon)}),
		right: makeTile(WeaponComponent{Weapon: NewWeapon("R", 0, KindCannon)}),
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

// TestActiveWeapons_SetsTileIdx: ActiveWeapons records which tile each weapon
// sits on, so firing can originate from the correct turret position.
func TestActiveWeapons_SetsTileIdx(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	wpos := hexmap.IdxXY(1, 0)

	w := NewWeapon("w", 0, KindCannon)
	tiles := map[hexmap.Index]*Tile{
		gen:  wireT(),
		wpos: makeTile(WeaponComponent{Weapon: w}),
	}
	tr := NewTurret(tiles, []hexmap.Index{gen}, 100)
	weapons := tr.ActiveWeapons()

	if len(weapons) != 1 {
		t.Fatalf("got %d active weapons, want 1", len(weapons))
	}
	if weapons[0].TileIdx != wpos {
		t.Errorf("weapon TileIdx = %v, want %v", weapons[0].TileIdx, wpos)
	}
}

// TestActiveWeapons_ExcludesJunk: junk and wire tiles never appear as weapons.
func TestActiveWeapons_ExcludesJunk(t *testing.T) {
	gen := hexmap.IdxXY(0, 0)
	junk := hexmap.IdxXY(1, 0)
	wire := hexmap.IdxXY(0, 1)
	tr := NewTurret(map[hexmap.Index]*Tile{
		gen:  wireT(),
		junk: junkT(),
		wire: wireT(),
	}, []hexmap.Index{gen}, 100)

	if w := tr.ActiveWeapons(); len(w) != 0 {
		t.Errorf("expected 0 weapons (only junk/wire), got %d", len(w))
	}
}

// ---- MuzzleOffset tests ----

// TestMuzzleOffset_FacingUpIsIdentity: with the default facing (-pi/2 = up),
// the muzzle offset equals the unrotated local tile offset.
func TestMuzzleOffset_FacingUpIsIdentity(t *testing.T) {
	idx := hexmap.IdxXY(1, 0)
	got := MuzzleOffset(idx, -math.Pi/2)
	wantX := 1 * TurretTileSize * 0.866
	wantY := 0.5 * TurretTileSize
	if !approx(got.X, wantX) || !approx(got.Y, wantY) {
		t.Errorf("MuzzleOffset facing up = (%v,%v), want (%v,%v)", got.X, got.Y, wantX, wantY)
	}
}

// TestMuzzleOffset_GeneratorIsZero: the generator tile at the origin has no
// offset regardless of facing.
func TestMuzzleOffset_GeneratorIsZero(t *testing.T) {
	got := MuzzleOffset(hexmap.IdxXY(0, 0), 1.23)
	if got.Abs() != 0 {
		t.Errorf("MuzzleOffset of generator = %v, want zero", got)
	}
}

// TestMuzzleOffset_RotationPreservesMagnitude: rotating to a different facing
// keeps the muzzle the same distance from the tank centre but moves it.
func TestMuzzleOffset_RotationPreservesMagnitude(t *testing.T) {
	idx := hexmap.IdxXY(2, 1)
	up := MuzzleOffset(idx, -math.Pi/2)
	right := MuzzleOffset(idx, 0)
	if !approx(up.Abs(), right.Abs()) {
		t.Errorf("magnitude changed under rotation: up=%v right=%v", up.Abs(), right.Abs())
	}
	if approx(up.X, right.X) && approx(up.Y, right.Y) {
		t.Errorf("muzzle did not move when facing changed: %v", up)
	}
}
