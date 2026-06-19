//go:build ignore
// +build ignore

// sndpak bundles the loose sound files in a source directory into a single
// obfuscated .pak that the game embeds. Keeping the raw, ready-to-play files out
// of the committed tree (see asset/sound/raw in .gitignore) means licensed
// sound assets are not directly reusable from the public repository, while the
// game still ships working audio in the pak.
//
// Each entry is keyed by the file's base name without extension (e.g. fire.wav
// -> "fire"); asset/sound.go maps those names to Sound values. Supported source
// extensions: .wav, .mp3, .ogg.
//
// Run: go run tools/sndpak/main.go asset/sound/raw asset/sound/sounds.pak
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noppikinatta/ebitenginegamejam2026/sndpak"
)

var supported = map[string]bool{".wav": true, ".mp3": true, ".ogg": true}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run tools/sndpak/main.go <srcDir> <out.pak>")
		os.Exit(2)
	}
	srcDir, outPath := os.Args[1], os.Args[2]

	dirEntries, err := os.ReadDir(srcDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read src dir:", err)
		os.Exit(1)
	}

	var entries []sndpak.Entry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(de.Name()))
		if !supported[ext] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, de.Name()))
		if err != nil {
			fmt.Fprintln(os.Stderr, "read", de.Name(), ":", err)
			os.Exit(1)
		}
		name := strings.TrimSuffix(de.Name(), filepath.Ext(de.Name()))
		entries = append(entries, sndpak.Entry{Name: name, Data: data})
	}

	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "no sound files found in %s (looked for .wav/.mp3/.ogg)\n", srcDir)
		os.Exit(1)
	}

	// Deterministic order so the packed file is stable across runs.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	blob := sndpak.Pack(entries)
	if err := os.WriteFile(outPath, blob, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write pak:", err)
		os.Exit(1)
	}

	fmt.Printf("packed %d sounds into %s (%d bytes):\n", len(entries), outPath, len(blob))
	for _, e := range entries {
		fmt.Printf("  %-12s %6d bytes\n", e.Name, len(e.Data))
	}
}
