// Package levelart builds the Druidarium-style ANTIC 4 tiles and the room
// screens (port of tools/make_level_art.py).
//
// Pixel values: 0 black, 1 light grey (PF0, also HUD text), 2 steel blue (PF1),
// 3 rust orange (PF2); tiles with bit 7 set draw their 3s in red (PF3).
// The cat is a PMG sprite, so no tile is reserved for it and the backdrop can
// be dense.
package levelart

import (
	"fmt"
	"math"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

// Tile IDs (character codes). OR a tile with Inv for the inverse (red) variant.
const (
	Inv = 128

	Void      = 64
	Ceil      = 65
	CeilEdge  = 66
	CatL      = 67 // playback cat head, left half (right half CatR); eyes blink, OR Inv = red REC eyes
	Cam       = 68
	RailL     = 69
	Floor     = 70 // play row: open floor (fixed ID, used by src/*.k65)
	RailR     = 71
	SrvA      = 72
	SrvB      = 73
	SrvC      = 74
	SrvD      = 75
	CapL      = 76
	Cap       = 77
	Shadow    = 78 // play row: rack shelter
	Beam      = 79 // play row: watched floor, drawn inverse (red)
	CatR      = 80
	Wall      = 81 // hatch closed
	Switch    = 82
	Relay     = 83
	Done      = 84
	Hatch     = 85
	CapR      = 86
	PlinthL   = 87
	Plinth    = 88
	PlinthR   = 89
	Edge      = 90
	DomeL     = 91 // vacuum / lift tiles: 91..94 are solid tops (platform levels)
	DomeM     = 92
	DomeR     = 93
	Lift      = 94
	BaseL     = 95 // vacuum base row (not solid)
	FrTL      = 96
	FrT       = 97
	FrTR      = 98
	FrL       = 99
	FrR       = 100
	FrBL      = 101
	FrB       = 102
	FrBR      = 103
	TxtA      = 104
	TxtB      = 105
	Conduit   = 106
	CeilJoint = 107
	Panel     = 108
	Cone      = 109 // 109..112: light cone, lit from pixel k (0 = full cell)
	ConeR     = 113 // 113..116: lit up to pixel k (116 = full cell)
	FloorLit  = 117 // play row inside a camera footprint
	CamL      = 118
	CamR      = 119
	HudFull   = 120 // HUD bar cell, filled (COLPF1: state colour)
	HudEmpty  = 121 // HUD bar cell, empty (COLPF2)
	HudL      = 122 // HUD bar brackets
	HudR      = 123
	BaseM     = 124
	BaseR     = 125
	Glass     = 126
	Crumb     = 127 // level 02 crumbling catwalk tile; the warning is the same glyph inverse (red)
	Ctrl      = 62  // level 02 controller console (solid; stored inverse = unpowered, red)
	AirA      = 8   // level 02 airflow streaks (alternate while the gust blows)
	AirB      = 9
	ChaseL    = 1 // pursuer dome (lamp band); same row-2 silhouette as Dome*
	ChaseM    = 2
	ChaseR    = 6
	DockBay   = 12 // charger contacts, row above the floor

	DockPlate = Plinth  // charger head, row above the bay
	ReadStrip = DockBay // service reader floor strip
	ReadHead  = Plinth  // service reader head
)

// Fan rotor tiles (TL, TR, BL, BR): FanA is the '+' pose, FanB the same rotor turned 45 degrees.
var (
	FanA = [4]int{21, 22, 23, 24}
	FanB = [4]int{25, 27, 28, 60}
)

// Screen is a 24-row by 40-column ANTIC 4 character screen, indexed [row][col].
type Screen [24][40]int

// Bytes flattens the screen row by row (960 bytes).
func (s *Screen) Bytes() []byte {
	out := make([]byte, 0, 960)
	for _, row := range s {
		for _, t := range row {
			out = append(out, byte(t))
		}
	}
	return out
}

// Servers is the cycle of rack front tiles.
var Servers = [9]int{SrvA, SrvB | Inv, SrvA, SrvC, SrvA, SrvD, SrvA, SrvA, SrvD | Inv}

// Rack draws a server rack (python rack(); defaults there were width=6, bottom=21, seed=0).
func Rack(s *Screen, left, top, width, bottom, seed int) {
	s[top][left] = CapL
	s[top][left+width-1] = CapR
	for x := left + 1; x < left+width-1; x++ {
		s[top][x] = Cap
	}
	for y := top + 1; y < bottom; y++ {
		s[y][left] = RailL
		s[y][left+width-1] = RailR
		for x := left + 1; x < left+width-1; x++ {
			s[y][x] = Servers[(x*7+y*5+seed*3+x*y)%len(Servers)]
		}
	}
	s[bottom][left] = PlinthL
	s[bottom][left+width-1] = PlinthR
	for x := left + 1; x < left+width-1; x++ {
		s[bottom][x] = Plinth
	}
}

// Frame draws a bordered panel whose interior tiles come from fill(x, y).
func Frame(s *Screen, left, top, right, bottom int, fill func(x, y int) int) {
	for x := left + 1; x < right; x++ {
		s[top][x] = FrT
		s[bottom][x] = FrB
	}
	for y := top + 1; y < bottom; y++ {
		s[y][left] = FrL
		s[y][right] = FrR
		for x := left + 1; x < right; x++ {
			s[y][x] = fill(x, y)
		}
	}
	s[top][left], s[top][right] = FrTL, FrTR
	s[bottom][left], s[bottom][right] = FrBL, FrBR
}

// Console draws the record console: a framed screen, a control strip and a
// plinth, rows top..21. Its footprint keeps the Shadow floor of a rack.
func Console(s *Screen, left, top, width int) {
	right := left + width - 1
	Frame(s, left, top, right, top+3, func(x, y int) int { return pick((x+y)%2 == 0, TxtB|Inv, TxtA) })
	for x := left; x <= right; x++ {
		s[top+4][x] = Panel
		s[21][x] = Plinth
	}
	s[21][left], s[21][right] = PlinthL, PlinthR
}

// PlaybackBlock is one wall-monitor interior (rows from Row, columns from Col)
// that shows the looped recording once the room-3 console has been used.
type PlaybackBlock struct {
	Row, Col int
	Tiles    [][]int
}

// Playback replaces the two monitor interiors of room 3 (src/l01_rooms.k65
// PlayL/PlayR, row-major, rows of 5 and 3 tiles).
var Playback = [2]PlaybackBlock{
	{7, 18, [][]int{
		{Beam | Inv, Beam | Inv, Beam | Inv, Beam | Inv, Beam | Inv},
		{Void, CatL | Inv, CatR | Inv, Void, Void},
		{Void, Void, Void, CatL | Inv, CatR | Inv},
		{Beam | Inv, Beam | Inv, Beam | Inv, Beam | Inv, Beam | Inv},
	}},
	{11, 33, [][]int{
		{Beam | Inv, Beam | Inv, Beam | Inv},
		{Void, CatL | Inv, CatR | Inv},
		{CatL | Inv, CatR | Inv, Void},
		{Beam | Inv, Beam | Inv, Beam | Inv},
	}},
}

// Shell is the empty room: void, ceiling rows, bottom edge, conduit columns and camera tiles.
func Shell(cameraCols, conduits []int) *Screen {
	s := &Screen{}
	for y := range s {
		for x := range s[y] {
			s[y][x] = Void
		}
	}
	for x := 0; x < 40; x++ {
		if x%8 == 3 {
			s[2][x], s[3][x] = CeilJoint, CeilJoint
		} else {
			s[2][x], s[3][x] = Ceil, CeilEdge
		}
		s[23][x] = Edge
	}
	for _, x := range conduits {
		for y := 4; y < 22; y++ {
			s[y][x] = Conduit
		}
	}
	for _, x := range cameraCols {
		s[4][x] = Cam
	}
	return s
}

// SpecialTile overrides one tile of the floor row.
type SpecialTile struct{ Col, Tile int }

// FloorRow fills row 22: open Floor where a camera can see it, Shadow under racks.
func FloorRow(s *Screen, specials ...SpecialTile) {
	for x := 0; x < 40; x++ {
		if s[21][x] == Void || s[21][x] == Conduit {
			s[22][x] = Floor
		} else {
			s[22][x] = Shadow
		}
	}
	for _, sp := range specials {
		s[22][sp.Col] = sp.Tile
	}
}

// Camera is one camera slot: column, sweep min/max (footprint centre column),
// frames per column, pause at each end, footprint half-width and whether the
// room-1 switch gates it. Col 0 marks an unused slot.
type Camera struct{ Col, Lo, Hi, Speed, Pause, HalfWidth, Gate int }

// Cameras holds two camera slots per room.
var Cameras = [3][2]Camera{
	{{20, 11, 28, 10, 90, 2, 0}, {0, 0, 0, 1, 1, 0, 0}},
	{{14, 10, 19, 5, 10, 2, 1}, {31, 23, 38, 9, 35, 2, 0}},
	// The exit camera must park far enough left that its end pause clears the
	// floor past the shelter: the cat is four columns wide and walks one in 12
	// frames, so it can only cross the open floor while the camera waits. With
	// centre 30 the shelter exit stayed lit and the hatch was unreachable; the
	// slower sweep and longer pause also leave room to hold the terminal.
	{{20, 9, 26, 8, 30, 2, 0}, {35, 27, 38, 9, 140, 2, 0}},
}

// ConeTop and ConeRows: the light cone spans rows 5..21, apex under the camera tile.
const (
	ConeTop  = 5
	ConeRows = 17
)

// CameraTable returns the 8-byte-per-slot camera table and the cone step table.
func CameraTable() (cams, steps []byte) {
	for _, slots := range Cameras {
		for _, c := range slots {
			// The widest cone row must cover exactly the detected columns
			// centre-hw..centre+hw. cone_hwh starts at 1 and gains one pixel per
			// carry out of the 8-bit fraction, so it must end at hw*4+1: a wider
			// cone lit a column (centre+hw+1) that cam_sees never alarms on.
			hwstep := int(math.RoundToEven((float64(c.HalfWidth*4) + 0.5) * 256 / ConeRows))
			if !(hwstep < 256 && (c.Col == 0 || c.Lo <= c.Hi)) {
				panic("levelart: bad camera slot")
			}
			if 1+ConeRows*hwstep/256 != c.HalfWidth*4+1 {
				panic("levelart: cone width mismatch")
			}
			cams = append(cams, byte(c.Col), byte(c.Lo), byte(c.Hi), byte(c.Speed),
				byte(c.Pause), byte(c.HalfWidth), byte(c.Gate), byte(hwstep))
		}
	}
	var step []int
	for d := -40; d < 40; d++ {
		step = append(step, int(math.RoundToEven(float64(d*4*256)/ConeRows))&0xFFFF)
	}
	for _, v := range step {
		steps = append(steps, byte(v))
	}
	for _, v := range step {
		steps = append(steps, byte(v>>8))
	}
	return cams, steps
}

func conds(ok bool, msg string) {
	if !ok {
		panic("levelart: " + msg)
	}
}

// Room returns the 960 tile bytes of data-center room 0, 1 or 2.
func Room(number int) []byte {
	var s *Screen
	switch number {
	case 0: // shelter racks 1..13, camera strip 14..25, racks 26..37
		s = Shell([]int{20}, []int{15, 25})
		Rack(s, 1, 6, 7, 21, 0)
		Rack(s, 8, 5, 6, 21, 1)
		Frame(s, 16, 7, 23, 12, func(x, y int) int { return pick((x+y)%3 != 0, TxtA, TxtB) })
		Rack(s, 26, 7, 6, 21, 2)
		Rack(s, 32, 5, 6, 21, 3)
		FloorRow(s)
	case 1: // switch col 6, open segment A 9..19, shelter 20..23, segment B
		s = Shell([]int{14, 31}, []int{9, 25})
		Rack(s, 1, 5, 4, 21, 4)
		for y := 16; y < 19; y++ {
			s[y][6] = Panel
		}
		Frame(s, 11, 8, 17, 13, func(x, y int) int { return pick((x+y)%3 != 0, TxtB, TxtA) })
		Rack(s, 20, 5, 4, 21, 6)
		Rack(s, 28, 7, 5, 21, 7)
		Rack(s, 36, 6, 4, 21, 8)
		FloorRow(s, SpecialTile{6, Switch})
	default: // zone 1 7..25 with cover 12..16, shelter (record console) 26..31, zone 2, hatch 38
		s = Shell([]int{20, 35}, []int{8, 24})
		Rack(s, 1, 6, 6, 21, 9)
		Rack(s, 12, 8, 5, 21, 10)
		Frame(s, 17, 6, 23, 11, func(x, y int) int { return pick((x*y)%3 != 0, TxtB, TxtA) })
		Console(s, 26, 16, 6)
		Frame(s, 32, 10, 36, 15, func(x, y int) int { return pick(y%2 != 0, TxtA, TxtB|Inv) })
		FloorRow(s, SpecialTile{38, Wall})
	}
	for _, c := range Cameras[number] {
		if c.Col != 0 {
			conds(s[4][c.Col] == Cam, "camera tile missing")
		}
	}
	return s.Bytes()
}

func pick(cond bool, a, b int) int {
	if cond {
		return a
	}
	return b
}

// VacuumBrushes returns the packed vacuum-brush table (assets/vacuum-brushes.bin).
func VacuumBrushes() []byte {
	var rows []string
	for _, phase := range VacBrushes {
		rows = append(rows, phase[:]...)
	}
	for _, phase := range VacBrushes {
		for _, r := range phase {
			rows = append(rows, reverseString(r))
		}
	}
	out := make([]byte, len(rows))
	for i, row := range rows {
		out[i] = PackGlyphRow(row)
	}
	return out
}

// PackGlyphRow packs a 4-digit pixel row into one ANTIC 4 byte (2 bits per pixel, leftmost high).
func PackGlyphRow(row string) byte {
	var v int
	for x, p := range row {
		v += int(p-'0') << (6 - 2*x)
	}
	return byte(v)
}

// WriteAssets writes vacuum-brushes.bin, level-rooms.pic, level-cams.bin and
// cone-step.bin into the assets directory of the repository.
func WriteAssets() error {
	var rooms []byte
	for n := 0; n < 3; n++ {
		rooms = append(rooms, Room(n)...)
	}
	cams, steps := CameraTable()
	for _, f := range []struct {
		name string
		data []byte
	}{
		{"vacuum-brushes.bin", VacuumBrushes()},
		{"level-rooms.pic", rooms},
		{"level-cams.bin", cams},
		{"cone-step.bin", steps},
	} {
		if err := os.WriteFile(k65src.Path("assets", f.name), f.data, 0o644); err != nil {
			return fmt.Errorf("levelart: %w", err)
		}
	}
	return nil
}
