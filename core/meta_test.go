package core

import "testing"

// TestEarnedCoins covers the run-end reward kills × (whole minutes + 1).
func TestEarnedCoins(t *testing.T) {
	cases := []struct {
		kills, tick, want int
	}{
		{0, 0, 0},          // no kills → no coins regardless of time
		{10, 0, 10},        // sub-minute run still pays kills (×1)
		{10, 3599, 10},     // 59.98 s is still 0 whole minutes → ×1
		{10, 3600, 20},     // exactly 1 minute → ×2
		{5, 3 * 3600, 20},  // 3 minutes → ×4
		{7, 10 * 3600, 77}, // 10 minutes → ×11
	}
	for _, c := range cases {
		if got := EarnedCoins(c.kills, c.tick); got != c.want {
			t.Errorf("EarnedCoins(%d, %d) = %d, want %d", c.kills, c.tick, got, c.want)
		}
	}
}

// TestMetaStatKeys checks every stat has a unique, non-empty slug and that the
// image key is derived from it.
func TestMetaStatKeys(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range MetaStats {
		k := MetaStatKey(s)
		if k == "" || k == "unknown" {
			t.Errorf("stat %d has no key", s)
		}
		if seen[k] {
			t.Errorf("duplicate stat key %q", k)
		}
		seen[k] = true
		if got, want := MetaStatImageKey(s), "meta_"+k; got != want {
			t.Errorf("MetaStatImageKey = %q, want %q", got, want)
		}
	}
	if len(MetaStats) != int(metaStatCount) {
		t.Errorf("MetaStats has %d entries, want %d", len(MetaStats), metaStatCount)
	}
}
