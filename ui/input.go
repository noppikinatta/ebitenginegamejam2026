package ui

import "github.com/noppikinatta/nyuuryoku"

// Input aggregates the player input devices used across scenes.
type Input struct {
	Mouse    *nyuuryoku.Mouse
	Keyboard *nyuuryoku.Keyboard
}
