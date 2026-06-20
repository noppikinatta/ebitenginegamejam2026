//go:build ignore
// +build ignore

// genprojimg writes one placeholder PNG per junk-emitter projectile sprite into
// the given directory (default asset/img). Each cosmetic junk projectile
// (balloon, coffee, toast, note, duck, firework) gets its own 16x16 file named
// by its core.Sprite* key, so the real art can be dropped in later by
// overwriting the matching file. Placeholders are deliberately plain: a filled
// circle in a hue unique to each projectile.
//
// Run: go run tools/genprojimg/main.go asset/img
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/noppikinatta/ebitenginegamejam2026/core"
)

const projSize = 16

// projSprites pairs each cosmetic projectile sprite key with a placeholder
// colour roughly evoking the device (red balloon, brown coffee, golden toast,
// purple note, yellow duck, magenta firework).
var projSprites = []struct {
	key string
	col color.RGBA
}{
	{core.SpriteBalloon, color.RGBA{0xE0, 0x40, 0x40, 0xFF}},
	{core.SpriteCoffee, color.RGBA{0x6B, 0x40, 0x22, 0xFF}},
	{core.SpriteToast, color.RGBA{0xD9, 0xA0, 0x4A, 0xFF}},
	{core.SpriteNote, color.RGBA{0x7A, 0x5C, 0xD0, 0xFF}},
	{core.SpriteDuck, color.RGBA{0xF2, 0xCC, 0x2E, 0xFF}},
	{core.SpriteFirework, color.RGBA{0xE0, 0x40, 0xC0, 0xFF}},
}

func main() {
	outDir := "asset/img"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	for _, s := range projSprites {
		img := circle(s.col)
		path := filepath.Join(outDir, s.key+".png")
		if err := writePNG(path, img); err != nil {
			fmt.Fprintln(os.Stderr, "write", path, ":", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", path)
	}
	fmt.Printf("generated %d projectile placeholder images in %s\n", len(projSprites), outDir)
}

// circle draws a filled disc of col centred in a projSize square, transparent
// outside, with a slightly darker rim for a touch of contrast.
func circle(col color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, projSize, projSize))
	c := float64(projSize-1) / 2
	r := float64(projSize)/2 - 1
	rim := color.RGBA{col.R / 2, col.G / 2, col.B / 2, 0xFF}
	for y := 0; y < projSize; y++ {
		for x := 0; x < projSize; x++ {
			d := math.Hypot(float64(x)-c, float64(y)-c)
			switch {
			case d <= r-1.5:
				img.Set(x, y, col)
			case d <= r:
				img.Set(x, y, rim)
			default:
				img.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
	return img
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
