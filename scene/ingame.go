package scene

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
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
	"github.com/noppikinatta/ebitenginegamejam2026/lang"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

// combatTileSize is the px-per-hex tile for the combat miniature. It is NOT a
// free knob: it must equal core.TurretTileSize so the drawn turret lines up with
// the muzzle world offsets, so it lives here rather than in tuning.go. All other
// scene rendering/layout knobs are in tuning.go.
const combatTileSize = core.TurretTileSize

// InGame is the main gameplay scene: a Vampire-Survivors-like run driven by the
// Ebiten-free core.World simulation. This scene only handles input and drawing.
type InGame struct {
	input      *ui.Input
	world      *core.World
	runSeed    *runSeed // shared seed source: opening's turret seed, or fresh on retry
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

	// popups are the active floating damage numbers (presentation only; spawned
	// from world.DamageEvents each tick). rng scatters their launch directions.
	popups []damagePopup
	rng    *rand.Rand

	// deaths are the fading enemy sprites left where enemies died (spawned from
	// world.DeathEvents each tick).
	deaths []deathFX

	// hpShake counts down the HP bar's post-hit shake (set when the tank takes
	// damage, decremented each tick).
	hpShake int
}

func NewInGame(input *ui.Input, seed *runSeed) *InGame {
	return &InGame{
		input:   input,
		runSeed: seed,
	}
}

// Outcome reports how the last run ended, for the result screen to branch on.
type Outcome int

const (
	OutcomeNone Outcome = iota // run still in progress (or not started)
	OutcomeWin                 // final boss defeated
	OutcomeLose                // player destroyed
)

// Outcome returns the result of the current world's run.
func (g *InGame) Outcome() Outcome {
	if g.world == nil {
		return OutcomeNone
	}
	switch g.world.State {
	case core.StateCleared:
		return OutcomeWin
	case core.StateGameOver:
		return OutcomeLose
	default:
		return OutcomeNone
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
	// Use the seed the opening generated its turret from (so the run matches the
	// cinematic); a retry from the result screen has no pending seed and rolls fresh.
	g.world = core.NewWorld(g.runSeed.take(), data.NewConfig())
	g.paused = false
	g.popups = g.popups[:0]
	g.deaths = g.deaths[:0]
	g.hpShake = 0
	if g.rng == nil {
		g.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	asset.PlayBGM(asset.BGMGame)
	// Snap the rendered turret angle to the fresh world's facing so it doesn't
	// spin from a stale value on scene entry.
	g.turretRenderedAngle = g.world.Player.FacingAngle
	// Snap the power gauge to its current value so it doesn't sweep up from empty.
	g.powerGaugeFill = g.powerGaugeTarget()
}

// soundSink routes core sound events to the asset audio backend. It is the
// Ebiten-side implementation of core.SoundSink injected each frame.
type soundSink struct{}

func (soundSink) PlaySound(e core.SoundEvent) {
	switch e {
	case core.SndFireCannon:
		asset.PlaySound(asset.SEFireCannon)
	case core.SndFireShotgun:
		asset.PlaySound(asset.SEFireShotgun)
	case core.SndFireSniper:
		asset.PlaySound(asset.SEFireSniper)
	case core.SndFireLaser:
		asset.PlaySound(asset.SEFireLaser)
	case core.SndFireGatling:
		asset.PlaySound(asset.SEFireGatling)
	case core.SndFireGrenade:
		asset.PlaySound(asset.SEFireGrenade)
	case core.SndFireCIWS:
		asset.PlaySound(asset.SEFireCIWS)
	case core.SndFireMissile:
		asset.PlaySound(asset.SEFireMissile)
	case core.SndExplosion:
		asset.PlaySound(asset.SEExplosion)
	case core.SndPlayerHit:
		asset.PlaySound(asset.SEPlayerHit)
	}
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
		g.world = core.NewWorld(g.runSeed.take(), data.NewConfig())
		g.turretRenderedAngle = g.world.Player.FacingAngle
	}
	if g.rng == nil {
		g.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
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
			core.DispatchSounds(g.world.SoundEvents, soundSink{})
			g.spawnDamagePopups()
			g.updateDamagePopups()
			g.spawnDeathFX()
			g.updateDeathFX()
			if g.hpShake > 0 {
				g.hpShake--
			}
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

	// Background scrolls with the camera (bgScrollMul = 1 locks the scenery 1:1
	// to the world), so moving the tank scrolls the scenery underneath it.
	drawScrollBG(screen, cam.X*bgScrollMul, cam.Y*bgScrollMul)
	g.drawGrid(screen, cam)

	for _, gem := range w.Gems {
		drawSprite(screen, cam, asset.ImgGem, gem.Pos, 8, 8, 0, 1, 1, 1, 1)
	}
	for _, pk := range w.Pickups {
		drawSprite(screen, cam, asset.ImgNipper, pk.Pos, 16, 16, 0, 1, 1, 1, 1)
	}
	g.drawDeathFX(screen, cam) // fading corpses, under the live enemies
	for _, e := range w.Enemies {
		sz := e.Radius * 2 // sprite footprint follows the collision radius
		drawSprite(screen, cam, enemySpriteKey(e), e.Pos, sz, sz, 0, 1, 1, 1, 1)
	}
	for _, p := range w.Projectiles {
		// Weapons and junk emitters tag projectiles with their own sprite key and
		// draw footprint; anything without a key falls back to the default bullet.
		key := p.Sprite
		if key == "" {
			key = asset.ImgProjectile
		}
		dw, dh := p.DrawW, p.DrawH
		if dw == 0 && dh == 0 {
			// Legacy fallback for projectiles that carry no explicit size: the
			// default bullet is small (8), tagged sprites (junk) are larger (16).
			if p.Sprite == "" {
				dw, dh = 8, 8
			} else {
				dw, dh = 16, 16
			}
		}
		// Elongated bullets (cannon/sniper/missile) are authored pointing up, so
		// rotate by the travel angle + 90° (same convention as the tank body).
		angle := 0.0
		if p.FaceVelocity && p.Vel.Abs() > 0 {
			angle = p.Vel.Angle() + math.Pi/2
		}
		drawSprite(screen, cam, key, p.Pos, dw, dh, angle, 1, 1, 1, 1)
	}
	g.drawBeams(screen, cam)

	// Player tank (tall sprite, authored pointing up; rotate to face movement
	// using the same smoothed angle as the turret so body and turret ease
	// together). Collision radius is separate, in core.
	drawSprite(screen, cam, asset.ImgTank, w.Player.Pos, tankDrawW, tankDrawH, g.turretRenderedAngle+math.Pi/2, 1, 1, 1, 1)

	// Turret miniature on top of the tank body, rotated to face movement direction.
	g.drawTurretCombat(screen, cam)

	g.drawExplosions(screen, cam)
	g.drawDamagePopups(screen, cam)

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
		drawing.DrawTextByKey(screen, "game-over", 48, opt)
		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(500, 380)
		drawing.DrawTextByKey(screen, "click-continue", 24, opt)
	case core.StateCleared:
		drawing.DrawRect(screen, 0, 0, screenW, screenH, 0.02, 0.10, 0.06, 0.6)
		opt := &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(430, 290)
		drawing.DrawTextByKey(screen, "mission-complete", 48, opt)
		opt = &ebiten.DrawImageOptions{}
		opt.GeoM.Translate(430, 360)
		drawing.DrawTextByKey(screen, "victory-detail", 22, opt)
	}

	// Power-per-tile gauge on the left edge, drawn last so it stays visible above
	// the pause and level-up overlays (the moments power changes most).
	if w.State != core.StateGameOver && w.State != core.StateCleared {
		g.drawPowerGauge(screen)
	}
}

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
	drawing.DrawTextByKey(screen, "hud-pwr", 14, opt)
	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(powerGaugeX-6, powerGaugeBottom+4)
	drawing.DrawTextTemplate(screen, "hud-pwr-mult", map[string]any{"Mult": fmt.Sprintf("%.2f", g.world.FireRateMultiplier())}, 14, opt)
}

func (g *InGame) drawLevelUp(screen *ebiten.Image) {
	drawing.DrawRect(screen, 0, 0, screenW, screenH, 0, 0, 0, 0.7)

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW/2-180, 120)
	drawing.DrawTextTemplate(screen, "levelup-title", map[string]any{"Level": g.world.Player.Level}, 30, opt)

	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW/2-270, 166)
	drawing.DrawTextByKey(screen, "levelup-instruction", 16, opt)

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
		drawing.DrawTextTemplate(screen, "card-doctor", map[string]any{"Name": doctorNameL(c.Doctor)}, 20, opt)

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
	drawing.DrawText(screen, offerItemText(it), 18, opt)
}

// offerLabel is the prefix shown for a proposal line ("" for nippers).
func offerLabel(k core.OfferKind) string {
	switch k {
	case core.OfferUpgrade:
		return lang.Text("offer-upgrade")
	case core.OfferNippers:
		return ""
	default: // adds
		return lang.Text("offer-add")
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
		return core.JunkImageKey(it.Text), false
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
// tile size and rotation. Weapons are drawn in two layers: a plain tile base,
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

	// Pass 1: tile bases, plus the flat junk device for regular junk. Every
	// weapon/junk tile is a plain base tile with its device drawn on top; weapon
	// barrels and tall junk get their taller sprite in pass 2, regular junk's
	// device is flat and tile-sized so it draws here with its base.
	for _, p := range ps {
		drawTileBase(screen, tr, p.idx, tiles[p.idx], power[p.idx], p.c.X, p.c.Y, size, theta)
	}

	// Pass 2: tall fixtures (weapon barrels + tall junk), drawn after all bases
	// (and in Y order) so a front tile's sprite overlays the tiles behind it.
	//   - Weapon barrels point where they fire (their aim angle) in combat, or the
	//     grid rotation when paused.
	//   - Tall junk (e.g. a pagoda) always points world-up (angle 0).
	for _, p := range ps {
		spriteTheta := 0.0 // tall junk: always world-up
		if wc, isWeapon := tiles[p.idx].Component.(core.WeaponComponent); isWeapon {
			if aimBarrels {
				spriteTheta = wc.Weapon.RenderAngle() + math.Pi/2
			} else {
				spriteTheta = theta
			}
		}
		drawTileFixture(screen, tr, p.idx, tiles[p.idx], power[p.idx], p.c.X, p.c.Y, size, spriteTheta)
	}
}

// drawTileBase draws a turret tile's base plate (plus the flat junk device for
// regular junk) centred at screen (cx, cy), rotated by theta and dimmed if
// unpowered. Shared by the in-game turret and the opening assembly cinematic.
func drawTileBase(screen *ebiten.Image, tr *core.Turret, idx hexmap.Index, tile *core.Tile, power, cx, cy, size, theta float64) {
	key, dim := tileBase(tr, idx, tile, power)
	drawing.DrawSprite(screen, drawing.Image(key), cx, cy, size, size, theta, dim, dim, dim, 1)
	if j, ok := tile.Component.(core.Junk); ok && !j.Tall {
		drawing.DrawSprite(screen, drawing.Image(core.JunkImageKey(j.DeviceName)), cx, cy, size, size, theta, dim, dim, dim, 1)
	}
}

// drawTileFixture draws a tile's tall fixture (weapon barrel or tall junk) if it
// has one: a rectangular sprite authored pointing "up" with its mount tile as the
// bottom TurretTileSize block. It is anchored at that mount-tile centre (cx, cy)
// and scaled by the tile zoom so the sprite's base sits on the socket, rotated by
// spriteTheta. No-op for tiles without a tall fixture.
func drawTileFixture(screen *ebiten.Image, tr *core.Turret, idx hexmap.Index, tile *core.Tile, power, cx, cy, size, spriteTheta float64) {
	key, ok := tallTileSprite(tile.Component)
	if !ok {
		return
	}
	dim := float32(1)
	if power <= 0 && !tr.IsGenerator(idx) {
		dim = 0.5 // unpowered (the generator reads as 0 power but is always live)
	}
	img := drawing.Image(key)
	b := img.Bounds()
	ax := float64(b.Dx()) / 2                     // mount tile is horizontally centred
	ay := float64(b.Dy()) - core.TurretTileSize/2 // ...and is the bottom tile block
	z := size / core.TurretTileSize
	drawing.DrawSpriteAnchored(screen, img, cx, cy, z, spriteTheta, ax, ay, dim, dim, dim, 1)
}

// tallTileSprite returns the tall-sprite image key for a component that is drawn
// as a fixture rising out of its tile (a weapon barrel or a tall junk), and
// whether it has one.
func tallTileSprite(comp core.Component) (key string, ok bool) {
	switch c := comp.(type) {
	case core.WeaponComponent:
		return weaponTileKey(c.Weapon.Kind), true
	case core.Junk:
		if c.Tall {
			return core.JunkImageKey(c.DeviceName), true
		}
	}
	return "", false
}

// drawPause renders the zoomed, upright cut view over a dimmed world: the tank
// and turret blown up so individual tiles can be clicked to cut. The tile under
// the mouse is highlighted.
func (g *InGame) drawPause(screen *ebiten.Image) {
	drawing.DrawRect(screen, 0, 0, screenW, screenH, 0, 0, 0, 0.7)

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW/2-280, 56)
	drawing.DrawTextTemplate(screen, "pause-title", map[string]any{"Nippers": g.world.Player.Nippers}, 24, opt)
	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW/2-280, 96)
	drawing.DrawTextByKey(screen, "pause-help", 16, opt)

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
		name := weaponName(c.Weapon.Kind)
		if c.Weapon.Level > 0 {
			name = fmt.Sprintf("%s  +%d", name, c.Weapon.Level)
		}
		return name, weaponDescL(c.Weapon.Kind), weaponTileKey(c.Weapon.Kind), true
	case core.Junk:
		return junkNameL(c.Name()), junkDescL(c.Name(), c.Tall), core.JunkImageKey(c.DeviceName), c.Tall
	case core.Capacitor:
		return lang.Text("comp-capacitor"), lang.Text("comp-capacitor-desc"), asset.ImgTileCapacitor, false
	default: // plain tile (or empty)
		return lang.Text("comp-wire-name"), lang.Text("comp-wire-desc"), asset.ImgTile, false
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
	if _, ok := tile.Component.(core.Capacitor); ok {
		if power <= 0 {
			return asset.ImgTileCapacitor, 0.45 // dim: disconnected capacitor
		}
		return asset.ImgTileCapacitor, 1
	}
	// Every other tile (weapon, junk, empty) sits on the same plain base tile; the
	// weapon barrel / junk device is layered on top afterwards.
	if power <= 0 {
		return asset.ImgTile, 0.45 // dim: unpowered tile
	}
	return asset.ImgTile, 1
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

// drawScrollBG tiles the layout-sized background image across the screen,
// scrolled by (ox, oy) in screen pixels. The art is authored to wrap seamlessly
// on all four edges, so a 2x2 block of copies always covers the screen with the
// seams falling off-screen. Only DrawImageOptions values are allocated per call
// (no per-frame images), and every copy shares one source image so the draws
// batch.
func drawScrollBG(screen *ebiten.Image, ox, oy float64) {
	img := drawing.Image(asset.ImgBackground)
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw == 0 || ih == 0 {
		return
	}
	tw, th := float64(screenW), float64(screenH)
	// Wrap the scroll into [0, tile) so only the 2x2 block is needed.
	sx := math.Mod(math.Mod(ox, tw)+tw, tw)
	sy := math.Mod(math.Mod(oy, th)+th, th)
	sclX, sclY := tw/float64(iw), th/float64(ih)
	for _, dx := range [2]float64{-sx, -sx + tw} {
		for _, dy := range [2]float64{-sy, -sy + th} {
			opt := &ebiten.DrawImageOptions{}
			opt.Filter = ebiten.FilterNearest
			opt.GeoM.Scale(sclX, sclY)
			opt.GeoM.Translate(dx, dy)
			screen.DrawImage(img, opt)
		}
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

	// HP bar: bottom-centre during play (easier to read than a corner); while
	// paused it falls back to the top-left so it stays clear of the pause view's
	// bottom tile-info panel. A recent hit shakes it (play only).
	hpLeft, hpTop := hpBarPauseX, hpBarPauseY
	if !g.paused {
		hpLeft = (screenW - hpBarW) / 2
		hpTop = screenH - hpBarBottomMargin - hpBarH
		if g.hpShake > 0 {
			frac := float64(g.hpShake) / hpShakeTicks
			amp := hpShakeAmp * frac
			ph := float64(g.hpShake) * hpShakeFreq
			hpLeft += amp * math.Sin(ph)
			hpTop += amp * 0.5 * math.Cos(ph*1.3)
		}
	}
	hp := &drawing.GaugeDrawer{
		Max:           int(p.MaxHP),
		Current:       int(p.HP),
		TopLeft:       geom.PointF{X: hpLeft, Y: hpTop},
		BottomRight:   geom.PointF{X: hpLeft + hpBarW, Y: hpTop + hpBarH},
		TextOffset:    geom.PointF{X: 6, Y: 2},
		FontSize:      16,
		ColorScaleMax: scale(0.2, 0.9, 0.3, 1),
		ColorScaleMin: scale(0.9, 0.2, 0.2, 1),
	}
	hp.Draw(screen)

	// XP bar: a thin full-width strip along the very top edge (Vampire-Survivors
	// style), with the stats line and cut hint just below-left.
	drawing.DrawRect(screen, 0, 0, screenW, xpBarH, 0.2, 0.2, 0.3, 1)
	if p.XPToNext > 0 {
		drawing.DrawRect(screen, 0, 0, screenW*float64(p.XP/p.XPToNext), xpBarH, 0.4, 0.6, 1, 1)
	}

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(hudTextX, hudStatsY)
	drawing.DrawTextTemplate(screen, "hud-stats", map[string]any{
		"Level":   p.Level,
		"Spd":     fmt.Sprintf("%.1f", p.Speed),
		"Fire":    fmt.Sprintf("%.2f", g.world.FireRateMultiplier()),
		"Nippers": p.Nippers,
	}, 18, opt)

	// Cut hint.
	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(hudTextX, hudHintY)
	drawing.DrawTextByKey(screen, "hud-hint", 14, opt)

	opt = &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(screenW-220, 20)
	secs := g.world.Tick / 60
	drawing.DrawTextTemplate(screen, "hud-time-kills", map[string]any{
		"Min":   secs / 60,
		"Sec":   fmt.Sprintf("%02d", secs%60),
		"Kills": g.world.Kills,
	}, 18, opt)
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
	return enemySpriteKeyFor(e.Sprite, e.Kind, e.IsBoss, e.DropsNipper)
}

// enemySpriteKeyFor selects the sprite from the fields that distinguish enemies,
// so the live draw (enemySpriteKey) and the death-fade effect (which only has a
// DeathEvent, not the Enemy) share one mapping. An explicit sprite override (set
// per boss) wins so each boss can use its own art.
func enemySpriteKeyFor(sprite string, kind core.EnemyKind, isBoss, dropsNipper bool) string {
	switch {
	case sprite != "":
		return sprite
	case dropsNipper:
		return asset.ImgCandlestick
	case isBoss:
		return asset.ImgBoss1
	case kind == core.EnemySwarmer:
		return asset.ImgEnemySwarmer
	case kind == core.EnemyBrute:
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
	drawing.DrawText(screen, bossNameL(b.Name), 18, opt)

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
