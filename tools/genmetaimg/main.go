//go:build ignore
// +build ignore

// genmetaimg writes placeholder art for the persistent-upgrade (workshop) UI
// into the given directory (default asset/img):
//   - one 24x24 icon per meta stat, named core.MetaStatImageKey (meta_hp.png …)
//   - one 16x16 coin sprite, coin.png
//
// Each file is keyed so the real art can be dropped in later by overwriting the
// matching file. Placeholders are deliberately plain: a hue unique to each stat
// plus a couple of marker cells. By default existing files are left untouched so
// real art is never clobbered; pass -force to overwrite.
//
// Run: go run tools/genmetaimg/main.go [-force] asset/img
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

const (
	iconSize = 24
	coinSize = 16
)

func main() {
	force := flag.Bool("force", false, "overwrite existing files (default: skip files that already exist)")
	flag.Parse()

	outDir := "asset/img"
	if flag.NArg() > 0 {
		outDir = flag.Arg(0)
	}

	wrote, skipped := 0, 0
	stats := core.MetaStats
	for i, s := range stats {
		key := core.MetaStatImageKey(s)
		if writeIfAbsent(filepath.Join(outDir, key+".png"), *force, func() image.Image {
			return statIcon(i, len(stats))
		}, &wrote, &skipped) {
			fmt.Printf("wrote %s (%s)\n", key+".png", core.MetaStatKey(s))
		}
	}

	if writeIfAbsent(filepath.Join(outDir, "coin.png"), *force, coinSprite, &wrote, &skipped) {
		fmt.Println("wrote coin.png")
	}

	fmt.Printf("generated %d meta placeholder images in %s (%d skipped)\n", wrote, outDir, skipped)
}

// writeIfAbsent writes the image produced by make to path unless it exists and
// force is false. Returns true when a file was written.
func writeIfAbsent(path string, force bool, make func() image.Image, wrote, skipped *int) bool {
	if !force && exists(path) {
		fmt.Printf("skip %s (exists; -force to overwrite)\n", path)
		*skipped++
		return false
	}
	if err := writePNG(path, make()); err != nil {
		fmt.Fprintln(os.Stderr, "write", path, ":", err)
		os.Exit(1)
	}
	*wrote++
	return true
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// statIcon builds a 24x24 placeholder for stat index i of total: a hue-tinted
// rounded body with a bordered edge and a small index glyph, distinct per stat.
func statIcon(i, total int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	hue := float64(i) / float64(maxInt(total, 1))
	body := hsv(hue, 0.5, 0.85)
	border := hsv(hue, 0.6, 0.45)
	marker := hsv(math.Mod(hue+0.5, 1), 0.45, 1)

	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			// Knock out the four corners so icons read as rounded chips.
			if corner(x, y, iconSize) {
				img.Set(x, y, color.RGBA{})
				continue
			}
			c := body
			if x <= 1 || y <= 1 || x >= iconSize-2 || y >= iconSize-2 {
				c = border
			}
			img.Set(x, y, c)
		}
	}

	// A 3x3 glyph encoding the low bits of i, so same-hue collisions still differ.
	for cell := 0; cell < 9; cell++ {
		if (i>>uint(cell))&1 == 0 {
			continue
		}
		fillRect(img, 5+(cell%3)*5, 5+(cell/3)*5, 3, 3, marker)
	}
	return img
}

// coinSprite builds a 16x16 gold disc with a lighter highlight.
func coinSprite() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, coinSize, coinSize))
	gold := color.RGBA{0xF4, 0xC4, 0x30, 0xFF}
	edge := color.RGBA{0xB8, 0x86, 0x0B, 0xFF}
	shine := color.RGBA{0xFF, 0xEE, 0x99, 0xFF}
	c := (coinSize - 1) / 2.0
	rOuter := coinSize/2.0 - 0.5
	for y := 0; y < coinSize; y++ {
		for x := 0; x < coinSize; x++ {
			dx, dy := float64(x)-c, float64(y)-c
			d := math.Hypot(dx, dy)
			switch {
			case d > rOuter:
				img.Set(x, y, color.RGBA{})
			case d > rOuter-1.5:
				img.Set(x, y, edge)
			case dx+dy < -3: // upper-left highlight
				img.Set(x, y, shine)
			default:
				img.Set(x, y, gold)
			}
		}
	}
	return img
}

// corner reports whether (x,y) is in one of the 2px clipped corners of a
// size×size tile, so the icons read as rounded chips.
func corner(x, y, size int) bool {
	const m = 2
	return (x < m && y < m && x+y < m) ||
		(x >= size-m && y < m && (size-1-x)+y < m) ||
		(x < m && y >= size-m && x+(size-1-y) < m) ||
		(x >= size-m && y >= size-m && (size-1-x)+(size-1-y) < m)
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

func maxInt(a, b int) int {
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
