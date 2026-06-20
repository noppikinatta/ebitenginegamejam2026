package scene

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

	// pauseTileSize is the px-per-hex tile in the zoomed pause/cut view (upright).
	pauseTileSize = 56.0

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
