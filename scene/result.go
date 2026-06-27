package scene

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/ebitenginegamejam2026/asset"
	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/drawing"
	"github.com/noppikinatta/ebitenginegamejam2026/lang"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

// Result shows the outcome of a run. A win returns to the opening; a loss lets
// the player retry the run or accept defeat (back to the opening).
type Result struct {
	input      *ui.Input
	inGame     *InGame     // source of the win/lose outcome, and the retry target
	opening    ebiten.Game // restart target
	workshop   ebiten.Game // retry detour when there are upgrades to buy
	meta       *metaHolder // persistent coins/upgrades; this run's reward is banked here
	sequence   *bamenn.Sequence
	transition bamenn.Transition
	kills      int     // enemies destroyed this run (for the reward formula)
	minutes    float64 // real (fractional) survived minutes (the time multiplier)
	junk       int     // junk tiles still mounted at run's end (for the reward formula)
	earned     int     // coins awarded for the run just finished (for display)
}

func NewResult(input *ui.Input) *Result {
	return &Result{input: input}
}

func (r *Result) Init(inGame *InGame, opening, workshop ebiten.Game, meta *metaHolder, sequence *bamenn.Sequence, transition bamenn.Transition) {
	r.inGame = inGame
	r.opening = opening
	r.workshop = workshop
	r.meta = meta
	r.sequence = sequence
	r.transition = transition
}

// Result button layout.
var (
	resReturnBtn = sceneButton{x: screenW/2 - 170, y: 470, w: 340, h: 60, labelKey: "btn-return"}
	resRetryBtn  = sceneButton{x: screenW/2 - 350, y: 470, w: 320, h: 60, labelKey: "btn-retry"}
	resAcceptBtn = sceneButton{x: screenW/2 + 30, y: 470, w: 320, h: 60, labelKey: "btn-accept"}
)

// spawnMultOptions are the enemy-spawn-frequency multipliers offered for the
// next run (a bonus high-difficulty option, picked radio-button style).
var spawnMultOptions = []int{1, 2, 4, 8}

// Enemy-spawn-rate selector layout: a centred row of icon+label cells near the
// bottom of the screen.
const (
	spawnSelY    = 600.0 // top of the option cells
	spawnCellW   = 116.0
	spawnCellH   = 54.0
	spawnCellGap = 16.0
)

// spawnCellRect returns the rectangle of the i-th spawn-rate option cell.
func spawnCellRect(i int) (x, y, w, h float64) {
	n := len(spawnMultOptions)
	total := float64(n)*spawnCellW + float64(n-1)*spawnCellGap
	x0 := float64(screenW)/2 - total/2
	return x0 + float64(i)*(spawnCellW+spawnCellGap), spawnSelY, spawnCellW, spawnCellH
}

// drawSpawnSelector draws the next-run enemy-spawn-frequency radio buttons: an
// enemy icon + "xN" per option with the current selection highlighted, a heading
// above and a short description below.
func (r *Result) drawSpawnSelector(screen *ebiten.Image, mx, my int) {
	cur := r.inGame.SpawnMult()
	drawTelopC(screen, lang.Text("result-spawn-label"), screenW/2, spawnSelY-30, 18, 0.9, 0.92, 0.98, 1)
	for i, m := range spawnMultOptions {
		x, y, w, h := spawnCellRect(i)
		selected := m == cur
		hovered := float64(mx) >= x && float64(mx) < x+w && float64(my) >= y && float64(my) < y+h
		bg := float32(0.16)
		switch {
		case selected:
			bg = 0.34
		case hovered:
			bg = 0.24
		}
		drawing.DrawRect(screen, x, y, w, h, bg, bg+0.02, bg+0.05, 1)
		if selected {
			drawing.DrawRect(screen, x, y, w, 3, 1, 0.55, 0.3, 1) // top accent marks the active option
		}
		drawing.DrawSprite(screen, drawing.Image(asset.ImgEnemy), x+28, y+h/2, 32, 32, 0, 1, 1, 1, 1)
		label := fmt.Sprintf("x%d", m)
		opt := &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(x+52, y+(h-22)/2)
		drawing.DrawText(screen, label, 22, opt)
	}
	drawTelopC(screen, lang.Text("result-spawn-desc"), screenW/2, spawnSelY+spawnCellH+14, 16, 0.82, 0.82, 0.88, 1)
}

// handleSpawnSelectorClick applies a click on the spawn-rate selector, setting
// the next run's multiplier. Returns true if a cell was hit (so the caller can
// skip its other click handling).
func (r *Result) handleSpawnSelectorClick(mx, my int) bool {
	for i, m := range spawnMultOptions {
		x, y, w, h := spawnCellRect(i)
		if float64(mx) >= x && float64(mx) < x+w && float64(my) >= y && float64(my) < y+h {
			r.inGame.SetSpawnMult(m)
			return true
		}
	}
	return false
}

// OnStart keeps the in-game BGM playing into the result screen (shared track, so
// re-requesting it is a no-op and the music continues seamlessly from the run),
// and banks the coins earned this run (win or lose) into the persistent state.
// It runs once per result entry, so each finished run is rewarded exactly once.
func (r *Result) OnStart() {
	asset.PlayBGM(asset.BGMGame)
	tick := 0
	r.kills, tick, r.junk = r.inGame.RunStats()
	const ticksPerMinute = 60 * 60 // 60 TPS × 60 s
	r.minutes = float64(tick) / ticksPerMinute
	r.earned = core.EarnedCoins(r.kills, r.minutes, r.junk)
	r.meta.state.Coins += r.earned
}

// drawReward shows the coin reward formula with the amount earned on the top
// line, and the new balance on the line below, both centred horizontally.
func (r *Result) drawReward(screen *ebiten.Image, y float64) {
	formula := lang.ExecuteTemplate("result-formula", map[string]any{
		"K": r.kills, "M": r.minutes, "J": r.junk, "E": r.earned,
	})
	total := lang.ExecuteTemplate("result-total", map[string]any{"N": r.meta.state.Coins})
	drawCoinLineC(screen, formula, y)
	drawCoinLineC(screen, total, y+34)
}

// drawCoinLineC draws a coin amount line centred horizontally at y.
func drawCoinLineC(screen *ebiten.Image, text string, y float64) {
	tw := drawing.MeasureText(text, 22)
	x := float64(screenW)/2 - (22+tw.X)/2
	drawCoinAmount(screen, text, x, y, 22)
}

func (r *Result) Update() error {
	if !r.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}
	mx, my := ebiten.CursorPosition()
	// The spawn-rate selector applies to the next run regardless of win/lose, so
	// handle it before the outcome branch; a hit only changes the selection.
	if r.handleSpawnSelectorClick(mx, my) {
		return nil
	}
	if r.inGame.Outcome() == OutcomeWin {
		if resReturnBtn.hit(mx, my) {
			r.sequence.SwitchWithTransition(r.opening, r.transition)
		}
		return nil
	}
	// Loss (or unknown): retry the run, or accept defeat and return to the opening.
	// Retry detours through the workshop when the player can afford an upgrade (the
	// run's coins were just banked), so they can spend before re-deploying; with
	// nothing buyable it goes straight back into the run.
	switch {
	case resRetryBtn.hit(mx, my):
		if metaShoppable(r.meta.state) {
			r.sequence.SwitchWithTransition(r.workshop, r.transition)
		} else {
			r.sequence.SwitchWithTransition(r.inGame, r.transition)
		}
	case resAcceptBtn.hit(mx, my):
		r.sequence.SwitchWithTransition(r.opening, r.transition)
	}
	return nil
}

func (r *Result) Draw(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	if r.inGame.Outcome() == OutcomeWin {
		screen.Fill(color.RGBA{10, 18, 12, 255})
		drawTelopC(screen, lang.Text("result-win"), screenW/2, 280, 40, 0.8, 1, 0.8, 1)
		r.drawReward(screen, 370)
		resReturnBtn.draw(screen, resReturnBtn.hit(mx, my))
		r.drawSpawnSelector(screen, mx, my)
		return
	}

	screen.Fill(color.RGBA{18, 10, 10, 255})
	drawTelopC(screen, lang.Text("result-lose-1"), screenW/2, 250, 38, 1, 0.8, 0.8, 1)
	drawTelopC(screen, lang.Text("result-lose-2"), screenW/2, 320, 26, 0.85, 0.7, 0.7, 1)
	r.drawReward(screen, 390)
	resRetryBtn.draw(screen, resRetryBtn.hit(mx, my))
	resAcceptBtn.draw(screen, resAcceptBtn.hit(mx, my))
	r.drawSpawnSelector(screen, mx, my)
}

func (r *Result) Layout(outsideWidth, outsideHeight int) (int, int) { return screenW, screenH }

// sceneButton is a simple clickable rectangle with a centred label. labelKey is
// a lang key resolved at draw time so the text follows the current language.
type sceneButton struct {
	x, y, w, h float64
	labelKey   string
}

func (b sceneButton) hit(mx, my int) bool {
	return float64(mx) >= b.x && float64(mx) < b.x+b.w && float64(my) >= b.y && float64(my) < b.y+b.h
}

func (b sceneButton) draw(screen *ebiten.Image, hovered bool) {
	bg := float32(0.16)
	if hovered {
		bg = 0.30
	}
	drawing.DrawRect(screen, b.x, b.y, b.w, b.h, bg, bg+0.02, bg+0.06, 1)
	drawing.DrawRect(screen, b.x, b.y, b.w, 3, 0.3, 0.7, 1, 1) // top accent
	label := lang.Text(b.labelKey)
	tw := drawing.MeasureText(label, 20)
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(b.x+(b.w-tw.X)/2, b.y+(b.h-tw.Y)/2)
	drawing.DrawText(screen, label, 20, opt)
}
