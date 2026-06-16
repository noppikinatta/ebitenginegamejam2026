package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/ebitenginegamejam2026/drawing"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

// Result shows the outcome of a run and returns to the title on click.
type Result struct {
	input      *ui.Input
	nextScene  ebiten.Game
	sequence   *bamenn.Sequence
	transition bamenn.Transition
}

func NewResult(input *ui.Input) *Result {
	return &Result{
		input: input,
	}
}

func (r *Result) Init(nextScene ebiten.Game, sequence *bamenn.Sequence, transition bamenn.Transition) {
	r.nextScene = nextScene
	r.sequence = sequence
	r.transition = transition
}

func (r *Result) Update() error {
	if r.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
		r.sequence.SwitchWithTransition(r.nextScene, r.transition)
	}
	return nil
}

func (r *Result) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{16, 10, 10, 255})

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(440, 340)
	drawing.DrawText(screen, "Result (stub) - Click to title", 24, opt)
}

func (r *Result) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1280, 720
}
