package core

import "testing"

// TestEarnedCoins covers the run-end reward: junk tiles + 1.
func TestEarnedCoins(t *testing.T) {
	cases := []struct {
		junk, want int
	}{
		{0, 1},   // a junk-free turret still pays a single coin
		{1, 2},   // one junk → 2
		{5, 6},   // five junk → 6
		{12, 13}, // a bloated turret pays out more
	}
	for _, c := range cases {
		if got := EarnedCoins(c.junk); got != c.want {
			t.Errorf("EarnedCoins(%d) = %d, want %d", c.junk, got, c.want)
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
