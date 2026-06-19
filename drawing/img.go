package drawing

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/noppikinatta/ebitenginegamejam2026/asset"
)

var (
	dummyImageBase = ebiten.NewImage(3, 3)

	// WhitePixel is useful to draw fill shape with DrawTriangles.
	WhitePixel = dummyImageBase.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	fallbackImage = ebiten.NewImage(32, 32)
)

func init() {
	dummyImageBase.Fill(color.White)
	fallbackImage.Fill(color.RGBA{G: 255, A: 255})
	DrawText(fallbackImage, "IMAGE\n NOT\n  FOUND", 9, &ebiten.DrawImageOptions{})
}

func Image(key string) *ebiten.Image {
	img, ok := asset.Images()[key]
	if !ok {
		return fallbackImage
	}
	return img
}

// DrawSprite draws img centred at screen position (cx, cy), scaled to fit a
// w×h box, rotated by angle radians around its centre, and tinted by (r,g,b,a).
// Scaling lets art authored at any resolution match the intended footprint.
// Only a DrawImageOptions value is allocated per call (no per-frame images).
func DrawSprite(dst, img *ebiten.Image, cx, cy, w, h, angle float64, r, g, b, a float32) {
	bounds := img.Bounds()
	iw, ih := bounds.Dx(), bounds.Dy()
	if iw == 0 || ih == 0 {
		return
	}

	opt := &ebiten.DrawImageOptions{}
	opt.Filter = ebiten.FilterNearest
	// Centre the image on its own origin, scale to the target box, rotate, then
	// move to the destination centre.
	opt.GeoM.Translate(-float64(iw)/2, -float64(ih)/2)
	opt.GeoM.Scale(w/float64(iw), h/float64(ih))
	opt.GeoM.Rotate(angle)
	opt.GeoM.Translate(cx, cy)
	opt.ColorScale.Scale(r, g, b, a)
	dst.DrawImage(img, opt)
}

// DrawSpriteAnchored draws img scaled uniformly by `scale`, rotated by `angle`
// around the sprite-local pivot (ax, ay) given in SOURCE pixels, positioned so
// that pivot lands at screen (cx, cy), and tinted by (r,g,b,a).
//
// Unlike DrawSprite — which stretches the image into a w×h box and pivots at its
// centre — this preserves the image's aspect ratio and lets the caller choose
// the pivot. It is used for rectangular turret-barrel art: anchoring at the
// centre of the barrel's mount tile makes the barrel swing about its base when
// the turret rotates. Only a DrawImageOptions value is allocated per call.
func DrawSpriteAnchored(dst, img *ebiten.Image, cx, cy, scale, angle, ax, ay float64, r, g, b, a float32) {
	if img.Bounds().Empty() {
		return
	}
	opt := &ebiten.DrawImageOptions{}
	opt.Filter = ebiten.FilterNearest
	// Pivot (source px) → origin, uniform scale, rotate, then to the destination.
	opt.GeoM.Translate(-ax, -ay)
	opt.GeoM.Scale(scale, scale)
	opt.GeoM.Rotate(angle)
	opt.GeoM.Translate(cx, cy)
	opt.ColorScale.Scale(r, g, b, a)
	dst.DrawImage(img, opt)
}
