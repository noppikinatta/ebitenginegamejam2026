package scene

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

// deathFX is a fading enemy sprite left where an enemy died. It mirrors the
// damagePopup pipeline: spawned from drained world.DeathEvents, aged each tick,
// drawn with decreasing alpha (and a slight swell), and culled when spent. The
// sprite key/size are resolved at spawn since the Enemy is already gone.
type deathFX struct {
	pos  geom.PointF
	key  string  // resolved sprite key
	size float64 // base draw size (Radius*2)
	age  int
}

// spawnDeathFX turns this tick's enemy deaths into fading sprites.
func (g *InGame) spawnDeathFX() {
	for _, ev := range g.world.DeathEvents {
		g.deaths = append(g.deaths, deathFX{
			pos:  ev.Pos,
			key:  enemySpriteKeyFor(ev.Sprite, ev.Kind, ev.IsBoss, ev.DropsNipper),
			size: ev.Radius * 2,
		})
	}
}

// updateDeathFX ages effects and drops spent ones, reusing the backing array.
func (g *InGame) updateDeathFX() {
	live := g.deaths[:0]
	for _, d := range g.deaths {
		d.age++
		if d.age < deathFadeTicks {
			live = append(live, d)
		}
	}
	g.deaths = live
}

// drawDeathFX renders each dying sprite fading out and swelling slightly.
func (g *InGame) drawDeathFX(screen *ebiten.Image, cam geom.PointF) {
	for i := range g.deaths {
		d := &g.deaths[i]
		frac := float64(d.age) / deathFadeTicks // 0 -> 1 across the fade
		a := float32(1 - frac)
		sz := d.size * (1 + (deathGrow-1)*frac)
		drawSprite(screen, cam, d.key, d.pos, sz, sz, 0, a, a, a, a)
	}
}
