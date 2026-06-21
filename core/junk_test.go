package core

import "testing"

// TestNewJunk_TallFlag: tall device names get Tall set; ordinary ones don't.
func TestNewJunk_TallFlag(t *testing.T) {
	if !newJunk("Five-storied Pagoda").Tall {
		t.Error("Five-storied Pagoda should be a Tall junk")
	}
	if newJunk("Toaster").Tall {
		t.Error("Toaster should not be a Tall junk")
	}
}

// TestJunkSpecs_Unique: device names in the pool are unique, so localisation
// slugs and random selection are unambiguous.
func TestJunkSpecs_Unique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range junkSpecs {
		if s.Name == "" {
			t.Error("junk spec with empty name")
		}
		if seen[s.Name] {
			t.Errorf("duplicate junk name %q", s.Name)
		}
		seen[s.Name] = true
	}
}
