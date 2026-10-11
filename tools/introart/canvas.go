// Package introart draws the intro scenes: the panoramic data center, the
// 160x96 tabby close-up and the boss video call (port of tools/make_intro_art.py).
package introart

import (
	"fmt"
	"math"
	"os"
)

// Height is the pixel height of the full-screen intro pictures.
const Height = 96

// Unset marks a canvas pixel that was never drawn (python None).
const Unset = -1

// Canvas is a 2-bit-per-pixel picture, Pixels[y][x].
type Canvas struct {
	Width  int
	Pixels [][]int
}

// NewCanvas returns a black Height-row canvas of the given width.
func NewCanvas(width int) *Canvas {
	c := &Canvas{Width: width, Pixels: make([][]int, Height)}
	for y := range c.Pixels {
		c.Pixels[y] = make([]int, width)
	}
	return c
}

// Dot sets one pixel; out-of-range coordinates are ignored.
func (c *Canvas) Dot(x, y, color int) {
	if 0 <= x && x < c.Width && 0 <= y && y < Height {
		c.Pixels[y][x] = color
	}
}

// Rect fills [x0,x1) x [y0,y1).
func (c *Canvas) Rect(x0, y0, x1, y1, color int) {
	for y := max(0, y0); y < min(Height, y1); y++ {
		for x := max(0, x0); x < min(c.Width, x1); x++ {
			c.Dot(x, y, color)
		}
	}
}

// Ellipse fills an ellipse centred on (cx, cy) with radii rx, ry.
func (c *Canvas) Ellipse(cx, cy, rx, ry, color int) {
	for y := max(0, cy-ry); y < min(Height, cy+ry+1); y++ {
		for x := max(0, cx-rx); x < min(c.Width, cx+rx+1); x++ {
			if inEllipse(x, y, cx, cy, rx, ry) {
				c.Dot(x, y, color)
			}
		}
	}
}

func inEllipse(x, y, cx, cy, rx, ry int) bool {
	a := float64(x-cx) / float64(rx)
	b := float64(y-cy) / float64(ry)
	return float64(a*a)+float64(b*b) <= 1
}

// Line draws a line from (x0, y0) to (x1, y1) inclusive.
func (c *Canvas) Line(x0, y0, x1, y1, color int) {
	steps := max(abs(x1-x0), abs(y1-y0), 1)
	for step := 0; step <= steps; step++ {
		c.Dot(round(float64(x0)+float64(float64((x1-x0)*step)/float64(steps))),
			round(float64(y0)+float64(float64((y1-y0)*step)/float64(steps))), color)
	}
}

// Triangle fills a triangle with its apex at (apexX, apexY) and its base on row baseY.
func (c *Canvas) Triangle(apexX, apexY, baseLeft, baseRight, baseY, color int) {
	for y := apexY; y <= baseY; y++ {
		t := float64(y-apexY) / float64(baseY-apexY)
		c.Rect(round(float64(apexX)+float64(float64(baseLeft-apexX)*t)), y,
			round(float64(apexX)+float64(float64(baseRight-apexX)*t))+1, y+1, color)
	}
}

// Mirrored reverses every row in place and returns the canvas.
func (c *Canvas) Mirrored() *Canvas {
	for _, row := range c.Pixels {
		for l, h := 0, len(row)-1; l < h; l, h = l+1, h-1 {
			row[l], row[h] = row[h], row[l]
		}
	}
	return c
}

// Packed packs 4 pixels per byte, leftmost in the high bits.
func (c *Canvas) Packed() []byte {
	var out []byte
	for _, row := range c.Pixels {
		for x := 0; x < c.Width; x += 4 {
			out = append(out, byte(row[x]<<6|row[x+1]<<4|row[x+2]<<2|row[x+3]))
		}
	}
	return out
}

// Preview writes a binary PPM; natural selects the in-game colour palette.
func (c *Canvas) Preview(path string, natural bool) error {
	colors := [4][3]byte{{0, 0, 0}, {34, 54, 76}, {88, 165, 159}, {215, 164, 68}}
	if natural {
		colors = [4][3]byte{{0, 0, 0}, {16, 30, 110}, {124, 125, 122}, {239, 199, 139}}
	}
	data := []byte(fmt.Sprintf("P6\n%d %d\n255\n", c.Width, Height))
	for _, row := range c.Pixels {
		for _, p := range row {
			data = append(data, colors[p][:]...)
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// round is python round(): ties to even.
func round(f float64) int { return int(math.RoundToEven(f)) }
