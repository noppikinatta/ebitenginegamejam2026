package scene

import (
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/bamenn"
	"github.com/noppikinatta/ebitenginegamejam2026/asset"
	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/drawing"
	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/ui"
)

// Opening is the intro cinematic: aliens swarm in, the unarmed tank rolls up
// from the bottom, then doctors bolt weapons onto it one by one until it reaches
// its battle-ready state, whereupon it advances to the title.
type Opening struct {
	input      *ui.Input
	nextScene  ebiten.Game // the title
	sequence   *bamenn.Sequence
	transition bamenn.Transition

	t        int // ticks since this play of the cinematic started
	switched bool
	rng      *rand.Rand
	aliens   []*openingAlien
	bubbles  []openingBubble
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

// Cinematic timeline (ticks at 60 TPS).
const (
	opAliensEnd   = 200 // aliens telop + swarm
	opTankStart   = 200 // tank starts rolling in
	opTankEnd     = 320 // tank reaches centre
	opFirstLine   = 330 // first doctor line appears
	opFirstArrive = 386 // first weapon snaps into place
	opStagger     = 20  // ticks between successive weapon arrivals
	opFlyDur      = 28  // ticks each weapon spends flying in
	opTile        = 40.0
	opZoom        = 2.2
)

var (
	opCenterX = float64(screenW) / 2
	opCenterY = 430.0
)

// openingWeapons is a decorative battle-ready turret (offsets from the tank
// centre + kind). It is intentionally fixed flavour, independent of the random
// run turret generated later.
var openingWeapons = []struct {
	dx, dy float64
	kind   core.WeaponKind
}{
	{0, -72, core.KindCannon},
	{-66, -40, core.KindSniper},
	{66, -40, core.KindLaser},
	{-86, 10, core.KindShotgun},
	{86, 10, core.KindGatling},
	{-54, 54, core.KindCIWS},
	{54, 54, core.KindMissile},
	{0, 74, core.KindGrenade},
}

const (
	opLineAliens = "エイリアンが攻めてきたぞ！"
	opLineDoctor = "一人では危険じゃ、これを授けよう"
)

func NewOpening(input *ui.Input) *Opening {
	return &Opening{input: input}
}

func (o *Opening) Init(nextScene ebiten.Game, sequence *bamenn.Sequence, transition bamenn.Transition) {
	o.nextScene = nextScene
	o.sequence = sequence
	o.transition = transition
}

// OnStart restarts the cinematic each time the scene is entered.
func (o *Opening) OnStart() { o.reset() }

func (o *Opening) reset() {
	o.t = 0
	o.switched = false
	o.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
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

func (o *Opening) doneTick() int {
	return opFirstArrive + (len(openingWeapons)-1)*opStagger + 90
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

	// Each weapon after the first pops a doctor speech bubble as it arrives, so
	// the rapid arm-up reads as a whole crowd of doctors piling on.
	for i := 1; i < len(openingWeapons); i++ {
		if o.t == opFirstArrive+i*opStagger {
			o.bubbles = append(o.bubbles, openingBubble{
				pos:  geom.PointF{X: 130 + o.rng.Float64()*(screenW-260), Y: 130 + o.rng.Float64()*(screenH-380)},
				born: o.t,
			})
		}
	}

	// Click skips ahead (after a short lockout so a click carried over from the
	// previous scene doesn't skip instantly); otherwise advance once assembly settles.
	skip := o.t > 20 && o.input.Mouse.IsJustPressed(ebiten.MouseButtonLeft)
	if !o.switched && (o.t >= o.doneTick() || skip) {
		o.switched = true
		o.sequence.SwitchWithTransition(o.nextScene, o.transition)
	}
	return nil
}

func (o *Opening) Draw(screen *ebiten.Image) {
	if o.t < opAliensEnd {
		screen.Fill(color.RGBA{26, 12, 14, 255}) // ominous red
	} else {
		screen.Fill(color.RGBA{12, 14, 22, 255}) // battle blue
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
		drawTelopC(screen, opLineAliens, opCenterX, 150, 44, 1, 0.5, 0.45, a)
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

	// Weapons fly in from outside to their mount, base sockets telegraphing the spot.
	for i, w := range openingWeapons {
		arrive := opFirstArrive + i*opStagger
		if o.t < arrive-opFlyDur {
			continue
		}
		target := geom.PointF{X: opCenterX + w.dx, Y: opCenterY + w.dy}
		ang := math.Atan2(w.dy, w.dx) + math.Pi/2 // barrel points outward from the tank
		drawing.DrawSprite(screen, drawing.Image(asset.ImgTileWire), target.X, target.Y, opTile, opTile, 0, 1, 1, 1, 1)

		pos := target
		if o.t < arrive {
			outward := geom.PointFFromPolar(1100, math.Atan2(w.dy, w.dx))
			pos = lerpPt(target.Add(outward), target, smooth(float64(o.t-(arrive-opFlyDur))/opFlyDur))
		}
		drawOpeningBarrel(screen, w.kind, pos.X, pos.Y, ang)
	}

	// The first doctor's line, then the crowd of repeated bubbles.
	if o.t >= opFirstLine && o.t < opFirstArrive+40 {
		drawTelopC(screen, opLineDoctor, opCenterX, 600, 30, 1, 0.95, 0.6, 1)
	}
	for _, b := range o.bubbles {
		a := float32(clamp01(1 - float64(o.t-b.born)/80))
		if a > 0 {
			drawTelopC(screen, opLineDoctor, b.pos.X, b.pos.Y, 18, 0.9, 0.9, 1, a)
		}
	}

	o.drawSkipHint(screen)
}

func (o *Opening) drawSkipHint(screen *ebiten.Image) {
	opt := &ebiten.DrawImageOptions{}
	opt.ColorScale.Scale(0.6, 0.6, 0.6, 0.6)
	opt.GeoM.Translate(screenW-180, screenH-36)
	drawing.DrawText(screen, "Click to skip", 16, opt)
}

func (o *Opening) Layout(outsideWidth, outsideHeight int) (int, int) { return screenW, screenH }

// drawOpeningBarrel draws a weapon barrel anchored at its mount-tile centre,
// the same way the in-game turret does, but at a fixed angle for the cinematic.
func drawOpeningBarrel(screen *ebiten.Image, kind core.WeaponKind, cx, cy, angle float64) {
	img := drawing.Image(weaponTileKey(kind))
	b := img.Bounds()
	ax := float64(b.Dx()) / 2
	ay := float64(b.Dy()) - core.TurretTileSize/2
	drawing.DrawSpriteAnchored(screen, img, cx, cy, opTile/core.TurretTileSize, angle, ax, ay, 1, 1, 1, 1)
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
