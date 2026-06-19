package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/ebitenginegamejam2026/asset"
	"github.com/noppikinatta/ebitenginegamejam2026/scene"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
	"github.com/noppikinatta/nyuuryoku"
)

func main() {
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Ebitengine Game Jam 2026")

	if err := asset.LoadSounds(); err != nil {
		log.Printf("load sounds: %v", err)
	}

	input := ui.Input{Mouse: nyuuryoku.NewMouse(), Keyboard: nyuuryoku.NewKeyboard()}
	seq := scene.CreateSequence(&input)
	ebiten.RunGame(seq)
}
