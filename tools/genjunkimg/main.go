//go:build ignore
// +build ignore

// genjunkimg writes one placeholder PNG per junk device into the given
// directory (default asset/img). Each junk type gets its own file named by
// core.JunkImageKey, so the real art can be dropped in later by overwriting the
// matching file. Flat junk is 24x24 (a tile); tall junk is 24x72 (matching the
// tall-fixture sprites). Placeholders are deliberately plain: a hue unique to
// each device plus a couple of marker cells, just enough to tell them apart.
//
// By default existing files are left untouched so real art is never clobbered;
// pass -force to overwrite.
//
// Run: go run tools/genjunkimg/main.go [-force] asset/img
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/noppikinatta/ebitenginegamejam2026/core"
)

const tileSize = 24

func main() {
	force := flag.Bool("force", false, "overwrite existing files (default: skip files that already exist)")
	flag.Parse()

	outDir := "asset/img"
	if flag.NArg() > 0 {
		outDir = flag.Arg(0)
	}

	names := core.JunkDeviceNames()
	wrote, skipped := 0, 0
	for i, name := range names {
		key := core.JunkImageKey(name)
		tall := core.JunkDeviceTall(name)
		path := filepath.Join(outDir, key+".png")
		if !*force && exists(path) {
			fmt.Printf("skip %s (exists; -force to overwrite)\n", path)
			skipped++
			continue
		}
		img := placeholder(i, len(names), tall)
		if err := writePNG(path, img); err != nil {
			fmt.Fprintln(os.Stderr, "write", path, ":", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%s%s)\n", path, name, tallSuffix(tall))
		wrote++
	}
	fmt.Printf("generated %d junk placeholder images in %s (%d skipped)\n", wrote, outDir, skipped)
}

// exists reports whether a file is already present at path.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func tallSuffix(tall bool) string {
	if tall {
		return ", tall"
	}
	return ""
}

// placeholder builds a simple, visually distinct placeholder for junk index i of
// total. Hue is spread evenly around the wheel so adjacent devices differ; a few
// marker cells encode the index so same-hue collisions still look different.
func placeholder(i, total int, tall bool) *image.RGBA {
	h := tileSize
	if tall {
		h = tileSize * 3
	}
	img := image.NewRGBA(image.Rect(0, 0, tileSize, h))

	hue := float64(i) / float64(max(total, 1))
	body := hsv(hue, 0.55, 0.85)
	border := hsv(hue, 0.65, 0.45)
	marker := hsv(math.Mod(hue+0.5, 1), 0.5, 1)

	for y := 0; y < h; y++ {
		for x := 0; x < tileSize; x++ {
			c := body
			// Tall fixtures taper to a narrower spire above the base tile.
			if tall && y < h-tileSize {
				inset := (h - tileSize - y) / 6
				if x < inset || x >= tileSize-inset {
					img.Set(x, y, color.RGBA{0, 0, 0, 0}) // transparent outside the spire
					continue
				}
			}
			if x == 0 || y == 0 || x == tileSize-1 || y == h-1 {
				c = border
			}
			img.Set(x, y, c)
		}
	}

	// Marker cells in the bottom tile: a 3x3 grid whose lit cells come from the
	// low bits of i, giving each device a small distinct glyph.
	baseY := h - tileSize
	for cell := 0; cell < 9; cell++ {
		if (i>>uint(cell))&1 == 0 {
			continue
		}
		cx := 4 + (cell%3)*6
		cy := baseY + 4 + (cell/3)*6
		fillRect(img, cx, cy, 4, 4, marker)
	}

	return img
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			img.Set(x+dx, y+dy, c)
		}
	}
}

// hsv converts h,s,v in [0,1] to an opaque RGBA color.
func hsv(h, s, v float64) color.RGBA {
	i := math.Floor(h * 6)
	f := h*6 - i
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	var r, g, b float64
	switch int(i) % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
