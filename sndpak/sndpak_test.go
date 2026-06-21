package sndpak_test

import (
	"bytes"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/sndpak"
)

func TestPackUnpackRoundTrip(t *testing.T) {
	entries := []sndpak.Entry{
		{Name: "fire", Data: []byte("RIFF....WAVEfire payload")},
		{Name: "explosion", Data: bytes.Repeat([]byte{0xAB, 0x00, 0xFF}, 50)},
		{Name: "empty", Data: nil},
	}

	blob := sndpak.Pack(entries)

	got, err := sndpak.Unpack(blob)
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

// TestPackEmpty covers the zero-entry edge case: a pak with no entries must
// round-trip to an empty (non-nil) map.
func TestPackEmpty(t *testing.T) {
	got, err := sndpak.Unpack(sndpak.Pack(nil))
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(got))
	}
}

func TestObfuscated(t *testing.T) {
	// The original payload (with its RIFF header) must not appear verbatim in the
	// packed bytes — otherwise renaming the file would expose a usable wav.
	raw := []byte("RIFF1234WAVEfmt the quick brown fox jumps over")
	blob := sndpak.Pack([]sndpak.Entry{{Name: "bgm", Data: raw}})
	if bytes.Contains(blob, raw) {
		t.Error("packed blob contains the original payload verbatim")
	}
	// The only allowed cleartext is the 4-byte magic header; no RIFF marker may
	// survive in the obfuscated payload.
	if bytes.Contains(blob[4:], []byte("RIFF")) {
		t.Error("packed blob exposes a RIFF marker in its payload")
	}
}

func TestUnpackBadMagic(t *testing.T) {
	if _, err := sndpak.Unpack([]byte("not a pak")); err == nil {
		t.Error("expected error for bad magic")
	}
}

func TestUnpackTooShort(t *testing.T) {
	// Shorter than the 4-byte magic — the length guard must reject it.
	if _, err := sndpak.Unpack([]byte("SPK")); err == nil {
		t.Error("expected error for input shorter than the magic header")
	}
}

// TestUnpackTruncated drives every error branch inside Unpack. The obfuscation
// keystream is positional, so slicing a valid blob still decodes the surviving
// prefix correctly while making the next read run off the end. With a single
// entry {Name:"fire"(4), Data:8 bytes} the plaintext layout is:
//
//	[count=1][nameLen=4][f i r e][dataLen=8][8 data bytes]   (15 payload bytes)
//
// Keeping N payload bytes after the 4-byte magic forces the failure at a known
// field.
func TestUnpackTruncated(t *testing.T) {
	const magicLen = 4
	blob := sndpak.Pack([]sndpak.Entry{
		{Name: "fire", Data: bytes.Repeat([]byte{0x7F}, 8)},
	})

	cases := []struct {
		name string
		keep int // payload bytes retained after the magic header
	}{
		{"count varint missing", 0},  // no payload at all
		{"nameLen varint missing", 1}, // count decoded, then nothing
		{"name truncated", 2},         // nameLen=4 decoded, name bytes missing
		{"dataLen varint missing", 6}, // name consumed, then nothing
		{"data truncated", 7},         // dataLen=8 decoded, data bytes missing
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			trunc := blob[:magicLen+c.keep]
			if _, err := sndpak.Unpack(trunc); err == nil {
				t.Errorf("expected error for truncation keeping %d payload bytes", c.keep)
			}
		})
	}

	// Sanity: the full blob still unpacks.
	if _, err := sndpak.Unpack(blob); err != nil {
		t.Fatalf("full blob should unpack, got %v", err)
	}
}
