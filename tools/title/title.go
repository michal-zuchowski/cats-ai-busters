// Package title packs the approved four-color title into 188 ANTIC E rows; the last four are blank.
package title

import (
	"fmt"
	"image/png"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

// Palette is the four title colors (RGB).
var Palette = [4][3]uint8{{10, 16, 25}, {36, 57, 76}, {236, 229, 203}, {236, 163, 69}}

// LoadRGB decodes a PNG into 8-bit RGB triples, row-major.
func LoadRGB(path string) (rgb []byte, w, h int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, 0, 0, err
	}
	b := img.Bounds()
	rgb = make([]byte, 0, b.Dx()*b.Dy()*3)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA() // composited over black, like PIL convert("RGB")
			rgb = append(rgb, byte(r>>8), byte(g>>8), byte(bl>>8))
		}
	}
	return rgb, b.Dx(), b.Dy(), nil
}

// TitleRGB returns assets/title-screen.png as RGB bytes.
func TitleRGB() ([]byte, error) {
	rgb, w, h, err := LoadRGB(k65src.Path("assets", "title-screen.png"))
	if err != nil {
		return nil, err
	}
	if w != 160 || h != 192 {
		return nil, fmt.Errorf("Title must be native 160x192")
	}
	return rgb, nil
}

// TitleBitmap packs the title into 188*40 bytes.
func TitleBitmap() ([]byte, error) {
	rgb, err := TitleRGB()
	if err != nil {
		return nil, err
	}
	pixels := make([]byte, 160*192)
	for i := range pixels {
		c := [3]uint8{rgb[3*i], rgb[3*i+1], rgb[3*i+2]}
		idx := -1
		for k, p := range Palette {
			if p == c {
				idx = k
			}
		}
		if idx < 0 {
			return nil, fmt.Errorf("pixel %d is not in the title palette", i)
		}
		pixels[i] = byte(idx)
	}
	for _, p := range pixels[188*160:] {
		if p != 0 {
			return nil, fmt.Errorf("Last four rows must be blank for the display-list gap")
		}
	}
	out := make([]byte, 0, 188*40)
	for i := 0; i < 188*160; i += 4 {
		out = append(out, pixels[i]<<6|pixels[i+1]<<4|pixels[i+2]<<2|pixels[i+3])
	}
	return out, nil
}

// Run writes assets/title-screen.pic.
func Run() error {
	data, err := TitleBitmap()
	if err != nil {
		return err
	}
	if len(data) != 7520 {
		return fmt.Errorf("unexpected title size %d", len(data))
	}
	return os.WriteFile(k65src.Path("assets", "title-screen.pic"), data, 0o644)
}
