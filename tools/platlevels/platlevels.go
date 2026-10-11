// Package platlevels builds the rooms of the platform levels 02-04 and the PCM
// computer-death cry (port of tools/make_plat_levels.py).
//
// Each room is a 1024-byte disk chunk: 960 tile bytes (platforms are Floor
// tiles, one-way) + 64 bytes of metadata read by src/plat_rooms.k65 into
// page 6 (see Meta).
package platlevels

import (
	"fmt"
	"math"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
	L "github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
	"github.com/michal-zuchowski/cats-ai-busters/tools/pyrand"
)

// Meta lists the metadata field names in chunk order (keep <= 32; src/plat_rooms.k65
// copies 32 at m_sx).
//
// L02 additions: swy/swc controller standing y + column (0 = none; it also gates
// the exit and, with lpow, the lift); fan = first column of the gust zone, fc1 =
// end column (exclusive), fr0/fr1 = body-row band [fr0, fr1); lspd = frames the
// lift dwells per row.
var Meta = []string{"sx", "sy", "fan", "lrow", "lcol", "lw", "lmin", "lmax", "arow", "amin",
	"amax", "aspd", "acol", "h0", "h1", "glass", "dlg", "desk", "exit_y",
	"swy", "swc", "lpow", "fc1", "fr0", "fr1", "lspd"}

// MetaIndex returns the offset of a Meta field name inside the metadata block.
func MetaIndex(name string) int {
	for i, n := range Meta {
		if n == name {
			return i
		}
	}
	panic("platlevels: unknown meta field " + name)
}

// L03Meta: level 03 only (cur_level == 2): offsets inside the 32 copied bytes.
// 3/4/6/7 reuse the lift fields (m_lw stays 0 there, so no lift exists);
// 26..30 are free (L02 owns 19..25).
var L03Meta = map[string]int{"xmax": 3, "dock": 4, "z0": 6, "z1": 7,
	"bcol": 26, "bmin": 27, "bmax": 28, "bspd": 29, "mode": 30}

// Values holds the metadata fields of one room by Meta name (missing = 0).
type Values map[string]int

// Ground is the standing y (PM byte) on a platform in this tile row.
func Ground(row int) int { return (row + 1) * 8 }

// Platform is a run of one-way Floor tiles: row, first and last column.
type Platform struct{ Row, C0, C1 int }

// RackSpec is a rack drawn by Build: left, top, width, bottom (seed = left).
type RackSpec struct{ Left, Top, Width, Bottom int }

// Spec describes one room for Build.
type Spec struct {
	Conduits  []int
	Platforms []Platform
	Racks     []RackSpec
	Decor     func(*L.Screen) // drawn before the platforms
	Extra     func(*L.Screen) // drawn after the platforms
	L03       map[string]int  // level 03 extra metadata (see L03Meta)
	Meta      Values
}

// Build renders one 1024-byte room chunk.
func Build(sp Spec) []byte {
	screen := L.Shell(nil, sp.Conduits)
	for _, r := range sp.Racks {
		L.Rack(screen, r.Left, r.Top, r.Width, r.Bottom, r.Left)
	}
	if sp.Decor != nil {
		sp.Decor(screen)
	}
	for _, p := range sp.Platforms {
		for c := p.C0; c <= p.C1; c++ {
			screen[p.Row][c] = L.Floor
		}
	}
	if sp.Extra != nil {
		sp.Extra(screen)
	}
	if sp.Meta["exit_y"] != 0 {
		exitRow := sp.Meta["exit_y"]/8 - 1
		for row := exitRow - 3; row < exitRow; row++ {
			if sp.Meta["swy"] != 0 {
				screen[row][39] = L.Wall
			} else {
				screen[row][39] = L.Hatch
			}
		}
	}
	m := Values{"sx": 8, "sy": Ground(22)}
	for k, v := range sp.Meta {
		m[k] = v
	}
	raw := make([]byte, 64)
	for i, k := range Meta {
		if m[k] < 0 || m[k] > 255 {
			panic(fmt.Sprintf("platlevels: meta %s=%d out of byte range", k, m[k]))
		}
		raw[i] = byte(m[k])
	}
	for name, value := range sp.L03 {
		if raw[L03Meta[name]] != 0 || sp.Meta["lw"] != 0 {
			panic("platlevels: L03 meta overlaps " + name)
		}
		raw[L03Meta[name]] = byte(value)
	}
	return append(screen.Bytes(), raw...)
}

// FloorAll is a full-width floor on row 22.
var FloorAll = []Platform{{22, 0, 39}}

// Pit is the nonwalkable column range of room 3 of level 03 (wider than any jump).
var Pit = [2]int{17, 32}

func computer(s *L.Screen) {
	L.Frame(s, 35, 7, 39, 13, func(x, y int) int { return pick((x*y)%3 != 0, L.TxtB, L.TxtA) })
	// Main computer unit: below and right of the shortened shelf.
	L.Frame(s, 35, 17, 38, 21, func(x, y int) int { return pick(y == 18, L.Panel, L.TxtB) })
}

func pick(cond bool, a, b int) int {
	if cond {
		return a
	}
	return b
}

func charger(s *L.Screen) { // marked alcove: robot base cols 22..24
	for c := 22; c < 25; c++ {
		s[20][c], s[21][c] = L.DockPlate, L.DockBay
	}
}

func reader(s *L.Screen) { // service reader over the vacuum's lane
	for c := 12; c < 15; c++ {
		s[20][c], s[21][c] = L.Relay, L.ReadStrip
	}
	for c := 23; c < 33; c++ {
		s[22][c], s[23][c] = L.Void, L.Void
	}
}

func pit(s *L.Screen) {
	for c := Pit[0]; c <= Pit[1]; c++ {
		s[22][c] = L.Beam | L.Inv
	}
}

// hang draws machinery hung from the ceiling (decorative, never supporting).
func hang(s *L.Screen, left, bottom, width int) {
	L.Rack(s, left, 4, width, bottom, left+bottom)
}

func rack(s *L.Screen, left, top, width, bottom, seed int) { L.Rack(s, left, top, width, bottom, seed) }

// CrumbRun is a crumbling catwalk run: row, first and last column.
type CrumbRun struct{ Row, C0, C1 int }

// crumbs draws CRUMB tiles over everything.
func crumbs(runs ...CrumbRun) func(*L.Screen) {
	return func(s *L.Screen) {
		for _, r := range runs {
			for c := r.C0; c <= r.C1; c++ {
				s[r.Row][c] = L.Crumb
			}
		}
	}
}

func conduitCols(s *L.Screen, r0, r1 int, cols ...int) {
	for r := r0; r < r1; r++ {
		for _, c := range cols {
			s[r][c] = L.Conduit
		}
	}
}

func room21(s *L.Screen) { // service shaft: zigzag ledges hang on cabinets, controller on the top-left ledge
	hang(s, 2, 8, 4)
	hang(s, 13, 7, 3)
	hang(s, 21, 9, 4)
	rack(s, 28, 5, 5, 21, 3)
	rack(s, 7, 13, 6, 21, 5)
	rack(s, 17, 15, 3, 17, 7)
	rack(s, 24, 17, 3, 19, 9)
	rack(s, 17, 19, 3, 21, 11)
	conduitCols(s, 10, 22, 32, 35) // lift guide rails
	conduitCols(s, 5, 13, 36)
}

func room21Late(s *L.Screen) {
	s[12][9] = L.Ctrl | L.Inv // controller console, unpowered (red) until FIRE
}

func room22(s *L.Screen) { // catwalks: crumbling shortcut above, stable stair below
	hang(s, 1, 6, 4)
	hang(s, 14, 8, 3)
	hang(s, 22, 7, 4)
	hang(s, 31, 8, 3)
	rack(s, 3, 14, 4, 21, 2)
	rack(s, 12, 16, 4, 21, 6)
	rack(s, 16, 14, 5, 21, 4)
	rack(s, 34, 12, 5, 21, 8)
	rack(s, 24, 20, 3, 21, 9)
	conduitCols(s, 15, 22, 10, 22)
}

func room23(s *L.Screen) { // cooling channel: fan on the left, airflow band over the gap
	hang(s, 1, 9, 4)
	hang(s, 20, 7, 4)
	hang(s, 31, 6, 4)
	rack(s, 1, 13, 6, 21, 2)
	rack(s, 29, 11, 5, 21, 4)
	rack(s, 36, 11, 4, 21, 8)
	L.Frame(s, 6, 6, 9, 9, func(x, y int) int { return L.Panel }) // fan housing
	for i, rc := range [4][2]int{{7, 7}, {7, 8}, {8, 7}, {8, 8}} {
		s[rc[0]][rc[1]] = L.FanA[i]
	}
}

func room24(s *L.Screen) { // roof lift: shaft, controller ledge, catwalk to the roof hatch
	hang(s, 1, 5, 4)
	hang(s, 8, 5, 3)
	rack(s, 2, 11, 5, 21, 2)
	rack(s, 20, 12, 5, 21, 6)
	rack(s, 27, 13, 4, 21, 3)
	rack(s, 34, 10, 5, 21, 9)
	conduitCols(s, 8, 22, 12, 16)
}

func room24Late(s *L.Screen) { s[10][26] = L.Ctrl | L.Inv }

func room41(s *L.Screen) {
	hang(s, 3, 13, 4)
	hang(s, 12, 10, 4)
	rack(s, 18, 17, 5, 21, 65)
	rack(s, 27, 19, 4, 21, 66)
	conduitCols(s, 7, 22, 31, 35)
	L.Frame(s, 18, 5, 26, 10, func(x, y int) int { return pick((x+y)%3 != 0, L.TxtB|L.Inv, L.TxtA) })
}

func room42(s *L.Screen) {
	hang(s, 2, 7, 4)
	hang(s, 12, 9, 4)
	rack(s, 18, 15, 4, 21, 67)
	rack(s, 26, 18, 4, 21, 68)
	computer(s)
}

func serviceRoom(s *L.Screen) {
	hang(s, 1, 14, 6)
	L.Frame(s, 11, 5, 18, 10, func(x, y int) int { return pick((x+y)%3 != 0, L.TxtA, L.TxtB) })
	rack(s, 20, 4, 7, 12, 71)
	rack(s, 30, 7, 7, 17, 73)
	rack(s, 8, 13, 3, 18, 74)
	for c, ch := range "SCAN" {
		s[18][11+c] = int(ch) - 32
	}
	for c, ch := range "EXIT" {
		s[18][34+c] = int(ch) - 32
	}
	conduitCols(s, 4, 17, 28)
}

// two builds a level 02 room: decor, then crumbling runs and the late overlay.
func two(builder, late func(*L.Screen), crumb []CrumbRun, platforms []Platform, meta Values) []byte {
	extra := func(s *L.Screen) {
		crumbs(crumb...)(s)
		if late != nil {
			late(s)
		}
	}
	return Build(Spec{Platforms: platforms, Decor: builder, Extra: extra, Meta: meta})
}

func cat(parts ...[]Platform) []Platform {
	var out []Platform
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// Rooms are the ten 1024-byte room chunks: level 02 (4), level 03 (4), level 04 (2).
var Rooms = buildRooms()

func buildRooms() [][]byte {
	rooms := [][]byte{
		// ---- Level 02 ----
		two(room21, room21Late, nil,
			cat(FloorAll, []Platform{{20, 22, 25}, {18, 16, 19}, {16, 23, 26}, {14, 16, 19}, {12, 6, 13}, {13, 35, 39}}),
			Values{"exit_y": Ground(13), "swy": Ground(12), "swc": 9, "lrow": 22, "lcol": 33, "lw": 2,
				"lmin": 13, "lmax": 22, "lpow": 1, "lspd": 18}),
		two(room22, nil, []CrumbRun{{13, 10, 13}, {12, 17, 20}, {11, 25, 28}},
			cat(FloorAll, []Platform{{13, 0, 6}, {20, 26, 29}, {18, 32, 35}, {16, 26, 29}, {14, 32, 35}, {11, 33, 39}}),
			Values{"exit_y": Ground(11), "sy": Ground(13)}),
		two(room23, nil, nil,
			cat(FloorAll, []Platform{{11, 0, 9}, {11, 18, 25}, {9, 29, 39}, {19, 12, 15}, {16, 17, 20}, {13, 22, 25}}),
			Values{"exit_y": Ground(9), "sy": Ground(11), "fan": 10, "fc1": 18, "fr0": 5, "fr1": 11}),
		two(room24, room24Late, []CrumbRun{{6, 18, 21}, {6, 25, 28}},
			cat(FloorAll, []Platform{{9, 0, 7}, {10, 17, 31}, {6, 33, 39}}),
			Values{"exit_y": Ground(6), "sy": Ground(9), "swy": Ground(10), "swc": 26, "lrow": 22,
				"lcol": 13, "lw": 3, "lmin": 6, "lmax": 22, "lspd": 12}),
		// ---- Level 03 ----
		// 1 cleaning corridor: niches at row 20, podium exit; one patrolling vacuum
		Build(Spec{Conduits: []int{5, 20}, Platforms: cat(FloorAll, []Platform{{20, 10, 14}, {20, 18, 22}, {20, 35, 39}}),
			Racks: []RackSpec{{28, 6, 5, 18}},
			Meta:  Values{"arow": 21, "amin": 16, "amax": 22, "aspd": 6, "acol": 16, "exit_y": Ground(20)}}),
		// 2 charger puzzle: lasers over the aisle until the struck vacuum docks in the bay
		Build(Spec{Conduits: []int{12, 38}, Platforms: FloorAll, Extra: charger,
			Meta: Values{"arow": 21, "amin": 14, "amax": 20, "aspd": 5, "acol": 14, "exit_y": Ground(22)},
			L03:  map[string]int{"xmax": 28, "dock": 22, "z0": 6, "z1": 33, "mode": 1}}),
		// 3 signature ride over the pit with a separate pursuer
		Build(Spec{Conduits: []int{3, 36}, Platforms: []Platform{{22, 0, Pit[0] - 1}, {22, Pit[1] + 1, 39}, {20, 34, 39}},
			Extra: pit,
			Meta:  Values{"arow": 21, "amin": 6, "amax": 29, "aspd": 12, "acol": 8, "exit_y": Ground(20)},
			L03:   map[string]int{"bcol": 0, "bmin": 0, "bmax": 26, "bspd": 6}}),
		// 4 scan -> one clear carrier -> service balcony -> jump to the floor-height exit
		Build(Spec{Conduits: []int{10, 33}, Platforms: []Platform{{22, 0, 22}, {22, 33, 39}, {18, 23, 27}},
			Decor: serviceRoom, Extra: reader,
			Meta: Values{"arow": 21, "amin": 8, "amax": 10, "aspd": 8, "acol": 8, "exit_y": Ground(22)},
			L03:  map[string]int{"xmax": 19, "z0": 12, "z1": 12, "mode": 2}}),
		// ---- Level 04 ----
		Build(Spec{Conduits: []int{4, 36}, Platforms: cat(FloorAll, []Platform{{20, 4, 8}, {18, 11, 15}, {16, 18, 22},
			{18, 27, 31}, {9, 35, 39}}), Decor: room41,
			Meta: Values{"sx": 0, "h0": 7, "h1": 34, "lrow": 18, "lcol": 32, "lw": 3, "lmin": 9, "lmax": 18,
				"lspd": 12, "exit_y": Ground(9)}}),
		Build(Spec{Conduits: []int{3, 25}, Platforms: cat(FloorAll, []Platform{{9, 0, 6}, {11, 11, 16}, {13, 20, 24},
			{16, 24, 28}, {14, 30, 34}}), Decor: room42,
			Meta: Values{"sx": 0, "sy": Ground(9), "h0": 7, "h1": 34, "glass": 33, "dlg": Ground(13), "desk": Ground(14)}}),
	}
	if len(rooms) != 10 {
		panic("platlevels: want 10 rooms")
	}
	for _, r := range rooms {
		if len(r) != 1024 {
			panic("platlevels: room is not 1024 bytes")
		}
	}
	return rooms
}

// Cry is the strained synthetic shriek followed by irregular sputters that
// drop and die (4-bit volumes); CryDefault gives the shipped parameters.
func Cry(n int, rate float64) []byte {
	rnd := pyrand.New(65)
	out := make([]byte, 0, n)
	phase := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		var f, amp float64
		if t < 0.30 { // shriek: wavering, slightly falling pitch
			f = 1500 - float64(700*t) + float64(160*math.Sin(float64(i)/25))
			amp = 7
		} else { // sputters: noise bursts and gaps, pitch collapses, fades out
			f = float64(800*math.Pow(1-t, 2)) + 60
			gate := 0.2
			if (i/60+rnd.RandInt(0, 1))%3 != 0 {
				gate = 1
			}
			amp = float64(7*math.Pow(1-t, 0.6)) * gate
		}
		phase = math.Mod(phase+f/rate, 1)
		v := -1.0
		if phase < 0.5 {
			v = 1
		}
		if t >= 0.30 && rnd.Random() < float64(0.35*t) {
			v = []float64{-1, 1}[rnd.Choice(2)]
		}
		if t < 0.97 {
			x := math.RoundToEven(7 + float64(amp*v))
			out = append(out, byte(math.Max(0, math.Min(15, x))))
		} else {
			out = append(out, 0)
		}
	}
	return out
}

// DefaultCry is the shipped computer-death cry (assets/plat-cry.pcm).
func DefaultCry() []byte { return Cry(2560, 5200) }

// WriteAssets writes plat-rooms.bin and plat-cry.pcm into the repository assets directory.
func WriteAssets() error {
	var all []byte
	for _, r := range Rooms {
		all = append(all, r...)
	}
	if err := os.WriteFile(k65src.Path("assets", "plat-rooms.bin"), all, 0o644); err != nil {
		return err
	}
	return os.WriteFile(k65src.Path("assets", "plat-cry.pcm"), DefaultCry(), 0o644)
}
