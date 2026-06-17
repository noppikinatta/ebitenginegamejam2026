package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/ebitenginegamejam2026/scene"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
	"github.com/noppikinatta/nyuuryoku"
)

func main() {
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Ebitengine Game Jam 2026")

	input := ui.Input{Mouse: nyuuryoku.NewMouse(), Keyboard: nyuuryoku.NewKeyboard()}
	seq := scene.CreateSequence(&input)
	ebiten.RunGame(seq)
}
