package core

import "testing"

// TestNewJunk_TallFlag: the tall device names get Tall set; ordinary ones don't.
func TestNewJunk_TallFlag(t *testing.T) {
	if !newJunk("Sagrada Familia").Tall {
		t.Error("Sagrada Familia should be a Tall junk")
	}
	if newJunk("Toaster").Tall {
		t.Error("Toaster should not be a Tall junk")
	}
	// Every name flagged tall must be in the device pool so it can actually spawn.
	for name := range tallJunkNames {
		found := false
		for _, n := range junkDeviceNames {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tall junk %q is not in junkDeviceNames", name)
		}
	}
}
