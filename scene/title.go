package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/ebitenginegamejam2026/asset"
	"github.com/noppikinatta/ebitenginegamejam2026/drawing"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

type Title struct {
	input      *ui.Input
	nextScene  ebiten.Game
	sequence   *bamenn.Sequence
	transition bamenn.Transition
}

func NewTitle(input *ui.Input) *Title {
	return &Title{
		input: input,
	}
}

func (t *Title) Init(nextScene ebiten.Game, sequence *bamenn.Sequence, transition bamenn.Transition) {
	t.nextScene = nextScene
	t.sequence = sequence
	t.transition = transition
}

// OnStart keeps the title BGM playing (started during the opening; the shared
// track makes the transition seamless, and re-requesting it is a no-op).
func (t *Title) OnStart() { asset.PlayBGM(asset.BGMTitle) }

func (t *Title) Update() error {
	if t.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
		t.sequence.SwitchWithTransition(t.nextScene, t.transition)
	}

	return nil
}

func (t *Title) Draw(screen *ebiten.Image) {
	// Set background color
	screen.Fill(color.RGBA{20, 20, 40, 255})

	titleImg := drawing.Image("title")
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Scale(2.0, 2.0)
	opt.GeoM.Translate(320, 0)
	screen.DrawImage(titleImg, opt)

	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(40, 400)
	drawing.DrawTextByKey(screen, "story-1", 24, opt)

	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(440, 640)
	drawing.DrawTextByKey(screen, "title-start", 28, opt)
}

func (t *Title) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1280, 720
}
