package lang_test

import (
	"strings"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/lang"
)

// restoreEnglish cycles the global language back to english so a test that
// calls Switch does not leak state into the others. There are a small, fixed
// number of languages, so a bounded loop is enough.
func restoreEnglish(t *testing.T) {
	t.Helper()
	for i := 0; i < 8; i++ {
		if lang.Switch() == "English" {
			return
		}
	}
	t.Fatal("could not restore english language")
}

func TestText_PlainKey(t *testing.T) {
	if got := lang.Text("title-start"); got != "Click to Start" {
		t.Fatalf("title-start: want %q, got %q", "Click to Start", got)
	}
}

func TestText_EscapedNewline(t *testing.T) {
	// story-1 contains a literal \n in the CSV that must be decoded to a real
	// newline by the loader.
	if got := lang.Text("story-1"); !strings.Contains(got, "\n") {
		t.Fatalf("story-1 should contain a newline, got %q", got)
	}
}

func TestText_MissingKey(t *testing.T) {
	got := lang.Text("no-such-key")
	if !strings.HasPrefix(got, "NO_TMPL:") {
		t.Fatalf("missing key should yield NO_TMPL marker, got %q", got)
	}
}

func TestHas(t *testing.T) {
	if !lang.Has("title-start") {
		t.Fatal("expected title-start to exist")
	}
	if lang.Has("no-such-key") {
		t.Fatal("did not expect no-such-key to exist")
	}
}

func TestTextWithDefault(t *testing.T) {
	if got := lang.TextWithDefault("title-start", "fallback"); got != "Click to Start" {
		t.Fatalf("existing key: want %q, got %q", "Click to Start", got)
	}
	if got := lang.TextWithDefault("no-such-key", "fallback"); got != "fallback" {
		t.Fatalf("missing key: want %q, got %q", "fallback", got)
	}
}

func TestExecuteTemplate(t *testing.T) {
	// hud-pwr-mult is "{{.Mult}} MW".
	got := lang.ExecuteTemplate("hud-pwr-mult", map[string]any{"Mult": "2.0"})
	if got != "2.0 MW" {
		t.Fatalf("want %q, got %q", "2.0 MW", got)
	}
}

func TestExecuteTemplate_MissingKey(t *testing.T) {
	got := lang.ExecuteTemplate("no-such-key", map[string]any{"X": 1})
	if !strings.HasPrefix(got, "NO_TMPL:") {
		t.Fatalf("want NO_TMPL marker, got %q", got)
	}
}

func TestSwitch(t *testing.T) {
	defer restoreEnglish(t)

	before := lang.Text("title-start")

	name := lang.Switch()
	if name == "English" {
		t.Fatalf("switching from the default english should change language, got %q", name)
	}
	// The display name is the language capitalised.
	if name[:1] != strings.ToUpper(name[:1]) {
		t.Fatalf("language name should be capitalised, got %q", name)
	}

	after := lang.Text("title-start")
	if after == before {
		t.Fatalf("text should differ after switching language, both %q", after)
	}
}
