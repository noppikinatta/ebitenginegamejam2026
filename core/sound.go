package core

// SoundEvent identifies a gameplay moment that should play a sound effect.
// The core simulation only records these as plain data (no Ebiten/audio
// dependency); the scene layer drains World.SoundEvents after each Update and
// translates them into actual playback. This keeps audio triggers unit-testable
// by asserting on the emitted slice.
type SoundEvent int

const (
	SndFire      SoundEvent = iota // a weapon fired a shot
	SndExplosion                   // an explosive projectile detonated
	SndPlayerHit                   // the tank took damage
)

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
// so a tick where several weapons fire triggers a single shot SE (instead of a
// dozen overlapping players). Call it once per frame after Update.
func DispatchSounds(evs []SoundEvent, sink SoundSink) {
	if sink == nil {
		return
	}
	var seen [3]bool // indexed by SoundEvent; grow if more events are added
	for _, e := range evs {
		if int(e) < len(seen) {
			if seen[e] {
				continue
			}
			seen[e] = true
		}
		sink.PlaySound(e)
	}
}
