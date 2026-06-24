//go:build ignore
// +build ignore

// genmagnetimg writes a 16x16 placeholder for the magnet pickup (magnet.png)
// into the given directory (default asset/img). The magnet pickup is dropped by
// the mid-bosses; collecting it magnetizes every gem and pickup on the field.
//
// The placeholder is a simple red horseshoe magnet with steel poles, sized to
// match the nipper/heart pickups. Drop in real art later by overwriting the
// file. By default an existing file is left untouched; pass -force to overwrite.
//
// Run: go run tools/genmagnetimg/main.go [-force] asset/img
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
)

const magnetSize = 16

func main() {
	force := flag.Bool("force", false, "overwrite existing file (default: skip if it already exists)")
	flag.Parse()

	outDir := "asset/img"
	if flag.NArg() > 0 {
		outDir = flag.Arg(0)
	}

	path := filepath.Join(outDir, "magnet.png")
	if !*force {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("skip %s (exists; -force to overwrite)\n", path)
			return
		}
	}
	if err := writePNG(path, magnetSprite()); err != nil {
		fmt.Fprintln(os.Stderr, "write", path, ":", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", path)
}

// magnetSprite builds the 16x16 horseshoe magnet: a red arc with two legs whose
// bottom tips are steel poles, opening downward.
func magnetSprite() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, magnetSize, magnetSize))
	red := color.RGBA{0xD0, 0x2B, 0x2B, 0xFF}
	redDark := color.RGBA{0x8E, 0x16, 0x16, 0xFF}
	steel := color.RGBA{0xC8, 0xCD, 0xD4, 0xFF}

	const (
		cx     = 7.5  // horizontal centre
		cyArc  = 8.0  // arc centre (legs hang below this row)
		rOuter = 7.0  // outer radius of the U
		rInner = 3.5  // inner radius (the magnet's mouth)
		tipTop = 12   // rows >= tipTop on the legs are steel poles
	)

	for y := 0; y < magnetSize; y++ {
		for x := 0; x < magnetSize; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cyArc

			var on bool
			if dy <= 0 { // top: the curved arc
				d := math.Hypot(dx, dy)
				on = d >= rInner && d <= rOuter
			} else { // bottom: the two straight legs
				ax := math.Abs(dx)
				on = ax >= rInner && ax <= rOuter
			}
			if !on {
				continue
			}

			c := red
			switch {
			case y >= tipTop && dy > 0: // pole tips
				c = steel
			case dy <= 0 && math.Hypot(dx, dy) >= rOuter-1: // outer rim shading on the arc
				c = redDark
			}
			img.Set(x, y, c)
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
