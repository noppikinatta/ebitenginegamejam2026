//go:build ignore
// +build ignore

// sndpak bundles sound effects into a single obfuscated .pak that the game
// embeds. Keeping the raw, ready-to-play files out of the committed tree (see
// asset/sound/raw in .gitignore) means licensed SE are not directly reusable
// from the public repository, while the game still ships working audio.
//
// MERGE behaviour: by default the existing output pak (if any) is loaded as the
// base, and the files found in <srcDir> OVERRIDE entries with the same name. So
// to swap one effect you only need to drop that single file into asset/sound/raw
// and repack — the others are preserved from the committed pak, even on a fresh
// clone where raw/ is otherwise empty. Pass -rebuild to ignore the existing pak
// and build solely from <srcDir> (the way to drop an effect: remove it from
// raw/ and rebuild).
//
// Each entry is keyed by the file's base name without extension (e.g. fire.wav
// -> "fire"); asset/sound.go maps those names to Sound values. Supported source
// extensions: .wav, .mp3, .ogg.
//
// Run: go run tools/sndpak/main.go asset/sound/raw asset/sound/se.pak
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noppikinatta/ebitenginegamejam2026/sndpak"
)

var supported = map[string]bool{".wav": true, ".mp3": true, ".ogg": true}

func main() {
	rebuild := flag.Bool("rebuild", false, "ignore the existing pak and build only from srcDir")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: go run tools/sndpak/main.go [-rebuild] <srcDir> <out.pak>")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	srcDir, outPath := flag.Arg(0), flag.Arg(1)

	// Base entries: start from the existing pak so unchanged effects survive a
	// partial repack (unless -rebuild asks for a clean build).
	base := map[string][]byte{}
	baseNames := map[string]bool{}
	if !*rebuild {
		if raw, err := os.ReadFile(outPath); err == nil {
			unpacked, err := sndpak.Unpack(raw)
			if err != nil {
				fmt.Fprintf(os.Stderr, "existing pak %s is unreadable: %v\n", outPath, err)
				os.Exit(1)
			}
			for name, data := range unpacked {
				base[name] = data
				baseNames[name] = true
			}
		}
	}

	// Overlay the loose source files; same-named entries override the base.
	replaced := map[string]bool{}
	added := map[string]bool{}
	dirEntries, err := os.ReadDir(srcDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read src dir:", err)
		os.Exit(1)
	}
	for _, de := range dirEntries {
		if de.IsDir() || !supported[strings.ToLower(filepath.Ext(de.Name()))] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, de.Name()))
		if err != nil {
			fmt.Fprintln(os.Stderr, "read", de.Name(), ":", err)
			os.Exit(1)
		}
		name := strings.TrimSuffix(de.Name(), filepath.Ext(de.Name()))
		if baseNames[name] {
			replaced[name] = true
		} else {
			added[name] = true
		}
		base[name] = data
	}

	if len(base) == 0 {
		fmt.Fprintf(os.Stderr, "nothing to pack: no existing pak and no sources in %s (.wav/.mp3/.ogg)\n", srcDir)
		os.Exit(1)
	}

	// Deterministic order so the packed file is stable across runs.
	names := make([]string, 0, len(base))
	for name := range base {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]sndpak.Entry, 0, len(names))
	for _, name := range names {
		entries = append(entries, sndpak.Entry{Name: name, Data: base[name]})
	}

	blob := sndpak.Pack(entries)
	if err := os.WriteFile(outPath, blob, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write pak:", err)
		os.Exit(1)
	}

	fmt.Printf("packed %d sounds into %s (%d bytes):\n", len(entries), outPath, len(blob))
	for _, e := range entries {
		fmt.Printf("  %-12s %7d bytes  [%s]\n", e.Name, len(e.Data), source(e.Name, replaced, added))
	}
}

// source labels where a final entry came from, for the summary line.
func source(name string, replaced, added map[string]bool) string {
	switch {
	case added[name]:
		return "added from raw"
	case replaced[name]:
		return "replaced from raw"
	default:
		return "kept from pak"
	}
}
