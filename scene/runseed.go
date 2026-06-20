package scene

import "time"

// runSeed carries the RNG seed for the upcoming run from the Opening cinematic to
// the InGame scene, so the turret shown assembling in the opening is the exact
// turret the player then fights with (same seed + same config => same turret;
// core.NewWorld consumes the RNG for turret generation first, so no data needs to
// cross scenes — only this one int64).
//
// pending is set by the Opening and consumed by InGame. When it is not pending
// (a retry straight from the result screen, which never passes through the
// opening) InGame rolls a fresh time-based seed instead, giving a new turret.
type runSeed struct {
	seed    int64
	pending bool
}

// set records the seed the opening generated its turret from and marks it for the
// next InGame entry to consume.
func (r *runSeed) set(seed int64) {
	r.seed = seed
	r.pending = true
}

// take returns the seed for a new run: the pending opening seed if one is waiting
// (clearing it so a later retry rolls fresh), otherwise a new time-based seed.
func (r *runSeed) take() int64 {
	if r.pending {
		r.pending = false
		return r.seed
	}
	return time.Now().UnixNano()
}
