package scene

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
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

// Opening is the intro cinematic: aliens swarm in, the unarmed tank rolls up
// from the bottom, then doctors bolt tiles onto it one by one — growing the
// ACTUAL run turret outward from the central generator — until it reaches its
// battle-ready state, whereupon it advances to the title. The turret shown is the
// real one the player will fight with: the opening seeds runSeed and builds the
// turret from it, and InGame later rebuilds the same turret from that seed.
type Opening struct {
	input      *ui.Input
	runSeed    *runSeed    // seeded here so InGame fights the turret shown assembling
	nextScene  ebiten.Game // the title
	sequence   *bamenn.Sequence
	transition bamenn.Transition

	t        int // ticks since this play of the cinematic started
	switched bool
	// startAtTitle makes the next entry jump straight to the title state (skipping
	// the cinematic). Set when returning from the workshop so the intro isn't replayed.
	startAtTitle bool
	rng          *rand.Rand
	aliens       []*openingAlien
	bubbles      []openingBubble

	// The run's real turret, plus a center-out ordering of its tiles and their
	// power map, used to animate the assembly and draw each tile faithfully.
	turret *core.Turret
	power  map[hexmap.Index]float64
	order  []hexmap.Index // tile indices sorted center-outward (assembly order)

	// Power meter, computed exactly like the in-game gauge so the cinematic shows
	// the fire-rate multiplier dropping as tiles pile on. powerCurve + bounds come
	// from the run config; powerFill is the smoothed [0,1] bar fill.
	powerCurve       []core.PowerPoint
	fireMin, fireMax float64
	powerFill        float64
}

type openingAlien struct {
	pos, vel geom.PointF
	key      string
	size     float64
}

// openingBubble is a doctor's repeated line popping up during the rapid arm-up.
type openingBubble struct {
	pos  geom.PointF
	born int
}

// The opening cinematic timeline, scroll speed and centre points are tunable in
// scene/tuning.go.

func NewOpening(input *ui.Input, seed *runSeed) *Opening {
	return &Opening{input: input, runSeed: seed}
}

func (o *Opening) Init(nextScene ebiten.Game, sequence *bamenn.Sequence, transition bamenn.Transition) {
	o.nextScene = nextScene
	o.sequence = sequence
	o.transition = transition
}

// SkipToTitle makes the next entry of the opening jump straight to the title
// state, skipping the cinematic. The workshop calls it when the player backs out
// so they don't have to rewatch the intro.
func (o *Opening) SkipToTitle() { o.startAtTitle = true }

// inTitle reports whether the cinematic has finished and the scene is now showing
// the title (assembled tank + title art), waiting for a click to continue.
func (o *Opening) inTitle() bool { return o.t >= o.doneTick() }

// OnStart restarts the cinematic each time the scene is entered and starts the
// title BGM (shared with the title screen, so the music carries over seamlessly).
func (o *Opening) OnStart() {
	o.reset()
	asset.PlayBGM(asset.BGMTitle)
}

func (o *Opening) reset() {
	o.t = 0
	o.switched = false
	o.rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	// Seed the run and build the real turret to show assembling, then hand the
	// seed to InGame so it fights the same turret. Built via NewWorld (the exact
	// in-game construction path) so the shapes are guaranteed identical.
	seed := time.Now().UnixNano()
	if o.runSeed != nil {
		o.runSeed.set(seed)
	}
	cfg := data.NewConfig()
	w := core.NewWorld(seed, cfg)
	o.turret = w.Turret()
	o.power = o.turret.ComputePower()
	o.buildOrder()

	// Capture the power curve and its multiplier bounds so the assembly meter
	// uses the same maths as the in-game gauge.
	o.powerCurve = cfg.PowerCurve
	o.fireMin, o.fireMax = fireRateBounds(cfg.PowerCurve)

	// Returning from the workshop jumps straight to the fully-assembled title so
	// the intro isn't replayed; the meter then reflects the final (diluted) power.
	if o.startAtTitle {
		o.t = o.doneTick()
		o.startAtTitle = false
	}
	o.powerFill = o.powerMeterTarget()

	o.aliens = o.aliens[:0]
	for i := 0; i < 9; i++ {
		key, size := asset.ImgEnemy, 30.0
		switch o.rng.Intn(3) {
		case 1:
			key, size = asset.ImgEnemySwarmer, 24
		case 2:
			key, size = asset.ImgEnemyBrute, 46
		}
		o.aliens = append(o.aliens, &openingAlien{
			pos:  geom.PointF{X: 80 + o.rng.Float64()*(screenW-160), Y: 80 + o.rng.Float64()*(screenH-160)},
			vel:  geom.PointFFromPolar(1.2+o.rng.Float64()*1.8, o.rng.Float64()*2*math.Pi),
			key:  key,
			size: size,
		})
	}
	o.bubbles = o.bubbles[:0]
}

// buildOrder sorts the turret's tile indices center-outward (by hex distance from
// the central generator, breaking ties by screen angle) so the assembly grows
// from the middle out in a stable order, independent of map iteration.
func (o *Opening) buildOrder() {
	o.order = o.order[:0]
	if o.turret == nil {
		return
	}
	origin := hexmap.IdxXY(0, 0)
	for idx := range o.turret.Tiles() {
		o.order = append(o.order, idx)
	}
	sort.Slice(o.order, func(i, j int) bool {
		di, dj := origin.Distance(o.order[i]), origin.Distance(o.order[j])
		if di != dj {
			return di < dj
		}
		axi, ayi := hexLocalOffset(o.order[i], 1)
		axj, ayj := hexLocalOffset(o.order[j], 1)
		return math.Atan2(ayi, axi) < math.Atan2(ayj, axj)
	})
}

// arriveTick is the tick the i-th tile (center-out order) snaps into place. The
// first tile lands at opFirstArrive; then there is an opArmPause beat before the
// doctors pile the rest on in a staggered stream, so the power meter visibly
// holds high and then craters.
func arriveTick(i int) int {
	if i <= 0 {
		return opFirstArrive
	}
	return opFirstArrive + opArmPause + (i-1)*opStagger
}

func (o *Opening) doneTick() int {
	n := len(o.order)
	if n == 0 {
		return opFirstArrive + 90
	}
	return arriveTick(n-1) + 90
}

func (o *Opening) Update() error {
	if o.rng == nil {
		o.reset()
	}
	o.t++

	for _, a := range o.aliens {
		a.pos = a.pos.Add(a.vel)
		if a.pos.X < 40 || a.pos.X > screenW-40 {
			a.vel.X = -a.vel.X
			a.pos.X = clampf(a.pos.X, 40, screenW-40)
		}
		if a.pos.Y < 40 || a.pos.Y > screenH-40 {
			a.vel.Y = -a.vel.Y
			a.pos.Y = clampf(a.pos.Y, 40, screenH-40)
		}
	}

	// Each weapon tile (skipping the central generator) pops a doctor speech bubble
	// as it arrives, so the rapid arm-up reads as a whole crowd of doctors piling on.
	if o.turret != nil {
		tiles := o.turret.Tiles()
		for i := 1; i < len(o.order); i++ {
			if o.t != arriveTick(i) {
				continue
			}
			if _, isWeapon := tiles[o.order[i]].Component.(core.WeaponComponent); !isWeapon {
				continue
			}
			o.bubbles = append(o.bubbles, openingBubble{
				pos:  geom.PointF{X: 130 + o.rng.Float64()*(screenW-260), Y: 130 + o.rng.Float64()*(screenH-380)},
				born: o.t,
			})
		}
	}

	// Ease the power meter toward its current target so it sweeps down smoothly as
	// each pile-on tile lands (mirroring the in-game gauge's easing).
	o.powerFill += (o.powerMeterTarget() - o.powerFill) * 0.18

	leftClick := o.input.Mouse != nil && o.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft)

	// During the aliens scene a click ends it and jumps into the assembly demo
	// (the tank rolls in and the doctors bolt on weapons). The lockout keeps a
	// click carried over from the previous scene from skipping instantly.
	if o.t > 20 && o.t < opAliensEnd && leftClick {
		o.t = opAliensEnd
	}

	// Holding Space skips the cinematic straight to the title state (it no longer
	// jumps scenes — the title is now this scene's terminal phase).
	if !o.inTitle() && o.spaceHeld() >= opSkipHoldTicks {
		o.t = o.doneTick()
	}

	// On the title (assembled tank + title art), a click advances to the next
	// scene (the workshop).
	if o.inTitle() && !o.switched && leftClick {
		o.switched = true
		o.sequence.SwitchWithTransition(o.nextScene, o.transition)
	}
	return nil
}

// spaceHeld returns how many ticks the Space key has been held (0 if released or
// no keyboard), used to require a deliberate hold to skip to the title.
func (o *Opening) spaceHeld() int {
	if o.input.Keyboard == nil {
		return 0
	}
	return o.input.Keyboard.PressDuration(ebiten.KeySpace)
}

func (o *Opening) Draw(screen *ebiten.Image) {
	// Once the cinematic settles, this scene becomes the title: the assembled tank
	// stays on screen under the title art, waiting for a click to continue.
	if o.inTitle() {
		o.drawTitle(screen)
		return
	}

	// Background: held still while the aliens attack, then scrolled top-to-bottom
	// during the launch demo so the screen-stationary tank reads as driving
	// upward (the scenery slides down past it). The scroll matches the in-game
	// camera convention: a tank moving up means a decreasing Y offset.
	var scrollY float64
	if o.t >= opTankStart {
		scrollY = -float64(o.t-opTankStart) * opScrollSpeed
	}
	drawScrollBG(screen, 0, scrollY)

	// A translucent mood tint over the backdrop (ominous red, then battle blue).
	if o.t < opAliensEnd {
		drawing.DrawRect(screen, 0, 0, screenW, screenH, 0.10, 0.05, 0.05, 0.5)
	} else {
		drawing.DrawRect(screen, 0, 0, screenW, screenH, 0.05, 0.06, 0.10, 0.4)
	}

	// Aliens, fading out as the tank rolls in.
	alienA := float32(1)
	if o.t >= opTankStart {
		alienA = float32(clamp01(1 - float64(o.t-opTankStart)/80))
	}
	if alienA > 0 {
		for _, a := range o.aliens {
			drawing.DrawSprite(screen, drawing.Image(a.key), a.pos.X, a.pos.Y, a.size, a.size, 0, alienA, alienA, alienA, alienA)
		}
	}

	if o.t < opAliensEnd {
		a := float32(clamp01(math.Min(float64(o.t)/30, float64(opAliensEnd-o.t)/30)))
		drawTelopC(screen, lang.Text("op-aliens"), opCenterX, 150, 44, 1, 0.5, 0.45, a)
		// Click advances out of the aliens scene; the prompt appears after the lockout.
		if o.t > 20 {
			drawTelopC(screen, lang.Text("op-hint-advance"), opCenterX, screenH-90, 22, 1, 1, 0.8, 0.8)
		}
		o.drawSkipHint(screen)
		return
	}

	// Tank rolls up from the bottom to the centre.
	tankY := 840.0
	if o.t >= opTankEnd {
		tankY = opCenterY
	} else if o.t > opTankStart {
		tankY = lerp(840, opCenterY, smooth(float64(o.t-opTankStart)/(opTankEnd-opTankStart)))
	}
	drawing.DrawSprite(screen, drawing.Image(asset.ImgTank), opCenterX, tankY, tankDrawW*opZoom, tankDrawH*opZoom, 0, 1, 1, 1, 1)

	// The real run turret grows outward from the central generator: each tile
	// flies in from off-screen to its hex slot, staggered in center-out order.
	o.drawAssembly(screen)

	// Power meter: full while the lone first tile sits, then dropping as the
	// doctors pile the rest on — the same gauge the player reads in combat.
	drawPowerGaugeBar(screen, o.powerFill, o.assemblyFireRate())

	// The first doctor's line, then the crowd of repeated bubbles.
	if o.t >= opFirstLine && o.t < opFirstArrive+40 {
		drawTelopC(screen, lang.Text("op-doctor"), opCenterX, 600, 30, 1, 0.95, 0.6, 1)
	}
	for _, b := range o.bubbles {
		a := float32(clamp01(1 - float64(o.t-b.born)/80))
		if a > 0 {
			drawTelopC(screen, lang.Text("op-doctor"), b.pos.X, b.pos.Y, 18, 0.9, 0.9, 1, a)
		}
	}

	o.drawSkipHint(screen)
}

// drawTitle renders the title screen: the real run's fully-assembled tank (kept
// from the cinematic) under the full-screen title art, with a pulsing prompt.
// The title art fills the layout, so dropping in a 1280x720 asset shows 1:1; the
// tank is drawn on top so it always stays visible (reposition via opCenterX/Y).
func (o *Opening) drawTitle(screen *ebiten.Image) {
	drawing.DrawSprite(screen, drawing.Image("title"), screenW/2, screenH/2, screenW, screenH, 0, 1, 1, 1, 1)

	drawing.DrawSprite(screen, drawing.Image(asset.ImgTank), opCenterX, opCenterY, tankDrawW*opZoom, tankDrawH*opZoom, 0, 1, 1, 1, 1)
	o.drawAssembly(screen) // every tile sits at its final slot once inTitle()

	blink := float32(0.7 + 0.3*math.Sin(float64(o.t)*0.12))
	drawTelopC(screen, lang.Text("title-start"), opCenterX, screenH-100, 26, 1, 1, 0.85, blink)
}

// drawSkipHint shows the "hold Space to skip" prompt bottom-right, brightening as
// the hold fills, with a small progress bar so the deliberate hold reads clearly.
func (o *Opening) drawSkipHint(screen *ebiten.Image) {
	held := o.spaceHeld()
	frac := clamp01(float64(held) / float64(opSkipHoldTicks))
	bright := float32(0.6 + 0.4*frac) // text brightens while held

	hint := lang.Text("op-hint-skip")
	w := drawing.MeasureText(hint, 16)
	x := float64(screenW) - w.X - 24
	y := float64(screenH) - 36

	opt := &ebiten.DrawImageOptions{}
	opt.ColorScale.Scale(bright, bright, bright, bright)
	opt.GeoM.Translate(x, y)
	drawing.DrawText(screen, hint, 16, opt)

	// Progress bar under the hint, only while actually holding.
	if held > 0 {
		const bh = 4
		by := y + 24
		drawing.DrawRect(screen, x, by, w.X, bh, 0.3, 0.3, 0.3, 0.6)
		drawing.DrawRect(screen, x, by, w.X*frac, bh, 0.9, 0.9, 0.6, 0.9)
	}
}

func (o *Opening) Layout(outsideWidth, outsideHeight int) (int, int) { return screenW, screenH }

// drawAssembly draws the real run turret mid-assembly: every tile that has begun
// arriving flies in from off-screen to its hex slot, drawn with the same
// tile/barrel art as the in-game turret. Each tile streaks in along the radial
// line through its final position, so pieces converge from every edge; the
// central generator drops straight down from above the tank. Tiles point world-up
// (theta 0), like the paused cut view, for a clear "being built" read.
func (o *Opening) drawAssembly(screen *ebiten.Image) {
	if o.turret == nil {
		return
	}
	center := geom.PointF{X: opCenterX, Y: opCenterY}

	// Current (flight-interpolated) positions of tiles that have started arriving.
	type vis struct {
		idx hexmap.Index
		pos geom.PointF
	}
	vs := make([]vis, 0, len(o.order))
	for i, idx := range o.order {
		arrive := arriveTick(i)
		if o.t < arrive-opFlyDur {
			continue // not started yet
		}
		dx, dy := hexLocalOffset(idx, opTile)
		target := geom.PointF{X: center.X + dx, Y: center.Y + dy}
		// Start off-screen, opFlyIn px out along the direction of the tile's slot.
		// The generator (no offset) has no radial direction, so it drops in from
		// straight above the tank instead.
		ang := math.Atan2(dy, dx)
		if dx == 0 && dy == 0 {
			ang = -math.Pi / 2
		}
		start := center.Add(geom.PointFFromPolar(opFlyIn, ang))
		p := smooth(float64(o.t-(arrive-opFlyDur)) / opFlyDur)
		vs = append(vs, vis{idx: idx, pos: lerpPt(start, target, p)})
	}
	// Paint top-to-bottom so nearer (lower) tiles and their tall fixtures overlay
	// the ones behind, matching the in-game turret's painter order.
	sort.Slice(vs, func(i, j int) bool {
		if vs[i].pos.Y != vs[j].pos.Y {
			return vs[i].pos.Y < vs[j].pos.Y
		}
		return vs[i].pos.X < vs[j].pos.X
	})

	tiles := o.turret.Tiles()
	for _, v := range vs {
		drawTileBase(screen, o.turret, v.idx, tiles[v.idx], o.power[v.idx], v.pos.X, v.pos.Y, opTile, 0)
	}
	for _, v := range vs {
		drawTileFixture(screen, o.turret, v.idx, tiles[v.idx], o.power[v.idx], v.pos.X, v.pos.Y, opTile, 0)
	}
}

// assemblyFireRate computes the fire-rate multiplier for the tiles that have
// landed so far, using the same maths as core's in-combat gauge: the power curve
// evaluated at the connected consumer-tile count, plus any capacitor FireRateAdd.
// Tiles arrive center-out, so the partial set stays connected and the count grows
// as the doctors pile on, dropping the multiplier.
func (o *Opening) assemblyFireRate() float64 {
	if o.turret == nil {
		return 1
	}
	tiles := o.turret.Tiles()
	consumers := 0
	var add float64
	for i, idx := range o.order {
		if o.t < arriveTick(i) {
			continue // not landed yet
		}
		if o.turret.IsGenerator(idx) {
			continue // the generator draws no power and adds no consumer count
		}
		consumers++
		if tile := tiles[idx]; tile != nil && tile.Component != nil {
			add += tile.Component.Mods().FireRateAdd
		}
	}
	return core.PowerMultiplier(o.powerCurve, consumers) + add
}

// powerMeterTarget normalises the current assembly multiplier into the [0,1] bar
// fill between the power curve's bounds (full bar = the curve's max multiplier).
func (o *Opening) powerMeterTarget() float64 {
	span := o.fireMax - o.fireMin
	if span <= 0 {
		return 0
	}
	return clamp01((o.assemblyFireRate() - o.fireMin) / span)
}

// fireRateBounds returns the min and max multiplier a power curve can produce,
// mirroring core's World.FireRateMultBounds for the cinematic gauge.
func fireRateBounds(curve []core.PowerPoint) (min, max float64) {
	if len(curve) == 0 {
		return 1, 1
	}
	min, max = curve[0].Mult, curve[0].Mult
	for _, p := range curve {
		if p.Mult < min {
			min = p.Mult
		}
		if p.Mult > max {
			max = p.Mult
		}
	}
	return min, max
}

// drawTelopC draws txt horizontally centred at cx with its top at y, tinted by
// (r,g,b) and faded by a (premultiplied).
func drawTelopC(screen *ebiten.Image, txt string, cx, y, size float64, r, g, b, a float32) {
	w := drawing.MeasureText(txt, size)
	opt := &ebiten.DrawImageOptions{}
	opt.ColorScale.Scale(r*a, g*a, b*a, a)
	opt.GeoM.Translate(cx-w.X/2, y)
	drawing.DrawText(screen, txt, size, opt)
}

func clamp01(v float64) float64 { return clampf(v, 0, 1) }
func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func lerp(a, b, t float64) float64 { return a + (b-a)*t }
func lerpPt(a, b geom.PointF, t float64) geom.PointF {
	return geom.PointF{X: lerp(a.X, b.X, t), Y: lerp(a.Y, b.Y, t)}
}

// smooth is a smoothstep ease over a t that is clamped to [0,1].
func smooth(t float64) float64 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}
