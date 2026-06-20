package sndpak

import (
	"bytes"
	"testing"
)

func TestPackUnpackRoundTrip(t *testing.T) {
	entries := []Entry{
		{Name: "fire", Data: []byte("RIFF....WAVEfire payload")},
		{Name: "explosion", Data: bytes.Repeat([]byte{0xAB, 0x00, 0xFF}, 50)},
		{Name: "empty", Data: nil},
	}

	blob := Pack(entries)

	got, err := Unpack(blob)
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	if len(got) != len(entries) {
		t.Fatalf("entry count = %d, want %d", len(got), len(entries))
	}
	for _, e := range entries {
		if !bytes.Equal(got[e.Name], e.Data) {
			t.Errorf("entry %q = %v, want %v", e.Name, got[e.Name], e.Data)
		}
	}
}

func TestObfuscated(t *testing.T) {
	// The original payload (with its RIFF header) must not appear verbatim in the
	// packed bytes — otherwise renaming the file would expose a usable wav.
	raw := []byte("RIFF1234WAVEfmt the quick brown fox jumps over")
	blob := Pack([]Entry{{Name: "bgm", Data: raw}})
	if bytes.Contains(blob, raw) {
		t.Error("packed blob contains the original payload verbatim")
	}
	if bytes.Contains(blob, []byte("RIFF")[:4]) && bytes.Contains(blob[4:], []byte("RIFF")) {
		// (magic is "SPK1"; the only allowed cleartext is the 4-byte header.)
		t.Error("packed blob exposes a RIFF marker")
	}
}

func TestUnpackBadMagic(t *testing.T) {
	if _, err := Unpack([]byte("not a pak")); err == nil {
		t.Error("expected error for bad magic")
	}
}
