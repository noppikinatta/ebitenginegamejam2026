package core

import (
	"math"
	"math/rand"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

// fakeSink records the events handed to it, standing in for the real Ebiten
// audio backend during tests.
type fakeSink struct{ played []SoundEvent }

func (f *fakeSink) PlaySound(s SoundEvent) { f.played = append(f.played, s) }

// TestDispatchSounds_DedupsPerFrame: duplicate events in a frame collapse to one
// play (so many simultaneous shots aren't deafening), while distinct events all
// play.
func TestDispatchSounds_DedupsPerFrame(t *testing.T) {
	sink := &fakeSink{}
	DispatchSounds([]SoundEvent{SndFire, SndFire, SndFire, SndExplosion, SndPlayerHit, SndExplosion}, sink)

	if got := countEvents(sink.played, SndFire); got != 1 {
		t.Errorf("SndFire played %d times, want 1", got)
	}
	if got := countEvents(sink.played, SndExplosion); got != 1 {
		t.Errorf("SndExplosion played %d times, want 1", got)
	}
	if got := countEvents(sink.played, SndPlayerHit); got != 1 {
		t.Errorf("SndPlayerHit played %d times, want 1", got)
	}
}

// TestDispatchSounds_NilSink: a nil sink (audio not loaded) is a safe no-op.
func TestDispatchSounds_NilSink(t *testing.T) {
	DispatchSounds([]SoundEvent{SndFire}, nil) // must not panic
}

func countEvents(evs []SoundEvent, want SoundEvent) int {
	n := 0
	for _, e := range evs {
		if e == want {
			n++
		}
	}
	return n
}

// TestSound_FireEmittedPerShot: a triggered weapon emits exactly one SndFire for
// the shot, regardless of how many pellets that shot spawns.
func TestSound_FireEmittedPerShot(t *testing.T) {
	w, wp := buildWeaponWorld(KindShotgun, hexmap.IdxXY(1, 0))
	wp.fireProgress = w.cfg.Weapons[KindShotgun].BaseInterval

	w.updateWeapons()

	if got := countEvents(w.SoundEvents, SndFire); got != 1 {
		t.Errorf("SndFire emitted %d times, want 1", got)
	}
}

// TestSound_ExplosionEmitted: explode() records an explosion sound event.
func TestSound_ExplosionEmitted(t *testing.T) {
	w, _ := buildWeaponWorld(KindGrenade, hexmap.IdxXY(1, 0))

	w.explode(geom.PointF{X: 1, Y: 2}, 32, 10)

	if got := countEvents(w.SoundEvents, SndExplosion); got != 1 {
		t.Errorf("SndExplosion emitted %d times, want 1", got)
	}
}

// TestSound_PlayerHitEmitted: taking damage records a player-hit sound event.
func TestSound_PlayerHitEmitted(t *testing.T) {
	w, _ := buildWeaponWorld(KindCannon, hexmap.IdxXY(1, 0))

	w.damagePlayer(5)

	if got := countEvents(w.SoundEvents, SndPlayerHit); got != 1 {
		t.Errorf("SndPlayerHit emitted %d times, want 1", got)
	}
}

// TestSound_ClearedEachTick: events from one tick do not leak into the next, and
// a frozen (non-playing) world produces no events.
func TestSound_ClearedEachTick(t *testing.T) {
	w := &World{
		Player: &Player{Pos: geom.PointF{}, FacingAngle: -math.Pi / 2, HP: 100, MaxHP: 100},
		State:  StatePlaying,
		rng:    rand.New(rand.NewSource(1)),
		cfg:    testConfig(),
	}
	w.SoundEvents = append(w.SoundEvents, SndExplosion) // stale event from a prior tick

	w.Update(geom.PointF{})
	for _, e := range w.SoundEvents {
		if e == SndExplosion {
			t.Fatal("stale SndExplosion survived into a new tick")
		}
	}

	w.State = StateLevelUp
	w.SoundEvents = append(w.SoundEvents, SndFire)
	w.Update(geom.PointF{})
	if len(w.SoundEvents) != 0 {
		t.Errorf("frozen world produced %d events, want 0", len(w.SoundEvents))
	}
}
