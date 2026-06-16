package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/ebitenginegamejam2026/drawing"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

// InGame is the main gameplay scene. Currently a stub: the actual
// Vampire-Survivors-like loop (tank, weapons, enemies, XP) is built here.
type InGame struct {
	input      *ui.Input
	nextScene  ebiten.Game
	sequence   *bamenn.Sequence
	transition bamenn.Transition
}

func NewInGame(input *ui.Input) *InGame {
	return &InGame{
		input: input,
	}
}

func (g *InGame) Init(nextScene ebiten.Game, sequence *bamenn.Sequence, transition bamenn.Transition) {
	g.nextScene = nextScene
	g.sequence = sequence
	g.transition = transition
}

func (g *InGame) Update() error {
	if g.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
		g.sequence.SwitchWithTransition(g.nextScene, g.transition)
	}
	return nil
}

func (g *InGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 16, 255})

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(440, 340)
	drawing.DrawText(screen, "InGame (stub) - Click to continue", 24, opt)
}

func (g *InGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1280, 720
}
