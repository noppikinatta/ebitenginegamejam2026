package scene

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/ebitenginegamejam2026/asset"
	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/data"
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
	pauseTileSize  = 56.0                // px per hex tile in the zoomed pause/cut view (upright)

	// Sprite draw sizes (px at the 1:1 camera). These are the asset footprints
	// and are intentionally independent of the core collision radii.
	tankDrawW = 48.0 // tank is tall (portrait)
	tankDrawH = 64.0

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

	// paused freezes the simulation (Space toggles it). While paused the turret
	// is shown zoomed and upright, and clicking a tile cuts it (Disconnect)
	// without resuming, so several tiles can be cut in a row.
	paused bool

	// powerGaugeFill is the smoothed [0,1] fill of the left-edge power gauge. It
	// eases toward the normalised fire-rate multiplier each frame so the bar
	// visibly rises when a tile is cut (power re-concentrates into faster fire)
	// and falls when a doctor adds a tile.
	powerGaugeFill float64
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
	g.world = core.NewWorld(time.Now().UnixNano(), data.NewConfig())
	g.paused = false
	// Snap the rendered turret angle to the fresh world's facing so it doesn't
	// spin from a stale value on scene entry.
	g.turretRenderedAngle = g.world.Player.FacingAngle
	// Snap the power gauge to its current value so it doesn't sweep up from empty.
	g.powerGaugeFill = g.powerGaugeTarget()
}

// powerGaugeTarget returns the [0,1] fill the left-edge power gauge should ease
// toward: the current fire-rate multiplier normalised between the power curve's
// minimum and maximum (full bar = the curve's max multiplier, i.e. power
// concentrated into the fewest tiles).
func (g *InGame) powerGaugeTarget() float64 {
	if g.world.Turret() == nil {
		return 0
	}
	min, max := g.world.FireRateMultBounds()
	span := max - min
	if span <= 0 {
		return 0
	}
	r := (g.world.FireRateMultiplier() - min) / span
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

func (g *InGame) Update() error {
	if g.world == nil {
		g.world = core.NewWorld(time.Now().UnixNano(), data.NewConfig())
		g.turretRenderedAngle = g.world.Player.FacingAngle
	}

	switch g.world.State {
	case core.StateGameOver, core.StateCleared:
		if g.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft) {
			g.sequence.SwitchWithTransition(g.nextScene, g.transition)
		}
	case core.StateLevelUp:
		g.handleLevelUpInput()
	default:
		// Space toggles pause. While paused the world is frozen and the turret
		// is shown zoomed for click-to-cut; otherwise WASD drives the tank.
		if kb := g.input.Keyboard; kb != nil && kb.IsJustPressed(ebiten.KeySpace) {
			g.paused = !g.paused
		}
		if g.paused {
			g.handlePauseCut()
		} else {
			g.world.Update(g.readMove())
		}
	}

	// Ease the rendered turret angle toward the tank's facing every frame.
	g.turretRenderedAngle = lerpAngle(g.turretRenderedAngle, g.world.Player.FacingAngle, 0.18)
	// Ease the power gauge toward its target so cutting a tile visibly raises it.
	g.powerGaugeFill += (g.powerGaugeTarget() - g.powerGaugeFill) * 0.18
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

// handlePauseCut cuts the turret tile under the mouse on a left click while
// paused. Cutting spends a nipper and cascades; pause is NOT released, so the
// player can cut several tiles in a row. Mouse only — no keyboard.
func (g *InGame) handlePauseCut() {
	m := g.input.Mouse
	if m == nil || !m.IsJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	if idx, ok := g.pauseTileAtCursor(); ok {
		g.world.CutTile(idx) // CutTile checks nippers and rejects the generator
	}
}

// pauseCenter is the screen position the zoomed pause turret is centred on.
func pauseCenter() (cx, cy float64) {
	return screenW / 2, screenH/2 + 40
}

// pauseTileAtCursor returns the cuttable (active, non-generator) tile whose
// zoomed centre is nearest the mouse, within half a tile.
func (g *InGame) pauseTileAtCursor() (hexmap.Index, bool) {
	tr := g.world.Turret()
	if tr == nil {
		return hexmap.Index{}, false
	}
	cx, cy := pauseCenter()
	mx, my := ebiten.CursorPosition()

	var best hexmap.Index
	bestD := pauseTileSize / 2
	found := false
	for idx, tile := range tr.Tiles() {
		if tile.IsPurged() || tr.IsGenerator(idx) {
			continue
		}
		c := tileScreenCenter(idx, cx, cy, pauseTileSize)
		if d := math.Hypot(float64(mx)-c.X, float64(my)-c.Y); d <= bestD {
			bestD = d
			best = idx
			found = true
		}
	}
	return best, found
}

// tileScreenCenter returns a tile's upright (unrotated) screen centre for a
// turret drawn centred at (cx, cy) with the given tile size.
func tileScreenCenter(idx hexmap.Index, cx, cy, size float64) geom.PointF {
	dx, dy := hexLocalOffset(idx, size)
	return geom.PointF{X: cx + dx, Y: cy + dy}
}

func (g *InGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{12, 14, 22, 255})

	w := g.world
	// Camera keeps the player centred on screen.
	cam := geom.PointF{X: w.Player.Pos.X - screenW/2, Y: w.Player.Pos.Y - screenH/2}

	g.drawGrid(screen, cam)

	for _, gem := range w.Gems {
		drawSprite(screen, cam, asset.ImgGem, gem.Pos, 8, 8, 0, 1, 1, 1, 1)
	}
	for _, pk := range w.Pickups {
		drawSprite(screen, cam, asset.ImgNipper, pk.Pos, 12, 12, 0, 1, 1, 1, 1)
	}
	for _, e := range w.Enemies {
		sz := e.Radius * 2 // sprite footprint follows the collision radius
		drawSprite(screen, cam, enemySpriteKey(e), e.Pos, sz, sz, 0, 1, 1, 1, 1)
	}
	for _, p := range w.Projectiles {
		drawSprite(screen, cam, asset.ImgProjectile, p.Pos, 8, 8, 0, 1, 1, 1, 1)
	}
	g.drawBeams(screen, cam)

	// Player tank (tall sprite, authored pointing up; rotate to face movement
	// using the same smoothed angle as the turret so body and turret ease
	// together). Collision radius is separate, in core.
	drawSprite(screen, cam, asset.ImgTank, w.Player.Pos, tankDrawW, tankDrawH, g.turretRenderedAngle+math.Pi/2, 1, 1, 1, 1)

	// Turret miniature on top of the tank body, rotated to face movement direction.
	g.drawTurretCombat(screen, cam)

	g.drawExplosions(screen, cam)

	g.drawHUD(screen)
	g.drawBossBar(screen)

	if g.paused && w.State == core.StatePlaying {
		g.drawPause(screen)
	}

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
	case core.StateCleared:
		drawing.DrawRect(screen, 0, 0, screenW, screenH, 0.02, 0.10, 0.06, 0.6)
		opt := &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(430, 290)
		drawing.DrawText(screen, "MISSION COMPLETE", 48, opt)
		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(430, 360)
		drawing.DrawText(screen, "The Disconnector is destroyed. Click to continue.", 22, opt)
	}

	// Power-per-tile gauge on the left edge, drawn last so it stays visible above
	// the pause and level-up overlays (the moments power changes most).
	if w.State != core.StateGameOver && w.State != core.StateCleared {
		g.drawPowerGauge(screen)
	}
}

// Left-edge power gauge geometry.
const (
	powerGaugeX      = 24.0
	powerGaugeW      = 22.0
	powerGaugeTop    = 132.0
	powerGaugeBottom = screenH - 40.0
)

// drawPowerGauge draws a vertical bar on the left edge whose height encodes the
// turret-wide fire-rate multiplier. It fills from the bottom up and brightens as
// the multiplier rises, so disconnecting tiles (which re-concentrates power into
// faster fire) makes the bar climb.
func (g *InGame) drawPowerGauge(screen *ebiten.Image) {
	trackH := powerGaugeBottom - powerGaugeTop

	// Track (dim background) and a 1px frame so the empty bar still reads.
	drawing.DrawRect(screen, powerGaugeX, powerGaugeTop, powerGaugeW, trackH, 0.10, 0.12, 0.16, 0.9)

	fill := g.powerGaugeFill
	if fill < 0 {
		fill = 0
	}
	if fill > 1 {
		fill = 1
	}
	fillH := trackH * fill
	if fillH > 0 {
		// Interpolate dim-blue (low) -> bright-cyan (high) so a charged turret reads hot.
		r := float32(0.15 + 0.25*fill)
		gr := float32(0.40 + 0.55*fill)
		b := float32(0.75 + 0.25*fill)
		drawing.DrawRect(screen, powerGaugeX, powerGaugeBottom-fillH, powerGaugeW, fillH, r, gr, b, 1)
	}

	// Label above the bar plus the exact fire-rate multiplier below it.
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(powerGaugeX-4, powerGaugeTop-26)
	drawing.DrawText(screen, "PWR", 14, opt)
	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(powerGaugeX-6, powerGaugeBottom+4)
	drawing.DrawText(screen, fmt.Sprintf("x%.2f", g.world.FireRateMultiplier()), 14, opt)
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

		// Header: pick number + doctor name.
		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(x+14, cardY+14)
		drawing.DrawText(screen, fmt.Sprintf("%d", i+1), 22, opt)
		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(x+46, cardY+16)
		drawing.DrawText(screen, "Dr. "+c.Doctor, 20, opt)

		// One line per item: label, icon, then name.
		itemY := cardY + 62.0
		for _, it := range c.Items {
			drawOfferItem(screen, it, x+14, itemY)
			itemY += 46
		}
	}
}

// drawOfferItem draws a single proposal line at (x, y): an "Add"/"Upgrade" label,
// the component's icon, then its name.
func drawOfferItem(screen *ebiten.Image, it core.OfferItem, x, y float64) {
	if label := offerLabel(it.Kind); label != "" {
		opt := &ebiten.DrawImageOptions{}
		opt.ColorScale.Scale(0.55, 0.70, 0.95, 1) // dim blue label
		opt.GeoM.Translate(x, y+11)
		drawing.DrawText(screen, label, 13, opt)
	}

	key, weapon := offerIcon(it)
	img := drawing.Image(key)
	icx, icy := x+90, y+16
	if weapon { // weapon barrels keep their tall aspect ratio
		b := img.Bounds()
		const h = 30.0
		drawing.DrawSprite(screen, img, icx, icy, h*float64(b.Dx())/float64(b.Dy()), h, 0, 1, 1, 1, 1)
	} else {
		drawing.DrawSprite(screen, img, icx, icy, 30, 30, 0, 1, 1, 1, 1)
	}

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(x+114, y+5)
	drawing.DrawText(screen, it.Text, 18, opt)
}

// offerLabel is the prefix shown for a proposal line ("" for nippers).
func offerLabel(k core.OfferKind) string {
	switch k {
	case core.OfferUpgrade:
		return "Upgrade"
	case core.OfferNippers:
		return ""
	default: // adds
		return "Add"
	}
}

// offerIcon returns the preview image key for a proposal item and whether it is a
// (tall-aspect) weapon barrel.
func offerIcon(it core.OfferItem) (key string, weapon bool) {
	switch it.Kind {
	case core.OfferAddWeapon, core.OfferUpgrade:
		return weaponTileKey(it.Weapon), true
	case core.OfferAddCapacitor:
		return asset.ImgTileCapacitor, false
	case core.OfferNippers:
		return asset.ImgNipper, false
	default: // OfferAddJunk
		return asset.ImgTileJunk, false
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

// drawTurretCombat draws the turret hex grid miniature centred on the player
// tank, rotated to match the player's FacingAngle. Called every frame during play.
func (g *InGame) drawTurretCombat(screen *ebiten.Image, cam geom.PointF) {
	psx := g.world.Player.Pos.X - cam.X
	psy := g.world.Player.Pos.Y - cam.Y
	// Rotate so that -pi/2 (default facing = up) maps to 0 rotation on screen.
	// Use the smoothed angle so the turret eases into new headings.
	theta := g.turretRenderedAngle + math.Pi/2
	g.drawTurretTiles(screen, psx, psy, combatTileSize, theta, true)
}

// drawTurretTiles renders the turret centred at screen (cx, cy) with the given
// tile size and rotation. Weapons are drawn in two layers: a wire socket base,
// then the barrel on top. Used by both the combat miniature (small, rotated)
// and the paused cut view (large, upright). theta=0 draws the turret upright.
//
// When aimBarrels is true each weapon barrel points along its own live aim angle
// (so lock-on weapons face their target / firing direction) instead of the grid
// rotation; the pause view passes false to keep barrels upright for clarity.
func (g *InGame) drawTurretTiles(screen *ebiten.Image, cx, cy, size, theta float64, aimBarrels bool) {
	tr := g.world.Turret()
	if tr == nil {
		return
	}
	power := tr.ComputePower()
	tiles := tr.Tiles()

	// Collect active tiles with their screen centres, sorted for painter order.
	type placed struct {
		idx hexmap.Index
		c   geom.PointF
	}
	ps := make([]placed, 0, len(tiles))
	for idx, t := range tiles {
		if t.IsPurged() {
			continue
		}
		dx, dy := hexLocalOffset(idx, size)
		off := geom.PointF{X: dx, Y: dy}
		rot := geom.PointFFromPolar(off.Abs(), off.Angle()+theta)
		ps = append(ps, placed{idx: idx, c: geom.PointF{X: cx + rot.X, Y: cy + rot.Y}})
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].c.Y != ps[j].c.Y {
			return ps[i].c.Y < ps[j].c.Y // top-to-bottom so lower (nearer) tiles draw on top
		}
		return ps[i].c.X < ps[j].c.X
	})

	// Pass 1: tile bases (weapons reuse the wire socket as their base).
	for _, p := range ps {
		key, dim := tileBase(tr, p.idx, tiles[p.idx], power[p.idx])
		drawing.DrawSprite(screen, drawing.Image(key), p.c.X, p.c.Y, size, size, theta, dim, dim, dim, 1)
	}

	// Pass 2: weapon barrels. These are rectangular sprites taller than a tile,
	// authored pointing "up" (= forward) with their mount tile as the bottom
	// TurretTileSize×TurretTileSize block. We anchor at that mount-tile centre and
	// scale uniformly by the tile zoom, so the barrel base sits on the socket and
	// the barrel swings about it as the turret rotates. Drawn after all bases (and
	// in Y order) so a front tile's barrel overlays the tiles behind it.
	z := size / core.TurretTileSize
	for _, p := range ps {
		wc, ok := tiles[p.idx].Component.(core.WeaponComponent)
		if !ok {
			continue
		}
		dim := float32(1)
		if power[p.idx] <= 0 {
			dim = 0.5
		}
		img := drawing.Image(weaponTileKey(wc.Weapon.Kind))
		b := img.Bounds()
		ax := float64(b.Dx()) / 2                     // mount tile is horizontally centred
		ay := float64(b.Dy()) - core.TurretTileSize/2 // ...and is the bottom tile block
		// Combat: each barrel points where it actually fires (source "up" = the aim
		// direction, hence +pi/2). Pause: keep the grid rotation (upright).
		barrelTheta := theta
		if aimBarrels {
			barrelTheta = wc.Weapon.RenderAngle() + math.Pi/2
		}
		drawing.DrawSpriteAnchored(screen, img, p.c.X, p.c.Y, z, barrelTheta, ax, ay, dim, dim, dim, 1)
	}
}

// drawPause renders the zoomed, upright cut view over a dimmed world: the tank
// and turret blown up so individual tiles can be clicked to cut. The tile under
// the mouse is highlighted.
func (g *InGame) drawPause(screen *ebiten.Image) {
	drawing.DrawRect(screen, 0, 0, screenW, screenH, 0, 0, 0, 0.7)

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW/2-280, 56)
	drawing.DrawText(screen, fmt.Sprintf("PAUSED — click a tile to cut  (Nippers %d)", g.world.Player.Nippers), 24, opt)
	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW/2-280, 96)
	drawing.DrawText(screen, "Cutting disconnects the tile and everything downstream. Space resumes.", 16, opt)

	cx, cy := pauseCenter()
	zoom := pauseTileSize / combatTileSize

	// Tank upright, behind the turret.
	drawing.DrawSprite(screen, drawing.Image(asset.ImgTank), cx, cy, tankDrawW*zoom, tankDrawH*zoom, 0, 1, 1, 1, 1)
	g.drawTurretTiles(screen, cx, cy, pauseTileSize, 0, false)
	g.drawTileLevels(screen, cx, cy)

	// Highlight the tile under the cursor, plus a cut preview: the collateral
	// tiles that would cascade-cut are framed in a dimmer white. The hovered
	// tile's name and description are shown in a panel at the bottom.
	if idx, ok := g.pauseTileAtCursor(); ok {
		for pidx := range g.world.Turret().CutPreview(idx) {
			if pidx == idx {
				continue // the target itself gets the bright frame below
			}
			c := tileScreenCenter(pidx, cx, cy, pauseTileSize)
			drawTileFrame(screen, c.X, c.Y, pauseTileSize, 0.7, 0.7, 0.7, 0.9) // dim: collateral
		}
		c := tileScreenCenter(idx, cx, cy, pauseTileSize)
		drawTileFrame(screen, c.X, c.Y, pauseTileSize, 1, 1, 1, 1) // bright: target
		g.drawPauseTileInfo(screen, idx)
	}
}

// drawTileLevels draws a "+N" badge on each upgraded weapon tile in the upright
// pause view, so the player can see how many times a weapon has been upgraded.
// Drawn after the turret so the badges sit on top of the barrels.
func (g *InGame) drawTileLevels(screen *ebiten.Image, cx, cy float64) {
	for idx, tile := range g.world.Turret().Tiles() {
		if tile.IsPurged() {
			continue
		}
		wc, ok := tile.Component.(core.WeaponComponent)
		if !ok || wc.Weapon.Level == 0 {
			continue
		}
		c := tileScreenCenter(idx, cx, cy, pauseTileSize)
		label := fmt.Sprintf("+%d", wc.Weapon.Level)
		sz := drawing.MeasureText(label, 16)
		opt := &ebiten.DrawImageOptions{}
		opt.ColorScale.Scale(1, 0.85, 0.30, 1) // gold
		opt.GeoM.Translate(c.X+pauseTileSize/2-sz.X-4, c.Y+pauseTileSize/2-sz.Y-2)
		drawing.DrawText(screen, label, 16, opt)
	}
}

// drawPauseTileInfo draws a bottom panel describing the hovered turret tile: a
// preview image, its name, and a one-line explanation, so the player knows what
// they are about to cut.
func (g *InGame) drawPauseTileInfo(screen *ebiten.Image, idx hexmap.Index) {
	tile := g.world.Turret().Tiles()[idx]
	if tile == nil {
		return
	}
	name, desc, imgKey, weapon := pauseTileInfo(tile.Component)

	const bx, bh = 24.0, 110.0
	bw := float64(screenW) - 2*bx
	by := float64(screenH) - bh - 16
	drawing.DrawRect(screen, bx, by, bw, bh, 0.06, 0.07, 0.10, 0.92)

	// Preview image on the left (weapon barrels keep their tall aspect ratio).
	icx, icy := bx+70, by+bh/2
	img := drawing.Image(imgKey)
	if weapon {
		b := img.Bounds()
		h := 84.0
		drawing.DrawSprite(screen, img, icx, icy, h*float64(b.Dx())/float64(b.Dy()), h, 0, 1, 1, 1, 1)
	} else {
		drawing.DrawSprite(screen, img, icx, icy, 72, 72, 0, 1, 1, 1, 1)
	}

	// Name + wrapped description on the right.
	tx := bx + 150
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(tx, by+14)
	drawing.DrawText(screen, name, 26, opt)
	drawWrapped(screen, desc, tx, by+58, bw-170, 18)
}

// pauseTileInfo returns the display name, description, preview image key, and
// whether the component is a weapon (a tall barrel sprite) for the pause info
// panel. These strings are UI copy; if the game grows localisation they can move
// to the lang CSVs.
func pauseTileInfo(comp core.Component) (name, desc, imgKey string, weapon bool) {
	switch c := comp.(type) {
	case core.WeaponComponent:
		name := c.Weapon.Name
		if c.Weapon.Level > 0 {
			name = fmt.Sprintf("%s  +%d", name, c.Weapon.Level)
		}
		return name, weaponDesc(c.Weapon.Kind), weaponTileKey(c.Weapon.Kind), true
	case core.Junk:
		return c.Name(), "A useless gadget a doctor bolted on. It conducts power but does nothing — a prime tile to cut.", asset.ImgTileJunk, false
	case core.Capacitor:
		return "Capacitor", "Equipment that raises the turret's fire rate while it stays connected.", asset.ImgTileCapacitor, false
	default: // Wire (or empty)
		return "Wire", "A bare conductor: it carries power but does nothing on its own. Cut it to reshape the layout.", asset.ImgTileWire, false
	}
}

// weaponDesc returns a short explanation of a weapon kind for the info panel.
func weaponDesc(k core.WeaponKind) string {
	switch k {
	case core.KindShotgun:
		return "Sprays a short-range spread of pellets at the nearest enemy."
	case core.KindSniper:
		return "Fires a fast, long-range round that hits hard."
	case core.KindLaser:
		return "Fires a sustained beam at the nearest enemy, piercing everything in its path."
	case core.KindGatling:
		return "Streams a rapid burst of pellets straight ahead."
	case core.KindGrenade:
		return "Lobs a shell outward that explodes where it lands."
	case core.KindCIWS:
		return "Point defence: holds fire until an enemy is close, then unleashes a burst."
	case core.KindMissile:
		return "Launches a homing missile that explodes on impact."
	default: // KindCannon
		return "Fires a balanced shell at the nearest enemy."
	}
}

// drawTileFrame draws an outline (four thin bars) around a tile centred at
// (cx, cy) in colour (r,g,b,a), so cut targets stand out against tile colours.
func drawTileFrame(screen *ebiten.Image, cx, cy, size float64, r, g, b, a float32) {
	const t = 2.0
	h := size / 2
	drawing.DrawRect(screen, cx-h, cy-h, size, t, r, g, b, a)   // top
	drawing.DrawRect(screen, cx-h, cy+h-t, size, t, r, g, b, a) // bottom
	drawing.DrawRect(screen, cx-h, cy-h, t, size, r, g, b, a)   // left
	drawing.DrawRect(screen, cx+h-t, cy-h, t, size, r, g, b, a) // right
}

// tileBase returns the under-layer image key for a tile plus a brightness
// multiplier (dim) so unpowered tiles render darker — power can't be encoded in
// a single sprite, so it is applied as a colour-scale tint. Weapon tiles use the
// wire socket as their base; the barrel is layered on top separately.
func tileBase(tr *core.Turret, idx hexmap.Index, tile *core.Tile, power float64) (key string, dim float32) {
	if tr.IsGenerator(idx) {
		return asset.ImgTileGenerator, 1
	}
	switch tile.Component.(type) {
	case core.Junk:
		return asset.ImgTileJunk, 1
	case core.Capacitor:
		if power <= 0 {
			return asset.ImgTileCapacitor, 0.45 // dim: disconnected capacitor
		}
		return asset.ImgTileCapacitor, 1
	case core.WeaponComponent:
		if power <= 0 {
			return asset.ImgTileWire, 0.45 // dim: unpowered weapon socket
		}
		return asset.ImgTileWire, 1
	default: // Wire
		if power <= 0 {
			return asset.ImgTileWire, 0.45 // dim: unpowered wire
		}
		return asset.ImgTileWire, 1
	}
}

// weaponTileKey maps a weapon kind to its turret-tile image key.
func weaponTileKey(k core.WeaponKind) string {
	switch k {
	case core.KindShotgun:
		return asset.ImgTileWeaponShotgun
	case core.KindSniper:
		return asset.ImgTileWeaponSniper
	case core.KindLaser:
		return asset.ImgTileWeaponLaser
	case core.KindGatling:
		return asset.ImgTileWeaponGatling
	case core.KindGrenade:
		return asset.ImgTileWeaponGrenade
	case core.KindCIWS:
		return asset.ImgTileWeaponCIWS
	case core.KindMissile:
		return asset.ImgTileWeaponMissile
	default:
		return asset.ImgTileWeaponCannon
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

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(20, 64)
	drawing.DrawText(screen, fmt.Sprintf("Lv %d  Spd %.1f  Fire x%.2f  Nippers %d", p.Level, p.Speed, g.world.FireRateMultiplier(), p.Nippers), 18, opt)

	// Cut hint.
	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(20, 88)
	drawing.DrawText(screen, "Space: pause & cut turret tiles", 14, opt)

	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW-220, 20)
	secs := g.world.Tick / 60
	drawing.DrawText(screen, fmt.Sprintf("Time %d:%02d  Kills %d", secs/60, secs%60, g.world.Kills), 18, opt)
}

func (g *InGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}

// drawSprite draws the image keyed by key centred on world position pos
// (transformed by cam), scaled to w×h, rotated by angle, and tinted by
// (r,gr,b,a). It is the sprite-based replacement for the old drawEntity.
func drawSprite(screen *ebiten.Image, cam geom.PointF, key string, pos geom.PointF, w, h, angle float64, r, gr, b, a float32) {
	drawing.DrawSprite(screen, drawing.Image(key), pos.X-cam.X, pos.Y-cam.Y, w, h, angle, r, gr, b, a)
}

// enemySpriteKey selects the sprite for an enemy: candlestick, boss, or the
// per-kind zako sprite.
func enemySpriteKey(e *core.Enemy) string {
	switch {
	case e.DropsNipper:
		return asset.ImgCandlestick
	case e.IsBoss:
		return asset.ImgBoss
	case e.Kind == core.EnemySwarmer:
		return asset.ImgEnemySwarmer
	case e.Kind == core.EnemyBrute:
		return asset.ImgEnemyBrute
	default:
		return asset.ImgEnemy
	}
}

// drawBossBar draws a name + health bar across the top when a boss is on the
// field, so the player can read the boss fight's progress.
func (g *InGame) drawBossBar(screen *ebiten.Image) {
	b := g.world.ActiveBoss()
	if b == nil {
		return
	}
	const bw, bh, by = 600.0, 16.0, 30.0
	bx := (screenW - bw) / 2

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(bx, by-22)
	drawing.DrawText(screen, b.Name, 18, opt)

	drawing.DrawRect(screen, bx, by, bw, bh, 0.15, 0.05, 0.08, 1)
	frac := 0.0
	if b.MaxHP > 0 {
		frac = b.HP / b.MaxHP
	}
	if frac < 0 {
		frac = 0
	}
	drawing.DrawRect(screen, bx, by, bw*frac, bh, 0.85, 0.25, 0.30, 1)
}

// drawExplosions renders each queued explosion as an orange circle that fades
// out (premultiplied alpha) as its Life counts down.
func (g *InGame) drawExplosions(screen *ebiten.Image, cam geom.PointF) {
	for _, e := range g.world.Explosions {
		if e.MaxLife <= 0 {
			continue
		}
		f := float32(e.Life) / float32(e.MaxLife)
		cx := float32(e.Pos.X - cam.X)
		cy := float32(e.Pos.Y - cam.Y)
		// Orange (255,140,0) with premultiplied alpha so it fades to transparent.
		c := color.RGBA{R: uint8(255 * f), G: uint8(140 * f), B: 0, A: uint8(255 * f)}
		vector.DrawFilledCircle(screen, cx, cy, float32(e.Radius), c, true)
	}
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
