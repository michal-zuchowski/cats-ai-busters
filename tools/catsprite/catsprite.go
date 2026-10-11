// Package catsprite generates the ginger agent cat as four single-line PMG
// players (16x24, two colours); port of tools/make_cat_sprite.py.
//
// Frame layout (128 bytes each): P0 dark-left, P1 dark-right, P2 fur-left,
// P3 fur-right, 32 bytes per player (24 used). Frames 0-11 walk right, 12-13
// idle right, 14-27 mirrored; 28-35 leap right, 36-43 leap left; 44-47 raised
// forepaw / hind kick; 48-49 strike; 50-53 seated; 54-57 desk-glass reach.
// Byte-identical logical frames share one physical frame; assets/cat-frame-map.bin
// maps logical -> physical.
package catsprite

import (
	"math"
)

// Sprite dimensions and pixel values.
const (
	W, H = 16, 24
	Fill = 1
	Dark = 2
)

// Img is H rows of W pixels (0 transparent, Fill, Dark).
type Img [][]int

var upper = [15]string{
	"................",
	"................",
	"................",
	"................",
	"...........F..F.",
	"...........FF.FF",
	"...........FFFFF",
	"..........FFF.FF",
	"...FFFFFF.FFF.FF",
	"..FdFFdFFdFFFFFF",
	"..FFdFFdFFdFFFFd",
	"..FFFFFFFFFFFFF.",
	"..FFFFFFFFFFFd..",
	"..FFFFFFFFFFF...",
	"...FFFFFFFFFF...",
}

var tail = [3][8]string{
	{".Fd..", "F....", "F....", "F....", "F....", ".F...", ".F...", "..F.."},
	{"..Fd.", ".F...", ".F...", ".F...", "F....", "F....", ".F...", "..F.."},
	{".....", "Fd...", "F....", "F....", "F....", ".F...", ".F...", "..F.."},
}

type paw struct {
	x    float64
	lift int
}

var hindTrack = [12]paw{{3.5, 0}, {3, 0}, {2, 0}, {1, 0}, {1, 1}, {2, 2}, {5, 2}, {6.5, 1},
	{7.5, 0}, {7, 0}, {6, 0}, {5, 0}}
var foreTrack = [12]paw{{12.5, 0}, {12, 0}, {11, 0}, {10, 0}, {9, 0}, {8, 0}, {8, 0}, {9, 1},
	{11, 2}, {13, 2}, {14, 1}, {14, 0}}

const (
	drop     = 2
	hip      = 4
	shoulder = 11
	top      = 15 + drop
	floor    = 23
)

func rnd(v float64) int { return int(math.RoundToEven(v)) }

func blank() Img {
	img := make(Img, H)
	for y := range img {
		img[y] = make([]int, W)
	}
	return img
}

func fromRows(rows []string) Img {
	img := make(Img, len(rows))
	for y, row := range rows {
		img[y] = make([]int, len(row))
		for x, ch := range row {
			switch ch {
			case 'F':
				img[y][x] = Fill
			case 'd':
				img[y][x] = Dark
			}
		}
	}
	return img
}

// Mirror returns img flipped left-right.
func Mirror(img Img) Img {
	out := make(Img, len(img))
	for y, row := range img {
		out[y] = make([]int, len(row))
		for x, v := range row {
			out[y][len(row)-1-x] = v
		}
	}
	return out
}

func put(img Img, x, y, c int) {
	if 0 <= x && x < W && 0 <= y && y < H {
		img[y][x] = c
	}
}

func line(img Img, x0, y0, x1, y1, c int) {
	steps := max(abs(x1-x0), abs(y1-y0), 1)
	for i := 0; i <= steps; i++ {
		put(img, rnd(float64(x0)+float64((x1-x0)*i)/float64(steps)),
			rnd(float64(y0)+float64((y1-y0)*i)/float64(steps)), c)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clearCols(img Img, y0, y1, x0, x1 int) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img[y][x] = 0
		}
	}
}

func hind(img Img, xf float64, lift, c, o int) {
	x := rnd(xf) + o
	hp := hip + o
	pawY := floor - lift
	hockX, hockY := rnd(float64(hp+x)/2)-1, 19+drop/2
	for dx := -1; dx <= 1; dx++ {
		put(img, hp+dx, top, c)
	}
	put(img, hp-1, top+1, c)
	line(img, hp, top, hockX, hockY, c)
	line(img, hockX, hockY, x, pawY, c)
	put(img, x+1, pawY, c)
}

func fore(img Img, xf float64, lift, c, o int) {
	x := rnd(xf) + o
	pawY := floor - lift
	if lift != 0 {
		line(img, shoulder+o, top, x, pawY-1, c)
		put(img, x-1, pawY, c)
	} else {
		line(img, shoulder+o, top, x, pawY, c)
		put(img, x+1, pawY, c)
	}
}

// Drawing is the standing/walking cat; k < 0 means the idle pose (python k=None).
func Drawing(tailIdx, k int) Img {
	rows := make([]string, 0, 15)
	for i, u := range upper[:8] {
		rows = append(rows, tail[tailIdx][i]+u[5:])
	}
	rows = append(rows, upper[8:]...)
	body := fromRows(rows)
	img := make(Img, 0, H)
	for i := 0; i < drop; i++ {
		img = append(img, make([]int, W))
	}
	img = append(img, body...)
	for len(img) < H {
		img = append(img, make([]int, W))
	}
	if k < 0 {
		hind(img, 5, 0, Fill, 0)
		fore(img, 12, 0, Fill, 0)
		hind(img, 4, 0, Fill, 0)
		fore(img, 11, 0, Fill, 0)
		return img
	}
	a, b := (k+6)%12, k
	hind(img, hindTrack[a].x, hindTrack[a].lift, Fill, 0)
	fore(img, foreTrack[a].x, foreTrack[a].lift, Fill, 0)
	hind(img, hindTrack[b].x, hindTrack[b].lift, Fill, 0)
	fore(img, foreTrack[b].x, foreTrack[b].lift, Fill, 0)
	return img
}

func idle() Img { return Drawing(0, -1) }

func leap(kind int) Img {
	if kind == 7 {
		return idle()
	}
	img := idle()
	for y := top; y < H; y++ {
		img[y] = make([]int, W)
	}
	clearCols(img, 0, 10, 0, 5)
	line(img, 0, 9, 2, 10, Fill)
	line(img, 2, 10, 3, 12, Fill)
	switch kind {
	case 1, 2:
		flight := blank()
		for y := 0; y < top; y++ {
			for x, colour := range img[y] {
				if colour != 0 {
					dx, dy := 0, 0
					if x >= 10 {
						dx = -2
					} else if x >= 2 {
						dx = -1
					}
					if x >= 9 {
						dy = -2
					} else if x >= 6 {
						dy = -1
					}
					put(flight, x+dx, y+dy, colour)
				}
			}
		}
		img = flight
		line(img, 4, 14, 9, 13, Fill)
		line(img, 4, 15, 9, 14, Fill)
		if kind == 1 {
			hind(img, 1, 2, Fill, 0)
		} else {
			hind(img, 3, 3, Fill, 0)
		}
		line(img, 9, 13, 15, 12, Fill)
		line(img, 9, 15, 14, 14, Fill)
		put(img, 15, 14, Fill)
		return img
	case 3, 4:
		tilted := blank()
		for y := 0; y < top; y++ {
			for x, colour := range img[y] {
				if colour != 0 {
					var dy int
					switch {
					case x >= 9:
						dy = 2
					case x >= 6:
						dy = 1
					case kind == 3:
						dy = -2
					default:
						dy = -1
					}
					put(tilted, x, y+dy, colour)
				}
			}
		}
		img = tilted
		if kind == 3 {
			line(img, hip, 15, 3, 18, Fill)
		} else {
			line(img, hip, 16, 3, 18, Fill)
		}
		line(img, 3, 18, 5, 20, Fill)
		put(img, 6, 20, Fill)
		line(img, 11, 19, 12, 22, Fill)
		line(img, 12, 23, 13, 23, Fill)
		return img
	case 5, 6:
		d, a, b := 1, 3, 11
		if kind == 5 {
			d, a, b = 2, 2, 10
		}
		shifted := make(Img, 0, H)
		for i := 0; i < d; i++ {
			shifted = append(shifted, make([]int, W))
		}
		img = append(shifted, img[:H-d]...)
		line(img, hip, top+d, a, 21, Fill)
		line(img, a, 21, 5, floor, Fill)
		put(img, 6, floor, Fill)
		line(img, shoulder, top+d, b, 21, Fill)
		line(img, b, 21, 12, floor, Fill)
		put(img, 13, floor, Fill)
		if kind == 6 {
			clearCols(img, 0, 11, 0, 5)
			line(img, 0, 6, 1, 7, Fill)
			line(img, 1, 7, 1, 10, Fill)
			line(img, 1, 10, 3, 12, Fill)
		}
		return img
	}
	hind(img, 5, 1, Fill, 0)
	fore(img, 11, 1, Fill, 0)
	return append(Img{make([]int, W)}, img[:H-1]...)
}

func pawAttack(back, strike bool) Img {
	img := append(Img{make([]int, W)}, idle()[:H-1]...)
	if !back {
		for y := 6; y < top; y++ {
			head := append([]int(nil), img[y][10:]...)
			clearCols(img, y, y+1, 10, 16)
			for i, colour := range head {
				if colour != 0 {
					img[y][8+i] = colour
				}
			}
		}
	}
	for y := top; y < H; y++ {
		img[y] = make([]int, W)
	}
	hind(img, 5, 0, Fill, 0)
	fore(img, 12, 0, Fill, 0)
	switch {
	case back:
		line(img, hip, top, 2, 19, Fill)
		line(img, 2, 19, 0, 19, Fill)
		put(img, 0, 20, Fill)
	case strike:
		line(img, shoulder, top, 13, 14, Fill)
		line(img, 13, 14, 15, 15, Fill)
		line(img, 15, 15, 15, 18, Fill)
		line(img, 15, 18, 14, 19, Fill)
		put(img, 13, 19, Fill)
	default:
		line(img, shoulder, top, 15, 10, Fill)
		line(img, 15, 10, 15, 4, Fill)
		put(img, 14, 3, Fill)
		put(img, 13, 4, Fill)
	}
	return img
}

var seatedRows = []string{
	".........F..F...",
	".........FF.FF..",
	".........FFFFF..",
	"........FFF.FFF.",
	"........FFF.FFF.",
	"........FFFFFd..",
	".........FFFF...",
	".........FFF....",
	"........FFFF....",
	".......FdFFF....",
	"......FFFFdF....",
	".....FFFFFFF....",
	"....FdFFFFFF....",
	"...FFFFFFdFF....",
	"..FFFFFFFFFF....",
	".FFFdFFFFFFF....",
	".FFFFFFF.FFF....",
	"F.FFFFFF..FFF...",
	"F.FFdFFF....F...",
	"F.FFFFFF....F...",
	"FF.FdFFF....F...",
	".FFFFFFFF...F...",
	"FFFFFFFFF...F...",
	"FFFFFFFFF..FFF..",
}

func seated(shove bool) Img {
	img := fromRows(seatedRows)
	if shove {
		clearCols(img, 19, 24, 0, 4)
		line(img, 3, 19, 1, 22, Fill)
		line(img, 2, 20, 0, 23, Fill)
		put(img, 1, 23, Fill)
		put(img, 2, 22, Fill)
	}
	return img
}

func glassPush(kind int) Img {
	img := idle()
	for y := top; y < H; y++ {
		img[y] = make([]int, W)
	}
	clearCols(img, 10, 17, 12, W)
	hind(img, 1, 0, Fill, 0)
	fore(img, 8, 0, Fill, 0)
	hind(img, 5, 0, Fill, 0)
	targets := [4][2]int{{10, 16}, {11, 14}, {15, 12}, {12, 14}}
	tx, ty := targets[kind][0], targets[kind][1]
	line(img, shoulder, top, tx, ty, Fill)
	put(img, tx, ty, Fill)
	if kind == 2 {
		put(img, min(W-1, tx+1), ty, Fill)
	}
	return img
}

func mirrorAll(l []Img) []Img {
	out := make([]Img, len(l))
	for i, f := range l {
		out[i] = Mirror(f)
	}
	return out
}

// Frames returns all 58 logical frames.
func Frames() []Img {
	var right []Img
	for k := 0; k < 12; k++ {
		right = append(right, Drawing(0, k))
	}
	right = append(right, Drawing(0, -1), Drawing(2, -1))
	var jr []Img
	for k := 0; k < 8; k++ {
		jr = append(jr, leap(k))
	}
	attacks := []Img{pawAttack(false, false), pawAttack(true, false)}
	strike := pawAttack(false, true)
	sit, shove := seated(false), seated(true)

	var all []Img
	all = append(all, right...)
	all = append(all, mirrorAll(right)...)
	all = append(all, jr...)
	all = append(all, mirrorAll(jr)...)
	all = append(all, attacks...)
	all = append(all, mirrorAll(attacks)...)
	all = append(all, strike, Mirror(strike), sit, Mirror(sit), shove, Mirror(shove))
	for k := 0; k < 4; k++ {
		all = append(all, glassPush(k))
	}
	return all
}

// Pack encodes one frame as 128 bytes (P0..P3, 32 bytes each).
func Pack(img Img) []byte {
	out := make([]byte, 0, 128)
	for _, colour := range [2]int{Dark, Fill} {
		for _, half := range [2]int{0, 8} {
			player := make([]byte, 32)
			for y := 0; y < H; y++ {
				var b byte
				for x := 0; x < 8; x++ {
					if img[y][half+x] == colour {
						b += 0x80 >> x
					}
				}
				player[y] = b
			}
			out = append(out, player...)
		}
	}
	return out
}

// PackedAtlas returns the physical frame bytes and the logical -> physical map.
func PackedAtlas(imgs []Img) (atlas, mapping []byte) {
	order := map[string]int{}
	n := 0
	for _, img := range imgs {
		data := Pack(img)
		idx, ok := order[string(data)]
		if !ok {
			idx = n
			order[string(data)] = n
			n++
			atlas = append(atlas, data...)
		}
		mapping = append(mapping, byte(idx))
	}
	return atlas, mapping
}
