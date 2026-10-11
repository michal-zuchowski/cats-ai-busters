// Package l03model is a frame-by-frame Go port of the level-03 (cur_level == 2) platformer logic
// of src/plat_*.k65.
//
// The model mirrors plat_frame: plat_input, plat_phys, plat_actor, plat_pursuer, paw_both,
// plat_hit, plat_beam, draw_player's counters. Data tables (ActT, PawX/PawY/PawK/PawE,
// PixelMask), the room art + metadata (assets/plat-rooms.bin) and the font (assets/term-font.pic)
// are read from the real files; the control flow is ported function by function and each port is
// named after its K65 function so a reviewer can diff it. Nothing is compiled or executed from K65.
package l03model

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

// Root is the repository root.
var Root = k65src.Root()

// SRC is the concatenated K65 source.
var SRC = k65src.Read()

var (
	// ROOMS is assets/plat-rooms.bin.
	ROOMS = mustRead("assets/plat-rooms.bin")
	// Font is assets/term-font.pic.
	Font = mustRead("assets/term-font.pic")
)

var metaNames = []string{"sx", "sy", "fan", "lrow", "lcol", "lw", "lmin", "lmax", "arow", "amin", "amax",
	"aspd", "acol", "h0", "h1", "glass", "dlg", "desk", "exit_y"}

var l03Meta = map[string]int{"xmax": 3, "dock": 4, "z0": 6, "z1": 7, "bcol": 26, "bmin": 27, "bmax": 28,
	"bspd": 29, "mode": 30}

const (
	base      = 4 // plat-rooms.bin chunk of L03 room 0 (LvBase[2] - 5)
	roomCount = 4 // LvRooms[2]
	// Floor is the floor glyph.
	Floor  = 70
	liftG  = 94
	beamG  = 79
	wallG  = 81
	rows   = 24
	cols   = 40
	chunk  = 1024
	pawRow = 24
)

var baseGlyphs = [3]int{95, 124, 125}

var (
	ActT      = data("ActT")
	pawX      = data("PawX")
	pawY      = data("PawY")
	pawK      = data("PawK")
	pawE      = data("PawE")
	pixelMask = data("PixelMask")
)

func init() {
	lb, lr := data("LvBase"), data("LvRooms")
	if lb[2]-lb[1] != base || lb[1] != 5 || lr[2] != roomCount {
		panic("l03model: unexpected LvBase/LvRooms")
	}
}

func mustRead(rel string) []int {
	b, err := os.ReadFile(k65src.Path(rel))
	if err != nil {
		panic(err)
	}
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

var (
	commentRe = regexp.MustCompile(`//[^\n]*`)
	numRe     = regexp.MustCompile(`\d+`)
)

func data(name string) []int {
	m := regexp.MustCompile(`(?s)data ` + name + ` \{([^}]*)\}`).FindStringSubmatch(SRC)
	if m == nil {
		panic("l03model: no data " + name)
	}
	var out []int
	for _, n := range numRe.FindAllString(commentRe.ReplaceAllString(m[1], ""), -1) {
		v, _ := strconv.Atoi(n)
		out = append(out, v)
	}
	return out
}

func s8(v int) int {
	if v&0x80 != 0 {
		return v - 256
	}
	return v
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// wrap emulates Python's negative list indexing.
func wrap(i, n int) int {
	if i < -n || i >= n {
		panic(fmt.Sprintf("index %d out of range %d", i, n))
	}
	if i < 0 {
		return i + n
	}
	return i
}

// Input is one frame of joystick state.
type Input struct{ Up, Left, Right, Fire bool }

// Game is the model state.
type Game struct {
	Facing, FirePrev, LevelFinished, Rtclok int
	Events                                  []string
	CurRoom                                 int
	Art, Scr                                [rows][cols]int
	M                                       map[string]int
	PlayerX, PlayerY, PlayerCol             int
	Air, Vy, Jprep, Lnd                     int
	PawT, PawKind, PawHit, Gt               int
	EMode, EBspd                            int
	// per-slot state, indexed by slot (0 = vacuum, 21 = pursuer)
	ActCol, ActDir, ActWob [22]int
	ActT                   int
	Ride, Carrier, Slot    int
	Dk, Gate, Bt, Bw, Bk   int
	Kk                     int
}

// New creates a game in the given room.
func New(room int) *Game {
	g := &Game{FirePrev: 1}
	g.EnterRoom(room, 0)
	return g
}

// Clone returns an independent copy.
func (g *Game) Clone() *Game {
	c := *g
	c.Events = append([]string(nil), g.Events...)
	return &c
}

func (g *Game) EnterRoom(room, entryCol int) {
	g.CurRoom = room
	ch := ROOMS[(base+room)*chunk : (base+room+1)*chunk]
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			g.Art[r][c] = ch[r*cols+c]
		}
	}
	g.Scr = g.Art
	raw := ch[960:]
	g.M = map[string]int{}
	for i, k := range metaNames {
		g.M[k] = raw[i]
	}
	for k, v := range l03Meta {
		g.M[k] = raw[v]
	}
	if g.M["fan"] != 0 || g.M["lw"] != 0 || g.M["h0"] != 0 {
		panic("l03model: unexpected room meta")
	}
	g.PlayerX, g.PlayerY = entryCol, g.M["sy"]
	g.Air, g.Vy, g.Jprep, g.Lnd, g.PawT, g.PawKind, g.PawHit = 0, 0, 0, 0, 0, 0, 0
	g.Gt = 0
	g.SetCol()
	g.EMode, g.EBspd = g.M["mode"], g.M["bspd"]
	g.ActInit()
}

func (g *Game) SetCol() {
	g.PlayerCol = (((g.PlayerX + 2) & 0xFF) >> 2) + 2
}

func (g *Game) cell(row, col int) *int {
	return &g.Scr[wrap(row, rows)][wrap(col, cols)]
}

func (g *Game) putRun(row, col, cnt, tile int) {
	for c := col; c < col+cnt; c++ {
		if tile == 0 {
			*g.cell(row, c) = g.Art[wrap(row, rows)][wrap(c, cols)]
		} else {
			*g.cell(row, c) = tile
		}
	}
}

func (g *Game) drawAct(slot int) {
	wm := 0
	if g.ActWob[slot] != 0 {
		wm = 0x80
	}
	if g.Dk != 0 && slot == 0 && g.Rtclok&8 == 0 {
		wm = 0x80
	}
	b := 0
	if slot == 21 && g.Bw != 0 {
		b = 6
	}
	col := g.ActCol[slot]
	for i := 0; i < 3; i++ {
		*g.cell(g.M["arow"], col+i) = ActT[b+i] | wm
	}
	for i := 0; i < 3; i++ {
		*g.cell(g.M["arow"]+1, col+i) = ActT[b+3+i] | wm
	}
}

func (g *Game) eraseAct(slot int) {
	col := g.ActCol[slot]
	g.putRun(g.M["arow"], col, 3, 0)
	g.putRun(g.M["arow"]+1, col, 3, 0)
}

func (g *Game) drawGate(tile int) {
	row := (g.M["exit_y"] >> 3) - 4
	for i := 0; i < 3; i++ {
		g.putRun(row+i, 39, 1, tile)
	}
}

func (g *Game) drawReader() {
	if g.EMode&2 != 0 {
		t := 83
		if g.Gate != 0 {
			t = 84
		}
		g.putRun(20, g.M["z0"], 3, t)
	}
}

func (g *Game) ActInit() {
	g.ActCol[0], g.ActCol[21] = g.M["acol"], g.M["bcol"]
	g.ActDir[0], g.ActDir[21] = 0, 0
	g.ActT = 0
	g.ActWob[0], g.ActWob[21] = 0, 0
	g.Ride, g.Carrier, g.Slot, g.Dk, g.Gate, g.Bt, g.Bw, g.Bk = 0, 0, 0, 0, 0, 0, 0, 0
	g.Kk = 0
	if g.M["arow"] == 0 {
		return
	}
	g.drawReader()
	g.drawAct(0)
	if g.EBspd != 0 {
		g.drawAct(21)
	}
	if g.EMode&3 != 0 {
		g.drawGate(wallG)
	}
}

func (g *Game) PlatReset() {
	if g.M["arow"] == 0 {
		return
	}
	g.eraseAct(0)
	if g.EBspd != 0 {
		g.eraseAct(21)
	}
	if g.EMode&3 != 0 {
		g.drawGate(0)
	}
	g.ActInit()
}

func (g *Game) Respawn() {
	g.Events = append(g.Events, "respawn")
	g.PlayerX, g.PlayerY = g.M["sx"], g.M["sy"]
	g.Air, g.Vy, g.Jprep, g.Lnd, g.PawT, g.PawKind, g.PawHit = 0, 0, 0, 0, 0, 0, 0
	g.SetCol()
	g.PlatReset()
}

func (g *Game) vacuumRoof() int { return (g.M["arow"] << 3) + 3 }

func (g *Game) roofOn(slot int) bool {
	return (g.PlayerCol-g.ActCol[slot])&0xFF < 3
}

func (g *Game) OnVacuum() int {
	if g.M["arow"] == 0 || g.vacuumRoof() != g.PlayerY {
		return 0
	}
	if g.roofOn(0) {
		g.Carrier = 0
		return 1
	}
	if g.EBspd == 0 || !g.roofOn(21) {
		return 0
	}
	if g.EMode&2 != 0 && g.Gate == 0 {
		return 0
	}
	g.Carrier = 21
	return 21
}

func (g *Game) Under() int {
	if g.OnVacuum() != 0 {
		return 1
	}
	if g.PlayerY&7 != 0 {
		return 0
	}
	row, col := (g.PlayerY>>3)-1, g.PlayerCol
	if row >= 24 {
		return 0 // the real code reads meta bytes here; none is a support glyph
	}
	gl := *g.cell(row, col) & 0x7F
	if gl == Floor || gl == liftG {
		return 1
	}
	for _, b := range baseGlyphs {
		if gl == b {
			return b2i(g.Art[wrap(row, rows)][wrap(col, cols)]&0x7F == Floor)
		}
	}
	return 0
}

func (g *Game) PlatInput(inp Input) {
	if g.Jprep != 0 {
		g.Jprep--
		if g.Jprep != 0 {
			return
		}
		g.Vy, g.Air, g.Gt, g.PawT = 0xFC, 1, 0, 0
	}
	if inp.Up && g.Air == 0 && g.Lnd == 0 {
		g.Jprep = 3
		return
	}
	g.PlatFire(inp)
	if g.LevelFinished != 0 || g.Lnd != 0 || g.PawT != 0 {
		return
	}
	if g.Air == 0 && g.Rtclok&1 != 0 {
		return
	}
	if inp.Left {
		g.Facing = 1
		if g.PlayerX == 0 {
			return
		}
		g.PlayerX--
		g.SetCol()
		return
	}
	if inp.Right {
		g.Facing = 0
		if g.PlayerX == 140 {
			g.PlatExit()
			return
		}
		g.PlayerX++
		g.SetCol()
	}
}

func (g *Game) PlatFire(inp Input) {
	if !inp.Fire {
		g.FirePrev = 1
		return
	}
	if g.FirePrev == 0 {
		return
	}
	g.FirePrev = 0
	if g.Air != 0 || g.PawT != 0 {
		return
	}
	g.Swat()
}

func (g *Game) Swat() {
	g.PawT, g.PawKind, g.PawHit = 10, 0, 0
	if g.OnVacuum() != 0 {
		g.PawKind = 2
	}
}

func (g *Game) PlatExitAllowed() int {
	if g.EMode&3 != 0 && g.Gate == 0 {
		return 0
	}
	if g.M["exit_y"] == 0 {
		return 1
	}
	return b2i(g.Air == 0 && g.PlayerY == g.M["exit_y"])
}

func (g *Game) PlatExit() {
	if g.PlatExitAllowed() == 0 {
		return
	}
	if g.CurRoom == roomCount-1 {
		g.LevelFinished = 1
		g.Events = append(g.Events, "level_end")
		return
	}
	g.Events = append(g.Events, fmt.Sprintf("room%d", g.CurRoom+1))
	g.EnterRoom(g.CurRoom+1, 0)
}

func (g *Game) PlatPhys() {
	if g.Air == 0 {
		if g.Under() != 0 {
			return
		}
		g.Jprep, g.Lnd, g.PawT = 0, 0, 0
		g.Air, g.Vy = 1, 0
	}
	g.Gt++
	if g.Gt == 3 {
		g.Gt = 0
		if g.Vy != 5 {
			g.Vy = (g.Vy + 1) & 0xFF
		}
		if g.Vy == 0 {
			g.Gt = 2
		}
	}
	if g.Vy&0x80 != 0 {
		g.PlayerY -= -s8(g.Vy)
	} else if g.Vy != 0 {
		cnt := g.Vy
		for cnt != 0 {
			g.PlayerY++
			if g.Under() != 0 {
				g.Air, g.Vy = 0, 0
				cnt = 1
				g.Lnd = 10
			}
			cnt--
		}
	}
	if g.PlayerY >= 196 {
		g.Respawn()
	}
}

func (g *Game) carry(slot int) {
	if g.Ride == 0 || g.Carrier != slot {
		return
	}
	if g.Kk == 1 && g.PlayerX < 136 {
		g.PlayerX += 4
	}
	if g.Kk == 2 && g.PlayerX >= 4 {
		g.PlayerX -= 4
	}
	g.SetCol()
}

func (g *Game) dockTick() {
	g.Dk--
	if g.Dk == 0 {
		g.ActWob[0], g.ActDir[0], g.ActT = 24, 1, 0
		g.Events = append(g.Events, "undock")
	}
	g.drawAct(0)
}

func (g *Game) aStep() {
	m := g.M
	wm := m["amax"]
	if g.EMode != 0 {
		if g.ActWob[0] != 0 || (g.EMode&2 != 0 && g.Gate != 0) {
			wm = m["xmax"]
		}
	}
	if g.ActDir[0] == 0 {
		if g.ActCol[0] >= wm {
			g.ActDir[0] = 1
		} else {
			g.ActCol[0]++
			g.Kk = 1
		}
	} else if m["amin"] >= g.ActCol[0] {
		g.ActDir[0] = 0
	} else {
		ok := true
		if g.EBspd != 0 && g.ActCol[0]-g.ActCol[21] < 4 {
			ok = false
		}
		if ok {
			g.ActCol[0]--
			g.Kk = 2
		}
	}
}

func (g *Game) AEvents() {
	m := g.M
	if g.EMode&1 != 0 && g.ActDir[0] == 0 && g.ActWob[0] != 0 && g.ActCol[0] >= m["dock"] {
		g.Dk, g.ActWob[0] = 240, 0
		g.Gate = 1
		g.drawGate(0)
		g.Events = append(g.Events, "dock")
	}
	if g.EMode&2 == 0 || g.Gate != 0 {
		return
	}
	if m["z0"] <= g.ActCol[0] && g.ActCol[0] <= m["z1"] {
		g.Gate = 1
		g.drawGate(0)
		g.drawReader()
		g.Events = append(g.Events, "gate")
	}
}

func (g *Game) PlatActor() {
	if g.M["arow"] == 0 {
		return
	}
	if g.ActWob[0] != 0 {
		g.ActWob[0]--
	}
	if g.ActWob[21] != 0 {
		g.ActWob[21]--
	}
	g.Ride = 0
	if g.Air == 0 && g.OnVacuum() != 0 {
		g.Ride = 1
	}
	g.Slot = 0
	if g.Dk != 0 {
		g.dockTick()
		return
	}
	g.ActT++
	if g.ActT < g.M["aspd"] {
		return
	}
	g.ActT = 0
	g.eraseAct(0)
	g.Kk = 0
	g.aStep()
	g.drawAct(0)
	g.AEvents()
	g.carry(0)
}

func (g *Game) bRight() {
	if g.ActCol[21] >= g.M["bmax"] {
		return
	}
	if g.ActCol[0]-g.ActCol[21] < 4 {
		return
	}
	g.ActCol[21]++
	g.Kk = 1
}

func (g *Game) bLeft() {
	if g.M["bmin"] >= g.ActCol[21] {
		return
	}
	g.ActCol[21]--
	g.Kk = 2
}

func (g *Game) bMove() {
	if g.ActWob[21] != 0 {
		if g.ActDir[21] == 0 {
			g.bRight()
		} else {
			g.bLeft()
		}
		return
	}
	if g.Ride != 0 {
		g.bRight()
		return
	}
	if g.M["bcol"] < g.ActCol[21] {
		g.bLeft()
	}
	if g.ActCol[21] < g.M["bcol"] {
		g.bRight()
	}
}

func (g *Game) BTouch() int {
	if g.Lnd != 0 || g.Jprep != 0 || g.Ride == 0 || g.Carrier != 0 || g.ActWob[21] != 0 {
		return 0
	}
	x := g.PlayerX
	if g.Facing != 0 {
		x += 2
	}
	return b2i(x < g.ActCol[21]*4+10)
}

func (g *Game) PlatPursuer() {
	if g.EBspd == 0 {
		return
	}
	if g.EMode&2 != 0 && g.Gate == 0 {
		return
	}
	g.Slot = 21
	g.Bw = 0
	if g.Ride != 0 {
		g.Bw = 1
	}
	if g.ActCol[21] != g.M["bcol"] {
		g.Bw = 1
	}
	if g.Bw != 0 {
		if g.BTouch() == 0 {
			if g.Bk-1 > 0 {
				g.Bk--
			} else {
				g.Bk = 0
			}
		} else {
			g.Bk++
			if g.Bk >= 48 {
				g.Slot = 0
				g.Respawn()
				return
			}
		}
	}
	g.Bt++
	if g.Bt < g.M["bspd"] {
		g.Slot = 0
		return
	}
	g.Bt = 0
	g.eraseAct(21)
	g.Kk = 0
	g.bMove()
	g.drawAct(21)
	g.carry(21)
	g.Slot = 0
}

func (g *Game) pawContact(slot int) {
	if g.PawT < 4 || g.PawT >= 8 || g.PawHit != 0 || g.M["arow"] == 0 {
		return
	}
	if g.Air != 0 || g.Jprep != 0 {
		return
	}
	if slot != 0 && g.EBspd == 0 {
		return
	}
	if g.Ride != 0 && g.Carrier == slot {
		return
	}
	kk := pawK[g.PawKind]
	for {
		px := pawX[kk]
		if g.Facing != 0 {
			px = ((^px & 0xFF) + 16) & 0xFF
		}
		px = (px + g.PlayerX - g.ActCol[slot]*4) & 0xFF
		if px < 12 {
			py := (pawY[kk] + g.PlayerY - (g.M["arow"]*8 + pawRow)) & 0xFF
			if py < 16 {
				tmp := 0
				if py&8 != 0 {
					tmp = 3
				}
				if slot != 0 && g.Bw != 0 {
					tmp += 6
				}
				glyph := ActT[(px>>2)+tmp]
				if Font[glyph*8+(py&7)]&pixelMask[px&3] != 0 {
					g.PawHit = 1
					d := g.Facing
					if g.PawKind != 0 {
						d = g.Facing ^ 1
					}
					g.ActDir[slot] = d
					g.ActWob[slot] = 40
					g.Bk = 0
					g.drawAct(slot)
					g.Events = append(g.Events, fmt.Sprintf("hit%d", slot))
					return
				}
			}
		}
		kk++
		if pawE[g.PawKind] == kk {
			return
		}
	}
}

func (g *Game) PawBoth() {
	g.pawContact(0)
	g.pawContact(21)
}

func (g *Game) hitCheck(slot int) {
	if g.ActWob[slot] != 0 {
		return
	}
	if slot == 0 {
		if g.Dk != 0 {
			return
		}
	} else if g.Bw == 0 {
		return
	}
	if g.PlayerY < g.vacuumRoof()+1 {
		return
	}
	if (g.PlayerCol-g.ActCol[slot])&0xFF >= 3 {
		return
	}
	g.Respawn()
}

func (g *Game) PlatHit() {
	if g.M["arow"] == 0 {
		return
	}
	g.hitCheck(0)
	if g.EBspd != 0 {
		g.hitCheck(21)
	}
}

func (g *Game) PlatBeam() {
	if g.EMode&1 == 0 {
		return
	}
	tile := 0xCF
	if g.Dk != 0 {
		tile = 0
	} else if g.Rtclok&16 != 0 {
		tile = 79
	}
	g.putRun(19, g.M["z0"], g.M["z1"]-g.M["z0"]+1, tile)
	if g.Dk != 0 {
		return
	}
	if g.PlayerCol < g.M["z0"] || g.M["z1"] < g.PlayerCol {
		return
	}
	if g.PlayerY < 153 || g.PlayerY >= 184 {
		return
	}
	g.Respawn()
}

func (g *Game) drawPlayer() {
	if g.Jprep != 0 || g.Air != 0 {
		return
	}
	if g.PawT != 0 {
		g.PawT--
		return
	}
	if g.Lnd != 0 {
		g.Lnd--
	}
}

// Frame advances the model by one frame.
func (g *Game) Frame(inp Input) {
	g.Rtclok = (g.Rtclok + 1) & 0xFF
	g.PlatInput(inp)
	if g.LevelFinished != 0 {
		return
	}
	g.PlatPhys()
	g.PlatActor()
	g.PlatPursuer()
	g.PawBoth()
	g.PlatHit()
	g.PlatBeam()
	g.drawPlayer()
}

// Run advances n frames with a constant input.
func (g *Game) Run(inp Input, n int) {
	for i := 0; i < n; i++ {
		g.Frame(inp)
	}
}
