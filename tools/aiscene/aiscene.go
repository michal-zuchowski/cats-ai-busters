// Package aiscene draws the streamed ANTIC D AI monitor: a 160x80 bitmap and two
// 24x5 mouth patches. The tiny paletted canvas reproduces the PIL ImageDraw
// primitives the Python generator used (rectangle, line, wide line, polygon).
package aiscene

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

// Palette is the four monitor colors (RGB).
var Palette = [4][3]uint8{{10, 16, 25}, {36, 100, 142}, {236, 229, 203}, {236, 163, 69}}

// Mouth is the (x0, y0, x1, y1) box of the mouth patch.
var Mouth = [4]int{64, 56, 88, 61}

// Image is a paletted canvas of color indices 0..3.
type Image struct {
	W, H int
	Pix  []uint8
}

func New(w, h int) *Image { return &Image{w, h, make([]uint8, w*h)} }

// Get returns the index at (x, y).
func (m *Image) Get(x, y int) uint8 { return m.Pix[y*m.W+x] }

func (m *Image) set(x, y int, c uint8) {
	if x >= 0 && y >= 0 && x < m.W && y < m.H {
		m.Pix[y*m.W+x] = c
	}
}

func (m *Image) hline(x0, y, x1 int, c uint8) {
	for x := x0; x <= x1; x++ {
		m.set(x, y, c)
	}
}

// Copy returns a deep copy.
func (m *Image) Copy() *Image {
	return &Image{m.W, m.H, append([]uint8(nil), m.Pix...)}
}

// Crop returns the sub-image [x0,x1) x [y0,y1) (PIL Image.crop).
func (m *Image) Crop(x0, y0, x1, y1 int) *Image {
	out := New(x1-x0, y1-y0)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			out.Pix[(y-y0)*out.W+x-x0] = m.Get(x, y)
		}
	}
	return out
}

// Paste copies src onto m at (x, y).
func (m *Image) Paste(src *Image, x, y int) {
	for j := 0; j < src.H; j++ {
		for i := 0; i < src.W; i++ {
			m.set(x+i, y+j, src.Get(i, j))
		}
	}
}

// Rect fills the inclusive rectangle.
func (m *Image) Rect(x0, y0, x1, y1 int, c uint8) {
	for y := y0; y <= y1; y++ {
		m.hline(x0, y, x1, c)
	}
}

// RectOutline draws a width-pixel border inside the inclusive rectangle.
func (m *Image) RectOutline(x0, y0, x1, y1 int, c uint8, width int) {
	for i := 0; i < width; i++ {
		if x0+i > x1-i || y0+i > y1-i {
			break
		}
		m.hline(x0+i, y0+i, x1-i, c)
		m.hline(x0+i, y1-i, x1-i, c)
		for y := y0 + i + 1; y < y1-i; y++ {
			m.set(x0+i, y, c)
			m.set(x1-i, y, c)
		}
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// Line is PIL's width-1 Bresenham line.
func (m *Image) Line(x0, y0, x1, y1 int, c uint8) {
	dx, dy := abs(x1-x0), abs(y1-y0)
	xs, ys := 1, 1
	if x1 < x0 {
		xs = -1
	}
	if y1 < y0 {
		ys = -1
	}
	switch {
	case dx == 0:
		for i := 0; i <= dy; i++ {
			m.set(x0, y0+i*ys, c)
		}
	case dy == 0:
		for i := 0; i <= dx; i++ {
			m.set(x0+i*xs, y0, c)
		}
	case dx > dy:
		e := 2*dy - dx
		for i := 0; i < dx; i++ {
			m.set(x0, y0, c)
			if e >= 0 {
				y0 += ys
				e -= 2 * dx
			}
			e += 2 * dy
			x0 += xs
		}
		m.set(x0, y0, c)
	default:
		e := 2*dx - dy
		for i := 0; i < dy; i++ {
			m.set(x0, y0, c)
			if e >= 0 {
				x0 += xs
				e -= 2 * dy
			}
			e += 2 * dx
			y0 += ys
		}
		m.set(x0, y0, c)
	}
}

func roundUp(f float64) int {
	if f >= 0 {
		return int(math.Floor(f + 0.5))
	}
	return -int(math.Floor(math.Abs(f) + 0.5))
}

func roundDown(f float64) int {
	if f >= 0 {
		return int(math.Ceil(f - 0.5))
	}
	return -int(math.Ceil(math.Abs(f) - 0.5))
}

// WideLine is PIL's width>1 line: a filled quadrilateral around the segment.
func (m *Image) WideLine(x0, y0, x1, y1 int, c uint8, width int) {
	dx, dy := x1-x0, y1-y0
	if dx == 0 && dy == 0 {
		m.set(x0, y0, c)
		return
	}
	big := math.Hypot(float64(dx), float64(dy))
	small := float64(width-1) / 2
	ratioMax := float64(roundUp(small)) / big
	ratioMin := float64(roundDown(small)) / big
	dxmin := roundDown(ratioMin * float64(dy))
	dxmax := roundDown(ratioMax * float64(dy))
	dymin := roundDown(ratioMin * float64(dx))
	dymax := roundDown(ratioMax * float64(dx))
	m.fillPolygon([][2]int{{x0 - dxmin, y0 + dymax}, {x1 - dxmin, y1 + dymax},
		{x1 + dxmax, y1 - dymin}, {x0 + dxmax, y0 - dymin}}, c)
}

type edge struct {
	xmin, xmax, ymin, ymax, x0, y0 int
	dx                             float32
}

func newEdge(x0, y0, x1, y1 int) edge {
	e := edge{xmin: min(x0, x1), xmax: max(x0, x1)}
	if y0 > y1 {
		x0, y0, x1, y1 = x1, y1, x0, y0
	}
	e.ymin, e.ymax, e.x0, e.y0 = y0, y1, x0, y0
	if y0 != y1 {
		e.dx = float32(x1-x0) / float32(y1-y0)
	}
	return e
}

func (m *Image) fillPolygon(pts [][2]int, c uint8) {
	n := len(pts)
	var edges []edge
	for i := range pts {
		j := (i + 1) % n
		edges = append(edges, newEdge(pts[i][0], pts[i][1], pts[j][0], pts[j][1]))
	}
	ymin, ymax := edges[0].ymin, edges[0].ymax
	var table []edge
	for _, e := range edges {
		ymin, ymax = min(ymin, e.ymin), max(ymax, e.ymax)
		if e.ymin == e.ymax {
			m.hline(e.xmin, e.ymin, e.xmax, c)
			continue
		}
		table = append(table, e)
	}
	for ; ymin <= ymax; ymin++ {
		var xx []float32
		for _, e := range table {
			if ymin >= e.ymin && ymin <= e.ymax {
				x := float32(ymin-e.y0)*e.dx + float32(e.x0)
				xx = append(xx, x)
				if ymin == e.ymax && ymin < ymax {
					xx = append(xx, x)
				}
			}
		}
		sort.Slice(xx, func(a, b int) bool { return xx[a] < xx[b] })
		xPos := 0
		for i := 1; i < len(xx); i += 2 {
			xEnd := roundDown(float64(xx[i]))
			if xEnd < xPos {
				continue
			}
			m.hline(roundUp(float64(xx[i-1])), ymin, xEnd, c)
			xPos = xEnd + 1
		}
	}
}

// Polygon fills the polygon then draws its 1-pixel outline.
func (m *Image) Polygon(pts [][2]int, fill, outline uint8) {
	m.fillPolygon(pts, fill)
	for i := range pts {
		j := (i + 1) % len(pts)
		m.Line(pts[i][0], pts[i][1], pts[j][0], pts[j][1], outline)
	}
}

// Pack packs 2-bit pixels, four per byte.
func Pack(m *Image) []byte {
	if m.W%4 != 0 {
		panic("aiscene.Pack: width must be a multiple of 4")
	}
	out := make([]byte, 0, len(m.Pix)/4)
	for i := 0; i < len(m.Pix); i += 4 {
		p := m.Pix[i : i+4]
		out = append(out, p[0]<<6|p[1]<<4|p[2]<<2|p[3])
	}
	return out
}

// Portrait returns the full monitor and the closed/open mouth patches.
func Portrait() (image, closed, opened *Image) {
	d := New(160, 80)
	d.RectOutline(8, 3, 151, 77, 1, 2)
	d.RectOutline(12, 7, 147, 70, 2, 1)
	for y := 13; y < 66; y += 8 {
		d.Line(16, y, 43, y, 1)
		d.Line(112, y, 143, y, 1)
		d.Rect(19, y-1, 21, y+1, 3)
		d.Rect(137, y-1, 139, y+1, 3)
	}
	d.Polygon([][2]int{{49, 17}, {104, 17}, {110, 23}, {110, 58}, {98, 66},
		{55, 66}, {43, 58}, {43, 23}}, 1, 2)
	d.Rect(49, 23, 104, 47, 0)
	d.WideLine(51, 29, 68, 34, 3, 2)
	d.WideLine(84, 34, 102, 29, 3, 2)
	d.Rect(56, 36, 69, 39, 2)
	d.Rect(84, 36, 97, 39, 2)
	d.WideLine(76, 39, 76, 49, 0, 2)
	d.Rect(62, 54, 89, 62, 0)
	for x := 24; x < 133; x += 12 {
		c := uint8(3)
		if x%24 != 0 {
			c = 2
		}
		d.Rect(x, 73, x+5, 74, c)
	}
	closed = New(24, 5)
	closed.Line(0, 2, 23, 2, 3)
	for x := 2; x < 24; x += 5 {
		closed.Line(x, 1, x, 3, 2)
	}
	opened = New(24, 5)
	opened.RectOutline(0, 0, 23, 4, 3, 1)
	for x := 2; x < 24; x += 4 {
		y1 := 3
		if x%8 != 0 {
			y1 = 2
		}
		opened.Line(x, 1, x, y1, 2)
	}
	d.Paste(closed, Mouth[0], Mouth[1])
	return d, closed, opened
}

// RGBA renders the image with the palette.
func (m *Image) RGBA() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, m.W, m.H))
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			p := Palette[m.Get(x, y)]
			out.SetRGBA(x, y, color.RGBA{p[0], p[1], p[2], 255})
		}
	}
	return out
}

// Run writes assets/ai-portrait.pic and out/ai-cutscene-preview.png.
func Run() error {
	img, closed, opened := Portrait()
	data := append(append(Pack(img), Pack(closed)...), Pack(opened)...)
	if len(data) != 3260 {
		return fmt.Errorf("unexpected portrait size %d", len(data))
	}
	if err := os.WriteFile(k65src.Path("assets", "ai-portrait.pic"), data, 0o644); err != nil {
		return err
	}
	if err := os.MkdirAll(k65src.Path("out"), 0o755); err != nil {
		return err
	}
	sheet := image.NewRGBA(image.Rect(0, 0, 640, 160))
	for n, pose := range []*Image{closed, opened} {
		frame := img.Copy()
		frame.Paste(pose, Mouth[0], Mouth[1])
		rgba := frame.RGBA()
		for y := 0; y < 160; y++ {
			for x := 0; x < 320; x++ {
				sheet.SetRGBA(n*320+x, y, rgba.RGBAAt(x/2, y/2))
			}
		}
	}
	f, err := os.Create(k65src.Path("out", "ai-cutscene-preview.png"))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, sheet)
}
