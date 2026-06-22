//go:build ignore
// +build ignore

// genbgimg writes a placeholder scrolling-background PNG (asset/img/background.png
// by default) sized to the game layout (1280x720). The art is built to wrap
// seamlessly on all four edges — a faint 80px grid (80 divides both 1280 and
// 720) plus a starfield plotted with toroidal wrap — so the scene can scroll it
// by simply tiling copies. Drop in the real seamless backdrop later by
// overwriting the file. By default an existing file is left untouched so real
// art is never clobbered; pass -force to overwrite.
//
// Run: go run tools/genbgimg/main.go [-force] asset/img/background.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
)

const (
	bgW  = 1280
	bgH  = 720
	grid = 80
)

func main() {
	force := flag.Bool("force", false, "overwrite an existing file (default: skip if it already exists)")
	flag.Parse()

	out := "asset/img/background.png"
	if flag.NArg() > 0 {
		out = flag.Arg(0)
	}

	if !*force {
		if _, err := os.Stat(out); err == nil {
			fmt.Printf("skip %s (exists; -force to overwrite)\n", out)
			return
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, bgW, bgH))
	base := color.RGBA{14, 16, 26, 255}
	for y := 0; y < bgH; y++ {
		for x := 0; x < bgW; x++ {
			img.Set(x, y, base)
		}
	}

	// Faint grid lines on the 80px lattice (wraps cleanly), with brighter nodes
	// at every 4th intersection for a sense of depth.
	line := color.RGBA{28, 32, 48, 255}
	node := color.RGBA{44, 52, 78, 255}
	for y := 0; y < bgH; y++ {
		for x := 0; x < bgW; x++ {
			onV := x%grid == 0
			onH := y%grid == 0
			if onV && onH && (x/grid)%4 == 0 && (y/grid)%4 == 0 {
				plot(img, x, y, node)
			} else if onV || onH {
				img.Set(x, y, line)
			}
		}
	}

	// Starfield: small dots placed with deterministic RNG and plotted with
	// toroidal wrap so none get clipped at the seams.
	rng := rand.New(rand.NewSource(20260620))
	for i := 0; i < 220; i++ {
		x := rng.Intn(bgW)
		y := rng.Intn(bgH)
		b := uint8(120 + rng.Intn(120))
		star := color.RGBA{b, b, uint8(min(255, int(b)+30)), 255}
		plot(img, x, y, star)
		if rng.Intn(3) == 0 { // a few brighter, 4-pixel stars
			plot(img, x+1, y, star)
			plot(img, x, y+1, star)
			plot(img, x-1, y, star)
			plot(img, x, y-1, star)
		}
	}

	f, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

// plot sets a pixel with toroidal wrap so edge-adjacent marks tile seamlessly.
func plot(img *image.RGBA, x, y int, c color.RGBA) {
	x = ((x % bgW) + bgW) % bgW
	y = ((y % bgH) + bgH) % bgH
	img.Set(x, y, c)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
