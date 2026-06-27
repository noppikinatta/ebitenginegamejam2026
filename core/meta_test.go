package core

import "testing"

// TestEarnedCoins covers the run-end reward kills × minutes × (junk + 1),
// truncated to a whole coin.
func TestEarnedCoins(t *testing.T) {
	cases := []struct {
		kills   int
		minutes float64
		junk    int
		want    int
	}{
		{0, 5, 0, 0},      // no kills → no coins
		{10, 0, 0, 0},     // a zero-length run pays nothing
		{10, 1, 0, 10},    // 1 minute, no junk → kills
		{10, 0.5, 1, 10},  // int(10 × 0.5 × 2)
		{300, 3, 2, 2700}, // 300 × 3 × 3
		{5, 3, 4, 75},     // 5 × 3 × 5
		{7, 2.5, 0, 17},   // int(17.5) truncates to 17
	}
	for _, c := range cases {
		if got := EarnedCoins(c.kills, c.minutes, c.junk); got != c.want {
			t.Errorf("EarnedCoins(%d, %g, %d) = %d, want %d", c.kills, c.minutes, c.junk, got, c.want)
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
