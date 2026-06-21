package lang_test

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/lang"
)

// TestCSVKeyParity guards against translation drift: every key defined in one
// language CSV must exist in all the others, so no string falls back to a
// NO_TMPL marker when the player switches language.
func TestCSVKeyParity(t *testing.T) {
	data := lang.LoadTemplates()
	if len(data) < 2 {
		t.Fatalf("expected at least 2 languages, got %d", len(data))
	}

	// Use english as the reference key set.
	ref, ok := data["english"]
	if !ok {
		t.Fatal("english language not loaded")
	}

	for lang, dict := range data {
		for k := range ref {
			if _, ok := dict[k]; !ok {
				t.Errorf("language %q is missing key %q", lang, k)
			}
		}
		for k := range dict {
			if _, ok := ref[k]; !ok {
				t.Errorf("language %q has key %q not present in english", lang, k)
			}
		}
	}
}
