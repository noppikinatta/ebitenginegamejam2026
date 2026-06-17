package scene

import (
	"fmt"
	"image/color"
	"math"
	"sort"
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

	// Turret overlay layout constants.
	tileSize       = 48.0                // px per hex tile square (level-up overlay)
	combatTileSize = core.TurretTileSize // px per hex tile (combat miniature; matches muzzle world offsets)
	turretAreaW    = 500.0               // width of the turret overlay panel
	turretAreaH    = 400.0               // height of the turret overlay panel
	turretAreaX    = (screenW - turretAreaW) / 2.0
	turretAreaY    = (screenH-turretAreaH)/2.0 + 20
	turretCenterX  = turretAreaX + turretAreaW/2.0
	turretCenterY  = turretAreaY + turretAreaH/2.0
)

// InGame is the main gameplay scene: a Vampire-Survivors-like run driven by the
// Ebiten-free core.World simulation. This scene only handles input and drawing.
type InGame struct {
	input      *ui.Input
	world      *core.World
	nextScene  ebiten.Game
	sequence   *bamenn.Sequence
	transition bamenn.Transition

	// hovered is the hex index the mouse is currently over during level-up.
	hovered    *hexmap.Index
	hoveredSet bool

	// turretRenderedAngle is the smoothed turret facing used for combat drawing.
	// It eases toward world.Player.FacingAngle so the turret rotates over a few
	// frames instead of snapping instantly.
	turretRenderedAngle float64

	// Combat cut-mode: while Shift is held (and nippers remain) the tank stops,
	// a cursor highlights one turret tile, WASD moves it, Space cuts it.
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
		// In cut-mode the tank stops (WASD drives the cursor instead), but the
		// simulation keeps running — enemies close in, creating the opening.
		move := g.readMove()
		if g.handleCombatCut() {
			move = geom.PointF{}
		}
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

// handleLevelUpInput handles click / Shift+click on the turret overlay.
// Left-click on a tile → tile-purge (Cut); Shift+left-click → weapon-purge (Disarm).
func (g *InGame) handleLevelUpInput() {
	mx, my := ebiten.CursorPosition()
	g.hoveredSet = false

	// Update hover state.
	if idx, ok := g.screenToHex(float64(mx), float64(my)); ok {
		g.hovered = &idx
		g.hoveredSet = true
	} else {
		g.hovered = nil
	}

	if g.input.Mouse != nil && g.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
		if g.hoveredSet && g.hovered != nil {
			idx := *g.hovered
			shiftHeld := ebiten.IsKeyPressed(ebiten.KeyShift)
			prefix := "Cut "
			if shiftHeld {
				prefix = "Disarm "
			}
			idxStr := idx.String()
			for i, c := range g.world.Choices {
				hasPrefix := len(c.Name) >= len(prefix) && c.Name[:len(prefix)] == prefix
				hasIdx := containsStr(c.Name, idxStr)
				if hasPrefix && hasIdx {
					g.world.ChooseUpgrade(i)
					return
				}
			}
		}
	}
}

func containsStr(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
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

// handleCombatCut runs the Shift+WASD cursor cut interaction. It returns true
// while cut-mode is active (the caller should freeze tank movement for the
// opening). Cut-mode requires Shift held and at least one nipper remaining.
func (g *InGame) handleCombatCut() bool {
	kb := g.input.Keyboard
	if kb == nil || !ebiten.IsKeyPressed(ebiten.KeyShift) || g.world.Player.Nippers <= 0 {
		g.cutCursorSet = false
		return false
	}

	tr := g.world.Turret()
	if tr == nil {
		g.cutCursorSet = false
		return false
	}

	// Drop the cursor if it points at a tile that no longer exists / was cut.
	if g.cutCursorSet {
		if tile, ok := tr.Tiles()[g.cutCursor]; !ok || tile.IsPurged() || tr.IsGenerator(g.cutCursor) {
			g.cutCursorSet = false
		}
	}
	// Initialise the cursor on the tile nearest the turret centre.
	if !g.cutCursorSet {
		if idx, ok := g.nearestCuttableTile(); ok {
			g.cutCursor = idx
			g.cutCursorSet = true
		} else {
			return true // no cuttable tiles, but still hold the tank stationary
		}
	}

	// WASD moves the cursor one tile in the pressed screen direction.
	if kb.IsJustPressed(ebiten.KeyW) || kb.IsJustPressed(ebiten.KeyArrowUp) {
		g.moveCutCursor(0, -1)
	}
	if kb.IsJustPressed(ebiten.KeyS) || kb.IsJustPressed(ebiten.KeyArrowDown) {
		g.moveCutCursor(0, 1)
	}
	if kb.IsJustPressed(ebiten.KeyA) || kb.IsJustPressed(ebiten.KeyArrowLeft) {
		g.moveCutCursor(-1, 0)
	}
	if kb.IsJustPressed(ebiten.KeyD) || kb.IsJustPressed(ebiten.KeyArrowRight) {
		g.moveCutCursor(1, 0)
	}

	// Space cuts the selected tile.
	if kb.IsJustPressed(ebiten.KeySpace) {
		if g.world.CutTile(g.cutCursor) {
			// The cut tile (and cascade) is gone; re-seat the cursor.
			g.cutCursorSet = false
		}
	}
	return true
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
	for _, e := range w.Enemies {
		s := float64(e.Radius) * 2
		drawEntity(screen, cam, e.Pos, s, s, 0.85, 0.25, 0.25, 1)
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
	drawing.DrawRect(screen, 0, 0, screenW, screenH, 0, 0, 0, 0.65)

	// Panel background.
	drawing.DrawRect(screen, turretAreaX-12, turretAreaY-60, turretAreaW+24, turretAreaH+80, 0.05, 0.07, 0.14, 0.97)

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(turretAreaX, turretAreaY-54)
	drawing.DrawText(screen, fmt.Sprintf("DISCONNECT  Lv %d", g.world.Player.Level), 28, opt)

	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(turretAreaX, turretAreaY-24)
	drawing.DrawText(screen, "Click tile to Cut  |  Shift+Click to Disarm weapon only  |  1-9 keys also work", 13, opt)

	// Draw turret hex grid.
	g.drawTurretGrid(screen)

	// Draw the currently hovered tile's choice description at the bottom.
	if g.hoveredSet && g.hovered != nil {
		idx := *g.hovered
		idxStr := idx.String()
		for _, c := range g.world.Choices {
			if containsStr(c.Name, idxStr) {
				opt = &ebiten.DrawImageOptions{}
				opt.GeoM.Translate(turretAreaX, turretAreaY+turretAreaH+4)
				drawing.DrawText(screen, c.Name, 16, opt)
				opt = &ebiten.DrawImageOptions{}
				opt.GeoM.Translate(turretAreaX, turretAreaY+turretAreaH+24)
				drawing.DrawText(screen, c.Desc, 13, opt)
				break
			}
		}
	} else {
		// Show numbered list as fallback when nothing is hovered.
		y := turretAreaY + turretAreaH + 4
		for i, c := range g.world.Choices {
			if i >= 6 {
				break
			}
			opt = &ebiten.DrawImageOptions{}
			opt.GeoM.Translate(turretAreaX, y)
			drawing.DrawText(screen, fmt.Sprintf("%d. %s", i+1, c.Name), 13, opt)
			y += 16
		}
	}
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
	if g.cutCursorSet && ebiten.IsKeyPressed(ebiten.KeyShift) {
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

// drawTurretGrid draws all tiles on the turret as coloured squares in hex brick layout.
// Tiles are sorted before drawing to ensure a deterministic draw order every frame,
// which prevents flicker caused by random map iteration order in Go.
func (g *InGame) drawTurretGrid(screen *ebiten.Image) {
	tr := g.world.Turret()
	if tr == nil {
		return
	}

	// Compute powered tiles for colour coding.
	power := tr.ComputePower()

	// Collect and sort indices: left-to-right (px), then top-to-bottom (py).
	// This eliminates flicker from random map iteration order.
	tiles := tr.Tiles()
	indices := make([]hexmap.Index, 0, len(tiles))
	for idx := range tiles {
		indices = append(indices, idx)
	}
	sort.Slice(indices, func(i, j int) bool {
		pxi, pyi := g.hexToScreen(indices[i])
		pxj, pyj := g.hexToScreen(indices[j])
		if pxi != pxj {
			return pxi < pxj
		}
		return pyi < pyj
	})

	for _, idx := range indices {
		tile := tiles[idx]
		px, py := g.hexToScreen(idx)
		const pad = 2.0
		x := px - tileSize/2 + pad
		y := py - tileSize/2 + pad
		w := tileSize - pad*2
		h := tileSize - pad*2

		if tile.IsPurged() {
			// Purged tiles are removed from the turret entirely; don't draw them.
			continue
		}

		r, gr, b := tileColorRGB(tr, idx, tile, power[idx])

		// Highlight hovered tile.
		if g.hoveredSet && g.hovered != nil && *g.hovered == idx {
			r = clamp32(r+0.3, 0, 1)
			gr = clamp32(gr+0.3, 0, 1)
			b = clamp32(b+0.3, 0, 1)
		}

		drawing.DrawRect(screen, x, y, w, h, r, gr, b, 1)

		// Draw left accent bar if the tile has a choice.
		if g.tileHasChoice(idx) {
			drawing.DrawRect(screen, x, y, 3, h, 1, 1, 1, 0.6)
		}

		// Label inside the tile.
		isGen := tr.IsGenerator(idx)
		lbl := tileShortLabel(tile, isGen)
		if lbl != "" {
			opt := &ebiten.DrawImageOptions{}
			opt.GeoM.Translate(px-tileSize/2+5, py-7)
			drawing.DrawText(screen, lbl, 11, opt)
		}
	}
}

func (g *InGame) tileHasChoice(idx hexmap.Index) bool {
	idxStr := idx.String()
	for _, c := range g.world.Choices {
		if containsStr(c.Name, idxStr) {
			return true
		}
	}
	return false
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

// hexToScreen converts a hex index to screen pixel position (centre of the tile).
// Vertical brick layout: tank "up" is screen up, so columns run vertically.
// Column q is offset horizontally by 0.866*tileSize; within each column, row r
// shifts down by half a tile for odd q values (brick stagger).
//
// Formula (pointy-top hex rendered as vertical bricks):
//
//	px = center + q * tileSize * 0.866
//	py = center + (r + q*0.5) * tileSize
func (g *InGame) hexToScreen(idx hexmap.Index) (px, py float64) {
	dx, dy := hexLocalOffset(idx, tileSize)
	return turretCenterX + dx, turretCenterY + dy
}

// screenToHex converts a screen pixel position to the nearest hex index,
// returning ok=false if the position is outside any tile bounding box.
func (g *InGame) screenToHex(sx, sy float64) (hexmap.Index, bool) {
	if g.world == nil || g.world.Turret() == nil {
		return hexmap.Index{}, false
	}
	for idx := range g.world.Turret().Tiles() {
		px, py := g.hexToScreen(idx)
		if sx >= px-tileSize/2 && sx < px+tileSize/2 &&
			sy >= py-tileSize/2 && sy < py+tileSize/2 {
			return idx, true
		}
	}
	return hexmap.Index{}, false
}

func tileShortLabel(tile *core.Tile, isGen bool) string {
	if isGen {
		return "GEN"
	}
	if tile == nil || tile.Component == nil {
		return ""
	}
	switch c := tile.Component.(type) {
	case core.WeaponComponent:
		n := c.Weapon.Name
		if len(n) > 4 {
			n = n[:4]
		}
		return n
	case core.Junk:
		return "JUNK"
	}
	return "W" // wire
}

func clamp32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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

	// Cut-mode hint.
	if p.Nippers > 0 {
		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(20, 88)
		drawing.DrawText(screen, "Hold Shift: aim with WASD, Space to cut a tile", 14, opt)
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
