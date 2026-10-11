package levelart

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// Palette maps pixel values (4 = inverse-tile red) to colours for previews.
var Palette = [5]color.RGBA{
	{0, 0, 0, 255}, {170, 170, 170, 255}, {48, 72, 168, 255}, {160, 88, 24, 255}, {200, 56, 40, 255},
}

// CatOverlay is a sprite (rows of 0/1/2 pixels) drawn at pixel column X0 on the play-row floor.
type CatOverlay struct {
	X0  int
	Img [][]int
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// Preview writes a PNG of room number at roughly the real ANTIC 4 pixel
// aspect (8:5); cat may be nil.
func Preview(path string, number int, cat *CatOverlay) error {
	data := Room(number)
	img := image.NewRGBA(image.Rect(0, 0, 160*8, 192*5))
	for row := 0; row < 24; row++ {
		for col := 0; col < 40; col++ {
			tile := data[row*40+col]
			glyph := TileGlyphs[int(tile&127)]
			for y, line := range glyph {
				for x, ch := range line {
					value := int(ch - '0')
					if value == 3 && tile&Inv != 0 {
						value = 4
					}
					px, py := (col*4+x)*8, (row*8+y)*5
					fillRect(img, px, py, px+8, py+5, Palette[value])
				}
			}
		}
	}
	if cat != nil {
		for y, line := range cat.Img {
			for x, value := range line {
				if value != 0 {
					c := color.RGBA{70, 30, 8, 255}
					if value == 1 {
						c = color.RGBA{240, 168, 80, 255}
					}
					px, py := (cat.X0+x)*8, (184-24+y)*5
					fillRect(img, px, py, px+8, py+5, c)
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
