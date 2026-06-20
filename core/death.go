package core

import "github.com/noppikinatta/ebitenginegamejam2026/geom"

// DeathEvent records an enemy dying this tick, carrying just enough to redraw
// its sprite where it fell. Like SoundEvent/DamageEvent this is plain data —
// core stays Ebiten-free and removes the enemy immediately (keeping w.Enemies
// strictly the live set), while the scene drains World.DeathEvents to spawn a
// short fading-sprite effect. Kind/IsBoss/DropsNipper let the scene pick the
// same sprite enemySpriteKey would; Radius gives the draw size.
type DeathEvent struct {
	Pos         geom.PointF
	Kind        EnemyKind
	Radius      float64
	IsBoss      bool
	DropsNipper bool
}

// emitDeath records an enemy death for the scene's fade-out effect.
func (w *World) emitDeath(e *Enemy) {
	w.DeathEvents = append(w.DeathEvents, DeathEvent{
		Pos:         e.Pos,
		Kind:        e.Kind,
		Radius:      e.Radius,
		IsBoss:      e.IsBoss,
		DropsNipper: e.DropsNipper,
	})
}
