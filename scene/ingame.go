package scene

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/drawing"
	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

const (
	screenW = 1280
	screenH = 720
	gridGap = 64
)

// InGame is the main gameplay scene: a Vampire-Survivors-like run driven by the
// Ebiten-free core.World simulation. This scene only handles input and drawing.
type InGame struct {
	input      *ui.Input
	world      *core.World
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

// OnStart is called by bamenn each time the scene begins, so every run starts
// from a fresh world.
func (g *InGame) OnStart() {
	g.world = core.NewWorld(1)
}

func (g *InGame) Update() error {
	if g.world == nil {
		g.world = core.NewWorld(1)
	}

	switch g.world.State {
	case core.StateGameOver:
		if g.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
			g.sequence.SwitchWithTransition(g.nextScene, g.transition)
		}
	case core.StateLevelUp:
		g.handleLevelUpInput()
	default:
		g.world.Update(g.readMove())
	}
	return nil
}

// handleLevelUpInput lets the player pick a disconnect by pressing 1–9.
func (g *InGame) handleLevelUpInput() {
	kb := g.input.Keyboard
	if kb == nil {
		return
	}
	keys := []ebiten.Key{
		ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3,
		ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6,
		ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9,
	}
	for i, k := range keys {
		if i < len(g.world.Choices) && kb.IsJustPressed(k) {
			g.world.ChooseUpgrade(i)
			return
		}
	}
}

func (g *InGame) readMove() geom.PointF {
	kb := g.input.Keyboard
	var m geom.PointF
	if kb == nil {
		return m
	}
	if kb.IsPressed(ebiten.KeyW) || kb.IsPressed(ebiten.KeyArrowUp) {
		m.Y -= 1
	}
	if kb.IsPressed(ebiten.KeyS) || kb.IsPressed(ebiten.KeyArrowDown) {
		m.Y += 1
	}
	if kb.IsPressed(ebiten.KeyA) || kb.IsPressed(ebiten.KeyArrowLeft) {
		m.X -= 1
	}
	if kb.IsPressed(ebiten.KeyD) || kb.IsPressed(ebiten.KeyArrowRight) {
		m.X += 1
	}
	return m
}

func (g *InGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{12, 14, 22, 255})

	w := g.world
	// Camera keeps the player centred on screen.
	cam := geom.PointF{X: w.Player.Pos.X - screenW/2, Y: w.Player.Pos.Y - screenH/2}

	g.drawGrid(screen, cam)

	for _, gem := range w.Gems {
		drawEntity(screen, cam, gem.Pos, 8, 8, 0.2, 0.8, 0.9, 1)
	}
	for _, e := range w.Enemies {
		s := float64(e.Radius) * 2
		drawEntity(screen, cam, e.Pos, s, s, 0.85, 0.25, 0.25, 1)
	}
	for _, p := range w.Projectiles {
		drawEntity(screen, cam, p.Pos, 8, 8, 1, 0.9, 0.3, 1)
	}

	// Player tank, blinking while invulnerable.
	pr := w.Player.Radius * 2
	drawEntity(screen, cam, w.Player.Pos, pr, pr, 0.3, 0.8, 0.5, 1)

	g.drawHUD(screen)

	switch w.State {
	case core.StateLevelUp:
		g.drawLevelUp(screen)
	case core.StateGameOver:
		drawing.DrawRect(screen, 0, 0, screenW, screenH, 0, 0, 0, 0.55)
		opt := &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(520, 300)
		drawing.DrawText(screen, "GAME OVER", 48, opt)
		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(500, 380)
		drawing.DrawText(screen, "Click to continue", 24, opt)
	}
}

func (g *InGame) drawLevelUp(screen *ebiten.Image) {
	drawing.DrawRect(screen, 0, 0, screenW, screenH, 0, 0, 0, 0.65)

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(515, 30)
	drawing.DrawText(screen, "DISCONNECT", 42, opt)

	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(410, 90)
	drawing.DrawText(screen, fmt.Sprintf("Lv %d — Press a number key to cut a node", g.world.Player.Level), 18, opt)

	const (
		cardW = 740
		cardH = 64
		gap   = 8
	)
	n := len(g.world.Choices)
	totalH := float64(n*(cardH+gap) - gap)
	x := float64(screenW-cardW) / 2
	y := (float64(screenH) - totalH) / 2

	for i, c := range g.world.Choices {
		drawing.DrawRect(screen, x, y, cardW, cardH, 0.12, 0.15, 0.25, 0.95)
		drawing.DrawRect(screen, x, y, 4, cardH, 0.6, 0.3, 1, 1) // left accent

		opt := &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(x+16, y+8)
		drawing.DrawText(screen, fmt.Sprintf("%d. %s", i+1, c.Name), 22, opt)

		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(x+16, y+38)
		drawing.DrawText(screen, c.Desc, 15, opt)

		y += cardH + gap
	}
}

func (g *InGame) drawGrid(screen *ebiten.Image, cam geom.PointF) {
	startX := math.Floor(cam.X/gridGap) * gridGap
	for x := startX; x < cam.X+screenW; x += gridGap {
		drawing.DrawRect(screen, x-cam.X, 0, 1, screenH, 1, 1, 1, 0.05)
	}
	startY := math.Floor(cam.Y/gridGap) * gridGap
	for y := startY; y < cam.Y+screenH; y += gridGap {
		drawing.DrawRect(screen, 0, y-cam.Y, screenW, 1, 1, 1, 1, 0.05)
	}
}

func (g *InGame) drawHUD(screen *ebiten.Image) {
	p := g.world.Player

	hp := &drawing.GaugeDrawer{
		Max:           int(p.MaxHP),
		Current:       int(p.HP),
		TopLeft:       geom.PointF{X: 20, Y: 20},
		BottomRight:   geom.PointF{X: 320, Y: 44},
		TextOffset:    geom.PointF{X: 6, Y: 2},
		FontSize:      16,
		ColorScaleMax: scale(0.2, 0.9, 0.3, 1),
		ColorScaleMin: scale(0.9, 0.2, 0.2, 1),
	}
	hp.Draw(screen)

	// XP bar under the HP gauge.
	drawing.DrawRect(screen, 20, 50, 300, 8, 0.2, 0.2, 0.3, 1)
	if p.XPToNext > 0 {
		drawing.DrawRect(screen, 20, 50, 300*float64(p.XP/p.XPToNext), 8, 0.4, 0.6, 1, 1)
	}

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(20, 64)
	drawing.DrawText(screen, fmt.Sprintf("Lv %d", p.Level), 18, opt)

	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW-220, 20)
	drawing.DrawText(screen, fmt.Sprintf("Time %d  Kills %d", g.world.Tick/60, g.world.Kills), 18, opt)
}

func (g *InGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}

// drawEntity draws a world-space rectangle centred on pos, transformed by cam.
func drawEntity(screen *ebiten.Image, cam, pos geom.PointF, w, h float64, r, gr, b, a float32) {
	drawing.DrawRect(screen, pos.X-cam.X-w/2, pos.Y-cam.Y-h/2, w, h, r, gr, b, a)
}

func scale(r, g, b, a float32) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.Scale(r, g, b, a)
	return cs
}
