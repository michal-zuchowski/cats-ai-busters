package catsprite

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"regexp"
	"strconv"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
	L "github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
	"github.com/michal-zuchowski/cats-ai-busters/tools/platlevels"
)

func fill(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func save(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// Preview writes a filmstrip PNG of all frames.
func Preview(path string, imgs []Img, scale int) error {
	pal := map[int]color.RGBA{0: {30, 40, 110, 255}, Fill: {236, 150, 60, 255}, Dark: {110, 45, 10, 255}}
	sheet := image.NewRGBA(image.Rect(0, 0, len(imgs)*(W+1)*2*scale, H*scale))
	for i, img := range imgs {
		for y := 0; y < H; y++ {
			for x := 0; x < W; x++ {
				x0 := (i*(W+1) + x) * 2 * scale
				fill(sheet, x0, y*scale, x0+2*scale, (y+1)*scale, pal[img[y][x]])
			}
		}
	}
	return save(path, sheet)
}

// GlassPreview writes the source-data filmstrip of the L04 glass beat.
// Deviation: the Python version also drew a text label per panel (PIL default
// font); the labels are omitted here.
func GlassPreview(path string, imgs []Img) error {
	const xscale, yscale = 4, 2
	const left, topRow, cols, rows = 20, 8, 20, 15
	panelW, panelH := cols*4, rows*8
	out := image.NewRGBA(image.Rect(0, 0, panelW*xscale, panelH*yscale*3))
	fill(out, 0, 0, panelW*xscale, panelH*yscale*3, color.RGBA{8, 12, 20, 255})
	room := platlevels.Rooms[9][:960]
	colours := [4]color.RGBA{{8, 12, 20, 255}, {150, 180, 190, 255}, {35, 100, 145, 255}, {30, 190, 220, 255}}
	poses := [3][3]int{{54, 33, 13}, {56, 33, 13}, {57, 36, 17}}

	source := k65src.Read()
	m := regexp.MustCompile(`(?s)data GlassPoses \{(.*?)\}`).FindStringSubmatch(source)
	if m == nil {
		return fmt.Errorf("data GlassPoses not found")
	}
	var glassPoses []int
	for _, v := range regexp.MustCompile(`\b\d+\b`).FindAllString(m[1], -1) {
		n, _ := strconv.Atoi(v)
		glassPoses = append(glassPoses, n)
	}
	if len(glassPoses) != 32 {
		return fmt.Errorf("GlassPoses has %d values, want 32", len(glassPoses))
	}

	rect := func(x, y int, c color.RGBA) {
		fill(out, x*xscale, y*yscale, x*xscale+xscale, y*yscale+yscale, c)
	}
	glyphAt := func(g L.Glyph, px0, py0 int) {
		for py, ln := range g {
			for px, ch := range ln {
				rect(px0+px, py0+py, colours[int(ch-'0')])
			}
		}
	}
	for panel, p := range poses {
		frame, glassCol, glassRow := p[0], p[1], p[2]
		oy := panel * panelH
		for row := 0; row < rows; row++ {
			for col := 0; col < cols; col++ {
				glyphAt(L.TileGlyphs[int(room[(topRow+row)*40+left+col]&127)], col*4, oy+row*8)
			}
		}
		gx, gy := (glassCol-left)*4, oy+(glassRow-topRow)*8
		if panel == 2 {
			for py, b := range glassPoses[24:32] {
				for px := 0; px < 4; px++ {
					rect(gx+px, gy+py, colours[(b>>(6-2*px))&3])
				}
			}
		} else {
			glyphAt(L.TileGlyphs[L.Glass], gx, gy)
		}
		visualTop := 120 - 24
		for y, ln := range imgs[frame] {
			for x, v := range ln {
				if v == 0 {
					continue
				}
				c := color.RGBA{220, 150, 55, 255}
				if v == Dark {
					c = color.RGBA{100, 45, 20, 255}
				}
				rect(118-left*4+x, oy+visualTop-topRow*8+y, c)
			}
		}
		if panel == 2 {
			for col := 35; col < 39; col++ {
				glyphAt(L.TileGlyphs[12], (col-left)*4, oy+(18-topRow)*8)
			}
		}
	}
	return save(path, out)
}
