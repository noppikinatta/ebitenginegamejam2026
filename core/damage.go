package core

import "github.com/noppikinatta/ebitenginegamejam2026/geom"

// DamageEvent records a hit landed during the current tick: where it happened,
// how much damage, and whether the tank (vs. an enemy) took it. Like SoundEvent
// this is plain data — core stays Ebiten-free and the scene layer drains
// World.DamageEvents after each Update to spawn floating damage numbers. Keeping
// it as an emitted slice makes the triggers unit-testable.
type DamageEvent struct {
	Pos      geom.PointF // hit location (enemy position, or the tank's position)
	Amount   float64     // raw damage dealt (the scene rounds it for display)
	ToPlayer bool        // true: the tank was hit (red); false: an enemy (white)
}

// emitDamage records a damage event for the current tick. Zero/negative amounts
// are ignored so cosmetic 0-damage blasts (e.g. the firework junk) show no
// number. Events accumulate during Update and are cleared at the next Update.
func (w *World) emitDamage(pos geom.PointF, amount float64, toPlayer bool) {
	if amount <= 0 {
		return
	}
	w.DamageEvents = append(w.DamageEvents, DamageEvent{Pos: pos, Amount: amount, ToPlayer: toPlayer})
}
