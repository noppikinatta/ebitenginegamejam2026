//go:build ignore
// +build ignore

// genprojimg writes one placeholder PNG per projectile sprite into the given
// directory (default asset/img): the cosmetic junk-emitter projectiles (balloon,
// coffee, toast, note, duck, firework) and the per-weapon bullets (cannon,
// shotgun, sniper, gatling, grenade, ciws, missile). Each gets its own file
// named by its core.Sprite* key so the real art can be dropped in later by
// overwriting the matching file. Placeholders are deliberately plain: round
// bullets are filled discs, elongated bullets (cannon/sniper/missile, authored
// pointing UP) are vertical capsules so their travel-facing rotation reads.
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

// projSprite pairs a sprite key with its placeholder colour, pixel footprint and
// shape. Elongated weapon bullets are authored pointing up (taller than wide).
type projSprite struct {
	key     string
	col     color.RGBA
	w, h    int
	capsule bool // true: vertical capsule (elongated, upright); false: disc
}

// projSprites lists every projectile placeholder. Junk emitters keep their 16×16
// discs; weapon bullets use sizes matching data/weapon.go's ProjDrawW/ProjDrawH.
var projSprites = []projSprite{
	// Junk-emitter cosmetic projectiles (discs).
	{key: core.SpriteBalloon, col: color.RGBA{0xE0, 0x40, 0x40, 0xFF}, w: 16, h: 16},
	{key: core.SpriteCoffee, col: color.RGBA{0x6B, 0x40, 0x22, 0xFF}, w: 16, h: 16},
	{key: core.SpriteToast, col: color.RGBA{0xD9, 0xA0, 0x4A, 0xFF}, w: 16, h: 16},
	{key: core.SpriteNote, col: color.RGBA{0x7A, 0x5C, 0xD0, 0xFF}, w: 16, h: 16},
	{key: core.SpriteDuck, col: color.RGBA{0xF2, 0xCC, 0x2E, 0xFF}, w: 16, h: 16},
	{key: core.SpriteFirework, col: color.RGBA{0xE0, 0x40, 0xC0, 0xFF}, w: 16, h: 16},
	// Weapon bullets. Round ones are discs; cannon/sniper/missile are upright
	// capsules (elongated, drawn rotated to face travel in-game).
	{key: core.SpriteCannon, col: color.RGBA{0xF0, 0xC0, 0x40, 0xFF}, w: 8, h: 14, capsule: true},
	{key: core.SpriteShotgun, col: color.RGBA{0xF0, 0x80, 0x30, 0xFF}, w: 6, h: 6},
	{key: core.SpriteSniper, col: color.RGBA{0x60, 0xE0, 0xF0, 0xFF}, w: 4, h: 16, capsule: true},
	{key: core.SpriteGatling, col: color.RGBA{0xF0, 0xE0, 0x60, 0xFF}, w: 6, h: 6},
	{key: core.SpriteGrenade, col: color.RGBA{0x70, 0x90, 0x50, 0xFF}, w: 14, h: 14},
	{key: core.SpriteCIWS, col: color.RGBA{0xF0, 0x50, 0x50, 0xFF}, w: 6, h: 6},
	{key: core.SpriteMissile, col: color.RGBA{0xD0, 0xD0, 0xD8, 0xFF}, w: 8, h: 12, capsule: true},
}

func main() {
	outDir := "asset/img"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	for _, s := range projSprites {
		var img *image.RGBA
		if s.capsule {
			img = capsule(s.col, s.w, s.h)
		} else {
			img = disc(s.col, s.w, s.h)
		}
		path := filepath.Join(outDir, s.key+".png")
		if err := writePNG(path, img); err != nil {
			fmt.Fprintln(os.Stderr, "write", path, ":", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", path)
	}
	fmt.Printf("generated %d projectile placeholder images in %s\n", len(projSprites), outDir)
}

// rim returns a darker shade of col for a thin contrast edge.
func rim(col color.RGBA) color.RGBA {
	return color.RGBA{col.R / 2, col.G / 2, col.B / 2, 0xFF}
}

// disc draws a filled ellipse of col filling a w×h box, transparent outside,
// with a darker rim.
func disc(col color.RGBA, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w-1)/2, float64(h-1)/2
	rx, ry := float64(w)/2-0.5, float64(h)/2-0.5
	edge := rim(col)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			nx := (float64(x) - cx) / rx
			ny := (float64(y) - cy) / ry
			d := math.Hypot(nx, ny)
			switch {
			case d <= 0.78:
				img.Set(x, y, col)
			case d <= 1.0:
				img.Set(x, y, edge)
			default:
				img.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
	return img
}

// capsule draws an upright stadium shape (rectangle with semicircular caps) of
// col filling a w×h box: a placeholder for elongated, travel-facing bullets.
func capsule(col color.RGBA, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx := float64(w-1) / 2
	r := float64(w) / 2 // cap radius = half width
	topCap := r         // centre of top semicircle
	botCap := float64(h-1) - r
	edge := rim(col)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x), float64(y)
			var d float64 // distance from the capsule's spine, in units of r
			switch {
			case fy < topCap:
				d = math.Hypot(fx-cx, fy-topCap) / r
			case fy > botCap:
				d = math.Hypot(fx-cx, fy-botCap) / r
			default:
				d = math.Abs(fx-cx) / r
			}
			switch {
			case d <= 0.7:
				img.Set(x, y, col)
			case d <= 1.0:
				img.Set(x, y, edge)
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
