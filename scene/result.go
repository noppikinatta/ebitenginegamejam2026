package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
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
	sequence   *bamenn.Sequence
	transition bamenn.Transition
}

func NewResult(input *ui.Input) *Result {
	return &Result{input: input}
}

func (r *Result) Init(inGame *InGame, opening ebiten.Game, sequence *bamenn.Sequence, transition bamenn.Transition) {
	r.inGame = inGame
	r.opening = opening
	r.sequence = sequence
	r.transition = transition
}

// Result button layout.
var (
	resReturnBtn = sceneButton{x: screenW/2 - 170, y: 470, w: 340, h: 60, labelKey: "btn-return"}
	resRetryBtn  = sceneButton{x: screenW/2 - 350, y: 470, w: 320, h: 60, labelKey: "btn-retry"}
	resAcceptBtn = sceneButton{x: screenW/2 + 30, y: 470, w: 320, h: 60, labelKey: "btn-accept"}
)

func (r *Result) Update() error {
	if !r.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}
	mx, my := ebiten.CursorPosition()
	if r.inGame.Outcome() == OutcomeWin {
		if resReturnBtn.hit(mx, my) {
			r.sequence.SwitchWithTransition(r.opening, r.transition)
		}
		return nil
	}
	// Loss (or unknown): retry the run, or accept defeat and return to the opening.
	switch {
	case resRetryBtn.hit(mx, my):
		r.sequence.SwitchWithTransition(r.inGame, r.transition)
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
		resReturnBtn.draw(screen, resReturnBtn.hit(mx, my))
		return
	}

	screen.Fill(color.RGBA{18, 10, 10, 255})
	drawTelopC(screen, lang.Text("result-lose-1"), screenW/2, 250, 38, 1, 0.8, 0.8, 1)
	drawTelopC(screen, lang.Text("result-lose-2"), screenW/2, 320, 26, 0.85, 0.7, 0.7, 1)
	resRetryBtn.draw(screen, resRetryBtn.hit(mx, my))
	resAcceptBtn.draw(screen, resAcceptBtn.hit(mx, my))
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
