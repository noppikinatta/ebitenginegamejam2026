package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/bamenn/bamennutil"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

func CreateSequence(input *ui.Input) ebiten.Game {
	rs := &runSeed{}
	opening := NewOpening(input, rs)
	title := NewTitle(input)
	inGame := NewInGame(input, rs)
	result := NewResult(input)
	seq := bamenn.NewSequence(opening)
	tran := bamenn.NewLinearTransition(5, 10, bamennutil.LinearFillFadingDrawer{Color: color.Black})

	opening.Init(title, seq, tran)
	title.Init(inGame, seq, tran)
	inGame.Init(result, seq, tran)
	result.Init(inGame, opening, seq, tran)

	return &wrapperGame{
		langSwitcher: &langSwitcher{},
		game:         seq,
	}
}

type wrapperGame struct {
	langSwitcher *langSwitcher
	game         ebiten.Game
}

func (w *wrapperGame) Update() error {
	w.langSwitcher.Update()
	return w.game.Update()
}

func (w *wrapperGame) Draw(screen *ebiten.Image) {
	w.game.Draw(screen)
	w.langSwitcher.Draw(screen)
}

func (w *wrapperGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return w.game.Layout(outsideWidth, outsideHeight)
}
