package scene

import "github.com/noppikinatta/ebitenginegamejam2026/core"

// metaHolder carries the persistent meta-progression state across scenes for one
// session. Persistence is memory-only by design (it resets on reload), so this is
// just a shared pointer wired into the scenes by CreateSequence: InGame reads it
// to build the run config, Result adds the run's earned coins, and Workshop
// spends them on upgrades.
type metaHolder struct {
	state core.MetaState
}
