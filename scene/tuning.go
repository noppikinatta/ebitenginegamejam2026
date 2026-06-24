package scene

import "math"

// This file is the single place to tune the SCENE (Ebiten) layer's adjustable
// numbers: rendering resolution, layout geometry, animation feel and scroll
// speeds. It is the presentation-layer counterpart to the `data` package, which
// owns core's gameplay balance — those live in `data` because they feed
// core.Config (Ebiten-free), whereas these only affect drawing/input here, so
// they stay in scene rather than leaking presentation concerns into data.
//
// To retune the look/feel, edit the values here. Constants that are NOT free
// knobs (e.g. combatTileSize, which must equal core.TurretTileSize so the
// miniature lines up with the muzzle world offsets) deliberately stay next to
// their usage instead of moving here.

// --- Rendering resolution -------------------------------------------------
// The internal layout size every scene reports from Layout(); the window is
// sized to match in app/main.go.
const (
	screenW = 1280
	screenH = 720
)

// --- Scrolling background -------------------------------------------------
const (
	// bgScrollMul scales how fast the in-game background scrolls relative to the
	// camera. 1.0 locks the scenery 1:1 to the world; <1 gives a parallax drift,
	// >1 exaggerates the sense of speed.
	bgScrollMul = 1.0

	// opScrollSpeed is the opening launch-demo background scroll in px/tick. The
	// backdrop slides top-to-bottom so the screen-stationary tank reads as
	// driving upward.
	opScrollSpeed = 2.4
)

// --- In-game world rendering ----------------------------------------------
const (
	gridGap = 64 // spacing of the faint world reference grid (px)

	// pauseTileSize is the px-per-hex tile in the upright pause/cut view. Kept at
	// combatTileSize (1:1) so even a large turret stays fully on screen and every
	// edge tile remains clickable to cut.
	pauseTileSize = combatTileSize

	// Tank sprite draw footprint (px at the 1:1 camera); independent of the core
	// collision radius. The tank is authored tall (portrait).
	tankDrawW = 48.0
	tankDrawH = 64.0
)

// --- Level-up doctor-card layout ------------------------------------------
const (
	cardW   = 360.0
	cardH   = 300.0
	cardGap = 28.0
	cardY   = 210.0
)

// --- XP bar + stats -------------------------------------------------------
// The XP bar is a thin full-width strip pinned to the very top edge (Vampire
// Survivors style); the stats line and cut hint sit just below it, left-aligned.
const (
	xpBarH    = 6.0  // thin bar height (px)
	hudTextX  = 20.0 // left margin for the stats line + cut hint
	hudStatsY = 12.0 // stats line baseline-top, just under the XP bar
	hudHintY  = 36.0 // cut hint, below the stats line
)

// --- Enemy death fade-out -------------------------------------------------
// When an enemy dies the scene leaves a fading sprite where it fell (spawned
// from world.DeathEvents), swelling slightly as it dissolves.
const (
	deathFadeTicks = 14   // ticks the dying sprite fades over (~0.23 s)
	deathGrow      = 1.15 // sprite scale reached by the end of the fade
)

// --- HP bar ---------------------------------------------------------------
// During play the HP bar sits bottom-centre (easier to read than a corner);
// while paused it falls back to the top-left corner so it stays clear of the
// pause view's bottom tile-info panel. A recent hit shakes it.
const (
	hpBarW            = 320.0 // bar width (px)
	hpBarH            = 22.0  // bar height (px)
	hpBarBottomMargin = 30.0  // gap from the screen bottom to the bar's bottom
	hpBarPauseX       = 20.0  // top-left position used while paused
	hpBarPauseY       = 20.0

	hpShakeTicks = 18  // how long the shake lasts after a hit
	hpShakeAmp   = 7.0 // peak shake amplitude (px)
	hpShakeFreq  = 1.1 // oscillation rate (radians/tick)
)

// --- Left-edge power gauge geometry ---------------------------------------
const (
	powerGaugeX      = 24.0
	powerGaugeW      = 22.0
	powerGaugeTop    = 132.0
	powerGaugeBottom = screenH - 40.0
)

// --- Opening cinematic timeline (ticks at 60 TPS) -------------------------
const (
	opAliensEnd   = 200 // aliens telop + swarm
	opTankStart   = 200 // tank starts rolling in (and the backdrop starts scrolling)
	opTankEnd     = 320 // tank reaches centre
	opFirstLine   = 330 // first doctor line appears
	opFirstArrive = 386 // first tile (the generator) snaps into place
	opArmPause    = 150 // pause after the first tile before the doctors pile the rest on (dramatizes the power drop)
	opStagger     = 12  // ticks between successive tile arrivals (turret has ~22 tiles)
	opFlyDur      = 40  // ticks each tile spends flying in from off-screen
	opFlyIn       = 950 // distance (px) off-screen each tile starts from before flying in
	// opTile / opZoom keep the cinematic at the SAME scale as the in-game combat
	// view so the assembled tank on the title matches the run and the transition is
	// seamless: opTile mirrors combatTileSize (the turret miniature's hex spacing)
	// and opZoom is 1.0 (the tank is drawn at its 1:1 in-game footprint). Raise
	// opZoom for a deliberately zoomed intro, at the cost of that seamlessness.
	opTile = combatTileSize
	opZoom = 1.0
	// opSkipHoldTicks is how long Space must be held to skip the whole opening to
	// the title (~1s at 60 TPS). A click only advances past the aliens scene.
	opSkipHoldTicks = 60
)

// opCenterX / opCenterY are where the tank settles in the cinematic and sits on
// the title. They match the in-game camera (which keeps the player at the screen
// centre), so the assembled tank lines up 1:1 with the run that starts on click.
var (
	opCenterX = float64(screenW) / 2
	opCenterY = float64(screenH) / 2
)

// --- Floating damage numbers ----------------------------------------------
// Numbers spawn at the hit point, dart a short distance along a random
// direction within an upward fan (so multi-hits scatter instead of stacking),
// hold, then fade out.
const (
	dmgFontSize  = 16          // roughly the HUD power-label size (14)
	dmgRise      = 24.0        // travel distance from the hit point (px)
	dmgRiseTicks = 8.0         // ticks to reach full travel (quick dart)
	dmgHoldTicks = 110         // ticks held in place before fading (~1.8 s)
	dmgFadeTicks = 30          // fade-out duration in ticks (~0.5 s)
	dmgFanRad    = math.Pi / 2 // total spread of the upward launch fan (90°)
)

// dmgEnemyRGB / dmgPlayerRGB tint the (white) glyph cache: enemies show white,
// the tank shows red.
var (
	dmgEnemyRGB  = [3]float32{1, 1, 1}
	dmgPlayerRGB = [3]float32{1, 0.25, 0.2}
)
