//go:build ignore
// +build ignore

// psd2png reads every .psd file in a directory and exports each layer as a
// separate transparent PNG. Output files are named after the source PSD plus a
// zero-padded sequence number, e.g. character.psd -> character_00.png,
// character_01.png, ... The sequence counts the exported layers in
// depth-first order; folder (group) layers carry no pixels and are skipped.
//
// Run:
//
//	go run tools/psd2png/main.go <input-dir> [output-dir]
//
// input-dir defaults to ".", output-dir defaults to input-dir.
package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/oov/psd"
)

func main() {
	inDir := "."
	if len(os.Args) > 1 {
		inDir = os.Args[1]
	}
	outDir := inDir
	if len(os.Args) > 2 {
		outDir = os.Args[2]
	}

	if err := run(inDir, outDir); err != nil {
		fmt.Fprintln(os.Stderr, "psd2png:", err)
		os.Exit(1)
	}
}

func run(inDir, outDir string) error {
	entries, err := os.ReadDir(inDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	found := 0
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".psd") {
			continue
		}
		found++
		if err := exportPSD(filepath.Join(inDir, e.Name()), outDir); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
	}
	if found == 0 {
		fmt.Printf("no .psd files found in %s\n", inDir)
	}
	return nil
}

func exportPSD(path, outDir string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	doc, _, err := psd.Decode(f, nil)
	if err != nil {
		return err
	}

	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	seq := 0
	if err := walk(doc.Layer, doc.Config.Rect, base, outDir, &seq); err != nil {
		return err
	}
	fmt.Printf("%s: exported %d layer(s)\n", filepath.Base(path), seq)
	return nil
}

// walk traverses the layer tree depth-first, writing one PNG per layer that
// carries pixels. Each PNG is sized to the full canvas (canvas) with the layer
// composited at its position, so every exported asset keeps the same
// dimensions and relative placement. Folders (groups) only descend into their
// children.
func walk(layers []psd.Layer, canvas image.Rectangle, base, outDir string, seq *int) error {
	for i := range layers {
		l := &layers[i]
		if l.HasImage() && l.Picker != nil && !l.Rect.Empty() {
			out := filepath.Join(outDir, fmt.Sprintf("%s_%02d.png", base, *seq))
			if err := writePNG(out, composeOnCanvas(canvas, l)); err != nil {
				return err
			}
			*seq++
		}
		if len(l.Layer) > 0 {
			if err := walk(l.Layer, canvas, base, outDir, seq); err != nil {
				return err
			}
		}
	}
	return nil
}

// composeOnCanvas returns a fully transparent image the size of the canvas with
// the layer's cropped pixels (l.Picker) drawn at its canvas position (l.Rect).
// The layer's Picker only holds the minimal non-empty rectangle, so without
// this the exported PNG would be cropped to that rectangle and lose both its
// surrounding transparency and its placement. draw.Draw clips automatically if
// the layer extends past the canvas bounds.
func composeOnCanvas(canvas image.Rectangle, l *psd.Layer) image.Image {
	dst := image.NewNRGBA(canvas)
	draw.Draw(dst, l.Rect, l.Picker, l.Picker.Bounds().Min, draw.Src)
	return dst
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return nil
}
