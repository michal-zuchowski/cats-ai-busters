package tests

// Route / behaviour checks for the four level-03 rooms, run on the faithful frame model
// (tests/l03model) over the generated maps with the unchanged jump physics. Bots are plain
// input policies; nothing here pokes an objective flag - docks, gates, ride and shoves happen only
// through the model's own contact/actor code.

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tests/l03model"
)

type l03Game = l03model.Game
type l03Input = l03model.Input
type l03Pol func(g *l03Game, i int) l03Input

var (
	l03None = l03Input{}
	l03R    = l03Input{Right: true}
	l03L    = l03Input{Left: true}
	l03UR   = l03Input{Up: true, Right: true}
	l03F    = l03Input{Fire: true}
	l03U    = l03Input{Up: true}
)

var (
	l03CommentRe = regexp.MustCompile(`//[^\n]*`)
	l03SpaceRe   = regexp.MustCompile(`\s+`)
	l03CharsetRe = regexp.MustCompile(`charset [^\n]*`)
	l03QuoteRe   = regexp.MustCompile(`"([^"\n]*)"`)
)

func l03Body(t *testing.T, name string) string {
	t.Helper()
	m := regexp.MustCompile(`(?s)func ` + name + ` \{(.*?)\n\}`).FindStringSubmatch(l03model.SRC)
	if m == nil {
		t.Fatalf("no func %s in source", name)
	}
	return strings.TrimSpace(l03SpaceRe.ReplaceAllString(l03CommentRe.ReplaceAllString(m[1], ""), " "))
}

func l03Need(t *testing.T, body, sub, what string) {
	t.Helper()
	if !strings.Contains(body, sub) {
		t.Errorf("%s: missing %q", what, sub)
	}
}

func l03Has(ev []string, name string) bool {
	for _, e := range ev {
		if e == name {
			return true
		}
	}
	return false
}

func l03Room(ev []string) bool {
	for _, e := range ev {
		if strings.HasPrefix(e, "room") {
			return true
		}
	}
	return false
}

func l03First(ev []string, n int) []string {
	if len(ev) < n {
		return ev
	}
	return ev[:n]
}

func l03Last(ev []string, n int) []string {
	if len(ev) < n {
		return ev
	}
	return ev[len(ev)-n:]
}

func l03Eq(a, b []string) bool {
	return len(a) == len(b) && (len(a) == 0 || reflect.DeepEqual(a, b))
}

func l03Count(ev []string, name string) int {
	n := 0
	for _, e := range ev {
		if e == name {
			n++
		}
	}
	return n
}

func l03Asset(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(l03model.Root + "/assets/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func l03Run(t *testing.T, g *l03Game, pol l03Pol, n int, stop ...string) ([]string, int) {
	t.Helper()
	if stop == nil {
		stop = []string{"respawn"}
	}
	var ev []string
	i := 0
	for i = 0; i < n; i++ {
		g.Frame(pol(g, i))
		if g.CurRoom == 3 {
			for c := g.ActCol[0]; c < g.ActCol[0]+3; c++ {
				if g.Art[22][c] != l03model.Floor {
					t.Fatalf("R4 robot footprint lost floor support: frame %d col %d", i, g.ActCol[0])
				}
			}
		}
		ev = append(ev, g.Events...)
		g.Events = nil
		stopped := false
		for _, s := range stop {
			if l03Has(ev, s) {
				stopped = true
			}
		}
		if stopped || g.LevelFinished != 0 || l03Room(ev) {
			break
		}
	}
	if i == n {
		i = n - 1
	}
	return ev, i
}

func l03ExitPol(g *l03Game) l03Input {
	if g.M["exit_y"] < 184 && g.PlayerCol >= 30 && g.PlayerY == 184 && g.Air == 0 && g.Jprep == 0 && g.Lnd == 0 {
		return l03UR
	}
	return l03R
}

func l03ExitFn(g *l03Game, i int) l03Input { return l03ExitPol(g) }

func l03SwatPol(dd, stopCol, hold int) l03Pol {
	f := 0
	return func(g *l03Game, i int) l03Input {
		d := g.ActCol[0] - g.PlayerCol
		if f != 0 && i < f+2 {
			return l03None
		}
		if f == 0 && 0 <= d && d <= dd && g.ActDir[0] == 1 && g.PawT == 0 && g.PlayerCol < stopCol {
			f = i
			return l03F
		}
		if hold != 0 && g.PlayerCol >= hold && f == 0 {
			return l03None
		}
		return l03ExitPol(g)
	}
}

func l03RidePol(shoves bool, jumpCol, boardCol, rear int) l03Pol {
	f := 0
	return func(g *l03Game, i int) l03Input {
		d := g.ActCol[0] - g.PlayerCol
		if g.Ride == 0 && g.PlayerY == 168 && g.PlayerCol >= 33 && g.Air == 0 {
			return l03R // on the exit ledge
		}
		if g.Ride == 0 {
			if g.Air != 0 {
				return l03R
			}
			if g.PlayerCol < boardCol {
				return l03R
			}
			if 0 <= d && d <= 4 && g.PlayerY == 184 {
				return l03UR
			}
			return l03None
		}
		if f != 0 && i < f+2 {
			return l03None
		}
		rel := g.PlayerCol - g.ActCol[0]
		if shoves && g.Bk < 6 && g.Air == 0 && g.Jprep == 0 && g.ActCol[0] < jumpCol {
			if rel > rear {
				return l03L
			}
			if rel < rear || g.Facing != 0 {
				return l03R
			}
		}
		if shoves && g.Bk >= 6 && g.Facing != 0 && g.PawT == 0 && g.Air == 0 {
			return l03R
		}
		if shoves && g.Bk >= 6 && g.PawT == 0 && g.Air == 0 && g.Jprep == 0 {
			f = i
			return l03F
		}
		if g.ActCol[0] >= jumpCol {
			return l03UR
		}
		return l03None
	}
}

func l03IdleUntil(g *l03Game, n int, pred func(g *l03Game) bool) bool {
	for i := 0; i < n; i++ {
		if pred(g) {
			return true
		}
		g.Frame(l03None)
	}
	return pred(g)
}

// l03Clean: screen buffer equals the art except where an actor is drawn.
func l03Clean(g *l03Game) bool {
	type rc struct{ r, c int }
	cols := map[rc]bool{}
	for _, s := range []int{0, 21} {
		if s == 21 && g.EBspd == 0 {
			continue
		}
		for r := 0; r < 2; r++ {
			for c := 0; c < 3; c++ {
				cols[rc{g.M["arow"] + r, g.ActCol[s] + c}] = true
			}
		}
	}
	for r := 0; r < 24; r++ {
		for c := 0; c < 40; c++ {
			if cols[rc{r, c}] {
				continue
			}
			expected := g.Art[r][c]
			if g.EMode&3 != 0 && g.Gate == 0 && c == 39 && (g.M["exit_y"]>>3)-4 <= r && r < (g.M["exit_y"]>>3)-1 {
				expected = 81
			}
			if g.EMode&1 != 0 && g.Dk == 0 && r == 19 && g.M["z0"] <= c && c <= g.M["z1"] {
				if g.Rtclok&16 != 0 {
					expected = 79
				} else {
					expected = 0xCF
				}
			}
			if g.EMode&2 != 0 && r == 20 && g.M["z0"] <= c && c < g.M["z0"]+3 {
				if g.Gate != 0 {
					expected = 84
				} else {
					expected = 83
				}
			}
			if g.Scr[r][c] != expected {
				return false
			}
		}
	}
	return true
}

func l03Rows(g *l03Game, col int, rows ...int) bool {
	for _, r := range rows {
		if g.Scr[r][col] != 81 {
			return false
		}
	}
	return true
}

// Bind the hand-written model's critical branches to the K65 implementation.
func TestL03SourceBindings(t *testing.T) {
	src := l03model.SRC
	for _, name := range []string{"paw_both", "plat_pursuer", "plat_beam"} {
		if !strings.HasPrefix(l03Body(t, name), "a=cur_level a?2 != { return }") {
			t.Errorf("%s: missing level-2 guard", name)
		}
	}
	roof := l03Body(t, "roof_on")
	l03Need(t, roof, "a?3 < { a=1 return }", "roof_on")
	if strings.Contains(roof, "a+1") {
		t.Errorf("roof_on contains a+1")
	}
	ae := l03Body(t, "a_events")
	l03Need(t, ae, "a=act_dir a?0 == { a=act_wob", "a_events")
	if strings.Contains(l03Body(t, "a_step"), "goto") {
		t.Errorf("a_step contains goto")
	}
	l03Need(t, ae, "a?m_dock >= { dk=a=240 act_wob=a=0 gate=a=1 tmp=a=0 draw_gate }", "a_events")
	l03Need(t, l03Body(t, "plat_pursuer"), "a?48 >= {", "plat_pursuer")
	l03Need(t, l03Body(t, "plat_beam"), "a?153 < { return } a?184 >= { return }", "plat_beam")
	l03Need(t, l03Body(t, "under"), "a=(ap),y a&0x7F a?70 == { a=1 return }", "under")
	pea := l03Body(t, "plat_exit_allowed")
	l03Need(t, pea, "a=e_mode a&3", "plat_exit_allowed")
	if strings.Contains(pea, "a=dk") {
		t.Errorf("plat_exit_allowed reads dk")
	}
	l03Need(t, pea, "a=gate a?0 == { return }", "plat_exit_allowed")
	for _, n := range []string{"act_init", "plat_reset"} {
		l03Need(t, l03Body(t, n), "a=e_mode a&3", n)
	}
	l03Need(t, l03Body(t, "plat_hud"), "a=e_mode a&1 a?0 != { a=gate a?0 != { x=9 } }", "plat_hud")
	l03Need(t, l03Body(t, "on_vacuum"), "a=e_mode a&2 a?0 != { a=gate a?0 == { return } }", "on_vacuum")
	l03Need(t, l03Body(t, "a_step"), "a=gate a?0 != { a=e_mode a&2 a?0 != { wm=a=m_xmax } }", "a_step")
	if !strings.Contains(src, `"EXIT   "`) || !strings.Contains(src, "HintOff { 0 7 14 21 28 35 42 49 56 63 }") {
		t.Errorf("EXIT hint / HintOff table missing")
	}
	resp := l03Body(t, "respawn")
	if strings.Count(resp, "crumble_reset") != 1 || !strings.Contains(resp, "== { plat_reset }") {
		t.Errorf("respawn: crumble_reset/plat_reset binding wrong: %s", resp)
	}
	frameOrder := l03Body(t, "plat_frame")
	prev := -1
	for _, n := range []string{"plat_phys", "plat_actor", "plat_pursuer", "paw_both", "plat_hit", "plat_beam"} {
		idx := strings.Index(frameOrder, n)
		if idx < 0 || idx < prev {
			t.Errorf("plat_frame order wrong at %s", n)
		}
		prev = idx
	}
	l03Need(t, l03Body(t, "draw_player"), "x?4 >= { x?8 < { a=52 } }", "draw_player")
}

func l03SeatedOverlap(atlas, mapping []byte, px, logical, bcol int) bool {
	sprite := atlas[int(mapping[logical])*128 : (int(mapping[logical])+1)*128]
	for y := 0; y < 24; y++ {
		for x := 0; x < 16; x++ {
			opaque := false
			for _, p := range []int{x / 8, 2 + x/8} {
				if int(sprite[p*32+y])&(128>>(x%8)) != 0 {
					opaque = true
				}
			}
			wx, wy := px+x-bcol*4, 171+y-192
			if opaque && 0 <= wx && wx < 12 && 0 <= wy && wy < 16 {
				tile := l03model.ActT[6+(wy/8)*3+wx/4]
				if l03model.Font[tile*8+wy%8]&(192>>(2*(wx%4))) != 0 {
					return true
				}
			}
		}
	}
	return false
}

// Compare the inexpensive seated-edge predicate with actual opaque art, not another rectangle model.
func TestL03SeatedTouchMatchesArt(t *testing.T) {
	atlas := l03Asset(t, "cat-sprites.bin")
	mapping := l03Asset(t, "cat-frame-map.bin")
	var g *l03Game
	for px := 66; px < 78; px++ { // all sub-pixel positions across the carrier's three support columns
		for bcol := 0; bcol < 17; bcol++ { // from the pursuer's home to dome-to-dome contact
			for logical := 50; logical < 54; logical++ { // seated/shove, both facings
				g = l03model.New(2)
				g.PlayerX, g.PlayerY = px, 171
				g.SetCol()
				g.Ride, g.Carrier, g.Facing = 1, 0, logical&1
				g.ActCol[0], g.ActCol[21] = 19, bcol
				if (g.BTouch() != 0) != l03SeatedOverlap(atlas, mapping, px, logical, bcol) {
					t.Fatalf("b_touch != art overlap at px=%d bcol=%d logical=%d", px, bcol, logical)
				}
			}
		}
	}
	g.Lnd = 10
	if g.BTouch() != 0 {
		t.Errorf("forepaw-first landing is not the seated silhouette yet")
	}
	g.Lnd, g.Jprep = 0, 3
	if g.BTouch() != 0 {
		t.Errorf("b_touch while jprep")
	}
	g.Jprep, g.PlayerX = 0, 62
	g.ActCol[21] = 0
	g.SetCol()
	if g.OnVacuum() != 0 {
		t.Errorf("unsupported one-column left overhang counts as on_vacuum")
	}
}

// ---------------------------------------------------------------- R1 cleaning corridor
func TestL03R1CleaningCorridor(t *testing.T) {
	ev, _ := l03Run(t, l03model.New(0), l03ExitFn, 3000)
	if !l03Has(ev, "respawn") || l03Has(ev, "hit0") {
		t.Fatalf("R1 floor bypass must fail: %v", ev)
	}
	ev, _ = l03Run(t, l03model.New(0), l03SwatPol(3, 28, 0), 3000)
	if !l03Eq(l03First(ev, 1), []string{"hit0"}) || !l03Has(ev, "room1") || l03Has(ev, "respawn") {
		t.Fatalf("R1 swat route: %v", ev)
	}
	g := l03model.New(0) // timed jump over the robot, no FIRE at all
	jumped := false
	jump := func(g *l03Game, i int) l03Input {
		d := g.ActCol[0] - g.PlayerCol
		if g.PlayerCol < 14 {
			return l03R
		}
		if !jumped && g.ActDir[0] == 1 && 0 <= d && d <= 4 && g.PlayerY == 184 && g.Air == 0 {
			jumped = true
			return l03UR
		}
		if jumped {
			return l03ExitPol(g)
		}
		return l03None
	}
	ev, _ = l03Run(t, g, jump, 3000)
	if !l03Has(ev, "room1") || l03Has(ev, "respawn") || l03Has(ev, "hit0") {
		t.Fatalf("R1 jump route: %v", ev)
	}
	g = l03model.New(0) // safe niche: climb the two platforms and wait while the robot patrols
	niche := func(g *l03Game, i int) l03Input {
		if g.Air != 0 || g.Jprep != 0 {
			return l03R
		}
		if g.PlayerY == 168 && g.PlayerCol >= 18 && g.Air == 0 {
			return l03None
		}
		if g.PlayerY == 184 && g.Air == 0 && g.PlayerCol >= 7 && g.PlayerCol < 12 {
			return l03UR
		}
		if g.PlayerY == 168 && g.PlayerCol < 14 && g.Air == 0 {
			return l03R
		}
		if g.PlayerY == 168 && g.Air == 0 && g.PlayerCol >= 14 {
			return l03UR
		}
		if g.PlayerY == 184 && g.PlayerCol < 7 {
			return l03R
		}
		return l03None
	}
	ev, n := l03Run(t, g, niche, 1000)
	if l03Has(ev, "respawn") || n != 999 || g.PlayerY != 168 || g.PlayerCol < 18 {
		t.Fatalf("R1 niche: ev=%v n=%d y=%d col=%d", ev, n, g.PlayerY, g.PlayerCol)
	}
}

// ---------------------------------------------------------------- R2 charger
func TestL03R2ChargerRoute(t *testing.T) {
	g := l03model.New(1)
	if g.Dk != 0 || g.Scr[21][22] != 12 || g.Art[21][22] != 12 || g.Scr[20][22] != 88 {
		t.Fatalf("R2 initial screen wrong")
	}
	if g.M["exit_y"] != 184 || g.Gate != 0 || !l03Rows(g, 39, 19, 20, 21) {
		t.Fatalf("R2 initial gate wrong")
	}
	policy := l03SwatPol(1, 14, 13)
	floorExit := func(g *l03Game, i int) l03Input {
		inp := policy(g, i)
		if inp.Up || g.PlayerY != 184 || g.Air != 0 {
			t.Fatalf("R2 floor route left the floor at frame %d", i)
		}
		return inp
	}
	ev, _ := l03Run(t, g, floorExit, 800, "room2") // real dock, then an uninterrupted floor route
	if !l03Eq(l03First(ev, 2), []string{"hit0", "dock"}) || !l03Has(ev, "room2") || l03Has(ev, "respawn") {
		t.Fatalf("R2 dock route: %v", ev)
	}
}

func TestL03R2DockTimeoutAndRecovery(t *testing.T) {
	g := l03model.New(1) // strike docks, then lasers (kill zone) are off; wait for the timeout; vacuum recovers
	pol := l03SwatPol(1, 14, 13)
	for i := 0; i < 400; i++ {
		inp := l03None
		if g.Dk == 0 {
			inp = pol(g, i)
		}
		g.Frame(inp)
		if g.Dk != 0 {
			break
		}
	}
	if g.Dk != 240 || g.ActCol[0] < 22 || !l03Has(g.Events, "dock") {
		t.Fatalf("dock state: dk=%d col=%d ev=%v", g.Dk, g.ActCol[0], g.Events)
	}
	if g.Gate == 0 || g.Scr[19][39] != 85 || g.Art[19][39] != 85 || g.Scr[20][39] != 85 || g.Art[20][39] != 85 ||
		g.Scr[21][39] != 85 || g.Art[21][39] != 85 {
		t.Fatalf("dock did not open the gate")
	}
	col0 := g.ActCol[0]
	if !l03IdleUntil(g, 400, func(g *l03Game) bool { return g.Dk == 0 }) {
		t.Fatalf("dock never timed out")
	}
	if !l03Has(g.Events, "undock") || g.ActWob[0] != 24 || g.ActDir[0] != 1 || g.ActCol[0] != col0 {
		t.Fatalf("undock state wrong")
	}
	g.Run(l03None, 80)
	if g.ActCol[0] >= col0 {
		t.Fatalf("it should patrol again (retreat toward the cat side)")
	}
}

func TestL03R2SlowPlayerKeepsDoorOpen(t *testing.T) {
	g := l03model.New(1) // reproduce a slow player: the laser timer ends, but the opened door stays open
	pol := l03SwatPol(1, 14, 13)
	found := false
	for i := 0; i < 800; i++ {
		g.Frame(pol(g, i))
		if g.Dk != 0 && g.PlayerCol >= 31 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no docked floor approach")
	}
	if g.Gate == 0 || g.Air != 0 || g.PlayerY != 184 {
		t.Fatalf("docked approach state wrong")
	}
	g.Events = nil
	g.Run(l03None, 350)
	if g.Dk != 0 || g.Gate == 0 || l03Has(g.Events, "respawn") {
		t.Fatalf("slow player: dk=%d gate=%d ev=%v", g.Dk, g.Gate, g.Events)
	}
	if !(g.Scr[19][39] == 85 && g.Scr[20][39] == 85 && g.Scr[21][39] == 85) || g.PlatExitAllowed() == 0 {
		t.Fatalf("door closed again")
	}
	if g.Scr[19][31]&127 != 79 { // laser returns without relocking the door
		t.Fatalf("laser did not return: %d", g.Scr[19][31])
	}
	ev, _ := l03Run(t, g, func(g *l03Game, i int) l03Input { return l03R }, 200, "room2")
	if !l03Has(ev, "room2") || l03Has(ev, "respawn") {
		t.Fatalf("exit after open door: %v", ev)
	}
}

func TestL03R2Hazards(t *testing.T) {
	g := l03model.New(1) // misses retry: no-fire walking into the vacuum respawns, state resets
	ev, _ := l03Run(t, g, l03ExitFn, 3000)
	if !l03Has(ev, "respawn") || g.Dk != 0 || g.ActCol[0] != g.M["acol"] || !l03Clean(g) {
		t.Fatalf("R2 miss must respawn and reset cleanly: %v", ev)
	}
	g = l03model.New(1) // lasers: a jump while the beam is live kills, a walk at floor height does not
	g.Run(l03None, 30)
	g.PlayerX, g.PlayerY, g.Air = 96, 168, 1
	g.SetCol()
	g.Events = nil
	for i := 0; i < 30; i++ {
		g.Frame(l03None)
		if len(g.Events) > 0 {
			break
		}
	}
	if !l03Has(g.Events, "respawn") {
		t.Fatalf("airborne in the laser band must die")
	}
	g = l03model.New(1)
	g.ActCol[0], g.ActWob[0] = 22, 0
	g.PlayerX, g.PlayerY = 96, 184
	g.SetCol()
	g.Dk = 100 // docked: the beam is off, so the same jump is safe
	g.Run(l03None, 5)
	if l03Has(g.Events, "respawn") {
		t.Fatalf("docked beam must be off")
	}
	g = l03model.New(1) // a cat standing at the robot's column (immune after a swat) keeps its floor
	g.PlayerX, g.PlayerY = 4*(g.ActCol[0]-1)-2, 184
	g.SetCol()
	if sg := g.Scr[22][g.PlayerCol]; !(sg == 95 || sg == 124 || sg == 125) || g.Art[22][g.PlayerCol] != 70 {
		t.Fatalf("base glyph expectation wrong")
	}
	g.PlayerY = 184
	g.Air = 0
	g.ActWob[0] = 40
	if g.Under() == 0 {
		t.Fatalf("immune cat must keep its floor")
	}
	g.Art[22][g.PlayerCol] = 79 | 0x80 // a pit beam under a base drawing is not support
	if g.Under() != 0 {
		t.Fatalf("pit beam under a base drawing must not support")
	}
	g = l03model.New(1)
	g.PlayerX, g.PlayerY = 140, 184
	g.SetCol()
	g.PlatExit()
	if g.CurRoom != 1 { // end jump / walking under the beam cannot skip the charger objective
		t.Fatalf("charger objective skipped")
	}
	g.Dk = 1
	if g.PlatExitAllowed() != 0 { // a timer alone is not the completed charger objective
		t.Fatalf("timer alone must not allow exit")
	}
	g.Gate = 1
	if g.PlatExitAllowed() == 0 {
		t.Fatalf("gate must allow exit")
	}
	g.Dk = 0
	if g.PlatExitAllowed() == 0 { // docking unlock persists after its short hazard window
		t.Fatalf("docking unlock must persist")
	}
	g.PlayerY = 168
	if g.PlatExitAllowed() != 0 { // former podium height is no longer the exit
		t.Fatalf("podium height must not be the exit")
	}
	g.Respawn()
	if g.Gate != 0 || g.Dk != 0 || !l03Rows(g, 39, 19, 20, 21) {
		t.Fatalf("respawn must relock the gate")
	}
	g.ActCol[0], g.ActWob[0], g.ActDir[0] = 23, 40, 1
	g.AEvents()
	if g.Dk != 0 { // striking from the wrong side must not dock the vacuum
		t.Fatalf("wrong-side strike docked the vacuum")
	}
	g.ActDir[0] = 0
	g.AEvents()
	if g.Dk != 240 {
		t.Fatalf("right-side strike must dock: dk=%d", g.Dk)
	}
	if g.Gate == 0 || g.Scr[19][39] != 85 || g.Scr[20][39] != 85 || g.Scr[21][39] != 85 {
		t.Fatalf("dock must open the gate")
	}
	g = l03model.New(1)
	g.PlayerX = 60
	g.SetCol()
	for _, c := range []struct {
		y      int
		danger bool
	}{{152, false}, {153, true}, {183, true}, {184, false}} {
		g.PlayerY = c.y
		g.Events = nil
		g.PlatBeam()
		if l03Has(g.Events, "respawn") != c.danger {
			t.Fatalf("beam at y=%d: events %v", c.y, g.Events)
		}
		g.PlayerX = 60
		g.SetCol()
	}
}

// ---------------------------------------------------------------- R3 ride and shove
func l03RearPol(relT, jumpCol int) l03Pol { // board, then stand at carrier column act+relT facing the travel direction
	base := l03RidePol(false, jumpCol, 4, 0)
	return func(g *l03Game, i int) l03Input {
		if g.Ride != 0 && g.Air == 0 {
			rel := g.PlayerCol - g.ActCol[0]
			if rel > relT {
				return l03L
			}
			if rel < relT || g.Facing != 0 {
				return l03R
			}
			return l03None
		}
		return base(g, i)
	}
}

func TestL03R3RideAndShove(t *testing.T) {
	// no ride at all: the floor stops at the pit
	g := l03model.New(2)
	ev, _ := l03Run(t, g, l03ExitFn, 3000)
	if !l03Has(ev, "respawn") {
		t.Fatalf("R3 floor must stop at the pit")
	}
	g = l03model.New(2) // rear of the roof, no FIRE: the pursuer's dome really touches the rider and catches it
	ev, _ = l03Run(t, g, l03RearPol(0, 28), 2500)
	if !l03Eq(ev, []string{"respawn"}) || g.Ride != 0 || g.Bk != 0 || !l03Clean(g) || g.ActCol[0] != g.M["acol"] {
		t.Fatalf("R3 rear rider: ev=%v ride=%d bk=%d", ev, g.Ride, g.Bk)
	}
	g = l03model.New(2) // documented dodge: standing at the very front edge never overlaps the gap-3 pursuer
	ev, _ = l03Run(t, g, l03RearPol(2, 99), 2500)
	if len(ev) != 0 || g.Bk != 0 {
		t.Fatalf("R3 front-edge dodge: ev=%v bk=%d", ev, g.Bk)
	}
	g = l03model.New(2)
	ev, _ = l03Run(t, g, l03RidePol(true, 28, 4, 0), 1500, "room3")
	if !l03Has(ev, "hit21") || l03Has(ev, "respawn") || !l03Has(ev, "room3") {
		t.Fatalf("R3 ride+shove route: %v", ev)
	}
	// a true seated shove leaves the carrier alone: clone the same moment with and without FIRE
	a := l03model.New(2)
	pa := l03RearPol(0, 99)
	for i := 0; i < 1500; i++ {
		a.Frame(pa(a, i))
		if a.Ride != 0 && a.Bk >= 6 && a.Facing == 0 && a.PawT == 0 {
			break
		}
	}
	if a.Ride != 1 || a.Carrier != 0 || a.Bk < 6 || (a.PawKind != 0 && a.PawKind != 2) {
		t.Fatalf("seated shove setup wrong")
	}
	b := a.Clone()
	a.Frame(l03F)
	b.Frame(l03None)
	if a.PawKind != 2 { // seated swing
		t.Fatalf("seated swing expected, paw_kind=%d", a.PawKind)
	}
	type trail [3]int
	var ta, tb []trail
	for i := 0; i < 40; i++ {
		a.Frame(l03None)
		b.Frame(l03None)
		ta = append(ta, trail{a.ActCol[0], a.ActDir[0], a.ActT})
		tb = append(tb, trail{b.ActCol[0], b.ActDir[0], b.ActT})
	}
	if !l03Has(a.Events, "hit21") || l03Has(b.Events, "hit21") || l03Has(a.Events, "respawn") {
		t.Fatalf("shove events: a=%v b=%v", a.Events, b.Events)
	}
	if !reflect.DeepEqual(ta, tb) {
		t.Fatalf("carrier column/direction/step cadence changed by the shove")
	}
	if !(a.ActWob[21] > 0 || a.ActCol[21] < b.ActCol[21] || (a.Bk == 0 && b.Bk > 0)) {
		t.Fatalf("shove had no effect on the pursuer")
	}
	if a.Bk != 0 || a.ActCol[21] >= b.ActCol[21] {
		t.Fatalf("shove must reset bk and push the pursuer back")
	}
	// early dismount: jumps from the roof at cols 20..24 fall into the pit; then retry/reboard works
	for _, jc := range []int{21, 23} {
		g = l03model.New(2)
		ev, _ = l03Run(t, g, l03RidePol(true, jc, 4, 0), 1500, "room3")
		if !l03Has(ev, "respawn") || l03Has(ev, "room3") {
			t.Fatalf("early dismount jc=%d: %v", jc, ev)
		}
		ev2, _ := l03Run(t, g, l03RidePol(true, 28, 4, 0), 1500, "room3")
		if !l03Has(ev2, "room3") {
			t.Fatalf("reboard after retry: %v", ev2)
		}
	}
	// a jump that lands on the post-pit floor (not the podium) still needs the exit jump
	g = l03model.New(2)
	ev, _ = l03Run(t, g, l03RidePol(true, 26, 4, 0), 1500, "room3")
	if l03Has(ev, "respawn") || g.PlayerY != 184 || g.PlayerCol < 33 {
		t.Fatalf("post-pit landing: ev=%v y=%d col=%d", ev, g.PlayerY, g.PlayerCol)
	}
	ev, _ = l03Run(t, g, l03ExitFn, 300, "room3")
	if !l03Has(ev, "room3") {
		t.Fatalf("exit jump: %v", ev)
	}
}

// ---------------------------------------------------------------- R4 reader, gate, ride, shove, exit
func l03ServicePol() l03Pol {
	sw, rd := l03SwatPol(2, 28, 7), l03RidePol(false, 19, 4, 0)
	return func(g *l03Game, i int) l03Input {
		if g.Gate == 0 {
			return sw(g, i)
		}
		if g.PlayerCol >= 33 && g.Air == 0 {
			return l03R
		}
		if g.PlayerY == 152 && g.Air == 0 {
			if g.PlayerCol < 27 {
				return l03R
			}
			return l03UR
		}
		return rd(g, i)
	}
}

func TestL03R4ReaderGateRoute(t *testing.T) {
	g := l03model.New(3)
	if g.Gate != 0 || g.EBspd != 0 || g.OnVacuum() != 0 {
		t.Fatalf("R4 initial state wrong")
	}
	g.Frame(l03U)
	g.Run(l03None, 60)
	if g.PlayerY != 184 || g.Ride != 0 || g.Gate != 0 { // the formerly stuck rear carrier is absent
		t.Fatalf("R4 idle state wrong")
	}
	// The shared roof helper also rejects any inactive parked pursuer if one is configured later.
	g.EBspd, g.PlayerX, g.PlayerY, g.ActCol[21] = 2, 0, 171, 0
	g.SetCol()
	if g.OnVacuum() != 0 {
		t.Fatalf("inactive pursuer must not carry")
	}
	g.Gate = 1
	if g.OnVacuum() != 21 {
		t.Fatalf("active pursuer must carry")
	}
	g = l03model.New(3)
	ev, _ := l03Run(t, g, l03ExitFn, 3000)
	if !l03Has(ev, "respawn") || l03Has(ev, "gate") { // walking the floor never opens the gate
		t.Fatalf("R4 floor walk: %v", ev)
	}
	g = l03model.New(3)
	ev, _ = l03Run(t, g, l03SwatPol(2, 28, 7), 600, "respawn", "gate")
	if !(l03Eq(l03Last(ev, 2), []string{"hit0", "gate"}) || (len(ev) > 0 && ev[len(ev)-1] == "gate")) {
		t.Fatalf("R4 gate route: %v", ev)
	}
	if g.Gate != 1 || !(g.M["z0"] <= g.ActCol[0] && g.ActCol[0] <= g.M["z1"]) || g.Scr[19][39] != g.Art[19][39] {
		t.Fatalf("R4 gate state wrong")
	}
	for _, c := range []int{12, 13, 14} {
		if g.Scr[20][c] != 84 {
			t.Fatalf("reader must show DONE")
		}
	}
	g = l03model.New(3) // no gate: even a jump onto the ledge cannot leave the room
	g.PlayerX, g.PlayerY = 140, 184
	g.SetCol()
	g.Frame(l03R)
	if g.LevelFinished != 0 || l03Has(g.Events, "level_end") {
		t.Fatalf("left the room without the gate")
	}

	g = l03model.New(3)
	pol := l03ServicePol()
	heights := map[int]bool{}
	for i := 0; i < 1500; i++ {
		g.Frame(pol(g, i))
		if g.Air == 0 {
			heights[g.PlayerY] = true
		}
		if g.LevelFinished != 0 || l03Has(g.Events, "respawn") {
			break
		}
	}
	if !l03Eq(g.Events, []string{"hit0", "gate", "level_end"}) {
		t.Fatalf("R4 service route events %v x=%d y=%d", g.Events, g.PlayerX, g.PlayerY)
	}
	if !(heights[171] && heights[152] && heights[184]) || g.PlayerY != 184 || g.PlayerCol < 34 {
		t.Fatalf("R4 service route heights %v y=%d col=%d", heights, g.PlayerY, g.PlayerCol)
	}
	// retry reset clears gate, ride, dk, and redraws the gate wall
	g.Respawn()
	if g.Gate != 0 || g.Ride != 0 || g.Dk != 0 || g.Bk != 0 || g.ActCol[0] != g.M["acol"] {
		t.Fatalf("respawn did not reset R4 state")
	}
	redrawn := false
	for r := 17; r < 21; r++ {
		if g.Scr[r][39] != g.Art[r][39] {
			redrawn = true
		}
	}
	if !redrawn { // gate wall redrawn
		t.Fatalf("gate wall not redrawn")
	}
	for _, c := range []int{12, 13, 14} {
		if g.Scr[20][c] != 83 {
			t.Fatalf("reader must show RELAY after reset")
		}
	}
	// every genuine rightward strike at a normal patrol position that connects reaches the reader within the stun
	for stand := 17; stand < 22; stand++ {
		for wait := 0; wait < 120; wait++ {
			g = l03model.New(3)
			g.PlayerX = stand
			g.SetCol()
			g.Run(l03None, wait)
			g.Frame(l03F)
			hitT := -1
			for tt := 0; tt < 140; tt++ {
				g.Frame(l03None)
				if l03Has(g.Events, "hit0") {
					hitT = tt
				}
				if l03Has(g.Events, "respawn") {
					t.Fatalf("respawn during strike stand=%d wait=%d", stand, wait)
				}
				if l03Has(g.Events, "gate") {
					if !(hitT >= 0 && tt-hitT < 40) { // gate while still stunned
						t.Fatalf("gate not within stun: stand=%d wait=%d", stand, wait)
					}
				}
				g.Events = nil
			}
			if (hitT < 0) != (g.Gate == 0) { // no hit-without-gate
				t.Fatalf("hit-without-gate: stand=%d wait=%d", stand, wait)
			}
		}
	}
}

// ---------------------------------------------------------------- R4 captions and reader hooks (source/data)
func TestL03R4CaptionsAndReaderHooks(t *testing.T) {
	src := l03model.SRC
	goal := regexp.MustCompile(`(?s)data R4Goal \{(.*?)\n\}`).FindStringSubmatch(src)[1]
	var lines []string
	for _, m := range l03QuoteRe.FindAllStringSubmatch(l03CharsetRe.ReplaceAllString(goal, ""), -1) {
		lines = append(lines, m[1])
	}
	if !strings.Contains(goal, "nocross") || len(lines) != 4 {
		t.Fatalf("R4Goal: %q", lines)
	}
	total := 0
	for _, x := range lines {
		if len(x) != 36 {
			t.Errorf("caption %q is not 36 chars", x)
		}
		total += len(x)
	}
	if !strings.HasPrefix(lines[0], "SWAT THE VACUUM") || !strings.Contains(lines[0], "READER") {
		t.Errorf("caption 0: %q", lines[0])
	}
	if !strings.Contains(lines[1], "BALCONY") || !strings.HasPrefix(lines[1], "RIDE") || !strings.Contains(lines[2], "GAP") || !strings.Contains(lines[2], "EXIT") {
		t.Errorf("captions 1/2: %q %q", lines[1], lines[2])
	}
	if !strings.HasPrefix(lines[3], "SERVICE ACCESS OPEN") || !strings.Contains(lines[3], "RIGHT") {
		t.Errorf("caption 3: %q", lines[3])
	}
	const allowed = " !~#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, ch := range strings.Join(lines, "") {
		if !strings.ContainsRune(allowed, ch) {
			t.Errorf("caption char %q not in charset", ch)
		}
	}
	if !strings.Contains(src, "data R4Off { 0 36 72 108 }") || total != 144 { // 144 bytes < one page
		t.Errorf("R4Off / caption total %d", total)
	}
	hints := l03CharsetRe.ReplaceAllString(regexp.MustCompile(`(?s)data Hints \{.*?\n\}`).FindString(src), "")
	var names []string
	for _, m := range l03QuoteRe.FindAllStringSubmatch(hints, -1) {
		names = append(names, strings.TrimSpace(m[1]))
	}
	var got []string
	for _, f := range strings.Fields(regexp.MustCompile(`R4Hint \{ ([\d ]+) \}`).FindStringSubmatch(src)[1]) {
		i, _ := strconv.Atoi(f)
		got = append(got, names[i])
	}
	if !l03Eq(got, []string{"SWAT", "RIDE", "JUMP", "EXIT"}) {
		t.Errorf("R4Hint names: %v (%v)", got, names)
	}
	rg := l03Body(t, "r4_goal")
	if !strings.HasPrefix(rg, "x=0 a=gate a?0 != { x=1 a=player_col a?23 >= { x=2 } a?33 >= { x=3 } }") {
		t.Errorf("r4_goal head: %s", rg)
	}
	for _, s := range []string{"msg_len=a=36 draw_row0_msg", "msg_src+1=a=&>R4Goal", "a=R4Off,x",
		"a=R4Hint,x x=a a=ride a?0 != { a=act_col a?17 >= { x=1 } }"} {
		l03Need(t, rg, s, "r4_goal")
	}
	l03Need(t, l03Body(t, "plat_hud"), "a=cur_room a?3 == { r4_goal }", "plat_hud")
	// draw_reader: reader room only; RELAY (83) until the crossing, then DONE (84); not the R2 dock.
	dr := l03Body(t, "draw_reader")
	if !strings.HasPrefix(dr, "a=cur_level a?2 != { return } a=e_mode a&2 a?0 == { return } tmp=a=83 a=gate a?0 != { tmp=a=84 }") {
		t.Errorf("draw_reader head: %s", dr)
	}
	l03Need(t, dr, "x=20 y=m_z0 cnt=a=3 put_run", "draw_reader")
	ae := l03Body(t, "a_events")
	parts := strings.Split(ae, "a=e_mode a&2 a?0 == { return }")
	if len(parts) != 2 {
		t.Fatalf("a_events split into %d parts", len(parts))
	}
	dock, reader := parts[0], parts[1]
	if strings.Contains(dock, "draw_reader") || !strings.Contains(dock, "draw_gate }") { // R2 dock: no reader redraw
		t.Errorf("a_events dock part wrong: %s", dock)
	}
	if !strings.HasSuffix(reader, "gate=a=1 tmp=a=0 draw_gate draw_reader") || strings.Count(reader, "draw_reader") != 1 {
		t.Errorf("a_events reader part wrong: %s", reader)
	}
	ai := l03Body(t, "act_init")
	l03Need(t, ai, "a=m_arow a?0 == { return } draw_reader draw_act", "act_init") // retry/reset restores RELAY
	if !strings.HasSuffix(l03Body(t, "plat_reset"), "act_init") || strings.Count(ai, "draw_reader") != 1 {
		t.Errorf("plat_reset/act_init draw_reader binding wrong")
	}
	sum := 0
	for _, m := range regexp.MustCompile(`func (\w+) \{`).FindAllStringSubmatch(src, -1) {
		sum += strings.Count(l03Body(t, m[1]), "draw_reader")
	}
	if sum != 2 { // act_init + a_events only
		t.Errorf("draw_reader call sites = %d, want 2", sum)
	}
}

// A genuine post-gate floor approach still cannot jump the first pit or the
// final gap. These starts are on supported floor, unlike a void-start probe.
func TestL03R4PostGateFloorBypass(t *testing.T) {
	for x := 74; x < 82; x++ {
		g := l03model.New(3)
		g.Gate, g.PlayerX, g.PlayerY = 1, x, 184
		g.SetCol()
		if g.Under() == 0 {
			t.Fatalf("x=%d col=%d: start not on supported floor", x, g.PlayerCol)
		}
		respawned := false
		for i := 0; i < 300; i++ {
			g.Frame(l03UR)
			if g.Air == 0 && g.PlayerY == 152 {
				t.Fatalf("x=%d balcony bypass", x)
			}
			if g.Air == 0 && g.PlayerY == 184 && g.PlayerCol >= 33 {
				t.Fatalf("x=%d final-floor bypass", x)
			}
			if g.LevelFinished != 0 {
				t.Fatalf("x=%d level finished", x)
			}
			if l03Has(g.Events, "respawn") {
				respawned = true
				break
			}
			g.Events = nil
		}
		if !respawned {
			t.Fatalf("x=%d never respawned (px=%d py=%d col=%d)", x, g.PlayerX, g.PlayerY, g.PlayerCol)
		}
	}
}

// R2 dock latch: after the wobble recovery the normal patrol limit applies again.
func TestL03R2DockLatch(t *testing.T) {
	g := l03model.New(1)
	pol := l03SwatPol(1, 14, 13)
	for i := 0; i < 400; i++ {
		inp := l03None
		if g.Dk == 0 {
			inp = pol(g, i)
		}
		g.Frame(inp)
		if g.Dk != 0 {
			break
		}
	}
	if g.Gate == 0 || g.EMode&1 == 0 || g.EMode&2 != 0 {
		t.Fatalf("dock latch setup wrong")
	}
	if !l03IdleUntil(g, 400, func(g *l03Game) bool { return g.Dk == 0 && g.ActWob[0] == 0 }) {
		t.Fatalf("never recovered from dock")
	}
	peak := 0
	for i := 0; i < 500; i++ {
		g.Frame(l03None)
		if g.ActCol[0] <= g.M["amax"] && g.ActCol[0] > peak {
			peak = g.ActCol[0]
		}
		if !(g.ActCol[0] <= g.M["amax"] || g.ActDir[0] == 1) { // may only be above amax while heading back
			t.Fatalf("vacuum above amax while heading right")
		}
	}
	if peak > g.M["amax"] {
		t.Fatalf("peak %d above amax", peak)
	}
}

// ---------------------------------------------------------------- R4 robustness: timing, dismount, recovery
type l03BotOpts struct {
	stand, edge, rear, dismount, delay int
	board                              [2]int
}

func l03DefaultBot() l03BotOpts {
	return l03BotOpts{stand: 17, board: [2]int{5, 7}, edge: 11, dismount: -1}
}

type l03BotState struct{ cool, swings, left, deaths int }

// l03R4Bot is an input policy only: swing again after a miss, wait at the first pit for the
// patrolling vacuum, jump to its roof, ride, leave at the balcony and cross the gaps by position.
func l03R4Bot(o l03BotOpts) (l03Pol, *l03BotState) {
	st := &l03BotState{}
	return func(g *l03Game, i int) l03Input {
		col, robot, x := g.PlayerCol, g.ActCol[0], g.PlayerX
		w0, w1 := o.board[0]-st.deaths, o.board[1]-st.deaths
		if i < o.delay {
			return l03None
		}
		if g.Gate == 0 {
			if st.cool != 0 {
				st.cool--
			} else if g.PawHit != 0 && g.ActWob[0] != 0 {
				return l03None // carried toward the reader; hits land at the patrol columns 8..10 and reach the reader in time
			} else if x >= o.stand && g.PawT == 0 && g.Air == 0 {
				st.swings++
				st.cool = []int{11, 16, 21}[st.swings%3]
				return l03F
			} else if x < o.stand {
				return l03R
			}
			return l03None
		}
		if g.PlayerY == 184 && g.Air == 0 && g.Ride == 0 && col >= 33 {
			return l03R
		}
		if g.PlayerY == 152 && g.Air == 0 {
			if col < 27 {
				return l03R
			}
			return l03UR
		}
		if g.Air != 0 {
			return l03R
		}
		if g.Ride != 0 {
			if o.dismount >= 0 && o.dismount <= robot && robot < 19 && st.left == 0 {
				st.left = 1
				return l03UR
			}
			if robot >= 19 {
				return l03UR
			}
			if col-robot < o.rear {
				return l03R
			}
			if col-robot > o.rear {
				return l03L
			}
			return l03None
		}
		if g.PlayerY == 184 {
			if g.ActDir[0] == 1 && w0 <= robot-col && robot-col <= w1 {
				return l03UR
			}
			if col < o.edge {
				return l03R
			}
		}
		return l03None
	}, st
}

func l03R4Run(t *testing.T, o l03BotOpts) (int, int, *l03BotState) {
	t.Helper()
	g := l03model.New(3)
	pol, st := l03R4Bot(o)
	deaths := 0
	seen := map[int]bool{}
	for i := 0; i < 6000; i++ {
		g.Frame(pol(g, i))
		for c := g.ActCol[0]; c < g.ActCol[0]+3; c++ {
			if g.Art[22][c] != l03model.Floor {
				t.Fatalf("R4 robot footprint lost floor support: frame %d col %d", i, g.ActCol[0])
			}
		}
		deaths += l03Count(g.Events, "respawn")
		st.deaths = deaths
		g.Events = nil
		if g.Air == 0 {
			seen[g.PlayerY] = true
		}
		if g.LevelFinished != 0 {
			if !(g.PlayerY == 184 && g.PlayerCol >= 34 && g.Gate != 0 && seen[184]) {
				t.Fatalf("R4 finish state wrong: %v", seen)
			}
			return i + 1, deaths, st
		}
	}
	t.Fatalf("R4 bot stalled %+v deaths=%d x=%d y=%d gate=%d ride=%d", o, deaths, g.PlayerX, g.PlayerY, g.Gate, g.Ride)
	return 0, 0, nil
}

func TestL03R4Robustness(t *testing.T) {
	runs := 0
	for _, stand := range []int{17, 19, 21} { // forepaw contact needs x 17..21 for the 8..10 patrol
		for _, board := range [][2]int{{5, 5}, {6, 6}, {7, 7}, {5, 7}} { // roof-jump distance: a 3-column, ~24 frame window
			for rear := 0; rear < 3; rear++ {
				o := l03DefaultBot()
				o.stand, o.board, o.rear, o.delay = stand, board, rear, 3*stand%17
				_, deaths, _ := l03R4Run(t, o)
				if deaths != 0 {
					t.Fatalf("stand=%d board=%v rear=%d deaths=%d", stand, board, rear, deaths)
				}
				runs++
			}
		}
	}
	for _, dismount := range []int{16, 17, 18} { // leaving before the balcony is fatal, not a soft-supported bypass
		o := l03DefaultBot()
		o.dismount = dismount
		_, deaths, st := l03R4Run(t, o)
		if !((dismount == 16 && deaths == 0) || (dismount != 16 && deaths >= 1)) || st.left != 1 {
			t.Fatalf("dismount=%d deaths=%d left=%d", dismount, deaths, st.left)
		}
	}
	for x := 90; x < 112; x++ { // the clear first pit and second gap are both fatal without the ride
		g := l03model.New(3)
		g.Gate, g.PlayerX, g.PlayerY = 1, x, 184
		g.SetCol()
		dead := false
		g.Frame(l03UR)
		for i := 0; i < 200; i++ {
			g.Frame(l03R)
			dead = dead || l03Has(g.Events, "respawn")
			if g.PlayerCol >= 34 && g.Air == 0 {
				break
			}
		}
		if !dead {
			t.Fatalf("x=%d crossed without the ride", x)
		}
	}
	var late []string
	for _, b := range []int{8, 9} { // a late jump falls in the pit, resets the room, retries
		o := l03DefaultBot()
		o.board = [2]int{b, b}
		_, deaths, _ := l03R4Run(t, o)
		if deaths < 1 {
			t.Fatalf("late jump board=%d had %d deaths", b, deaths)
		}
		late = append(late, fmt.Sprintf("(%d, %d)", b, deaths))
	}
	t.Logf("R4 model timing/boarding variants: %d complete with 0 respawns; early dismount retries; late jumps [%s]", runs, strings.Join(late, ", "))
}

// A rider boarded before the reader opens cannot carry the cat past the first pit: no softlock, no bypass.
func TestL03R4RiderBeforeReaderOpens(t *testing.T) {
	g := l03model.New(3)
	for i := 0; i < 3000; i++ {
		robot, col := g.ActCol[0], g.PlayerCol
		var inp l03Input
		switch {
		case g.Ride != 0:
			if robot >= 9 && g.Air == 0 {
				inp = l03UR
			}
		case g.Air != 0 || (g.PlayerY == 184 && g.PlayerX < 18):
			inp = l03R
		case g.PlayerY == 184 && g.ActDir[0] == 1 && 0 <= robot-col && robot-col <= 4:
			inp = l03UR
		}
		g.Frame(inp)
		g.Events = nil
		if g.Gate != 0 || g.LevelFinished != 0 {
			t.Fatalf("frame %d: gate=%d finished=%d", i, g.Gate, g.LevelFinished)
		}
	}
	if g.M["xmax"] != 19 || g.M["amax"] != 10 {
		t.Fatalf("xmax=%d amax=%d", g.M["xmax"], g.M["amax"])
	}
}
