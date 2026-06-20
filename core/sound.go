package core

// SoundEvent identifies a gameplay moment that should play a sound effect.
// The core simulation only records these as plain data (no Ebiten/audio
// dependency); the scene layer drains World.SoundEvents after each Update and
// translates them into actual playback. This keeps audio triggers unit-testable
// by asserting on the emitted slice.
type SoundEvent int

const (
	SndExplosion SoundEvent = iota // an explosive projectile detonated
	SndPlayerHit                   // the tank took damage
	// Per-weapon fire events. These MUST stay contiguous and in WeaponKind order
	// so FireSound can index into them; each weapon gets its own fire SE.
	SndFireCannon
	SndFireShotgun
	SndFireSniper
	SndFireLaser
	SndFireGatling
	SndFireGrenade
	SndFireCIWS
	SndFireMissile

	numSoundEvents // sentinel: count of sound events (for dedup sizing)
)

// FireSound returns the fire SoundEvent for a weapon kind. The per-kind events
// are laid out contiguously from SndFireCannon in WeaponKind order.
func FireSound(k WeaponKind) SoundEvent {
	return SndFireCannon + SoundEvent(k)
}

// emit records a sound event for the current tick. Events accumulate during
// Update and are cleared at the start of the next Update.
func (w *World) emit(s SoundEvent) {
	w.SoundEvents = append(w.SoundEvents, s)
}

// SoundSink is the audio backend the scene layer injects to actually play
// sounds. Keeping it an interface (rather than calling Ebiten directly) lets the
// dispatch logic be unit-tested with a fake, and keeps core Ebiten-free.
type SoundSink interface {
	PlaySound(SoundEvent)
}

// DispatchSounds plays the events emitted during a tick, collapsing duplicates
// so a tick where several tiles of the same weapon fire triggers a single shot
// SE (instead of a dozen overlapping players). Distinct events (e.g. two
// different weapon kinds firing) still each play. Call it once per frame after
// Update.
func DispatchSounds(evs []SoundEvent, sink SoundSink) {
	if sink == nil {
		return
	}
	var seen [numSoundEvents]bool
	for _, e := range evs {
		if e >= 0 && int(e) < len(seen) {
			if seen[e] {
				continue
			}
			seen[e] = true
		}
		sink.PlaySound(e)
	}
}
