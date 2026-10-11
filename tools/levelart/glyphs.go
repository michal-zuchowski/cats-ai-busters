package levelart

import "math"

// Glyph is one ANTIC 4 tile: 8 rows of 4 pixels, each pixel a digit 0..3.
type Glyph [8]string

// VacBrushes are the four vacuum-brush animation phases (3 rows each).
var VacBrushes = [4][3]string{
	{"0100", "1331", "0010"},
	{"1001", "0330", "1001"},
	{"0010", "1331", "0100"},
	{"0011", "0330", "1100"},
}

func reverseRows(rows [3]string) [3]string {
	var out [3]string
	for i, r := range rows {
		b := []byte(r)
		for l, h := 0, len(b)-1; l < h; l, h = l+1, h-1 {
			b[l], b[h] = b[h], b[l]
		}
		out[i] = string(b)
	}
	return out
}

func reverseString(s string) string { return reverseRows([3]string{s})[0] }

func g(rows ...string) Glyph {
	if len(rows) != 8 {
		panic("levelart: glyph needs 8 rows")
	}
	var out Glyph
	copy(out[:], rows)
	return out
}

func same(row string) Glyph { return g(row, row, row, row, row, row, row, row) }

// coneGlyph is a light-cone tile: the dither pattern kept where keep(i) holds.
func coneGlyph(keep func(i int) bool) Glyph {
	var out Glyph
	pattern := [4]string{"1010", "0000", "0101", "0000"}
	for r := range out {
		row := []byte(pattern[r%4])
		for i := range row {
			if !keep(i) {
				row[i] = '0'
			}
		}
		out[r] = string(row)
	}
	return out
}

// TileGlyphs maps every tile ID (0..127, bit 7 excluded) that has a glyph to its pixels.
var TileGlyphs = buildTileGlyphs()

func buildTileGlyphs() map[int]Glyph {
	t := map[int]Glyph{
		Void:      same("0000"),
		Ceil:      g("2222", "2212", "2222", "1222", "2222", "2221", "2222", "2122"),
		CeilEdge:  g("2222", "2212", "2222", "1111", "2222", "0000", "0000", "0000"),
		CeilJoint: g("2112", "2112", "2112", "1111", "2112", "0220", "0000", "0000"),
		Tray:      g("0000", "2002", "2332", "3333", "3133", "2222", "0000", "0000"),
		Cam:       g("0220", "0220", "2222", "2112", "2222", "2232", "0220", "0000"),
		CapL:      g("0111", "1122", "1222", "1222", "1222", "1222", "1200", "1211"),
		Cap:       g("1111", "2222", "2212", "2222", "2222", "2222", "0000", "1111"),
		CapR:      g("1110", "2220", "2220", "2220", "2220", "2220", "0020", "1020"),
		RailL:     g("1222", "1232", "1222", "1200", "1211", "1222", "1222", "1200"),
		RailR:     g("2220", "2220", "2020", "0020", "1120", "2220", "2220", "0020"),
		// calm 1U fronts; SrvB/SrvD LEDs are blinked at runtime by src/l01_rooms.k65 blink_leds
		SrvA:    g("2222", "2222", "2222", "2222", "2222", "2222", "2222", "0000"),
		SrvB:    g("2222", "2222", "2232", "2222", "2222", "2222", "2222", "0000"),
		SrvC:    g("2222", "2222", "2222", "2112", "2222", "2222", "2222", "0000"),
		SrvD:    g("2222", "2222", "2222", "2222", "2322", "2222", "2222", "0000"),
		PlinthL: g("1111", "1222", "1202", "1222", "0000", "0000", "0000", "0000"),
		Plinth:  g("1111", "2222", "0202", "2222", "0000", "0000", "0000", "0000"),
		PlinthR: g("1110", "2220", "0220", "2220", "0000", "0000", "0000", "0000"),
		// play row (row 18): black space above a thin floor surface
		Floor:    g("0000", "0000", "0000", "0000", "0000", "0000", "2121", "2222"),
		Shadow:   g("0000", "0000", "0000", "0000", "0000", "0000", "0202", "2222"),
		Safe:     g("0000", "0000", "0000", "0000", "0000", "0000", "0200", "2222"),
		Beam:     g("0300", "0000", "0003", "3000", "0030", "0303", "3333", "3333"),
		Wall:     g("1222", "1222", "1232", "1222", "1222", "1222", "1222", "2222"),
		Switch:   g("0000", "0110", "0130", "0110", "0110", "0020", "2121", "2222"),
		Relay:    g("1111", "1331", "1311", "1331", "1111", "0220", "2121", "2222"),
		Done:     g("1111", "1221", "1211", "1221", "1111", "0220", "2121", "2222"),
		Hatch:    g("1000", "1000", "1000", "1000", "1000", "1000", "1121", "2222"),
		Edge:     g("1111", "1111", "2222", "2212", "2222", "2122", "2222", "0000"),
		FrTL:     g("0111", "1122", "1200", "1200", "1200", "1200", "1200", "1200"),
		FrT:      g("1111", "2222", "0000", "0000", "0000", "0000", "0000", "0000"),
		FrTR:     g("1110", "2220", "0020", "0020", "0020", "0020", "0020", "0020"),
		FrL:      same("1200"),
		FrR:      same("0020"),
		FrBL:     g("1200", "1200", "1200", "1200", "1200", "1222", "1222", "0000"),
		FrB:      g("0000", "0000", "0000", "0000", "0000", "2222", "2222", "0000"),
		FrBR:     g("0020", "0020", "0020", "0020", "0020", "2220", "2220", "0000"),
		TxtA:     g("0000", "3303", "0000", "3033", "0000", "3330", "0000", "0000"),
		TxtB:     g("0000", "0333", "0000", "3300", "0000", "3033", "0000", "0000"),
		Conduit:  g("0210", "0210", "0220", "0210", "0210", "0210", "0220", "0210"),
		FloorLit: g("1010", "0000", "0101", "0000", "1010", "0101", "1111", "2222"),
		CamL:     g("0220", "0220", "2222", "2112", "2222", "3222", "0220", "0000"),
		CamR:     g("0220", "0220", "2222", "2112", "2222", "2223", "0220", "0000"),
		Panel:    g("2222", "2112", "2332", "2112", "2222", "2002", "2222", "0000"),
		DomeL:    g("0000", "0000", "0011", "0122", "1222", "1222", "1232", "1222"),
		DomeM:    g("0000", "0000", "1111", "2222", "2222", "2222", "2332", "2222"),
		DomeR:    g("0000", "0000", "1100", "2210", "2221", "2221", "2321", "2221"),
		Lift:     g("0000", "0000", "0000", "0000", "0000", "3030", "3333", "2222"),
		BaseL:    g("1222", "1222", "1222", "0222", "0010", VacBrushes[0][0], VacBrushes[0][1], VacBrushes[0][2]),
		BaseM:    g("2222", "2112", "2222", "2222", "0220", "2002", "0000", "0000"),
		BaseR: func() Glyph {
			r := reverseRows(VacBrushes[0])
			return g("2221", "2221", "2221", "2220", "0100", r[0], r[1], r[2])
		}(),
		ChaseL:   g("0000", "0000", "0011", "0133", "1333", "1333", "1232", "1222"),
		ChaseM:   g("0000", "0000", "1111", "3333", "3333", "3333", "2332", "2222"),
		ChaseR:   g("0000", "0000", "1100", "3310", "3331", "3331", "2321", "2221"),
		DockBay:  g("0000", "0000", "0000", "0000", "0000", "0330", "3333", "1111"),
		Glass:    g("0000", "0000", "1001", "1001", "1221", "1221", "1111", "2222"),
		HudFull:  g("0000", "2220", "2220", "2220", "2220", "2220", "0000", "0000"),
		HudEmpty: g("0000", "3330", "3030", "3030", "3030", "3330", "0000", "0000"),
		HudL:     g("0000", "0033", "0030", "0030", "0030", "0033", "0000", "0000"),
		HudR:     g("0000", "3300", "0300", "0300", "0300", "3300", "0000", "0000"),
		Crumb:    g("0000", "0000", "0000", "0000", "0000", "3003", "3333", "3131"),
		Ctrl:     g("0000", "0110", "0110", "1331", "1221", "1111", "2121", "2222"),
		AirA:     g("0000", "1100", "0000", "0000", "0000", "0011", "0000", "0000"),
		AirB:     g("0000", "0011", "0000", "0000", "0000", "1100", "0000", "0000"),
	}
	for k := 0; k < 4; k++ {
		t[Cone+k] = coneGlyph(func(i int) bool { return i >= k })
		t[ConeR+k] = coneGlyph(func(i int) bool { return i <= k })
	}
	for i, tile := range Rotor(0) {
		t[FanA[i]] = tile
	}
	for i, tile := range Rotor(math.Pi / 4) {
		t[FanB[i]] = tile
	}
	for k := range t {
		if k < 0 || k >= 128 {
			panic("levelart: tile id out of range")
		}
	}
	return t
}

// Rotor draws the 8x16 pixel 4-blade rotor (a pixel is 2 colour clocks wide:
// 16x16 clocks) as its 2x2 glyphs (TL, TR, BL, BR).
func Rotor(angle float64) [4]Glyph {
	var pix [16][8]int
	for y := 0; y < 16; y++ {
		for x := 0; x < 8; x++ {
			px, py := float64(x*2+1-8), float64(y)+0.5-8
			dist := math.Hypot(px, py)
			hit := 0
			for k := 0; k < 4; k++ {
				a := angle + float64(k)*math.Pi/2
				// float64() blocks fused multiply-add so results match CPython.
				along := float64(px*math.Cos(a)) + float64(py*math.Sin(a))
				side := math.Abs(float64(-px*math.Sin(a)) + float64(py*math.Cos(a)))
				limit := 1.3
				if k%2 == 1 {
					limit += 0.12
				}
				if 1.5 <= along && along <= 8 && side <= limit {
					hit = 3
				}
			}
			if dist < 2.2 {
				pix[y][x] = 1
			} else {
				pix[y][x] = hit
			}
		}
	}
	var tiles [4]Glyph
	n := 0
	for _, ty := range []int{0, 8} {
		for _, tx := range []int{0, 4} {
			for r := 0; r < 8; r++ {
				row := make([]byte, 4)
				for c := 0; c < 4; c++ {
					row[c] = byte('0' + pix[ty+r][tx+c])
				}
				tiles[n][r] = string(row)
			}
			n++
		}
	}
	return tiles
}
