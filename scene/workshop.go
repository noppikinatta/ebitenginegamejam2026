package scene

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/ebitenginegamejam2026/asset"
	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/data"
	"github.com/noppikinatta/ebitenginegamejam2026/drawing"
	"github.com/noppikinatta/ebitenginegamejam2026/lang"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

// Workshop is the persistent-upgrade shop, opened from the Title. The player
// spends coins earned across runs on five tracks (HP / armor / regen / speed /
// attack), then starts a run. It reads and writes the shared metaHolder.
type Workshop struct {
	input      *ui.Input
	meta       *metaHolder
	startScene ebiten.Game // InGame: begin a run with the current upgrades
	opening    *Opening    // back target: the opening, jumped straight to its title state
	sequence   *bamenn.Sequence
	transition bamenn.Transition
}

func NewWorkshop(input *ui.Input) *Workshop {
	return &Workshop{input: input}
}

func (w *Workshop) Init(startScene ebiten.Game, opening *Opening, meta *metaHolder, sequence *bamenn.Sequence, transition bamenn.Transition) {
	w.startScene = startScene
	w.opening = opening
	w.meta = meta
	w.sequence = sequence
	w.transition = transition
}

// OnStart keeps the title BGM playing (the workshop sits between title and game).
func (w *Workshop) OnStart() { asset.PlayBGM(asset.BGMTitle) }

// Workshop layout.
const (
	wsRowTop = 150.0
	wsRowH   = 84.0
	wsPanelX = 150.0
	wsBuyX   = wsPanelX + 760
	wsBuyW   = 210.0
	wsBuyH   = 48.0
)

var (
	wsStartBtn = sceneButton{x: screenW/2 + 20, y: 648, w: 300, h: 56, labelKey: "workshop-start"}
	wsBackBtn  = sceneButton{x: screenW/2 - 320, y: 648, w: 300, h: 56, labelKey: "workshop-back"}
)

// buyRect is the clickable buy button for stat row i.
func wsBuyRect(i int) sceneButton {
	return sceneButton{x: wsBuyX, y: wsRowTop + float64(i)*wsRowH + (wsRowH-wsBuyH)/2, w: wsBuyW, h: wsBuyH}
}

func (w *Workshop) Update() error {
	if !w.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}
	mx, my := ebiten.CursorPosition()

	for i, s := range core.MetaStats {
		if wsBuyRect(i).hit(mx, my) {
			if next, ok := data.BuyMeta(w.meta.state, s); ok {
				w.meta.state = next
			}
			return nil
		}
	}

	switch {
	case wsStartBtn.hit(mx, my):
		w.sequence.SwitchWithTransition(w.startScene, w.transition)
	case wsBackBtn.hit(mx, my):
		w.opening.SkipToTitle() // return to the title without replaying the cinematic
		w.sequence.SwitchWithTransition(w.opening, w.transition)
	}
	return nil
}

func (w *Workshop) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{16, 18, 28, 255})

	// Header: title + current coin balance.
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(wsPanelX, 70)
	drawing.DrawTextByKey(screen, "workshop-title", 34, opt)
	drawCoinAmount(screen, fmt.Sprintf("%d", w.meta.state.Coins), screenW-260, 80, 26)

	mx, my := ebiten.CursorPosition()
	for i, s := range core.MetaStats {
		w.drawRow(screen, i, s, mx, my)
	}

	wsStartBtn.draw(screen, wsStartBtn.hit(mx, my))
	wsBackBtn.draw(screen, wsBackBtn.hit(mx, my))
}

// drawRow renders one upgrade track: icon, name, level, per-level bonus, and a
// buy button showing the cost (or MAX).
func (w *Workshop) drawRow(screen *ebiten.Image, i int, s core.MetaStat, mx, my int) {
	y := wsRowTop + float64(i)*wsRowH
	level := w.meta.state.Level(s)
	maxed := data.MetaMaxed(s, level)

	// Row backdrop.
	drawing.DrawRect(screen, wsPanelX, y+6, screenW-2*wsPanelX, wsRowH-12, 0.10, 0.12, 0.18, 0.95)

	// Icon.
	drawing.DrawSprite(screen, drawing.Image(core.MetaStatImageKey(s)), wsPanelX+26, y+wsRowH/2, 36, 36, 0, 1, 1, 1, 1)

	// Name.
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(wsPanelX+58, y+18)
	drawing.DrawTextByKey(screen, "meta-"+core.MetaStatKey(s)+"-name", 22, opt)

	// Level + per-level bonus.
	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(wsPanelX+58, y+46)
	drawing.DrawText(screen, fmt.Sprintf("Lv %d/%d", level, data.MetaSpec(s).MaxLevel), 16, opt)
	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(wsPanelX+200, y+46)
	drawing.DrawText(screen, metaBonusText(s), 16, opt)

	// Buy button.
	btn := wsBuyRect(i)
	hovered := !maxed && btn.hit(mx, my)
	bg := float32(0.16)
	if maxed {
		bg = 0.10
	} else if hovered {
		bg = 0.32
	}
	drawing.DrawRect(screen, btn.x, btn.y, btn.w, btn.h, bg, bg+0.04, bg+0.02, 1)
	if maxed {
		tw := drawing.MeasureText(lang.Text("workshop-maxed"), 20)
		o := &ebiten.DrawImageOptions{}
		o.GeoM.Translate(btn.x+(btn.w-tw.X)/2, btn.y+(btn.h-tw.Y)/2)
		drawing.DrawText(screen, lang.Text("workshop-maxed"), 20, o)
		return
	}
	cost := data.MetaCost(s, level)
	afford := w.meta.state.Coins >= cost
	tint := float32(1.0)
	if !afford {
		tint = 0.5
	}
	drawCoinAmountTint(screen, fmt.Sprintf("%d", cost), btn.x+24, btn.y+(btn.h-22)/2, 20, tint)
}

func (w *Workshop) Layout(outsideWidth, outsideHeight int) (int, int) { return screenW, screenH }

// metaBonusText is the per-level effect label for a stat, e.g. "+20 / Lv" or
// "+5% / Lv" for the attack multiplier.
func metaBonusText(s core.MetaStat) string {
	b := data.MetaSpec(s).Bonus
	if s == core.MetaAttack {
		return fmt.Sprintf("+%g%% / Lv", b*100)
	}
	return fmt.Sprintf("+%g / Lv", b)
}

// drawCoinAmount draws the 16x16 coin sprite followed by text at (x, y).
func drawCoinAmount(screen *ebiten.Image, text string, x, y, size float64) {
	drawCoinAmountTint(screen, text, x, y, size, 1)
}

func drawCoinAmountTint(screen *ebiten.Image, text string, x, y, size float64, tint float32) {
	drawing.DrawSprite(screen, drawing.Image("coin"), x+8, y+size/2, 16, 16, 0, tint, tint, tint*0.85, 1)
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(x+22, y)
	opt.ColorScale.Scale(tint, tint, tint, 1)
	drawing.DrawText(screen, text, size, opt)
}
