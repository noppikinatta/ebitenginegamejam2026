package asset

import (
	"bytes"
	"embed"
	"image"
	"io/fs"
	"log"
	"path"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	//go:embed font/Mplus2-Regular.ttf
	mplus2regularttf []byte

	Mplus2RegularFaceSource *text.GoTextFaceSource

	fontFaces map[float64]text.Face

	//go:embed img/*.png
	imgDir embed.FS

	imgs map[string]*ebiten.Image
)

func init() {
	initFont()
	initImages()
}

func initFont() {
	var err error

	Mplus2RegularFaceSource, err = text.NewGoTextFaceSource(bytes.NewReader(mplus2regularttf))
	if err != nil {
		log.Fatal("cannot read embedded font:", err)
	}

	fontFaces = make(map[float64]text.Face)
}

func FontFace(size float64) text.Face {
	f, ok := fontFaces[size]
	if !ok {
		f = &text.GoTextFace{
			Source: Mplus2RegularFaceSource,
			Size:   size,
		}
		fontFaces[size] = f
	}

	return f
}

func initImages() {
	imgs = make(map[string]*ebiten.Image)

	const imgDirPath = "img"
	entries, err := imgDir.ReadDir(imgDirPath)
	if err != nil {
		log.Fatal("canot open image directory:", err)
	}

	for _, e := range entries {
		err := addImage(e, imgDirPath)
		if err != nil {
			log.Fatal("cannot load image for ", e.Name(), ":", err)
		}
	}
}
func addImage(entry fs.DirEntry, dirPath string) error {
	fpath := path.Join(dirPath, entry.Name())
	file, err := imgDir.Open(fpath)
	if err != nil {
		return err
	}

	imageImg, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	ebitenImg := ebiten.NewImageFromImage(imageImg)
	imgs[strings.TrimSuffix(entry.Name(), ".png")] = ebitenImg
	return nil
}

func Images() map[string]*ebiten.Image {
	return imgs
}
