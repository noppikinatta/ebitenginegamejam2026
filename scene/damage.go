package scene

import (
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/ebitenginegamejam2026/drawing"
	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

// damagePopup is a floating damage number. It spawns at the hit point and darts
// a short way along dir (a unit vector in an upward fan), holds, then fades.
// text is the pre-formatted integer string so drawing only ever renders single
// digits — drawing.DrawText caches one image per (string, size), and feeding it
// whole numbers like "137" would cache an entry per distinct number; per-digit
// drawing caps the cache at the ten digits.
type damagePopup struct {
	pos  geom.PointF // world-space hit location (the launch travel is added at draw)
	dir  geom.PointF // unit launch direction (within the upward fan)
	text string      // digits only, e.g. "42"
	red  bool        // true: tank damage (red), false: enemy damage (white)
	age  int         // ticks since spawn
}

// dmgLife is the total lifetime of a popup in ticks.
const dmgLife = int(dmgRiseTicks) + dmgHoldTicks + dmgFadeTicks

// spawnDamagePopups drains the world's damage events for this tick into floating
// numbers, each launched along a random direction within an upward (±half-fan)
// cone so several hits on the same spot scatter instead of overlapping.
func (g *InGame) spawnDamagePopups() {
	for _, ev := range g.world.DamageEvents {
		n := int(math.Round(ev.Amount))
		if n <= 0 {
			continue
		}
		// Up is -Y; spread by ±dmgFanRad/2 around it.
		angle := -math.Pi/2 + (g.rng.Float64()-0.5)*dmgFanRad
		g.popups = append(g.popups, damagePopup{
			pos:  ev.Pos,
			dir:  geom.PointFFromPolar(1, angle),
			text: strconv.Itoa(n),
			red:  ev.ToPlayer,
		})
	}
}

// updateDamagePopups ages popups and drops expired ones, reusing the backing
// array (no per-frame allocation).
func (g *InGame) updateDamagePopups() {
	live := g.popups[:0]
	for _, p := range g.popups {
		p.age++
		if p.age < dmgLife {
			live = append(live, p)
		}
	}
	g.popups = live
}

// drawDamagePopups renders each popup as camera-relative digits, tinted by
// target and faded near end of life.
func (g *InGame) drawDamagePopups(screen *ebiten.Image, cam geom.PointF) {
	for i := range g.popups {
		p := &g.popups[i]

		// Travel: ease quickly to full distance, then stop.
		prog := smooth(math.Min(float64(p.age)/dmgRiseTicks, 1))
		off := p.dir.Multiply(dmgRise * prog)

		// Alpha: solid through rise+hold, then linear fade.
		alpha := float32(1)
		if fadeStart := int(dmgRiseTicks) + dmgHoldTicks; p.age > fadeStart {
			alpha = float32(clamp01(1 - float64(p.age-fadeStart)/dmgFadeTicks))
		}

		rgb := dmgEnemyRGB
		if p.red {
			rgb = dmgPlayerRGB
		}

		// Centre the number horizontally over the hit point.
		total := 0.0
		for _, ch := range p.text {
			total += digitWidth(ch)
		}
		x := p.pos.X - cam.X + off.X - total/2
		y := p.pos.Y - cam.Y + off.Y - dmgFontSize/2

		for _, ch := range p.text {
			opt := &ebiten.DrawImageOptions{}
			opt.GeoM.Translate(x, y)
			// Premultiplied tint+alpha; the white glyph becomes the target colour
			// and the baked shadow fades with it.
			opt.ColorScale.Scale(rgb[0]*alpha, rgb[1]*alpha, rgb[2]*alpha, alpha)
			drawing.DrawText(screen, string(ch), dmgFontSize, opt)
			x += digitWidth(ch)
		}
	}
}

// digitWidthCache memoises each digit glyph's advance at dmgFontSize so the hot
// draw loop avoids repeated MeasureText calls.
var digitWidthCache [10]float64

func digitWidth(ch rune) float64 {
	if ch < '0' || ch > '9' {
		return drawing.MeasureText(string(ch), dmgFontSize).X
	}
	if digitWidthCache[ch-'0'] == 0 {
		digitWidthCache[ch-'0'] = drawing.MeasureText(string(ch), dmgFontSize).X
	}
	return digitWidthCache[ch-'0']
}
