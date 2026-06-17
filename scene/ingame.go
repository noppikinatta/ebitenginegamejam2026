package scene

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/drawing"
	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

const (
	screenW = 1280
	screenH = 720
	gridGap = 64

	combatTileSize = core.TurretTileSize // px per hex tile (combat miniature; matches muzzle world offsets)

	// Level-up doctor-card layout.
	cardW   = 360.0
	cardH   = 300.0
	cardGap = 28.0
	cardY   = 210.0
)

// InGame is the main gameplay scene: a Vampire-Survivors-like run driven by the
// Ebiten-free core.World simulation. This scene only handles input and drawing.
type InGame struct {
	input      *ui.Input
	world      *core.World
	nextScene  ebiten.Game
	sequence   *bamenn.Sequence
	transition bamenn.Transition

	// turretRenderedAngle is the smoothed turret facing used for combat drawing.
	// It eases toward world.Player.FacingAngle so the turret rotates over a few
	// frames instead of snapping instantly.
	turretRenderedAngle float64

	// Combat cut cursor: IJKL moves the cursor over turret tiles; Space cuts the
	// selected tile (consumes one nipper). Always visible when nippers > 0.
	// Tank movement (WASD) is unaffected — cutting is a parallel input channel.
	cutCursor    hexmap.Index
	cutCursorSet bool
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
// from a fresh world. The seed is time-based so runs vary; use core.NewWorld
// directly with a fixed seed for deterministic tests.
func (g *InGame) OnStart() {
	g.world = core.NewWorld(time.Now().UnixNano())
	// Snap the rendered turret angle to the fresh world's facing so it doesn't
	// spin from a stale value on scene entry.
	g.turretRenderedAngle = g.world.Player.FacingAngle
}

func (g *InGame) Update() error {
	if g.world == nil {
		g.world = core.NewWorld(time.Now().UnixNano())
		g.turretRenderedAngle = g.world.Player.FacingAngle
	}

	switch g.world.State {
	case core.StateGameOver:
		if g.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
			g.sequence.SwitchWithTransition(g.nextScene, g.transition)
		}
	case core.StateLevelUp:
		g.handleLevelUpInput()
	default:
		// WASD drives the tank; IJKL drives the cut cursor in parallel.
		move := g.readMove()
		g.handleCombatCut()
		g.world.Update(move)
	}

	// Ease the rendered turret angle toward the tank's facing every frame.
	g.turretRenderedAngle = lerpAngle(g.turretRenderedAngle, g.world.Player.FacingAngle, 0.18)
	return nil
}

// lerpAngle eases a toward b by fraction t along the shortest angular path,
// so wrapping across ±pi rotates the short way rather than spinning around.
func lerpAngle(a, b, t float64) float64 {
	diff := math.Atan2(math.Sin(b-a), math.Cos(b-a))
	return a + diff*t
}

// handleLevelUpInput handles the doctor-card selection: click a card or press
// the matching number key (1-3) to apply that level-up offer.
func (g *InGame) handleLevelUpInput() {
	n := len(g.world.Choices)
	if n == 0 {
		return
	}

	if g.input.Mouse != nil && g.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		for i := 0; i < n; i++ {
			x := cardX(i, n)
			if float64(mx) >= x && float64(mx) < x+cardW &&
				float64(my) >= cardY && float64(my) < cardY+cardH {
				g.world.ChooseUpgrade(i)
				return
			}
		}
	}

	kb := g.input.Keyboard
	if kb != nil {
		keys := []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3}
		for i := 0; i < n && i < len(keys); i++ {
			if kb.IsJustPressed(keys[i]) {
				g.world.ChooseUpgrade(i)
				return
			}
		}
	}
}

// cardX returns the left x of the i-th level-up card when laying out n cards
// centred horizontally on screen.
func cardX(i, n int) float64 {
	total := float64(n)*cardW + float64(n-1)*cardGap
	startX := (screenW - total) / 2
	return startX + float64(i)*(cardW+cardGap)
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

// handleCombatCut manages the IJKL cut cursor. IJKL moves the cursor one tile
// in the pressed screen direction; Space cuts the highlighted tile (costs one
// nipper). The tank continues moving with WASD — no mode switch required.
func (g *InGame) handleCombatCut() {
	if g.world.Player.Nippers <= 0 {
		g.cutCursorSet = false
		return
	}

	tr := g.world.Turret()
	if tr == nil {
		g.cutCursorSet = false
		return
	}

	// Drop the cursor if it points at a tile that no longer exists / was cut.
	if g.cutCursorSet {
		if tile, ok := tr.Tiles()[g.cutCursor]; !ok || tile.IsPurged() || tr.IsGenerator(g.cutCursor) {
			g.cutCursorSet = false
		}
	}
	// Auto-initialise the cursor when we have nippers but no position yet.
	if !g.cutCursorSet {
		if idx, ok := g.nearestCuttableTile(); ok {
			g.cutCursor = idx
			g.cutCursorSet = true
		} else {
			return // no cuttable tiles
		}
	}

	kb := g.input.Keyboard
	if kb == nil {
		return
	}

	// IJKL moves the cursor one tile in the pressed screen direction.
	if kb.IsJustPressed(ebiten.KeyI) {
		g.moveCutCursor(0, -1)
	}
	if kb.IsJustPressed(ebiten.KeyK) {
		g.moveCutCursor(0, 1)
	}
	if kb.IsJustPressed(ebiten.KeyJ) {
		g.moveCutCursor(-1, 0)
	}
	if kb.IsJustPressed(ebiten.KeyL) {
		g.moveCutCursor(1, 0)
	}

	// Space cuts the selected tile.
	if kb.IsJustPressed(ebiten.KeySpace) {
		if g.world.CutTile(g.cutCursor) {
			// The cut tile (and cascade) is gone; re-seat the cursor.
			g.cutCursorSet = false
		}
	}
}

// tileRotOffset returns a tile's screen-space offset from the tank centre under
// the current rendered turret rotation (the same transform drawTurretCombat uses).
func (g *InGame) tileRotOffset(idx hexmap.Index) geom.PointF {
	theta := g.turretRenderedAngle + math.Pi/2
	dx, dy := hexLocalOffset(idx, combatTileSize)
	off := geom.PointF{X: dx, Y: dy}
	return geom.PointFFromPolar(off.Abs(), off.Angle()+theta)
}

// nearestCuttableTile returns the active, non-generator tile closest to the
// turret centre — a stable starting point for the cut cursor.
func (g *InGame) nearestCuttableTile() (hexmap.Index, bool) {
	tr := g.world.Turret()
	var best hexmap.Index
	bestD := math.Inf(1)
	found := false
	for idx, tile := range tr.Tiles() {
		if tile.IsPurged() || tr.IsGenerator(idx) {
			continue
		}
		dx, dy := hexLocalOffset(idx, combatTileSize)
		if d := math.Hypot(dx, dy); d < bestD {
			bestD = d
			best = idx
			found = true
		}
	}
	return best, found
}

// moveCutCursor moves the cursor to the active tile best matching the given
// unit screen direction (dirX, dirY) from the current cursor position. Tiles
// behind the direction are ignored; off-axis tiles are penalised so the cursor
// follows the most aligned, nearest tile.
func (g *InGame) moveCutCursor(dirX, dirY float64) {
	tr := g.world.Turret()
	cur := g.tileRotOffset(g.cutCursor)
	dir := geom.PointF{X: dirX, Y: dirY}

	var best hexmap.Index
	bestScore := math.Inf(1)
	found := false
	for idx, tile := range tr.Tiles() {
		if tile.IsPurged() || tr.IsGenerator(idx) || idx == g.cutCursor {
			continue
		}
		off := g.tileRotOffset(idx).Subtract(cur)
		proj := off.InnerProduct(dir)
		if proj <= 0 {
			continue // not in the pressed direction
		}
		dist := off.Abs()
		cos := proj / dist // dir is unit length
		score := dist / (cos * cos)
		if score < bestScore {
			bestScore = score
			best = idx
			found = true
		}
	}
	if found {
		g.cutCursor = best
	}
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
	for _, pk := range w.Pickups {
		// Nipper drop: bright yellow-green diamond-ish square.
		drawEntity(screen, cam, pk.Pos, 12, 12, 0.8, 1, 0.2, 1)
	}
	for _, e := range w.Enemies {
		s := float64(e.Radius) * 2
		if e.DropsNipper {
			drawEntity(screen, cam, e.Pos, s, s, 0.95, 0.8, 0.2, 1) // gold candlestick
		} else {
			drawEntity(screen, cam, e.Pos, s, s, 0.85, 0.25, 0.25, 1)
		}
	}
	for _, p := range w.Projectiles {
		drawEntity(screen, cam, p.Pos, 8, 8, 1, 0.9, 0.3, 1)
	}
	g.drawBeams(screen, cam)

	// Player tank.
	pr := w.Player.Radius * 2
	drawEntity(screen, cam, w.Player.Pos, pr, pr, 0.3, 0.8, 0.5, 1)

	// Turret miniature on top of the tank body, rotated to face movement direction.
	g.drawTurretCombat(screen, cam)

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
	drawing.DrawRect(screen, 0, 0, screenW, screenH, 0, 0, 0, 0.7)

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW/2-180, 120)
	drawing.DrawText(screen, fmt.Sprintf("LEVEL UP — Lv %d", g.world.Player.Level), 30, opt)

	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW/2-270, 166)
	drawing.DrawText(screen, "Three doctors approach. Choose one (click a card or press 1/2/3).", 16, opt)

	n := len(g.world.Choices)
	mx, my := ebiten.CursorPosition()
	for i, c := range g.world.Choices {
		x := cardX(i, n)
		hovered := float64(mx) >= x && float64(mx) < x+cardW &&
			float64(my) >= cardY && float64(my) < cardY+cardH

		bg := float32(0.10)
		if hovered {
			bg = 0.18
		}
		drawing.DrawRect(screen, x, cardY, cardW, cardH, bg, bg+0.02, bg+0.06, 0.98)
		drawing.DrawRect(screen, x, cardY, cardW, 4, 0.3, 0.7, 1, 1) // top accent

		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(x+14, cardY+16)
		drawing.DrawText(screen, fmt.Sprintf("%d", i+1), 22, opt)

		drawWrapped(screen, c.Name, x+14, cardY+56, cardW-28, 18)
		drawWrapped(screen, c.Desc, x+14, cardY+150, cardW-28, 14)
	}
}

// drawWrapped draws txt word-wrapped within maxWidth starting at (x, y), using
// the given font size. Lines advance by ~1.35x the font size.
func drawWrapped(screen *ebiten.Image, txt string, x, y, maxWidth, fontSize float64) {
	words := strings.Fields(txt)
	if len(words) == 0 {
		return
	}
	lineH := fontSize * 1.35
	line := ""
	flush := func() {
		if line == "" {
			return
		}
		opt := &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(x, y)
		drawing.DrawText(screen, line, fontSize, opt)
		y += lineH
		line = ""
	}
	for _, word := range words {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if drawing.MeasureText(candidate, fontSize).X > maxWidth && line != "" {
			flush()
			line = word
		} else {
			line = candidate
		}
	}
	flush()
}

// drawTurretCombat draws the turret hex grid miniature centred on the player tank,
// rotated to match the player's FacingAngle. Called every frame during play.
func (g *InGame) drawTurretCombat(screen *ebiten.Image, cam geom.PointF) {
	tr := g.world.Turret()
	if tr == nil {
		return
	}

	psx := g.world.Player.Pos.X - cam.X
	psy := g.world.Player.Pos.Y - cam.Y

	// Rotate so that -pi/2 (default facing = up) maps to 0 rotation on screen.
	// Use the smoothed angle so the turret eases into new headings.
	theta := g.turretRenderedAngle + math.Pi/2

	power := tr.ComputePower()

	// Collect and sort tiles for deterministic draw order.
	tiles := tr.Tiles()
	indices := make([]hexmap.Index, 0, len(tiles))
	for idx := range tiles {
		if tiles[idx].IsPurged() {
			continue
		}
		indices = append(indices, idx)
	}
	sort.Slice(indices, func(i, j int) bool {
		dxi, dyi := hexLocalOffset(indices[i], combatTileSize)
		offi := geom.PointF{X: dxi, Y: dyi}
		roti := geom.PointFFromPolar(offi.Abs(), offi.Angle()+theta)
		dxj, dyj := hexLocalOffset(indices[j], combatTileSize)
		offj := geom.PointF{X: dxj, Y: dyj}
		rotj := geom.PointFFromPolar(offj.Abs(), offj.Angle()+theta)
		cxi := psx + roti.X
		cxj := psx + rotj.X
		if cxi != cxj {
			return cxi < cxj
		}
		return psy+roti.Y < psy+rotj.Y
	})

	for _, idx := range indices {
		tile := tiles[idx]
		dx, dy := hexLocalOffset(idx, combatTileSize)
		off := geom.PointF{X: dx, Y: dy}
		rot := geom.PointFFromPolar(off.Abs(), off.Angle()+theta)
		cx := psx + rot.X
		cy := psy + rot.Y

		r, gr, b := tileColorRGB(tr, idx, tile, power[idx])
		drawing.DrawRect(screen, cx-combatTileSize/2, cy-combatTileSize/2, combatTileSize-2, combatTileSize-2, r, gr, b, 1)
	}

	// Cut cursor highlight: a white frame around the selected tile.
	if g.cutCursorSet {
		rot := g.tileRotOffset(g.cutCursor)
		cx := psx + rot.X
		cy := psy + rot.Y
		drawCursorFrame(screen, cx, cy, combatTileSize)
	}
}

// drawCursorFrame draws a white outline (four thin bars) around a tile centred
// at (cx, cy) so the selected cut target stands out against tile colours.
func drawCursorFrame(screen *ebiten.Image, cx, cy, size float64) {
	const t = 2.0
	h := size / 2
	drawing.DrawRect(screen, cx-h, cy-h, size, t, 1, 1, 1, 1)   // top
	drawing.DrawRect(screen, cx-h, cy+h-t, size, t, 1, 1, 1, 1) // bottom
	drawing.DrawRect(screen, cx-h, cy-h, t, size, 1, 1, 1, 1)   // left
	drawing.DrawRect(screen, cx+h-t, cy-h, t, size, 1, 1, 1, 1) // right
}

// tileColorRGB returns the display colour for a tile given its power level.
func tileColorRGB(tr *core.Turret, idx hexmap.Index, tile *core.Tile, power float64) (r, gr, b float32) {
	isGen := tr.IsGenerator(idx)
	switch tile.Component.(type) {
	case core.WeaponComponent:
		if power > 0 {
			return 1, 0.6, 0.1 // orange: active weapon
		}
		return 0.5, 0.25, 0.05 // dim orange: unpowered weapon
	case core.Junk:
		return 0.6, 0.4, 0.7 // purple: useless junk device
	default: // Wire or generator
		if isGen {
			return 1, 1, 0.2 // yellow: generator
		}
		if power > 0 {
			return 0.25, 0.45, 0.85 // blue: powered wire
		}
		return 0.12, 0.18, 0.35 // dark: unpowered wire
	}
}

// hexLocalOffset returns the tile centre offset (px) from the turret centre,
// for a tile at hex index idx, using the given tile size. Vertical brick layout.
func hexLocalOffset(idx hexmap.Index, size float64) (dx, dy float64) {
	q := float64(idx.X())
	r := float64(idx.Y())
	dx = q * size * 0.866
	dy = (r + q*0.5) * size
	return
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

	powerPerTile := 0.0
	if tr := g.world.Turret(); tr != nil {
		powerPerTile = tr.PowerPerTile()
	}
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(20, 64)
	drawing.DrawText(screen, fmt.Sprintf("Lv %d  Spd %.1f  Pwr/Tile %.1f  Nippers %d", p.Level, p.Speed, powerPerTile, p.Nippers), 18, opt)

	// Cut cursor hint.
	if p.Nippers > 0 {
		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(20, 88)
		drawing.DrawText(screen, "IJKL: move cut cursor  Space: cut tile", 14, opt)
	}

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

// drawBeams renders active laser beams as rotated quads using DrawTriangles.
func (g *InGame) drawBeams(screen *ebiten.Image, cam geom.PointF) {
	for _, b := range g.world.ActiveBeams() {
		end := b.Origin.Add(b.Dir.Multiply(b.Length))

		// Perpendicular unit vector (90° CCW rotation of Dir).
		perp := geom.PointF{X: -b.Dir.Y, Y: b.Dir.X}
		hw := float32(b.Width / 2)

		ox, oy := float32(b.Origin.X-cam.X), float32(b.Origin.Y-cam.Y)
		ex, ey := float32(end.X-cam.X), float32(end.Y-cam.Y)
		px, py := float32(perp.X)*hw, float32(perp.Y)*hw

		vertices := []ebiten.Vertex{
			{DstX: ox + px, DstY: oy + py, SrcX: 0, SrcY: 0, ColorR: 0.2, ColorG: 1, ColorB: 0.4, ColorA: 0.9},
			{DstX: ox - px, DstY: oy - py, SrcX: 0, SrcY: 0, ColorR: 0.2, ColorG: 1, ColorB: 0.4, ColorA: 0.9},
			{DstX: ex + px, DstY: ey + py, SrcX: 0, SrcY: 0, ColorR: 0.2, ColorG: 1, ColorB: 0.4, ColorA: 0.9},
			{DstX: ex - px, DstY: ey - py, SrcX: 0, SrcY: 0, ColorR: 0.2, ColorG: 1, ColorB: 0.4, ColorA: 0.9},
		}
		screen.DrawTriangles(vertices, []uint16{0, 1, 2, 1, 3, 2}, drawing.WhitePixel, &ebiten.DrawTrianglesOptions{})
	}
}
