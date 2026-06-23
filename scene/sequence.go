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
	meta := &metaHolder{}
	opening := NewOpening(input, rs)
	workshop := NewWorkshop(input)
	inGame := NewInGame(input, rs, meta)
	result := NewResult(input)
	seq := bamenn.NewSequence(opening)
	tran := bamenn.NewLinearTransition(5, 10, bamennutil.LinearFillFadingDrawer{Color: color.Black})

	// The opening cinematic ends on the title (assembled tank + title art); clicking
	// it advances to the workshop. The workshop's "back" returns to that title state
	// (the opening skips its cinematic) rather than replaying the intro.
	opening.Init(workshop, seq, tran)               // title → workshop (spend coins, then start)
	workshop.Init(inGame, opening, meta, seq, tran) // start → run, back → title
	inGame.Init(result, seq, tran)
	result.Init(inGame, opening, meta, seq, tran)

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
