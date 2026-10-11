package introart

// Boss video-call picture geometry.
const (
	CallW = 160
	CallH = 80
)

// CallMouth are the picture byte offsets the mouth animation rewrites (rows 47-49, bytes 16-17).
var CallMouth = [6]int{1896, 1897, 1936, 1937, 1976, 1977}

type pred func(x, y int) bool

func ell(cx, cy, rx, ry int) pred {
	return func(x, y int) bool { return inEllipse(x, y, cx, cy, rx, ry) }
}

func fl(v float64) float64 { return v } // forces rounding of a product before it is added (no FMA)

// BossCall draws the Blofeld video-call picture (160x96 canvas, 80 rows used).
func BossCall() *Canvas {
	art := NewCanvas(160)

	fill := func(test pred, color int) {
		for y := 0; y < CallH; y++ {
			for x := 0; x < CallW; x++ {
				if test(x, y) {
					art.Dot(x, y, color)
				}
			}
		}
	}

	// armchair: dithered leather with channel seams, solid rim
	chairEll := ell(80, 30, 66, 26)
	chair := func(x, y int) bool { return 14 <= x && x <= 145 && (y >= 6 || chairEll(x, y)) }
	fill(func(x, y int) bool { return chair(x, y) && (x+y)%2 == 0 && x%12 != 2 }, 2)
	fill(func(x, y int) bool { return chair(x, y) && (!chair(x, y-2) || !chair(x-2, y) || !chair(x+2, y)) }, 2)
	// armrests
	for _, r := range [][2]int{{4, 32}, {128, 156}} {
		a, b := r[0], r[1]
		fill(func(x, y int) bool { return a <= x && x <= b && 48 <= y && (x+y)%2 == 0 }, 2)
		fill(func(x, y int) bool { return a <= x && x <= b && 46 <= y && y <= 48 }, 2)
	}
	// Blofeld: sleeves, jacket, lap
	torso := func(x, y int) bool {
		fy := float64(y)
		return 34+fl(fy*0.15) <= float64(x) && float64(x) <= 126-fl(fy*0.15) && y <= 60
	}
	sleeveL := func(x, y int) bool {
		fy := float64(y)
		return 26+fl(fy*0.1) <= float64(x) && float64(x) <= 44+fl(fy*0.05) && y <= 52
	}
	sleeveR := func(x, y int) bool {
		fy := float64(y)
		return 116-fl(fy*0.05) <= float64(x) && float64(x) <= 134-fl(fy*0.1) && y <= 44
	}
	lapA, lapB := ell(56, 68, 30, 16), ell(104, 68, 30, 16)
	lap := func(x, y int) bool { return y >= 56 && (lapA(x, y) || lapB(x, y) || (40 <= x && x <= 120)) }
	body := func(x, y int) bool { return torso(x, y) || sleeveL(x, y) || sleeveR(x, y) || lap(x, y) }
	fill(body, 0)
	fill(func(x, y int) bool { return body(x, y) && body(x-1, y) && body(x+1, y) }, 2)
	// seams: sleeve edges, knees
	fill(func(x, y int) bool { return y <= 52 && absf(float64(x)-(44+fl(float64(y)*0.05))) < 0.6 }, 0)
	fill(func(x, y int) bool { return y <= 44 && absf(float64(x)-(116-fl(float64(y)*0.05))) < 0.6 }, 0)
	fill(func(x, y int) bool { return y >= 66 && abs(x-80) < 1 }, 0)
	// neck and Nehru collar at the top edge; the face stays out of frame
	art.Rect(73, 0, 88, 4, 3)
	fill(func(x, y int) bool {
		a := float64(x-80) / 10
		b := float64(y+2) / 9
		return y <= 7 && 70 <= x && x <= 90 && !(73 <= x && x < 88 && y < 4) && float64(a*a)+float64(b*b) <= 1.15
	}, 0)
	fill(func(x, y int) bool { return x == 80 && 7 <= y && y <= 30 }, 0)
	for y := 11; y < 31; y += 6 {
		art.Dot(78, y, 0)
	}
	// left hand curled over the armrest
	fill(ell(30, 47, 6, 3), 3)
	for _, fx := range []int{25, 28, 31, 34} {
		art.Rect(fx, 48, fx+2, 53, 3)
		art.Dot(fx+2, 52, 0)
	}
	art.Line(26, 50, 35, 50, 3)
	// the white Persian
	catBody := ell(90, 55, 30, 12)
	catHead := ell(66, 40, 15, 13)
	tailA, tailB := ell(94, 60, 26, 7), ell(96, 57, 24, 5)
	tail := func(x, y int) bool { return tailA(x, y) && !tailB(x, y) && y >= 60 && x >= 72 }
	pawA, pawB := ell(59, 58, 5, 3), ell(71, 58, 5, 3)
	paws := func(x, y int) bool { return pawA(x, y) || pawB(x, y) }
	hang := func(x, y int) bool {
		if !(56 <= y && y <= 74) {
			return false
		}
		c := 117 + fl(float64(y-56)*0.35) - fl(float64(max(0, y-69))*0.9)
		return absf(float64(x)-c) <= 2.6
	}
	cat := func(x, y int) bool { return catBody(x, y) || catHead(x, y) || tail(x, y) || paws(x, y) || hang(x, y) }
	fill(cat, 1)
	// fluffy outline: jagged tufts
	for y := 0; y < CallH; y++ {
		for x := 0; x < CallW; x++ {
			if cat(x, y) && !cat(x, y-2) && (x*7+y)%5 == 0 {
				art.Dot(x, y-1, 1)
			}
			if cat(x, y) && !cat(x-2, y) && (y*3)%4 == 0 {
				art.Dot(x-1, y, 1)
			}
			if cat(x, y) && !cat(x+2, y) && (y*5)%4 == 1 {
				art.Dot(x+1, y, 1)
			}
		}
	}
	// ears
	art.Triangle(55, 25, 51, 60, 32, 1)
	art.Triangle(77, 25, 72, 81, 32, 1)
	art.Triangle(55, 28, 53, 58, 32, 3)
	art.Triangle(77, 28, 74, 79, 32, 3)
	// fur shading: gray strokes below and between shapes
	fill(func(x, y int) bool { return catBody(x, y) && !catBody(x, y+2) && !tail(x, y) && (x+y)%2 == 0 }, 2)
	fill(func(x, y int) bool { return catHead(x, y) && !catHead(x, y+2) && (x+y)%2 == 0 }, 2)
	for _, p := range [][2]int{{96, 48}, {104, 50}, {86, 52}, {112, 54}, {100, 57}, {78, 58}} {
		art.Line(p[0], p[1], p[0]+2, p[1]+2, 2)
	}
	fill(func(x, y int) bool { return tail(x, y) && !tail(x, y+1) && x%2 == 0 }, 2)
	fill(func(x, y int) bool { return hang(x, y) && !hang(x+1, y) && y > 60 }, 2)
	for _, px := range []int{56, 59, 62, 68, 71, 74} {
		art.Line(px, 60, px, 61, 2)
	}
	// copper eyes with slit pupils, nose, mouth
	for _, ex := range []int{60, 72} {
		art.Rect(ex-3, 38, ex+3, 42, 3)
		art.Dot(ex-3, 38, 1)
		art.Dot(ex+2, 38, 1)
		art.Rect(ex-1, 38, ex+1, 42, 0)
		dy := 0
		if ex < 66 {
			dy = 1
		}
		art.Line(ex-4, 36, ex+2, 36-dy, 2)
	}
	art.Rect(65, 44, 68, 46, 3)
	art.Dot(66, 46, 0)
	art.Line(66, 47, 64, 48, 0)
	art.Line(67, 47, 69, 48, 0)
	// diamond collar
	for cx := 56; cx < 78; cx += 3 {
		art.Dot(cx, 52+abs(cx-66)/8, 3)
	}
	// right hand stroking the cat's back
	fill(ell(116, 43, 6, 4), 3)
	for i, fx := range []int{109, 112, 115, 118} {
		off := 0
		if i == 0 {
			off = 1
		}
		art.Line(fx, 45, fx-4, 50-off, 3)
		art.Line(fx+1, 45, fx-3, 50-off, 3)
	}
	art.Line(120, 40, 122, 44, 2)
	// video-call viewfinder corners and REC dot
	for _, c := range [][4]int{{1, 1, 1, 1}, {158, 1, -1, 1}, {1, 78, 1, -1}, {158, 78, -1, -1}} {
		x0, y0, dx, dy := c[0], c[1], c[2], c[3]
		art.Line(x0, y0, x0+8*dx, y0, 1)
		art.Line(x0, y0, x0, y0+5*dy, 1)
	}
	art.Rect(5, 4, 8, 7, 3)
	return art
}

// OpenMouth opens the Persian's mouth on a BossCall canvas.
func OpenMouth(art *Canvas) {
	art.Rect(65, 47, 69, 50, 0)
	art.Rect(66, 49, 68, 50, 3)
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
