package tests

// Level 02 redesign: source wiring plus route/mechanic proofs on tools/platmodel.
//
// platmodel is a frame-level model of src/plat_*.k65's plat_* functions. These checks prove the
// designed routes, crumble/fan/switch/lift behaviour and softlock freedom at the source+asset
// level; they do not execute a compiled XEX.

import (
	"bytes"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	L "github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
	P "github.com/michal-zuchowski/cats-ai-busters/tools/platlevels"
	M "github.com/michal-zuchowski/cats-ai-busters/tools/platmodel"
)

type lv2Span [3]int

var lv2Expect = map[int][]lv2Span{
	0: {{22, 0, 39}, {20, 22, 25}, {18, 16, 19}, {16, 23, 26}, {14, 16, 19}, {12, 6, 13}, {13, 35, 39}},
	1: {{22, 0, 39}, {13, 0, 6}, {13, 10, 13}, {12, 17, 20}, {11, 25, 28}, {20, 26, 29}, {18, 32, 35},
		{16, 26, 29}, {14, 32, 35}, {11, 33, 39}},
	2: {{22, 0, 39}, {11, 0, 9}, {11, 18, 25}, {9, 29, 39}, {19, 12, 15}, {16, 17, 20}, {13, 22, 25}},
	3: {{22, 0, 39}, {9, 0, 7}, {10, 17, 31}, {6, 18, 21}, {6, 25, 28}, {6, 33, 39}},
}

func lv2Need(t *testing.T, ok bool, format string, args ...any) {
	t.Helper()
	if !ok {
		t.Fatalf(format, args...)
	}
}

func lv2Metas() []map[string]int {
	out := make([]map[string]int, 4)
	for n := range out {
		_, out[n] = M.Unpack(P.Rooms[n])
	}
	return out
}

func lv2Blank(chunk []byte, spans ...lv2Span) []byte {
	b := append([]byte(nil), chunk...)
	for _, s := range spans {
		for c := s[1]; c <= s[2]; c++ {
			b[s[0]*40+c] = L.Void
		}
	}
	return b
}

func lv2Plan(chunk []byte, start M.Node, o *M.PlanOpts) M.Parents { return M.Plan(chunk, start, o) }

func lv2Reach(p M.Parents, n M.Node) bool { _, ok := p[n]; return ok }

func lv2Replay(t *testing.T, chunk []byte, start, goal M.Node, o *M.PlanOpts) (int, *M.Sim, M.Parents) {
	t.Helper()
	par := lv2Plan(chunk, start, o)
	lv2Need(t, lv2Reach(par, goal), "unreachable %v -> %v", start, goal)
	sim := M.New(chunk, 0)
	sim.X, sim.Y = start[0], start[1]
	lv2Need(t, M.Drive(sim, M.PathTo(par, goal), 0), "replay left the plan")
	lv2Need(t, sim.Deaths == 0 && (M.Node{sim.X, sim.Y}) == goal, "replay ended at %d,%d deaths %d", sim.X, sim.Y, sim.Deaths)
	return M.CostOf(par, goal), sim, par
}

func lv2Jumps(par M.Parents, goal M.Node) int {
	n := 0
	for _, h := range M.PathTo(par, goal) {
		if h.Macro.Kind == "jump" {
			n++
		}
	}
	return n
}

func lv2SettlePress(s *M.Sim, key int) {
	s.Step(key)
	s.Step(0)
}

func lv2FireAt(chunk []byte, x, y, air int) *M.Sim {
	s := M.New(chunk, 0)
	s.X, s.Y, s.Air = x, y, air
	s.Step(M.Fire)
	return s
}

func lv2Absi(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestLevel02SourceWiring(t *testing.T) {
	main := src()
	body := func(n string) string { return funcBodyBraced(main, n) }
	has := func(s, sub string) { t.Helper(); ensure(t, strings.Contains(s, sub), "missing %q", sub) }

	plats := readAsset("plat-rooms.bin")
	for i := 0; i < 4; i++ {
		ensure(t, bytes.Equal(plats[i*1024:(i+1)*1024], P.Rooms[i]), "assets/plat-rooms.bin room %d is stale", i)
	}

	fire := body("plat_fire")
	has(fire, "a=air a?0 != { return }") // grounded only
	has(fire, "a=TRIG0 a&1 a?0 != { fire_prev=a=1 return }")
	has(fire, "fire_prev=a=0") // press edge
	has(fire, "a=cur_level a?1 == { plat_switch return } a?2 == { swat return }")
	ready := body("sw_ready")
	for _, n := range []string{"a=m_swy", "a?player_y", "a=sw_on", "a=air", "c+ a-m_swc", "c- a+2", "a?5 >= { a=0 return }"} {
		has(ready, n)
	}
	sw := body("plat_switch")
	ensure(t, strings.HasPrefix(sw, "sw_ready a?0 == { return }"), "plat_switch prefix")
	has(sw, "sw_on=a")
	has(sw, "tmp=a=62")
	has(sw, "tmp=a=85")
	has(sw, "a=m_lpow a?0 != { draw_lift }")
	under := body("under")
	for _, g := range []string{"a?70", "a?94", "a?127", "a?62"} {
		has(under, g)
	}
	ensure(t, strings.Count(body("respawn"), "crumble_reset") == 1, "respawn crumble_reset count")
	ensure(t, strings.HasPrefix(body("crumble_reset"), "a=cur_level a?1 != { return } x=0"), "crumble_reset prefix") // L03/L04 alias these bytes
	ensure(t, strings.HasPrefix(body("plat_dyn_init"), "sw_on=a=0 fan_t=a x=0 { cr_t,x=a x++ x?8 } !="), "plat_dyn_init prefix")
	lift := body("plat_lift")
	has(lift, "a=m_lpow a?0 != { a=sw_on a?0 == { return } }")
	has(lift, "a?tmp < { return }")
	has(lift, "a=m_lspd a?0 == { a=24 }") // Levels 03/04 keep the 24 frame lift period
	frame := body("plat_frame")
	after := strings.SplitN(frame, "plat_lift", 2)
	lv2Need(t, len(after) == 2, "plat_frame lacks plat_lift")
	has(strings.SplitN(after[1], "plat_actor", 2)[0], "plat_crumble")
	fan := body("plat_fan")
	has(fan, "a=fan_t a&64")
	has(fan, "a?m_fan < { return }")
	has(fan, "a?m_fc1 >= { return }")
	has(fan, "a=air a?0 == { return }")
	ensure(t, !strings.Contains(fan, "RTCLOK") && !strings.Contains(fan, "COLOR2"), "fan reads RTCLOK/COLOR2")
	has(fan, "fan_draw")
	has(body("fan_draw"), "FanT,x")
	has(body("fan_draw"), "put_run")
	hud := body("plat_hud")
	has(hud, "a=fan_t a&64")
	has(hud, "sw_ready a?0 != { x=7 }")
	ensure(t, regexp.MustCompile(`"PUSH   "\s+"FIRE   "\s+"DOCK   "`).MatchString(main), "hint strings")
	has(main, "HintOff { 0 7 14 21 28 35 42 49 56 63 }")
	for _, n := range []string{"paw_both", "plat_pursuer", "plat_beam"} {
		ensure(t, strings.HasPrefix(body(n), "a=cur_level a?2 != { return }"), "%s prefix", n)
	}
	ensure(t, strings.HasPrefix(body("plat_exit_allowed"), "a=cur_level a?2 == {"), "plat_exit_allowed prefix")
	inp, phys := body("plat_input"), body("plat_phys") // physics / pose-driving constants are untouched
	has(inp, "vy=a=0xFC")
	has(inp, "jprep=a=3")
	has(inp, "walk_timer=a=4")
	has(phys, "lnd=a=10")
	has(phys, "a?3 == {")
	has(phys, "gt=a=2")

	addr := map[string]int{}
	for _, m := range regexp.MustCompile(`var (\w+) = (0x[0-9A-Fa-f]+)`).FindAllStringSubmatch(main, -1) {
		v, _ := strconv.ParseInt(m[2][2:], 16, 64)
		addr[m[1]] = int(v)
	}
	used := []int{addr["sw_on"], addr["fan_t"], addr["cr_i"]}
	for _, n := range []string{"cr_t", "cr_r", "cr_c"} {
		for i := 0; i < 8; i++ {
			used = append(used, addr[n]+i)
		}
	}
	seen := map[int]bool{}
	lo, hi := used[0], used[0]
	for _, u := range used {
		seen[u] = true
		lo, hi = min(lo, u), max(hi, u)
	}
	ensure(t, len(seen) == len(used) && lo > addr["paw_hit"] && hi <= 0x06FF, "dynamic RAM layout")

	var fant []int
	for _, f := range strings.Fields(regexp.MustCompile(`data FanT \{([^}]*)\}`).FindStringSubmatch(main)[1]) {
		v, err := strconv.Atoi(f)
		lv2Need(t, err == nil, "FanT value %q", f)
		fant = append(fant, v)
	}
	want := append(append([]int{}, L.FanA[:]...), L.FanB[:]...)
	ensure(t, len(fant) == len(want), "FanT %v != %v", fant, want)
	for i := range want {
		ensure(t, i < len(fant) && fant[i] == want[i], "FanT %v != %v", fant, want)
	}
}

func TestLevel02MapInvariants(t *testing.T) {
	metas := lv2Metas()
	for n := 0; n < 4; n++ {
		spans := lv2Expect[n]
		want := map[[2]int]bool{}
		for _, s := range spans {
			for c := s[1]; c <= s[2]; c++ {
				want[[2]int{s[0], c}] = true
			}
		}
		got := map[[2]int]bool{}
		for r := 0; r < 24; r++ {
			for c := 0; c < 40; c++ {
				if M.IsSupport(int(P.Rooms[n][r*40+c])) {
					got[[2]int{r, c}] = true
				}
			}
		}
		ensure(t, len(got) == len(want), "room %d: support tile count %d != %d", n, len(got), len(want))
		for k := range want { // decorative tiles never support: exactly the designed platforms
			ensure(t, got[k], "room %d: missing support %v", n, k)
		}
		for k := range got {
			ensure(t, want[k], "room %d: unexpected support %v", n, k)
		}
		row := metas[n]["sy"]/8 - 1
		ensure(t, want[[2]int{row, 2}] && want[[2]int{row, 4}] && metas[n]["sx"] == 8, "room %d entry/respawn", n)
	}
	var sy, exitY []int
	for _, m := range metas {
		sy, exitY = append(sy, m["sy"]), append(exitY, m["exit_y"])
	}
	ensure(t, intsEq(sy, []int{184, 112, 96, 80}), "sy %v", sy)
	ensure(t, intsEq(exitY, []int{112, 96, 80, 56}), "exit_y %v", exitY)
	for i, m := range metas {
		prev := 184
		if i > 0 {
			prev = metas[i-1]["exit_y"]
		}
		ensure(t, m["sy"] == prev, "room %d entry is not previous exit height", i)
		ensure(t, m["arow"] == 0 && m["h0"] == 0 && m["glass"] == 0 && m["dlg"] == 0, "room %d has vacuums/hazards", i) // no vacuums/hazards
	}
	for n := 0; n < 4; n++ {
		row := metas[n]["exit_y"]/8 - 1
		for _, c := range []int{37, 38, 39} {
			ensure(t, M.IsSupport(int(P.Rooms[n][row*40+c])), "room %d exit ledge col %d", n, c)
		}
		closed := byte(L.Hatch)
		if metas[n]["swy"] != 0 {
			closed = L.Wall // controller rooms start with the exit sealed
		}
		for r := row - 3; r < row; r++ {
			ensure(t, P.Rooms[n][r*40+39] == closed, "room %d exit seal row %d", n, r)
		}
	}
	type swm struct{ y, c, p int }
	var sws []swm
	for _, m := range metas {
		sws = append(sws, swm{m["swy"], m["swc"], m["lpow"]})
	}
	ensure(t, sws[0] == swm{104, 9, 1} && sws[1] == swm{} && sws[2] == swm{} && sws[3] == swm{88, 26, 0}, "switch metas %v", sws)
	ensure(t, P.Rooms[0][12*40+9] == L.Ctrl|L.Inv && P.Rooms[3][10*40+26] == L.Ctrl|L.Inv, "console tiles")
	ensure(t, metas[0]["fan"] == 0 && metas[1]["fan"] == 0 && metas[2]["fan"] == 10 && metas[3]["fan"] == 0, "fan metas")
	fm := metas[2]
	ensure(t, fm["fan"] == 10 && fm["fc1"] == 18 && fm["fr0"] == 5 && fm["fr1"] == 11, "fan zone metas")
	for r := 5; r < 11; r++ {
		for c := 10; c < 18; c++ {
			ensure(t, P.Rooms[2][r*40+c] == L.Void, "fan zone not empty at %d,%d", r, c) // zone is empty air
		}
	}
	var blade []int
	for dr := 0; dr < 2; dr++ {
		for dc := 0; dc < 2; dc++ {
			blade = append(blade, int(P.Rooms[2][(fm["fr0"]+2+dr)*40+fm["fan"]-3+dc]))
		}
	}
	ensure(t, intsEq(blade, L.FanA[:]), "blade tiles %v", blade)
	setA, setB := map[int]bool{}, map[int]bool{}
	for i := range L.FanA {
		setA[L.FanA[i]], setB[L.FanB[i]] = true, true
	}
	sameSet := len(setA) == len(setB)
	for k := range setA {
		sameSet = sameSet && setB[k]
	}
	ensure(t, !sameSet, "FanA and FanB use the same tile set")
	font := readAsset("term-font.pic")
	glyphs := append(append([]int{}, L.FanA[:]...), L.FanB[:]...)
	glyphs = append(glyphs, L.AirA, L.AirB, L.Crumb, L.Ctrl)
	for _, g := range glyphs {
		ensure(t, !bytes.Equal(font[g*8:(g+1)*8], make([]byte, 8)), "empty font glyph %d", g)
	}
	ensure(t, !bytes.Equal(font[L.AirA*8:L.AirA*8+8], font[L.AirB*8:L.AirB*8+8]), "air glyphs equal")
}

func intsEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLevel02Room1ServiceShaft(t *testing.T) {
	c1 := P.Rooms[0]
	ex := lv2Expect[0]
	centres := make([]int, 5) // doubled centres: (a+b) avoids fractions
	for i := 0; i < 5; i++ {
		centres[i] = ex[i+1][1] + ex[i+1][2]
	}
	ensure(t, !(centres[1] > centres[0]) && centres[2] > centres[1] && !(centres[3] > centres[2]), "climb zigzag") // left, right, left ...
	for i, r := range []int{20, 18, 16, 14, 12} {
		ensure(t, ex[i+1][0] == r, "climb row %d", i) // ... rising two rows each time
	}
	par1 := lv2Plan(c1, M.Node{0, 184}, nil)
	_, sim, _ := lv2Replay(t, c1, M.Node{0, 184}, M.Node{28, 104}, nil) // controller ledge with the unchanged jump
	ensure(t, sim.Col() == 9, "controller column %d", sim.Col())
	for n := range par1 {
		ensure(t, !(n[1] == 112 && n[0] >= 130), "bypass to exit ledge %v", n) // no walking/jumping bypass to the exit ledge
		ensure(t, n[1] >= 104, "above console ledge %v", n)
	}
	for n := range lv2Plan(c1, M.Node{28, 104}, nil) {
		ensure(t, !(n[1] == 112 && n[0] >= 130), "bypass from console ledge %v", n)
	}
	idle := M.New(c1, 0)
	for i := 0; i < 500; i++ {
		idle.Step(0)
	}
	ensure(t, idle.LiftRow == 22 && idle.SwOn == 0 && idle.Scr[22][33] == L.Lift|0x80, "idle lift") // stationary, red
	for x := 0; x < 141; x++ {                                                                      // FIRE works within 2 columns of the console only
		want := lv2Absi(((x+2)>>2)+2-9) <= 2
		ensure(t, (lv2FireAt(c1, x, 104, 0).SwOn != 0) == want, "fire reach x=%d", x)
	}
	ensure(t, lv2FireAt(c1, 28, 96, 0).SwOn == 0, "wrong height")
	ensure(t, lv2FireAt(c1, 28, 104, 1).SwOn == 0, "airborne")
	held := M.New(c1, 0)
	held.X, held.Y, held.FirePrev = 28, 104, 0 // FIRE already held: no edge
	held.Step(M.Fire)
	held.Step(M.Fire)
	ensure(t, held.SwOn == 0, "held fire switched")
	held.Step(0)
	held.Step(M.Fire)
	ensure(t, held.SwOn != 0, "release+press did not switch")
	s := lv2FireAt(c1, 28, 104, 0)
	ensure(t, s.SwOn != 0 && s.Scr[12][9] == L.Ctrl, "switch/console tile")
	for _, r := range []int{10, 11, 12} {
		ensure(t, s.Scr[r][39] == L.Hatch, "hatch row %d", r)
	}
	for i := 0; i < 40; i++ {
		s.Step(0)
	}
	ensure(t, s.LiftRow != 22 && s.Scr[s.LiftRow][33] == L.Lift, "powered lift") // powered lift runs, no longer red
	exitTry := func(sw, y, air int) bool {
		g := M.New(c1, 0)
		g.X, g.Y, g.SwOn, g.Air = 140, y, sw, air
		g.Step(M.Right)
		return g.Exited
	}
	ensure(t, !exitTry(0, 112, 0) && !exitTry(1, 104, 0) && !exitTry(1, 112, 1), "exit gate open too early") // needs switch, height, ground
	ensure(t, exitTry(1, 112, 0), "exit gate shut")
	sim = M.New(c1, 0) // complete: climb, FIRE, drop, ride the lift, exit
	M.Drive(sim, M.PathTo(par1, M.Node{28, 104}), 0)
	lv2SettlePress(sim, M.Fire)
	lv2Need(t, sim.SwOn != 0, "switch not pressed")
	lv2Need(t, M.Drive(sim, M.PathTo(lv2Plan(c1, M.Node{28, 104}, nil), M.Node{124, 184}), 0), "drop to lift")
	lv2Need(t, M.HoldUntil(sim, 0, func(t *M.Sim) bool { return t.LiftRow == 22 && t.Y == 184 }, 0), "lift bottom")
	lv2Need(t, M.HoldUntil(sim, 0, func(t *M.Sim) bool { return t.LiftRow == 13 }, 0) && sim.Y == 112, "lift top")
	lv2Need(t, M.HoldUntil(sim, M.Right, func(t *M.Sim) bool { return t.X == 140 }, 0) && sim.Y == 112, "walk to gate")
	sim.Step(M.Right)
	sim.Step(M.Right)
	ensure(t, sim.Exited && sim.Deaths == 0, "did not exit")
	cyc := M.New(c1, 0)
	cyc.SwOn = 1
	seen := map[int]bool{}
	for i := 0; i < 2*9*18+20; i++ { // the lift returns to the bottom every cycle: reboard is always possible
		cyc.Step(0)
		seen[cyc.LiftRow] = true
	}
	ensure(t, len(seen) == 10, "lift rows %v", seen)
	for r := 13; r < 23; r++ {
		ensure(t, seen[r], "lift never at row %d", r)
	}
}

func TestLevel02Room2Catwalks(t *testing.T) {
	c2 := P.Rooms[1]
	start, goal := M.Node{0, 112}, M.Node{140, 96}
	crumbOnly := lv2Blank(c2, lv2Span{22, 0, 39}, lv2Span{20, 26, 29}, lv2Span{18, 32, 35}, lv2Span{16, 26, 29}, lv2Span{14, 32, 35})
	stableOnly := lv2Blank(c2, lv2Span{13, 10, 13}, lv2Span{12, 17, 20}, lv2Span{11, 25, 28})
	costS, simS, parS := lv2Replay(t, crumbOnly, start, goal, nil) // each route works alone
	costT, simT, parT := lv2Replay(t, stableOnly, start, goal, nil)
	costF, _, _ := lv2Replay(t, c2, start, goal, nil)
	ensure(t, costF <= costS && costS <= costT-30, "costs %d %d %d", costF, costS, costT) // the shortcut is shorter in frames
	ensure(t, lv2Jumps(parS, goal) < lv2Jumps(parT, goal) && simS.T < simT.T, "shortcut not shorter")
	art := M.New(c2, 0).Art
	ensure(t, art[13][10] == L.Crumb || art[12][17] == L.Crumb || art[11][25] == L.Crumb, "no crumble tiles")
	row := 13
	cm := M.New(c2, 0)
	cm.X, cm.Y = 44, 112 // on the first crumbling run
	col := cm.Col()
	ensure(t, cm.Scr[row][col] == L.Crumb, "tile under is not crumble")
	var log []int
	for i := 0; i < 135; i++ {
		cm.Step(0)
		log = append(log, cm.Scr[row][col])
	}
	ensure(t, log[0] == L.Crumb|0x80 && cm.Deaths == 0, "no warning on first contact") // warning from the first contact frame ...
	for i, v := range log[:39] {
		ensure(t, v == L.Crumb|0x80, "warning frame %d", i) // ... solid and visibly red for 40 frames
	}
	for i, v := range log[39:119] {
		ensure(t, v == L.Void, "gone frame %d", i+39) // then gone
	}
	ensure(t, log[119] == L.Crumb && log[len(log)-1] == L.Crumb, "not restored at frame 120") // and restored at frame 120 without any retry
	ensure(t, cm.Y == 184 && cm.Air == 0, "crumbled cat not caught by floor")                 // no softlock
	rr := M.New(c2, 0)
	rr.X, rr.Y = 44, 112
	for i := 0; i < 45; i++ {
		rr.Step(0)
	}
	ensure(t, rr.Scr[row][col] == L.Void, "crumble not gone")
	rr.Y, rr.Air = 196, 1
	rr.Step(0) // retry (respawn) restores every active crumble at once
	active := false
	for _, sl := range rr.Cr {
		active = active || sl[0] != 0
	}
	ensure(t, rr.Deaths == 1 && rr.Scr[row][col] == L.Crumb && !active, "respawn restore")
	full := M.New(c2, 0)
	full.X, full.Y = 44, 112
	for i := range full.Cr {
		full.Cr[i] = [3]int{1, 0, 0} // every slot busy: the tile stays a plain solid tile, never silently corrupt
	}
	full.Step(0)
	ensure(t, full.Scr[row][full.Col()] == L.Crumb && full.Air == 0, "full slots")
	exp := map[[2]int]bool{}
	for _, s := range lv2Expect[1] {
		for c := s[1]; c <= s[2]; c++ {
			exp[[2]int{s[0], c}] = true
		}
	}
	for _, first := range []int{10, 17, 25} { // lower catches under all crumbles
		found := false
		for r := 14; r < 23; r++ {
			found = found || exp[[2]int{r, first}]
		}
		ensure(t, found, "no catch below column %d", first)
	}
	for x := 0; x <= 140; x += 8 { // from anywhere on the floor the stable route still reaches the exit
		ensure(t, lv2Reach(lv2Plan(stableOnly, M.Node{x, 184}, &M.PlanOpts{XStep: 2}), M.Node{140, 96}), "stable route from x=%d", x)
	}
}

func TestLevel02Room3CoolingChannel(t *testing.T) {
	c3 := P.Rooms[2]
	start, goal := M.Node{0, 96}, M.Node{140, 80}
	windy := lv2Blank(c3, lv2Span{22, 0, 39}, lv2Span{19, 12, 15}, lv2Span{16, 17, 20}, lv2Span{13, 22, 25})
	gusts := M.Range(64, 128, 2)
	costW, _, _ := lv2Replay(t, windy, start, goal, &M.PlanOpts{T0s: gusts}) // the wind-assisted gap alone gets through
	ensure(t, !lv2Reach(lv2Plan(windy, start, nil), goal), "gap passable without wind")
	noGap := lv2Blank(c3, lv2Span{11, 0, 9})
	costSt, _, _ := lv2Replay(t, noGap, M.Node{8, 184}, goal, nil) // longer stable alternative with no wind
	ensure(t, costSt > costW, "stable %d <= wind %d", costSt, costW)

	pushed := func(x, y, before, air int) int {
		f := M.New(c3, 0)
		f.X, f.Y, f.Air, f.Vy, f.FanT = x, y, air, 0, before
		f.Fan()
		return f.X - x
	}
	ensure(t, pushed(34, 80, 64, 1) == 1, "gust push") // gust phase, airborne, inside the visible zone
	ensure(t, pushed(34, 80, 0, 1) == 0, "calm push")
	ensure(t, pushed(34, 80, 64, 0) == 0, "grounded push")
	ensure(t, pushed(20, 80, 64, 1) == 0 && pushed(24, 80, 64, 1) == 0, "nook push")                                                              // waiting nook (cols <= 9) outside the zone
	ensure(t, pushed(28, 80, 64, 1) == 0 && pushed(32, 80, 64, 1) == 1 && pushed(60, 80, 64, 1) == 1 && pushed(64, 80, 64, 1) == 0, "zone edges") // zone starts at col 10 and ends before col 18
	rows := []int{pushed(34, 48, 64, 1), pushed(34, 56, 64, 1), pushed(34, 96, 64, 1), pushed(34, 104, 64, 1)}
	ensure(t, intsEq(rows, []int{0, 1, 1, 0}), "body rows %v", rows) // body rows 5..10 only
	ensure(t, pushed(140, 80, 64, 1) == 0, "push at gate")
	nook := M.New(c3, 0)
	for i := 0; i < 300; i++ {
		nook.Step(0) // waits through gust and calm phases on ledge A, never moved
	}
	ensure(t, nook.X == 0 && nook.Y == 96 && nook.Deaths == 0, "nook cat moved")
	vis := M.New(c3, 0)
	poses := map[bool]map[[4]int]bool{true: {}, false: {}}
	for i := 0; i < 256; i++ {
		vis.Step(0)
		gust := vis.FanT&64 != 0
		for r := 5; r < 11; r++ {
			for c := 10; c < 18; c++ {
				v := vis.Scr[r][c]
				ok := v == L.Void
				if gust {
					ok = v == L.AirA || v == L.AirB
				}
				ensure(t, ok, "airflow tile %d at %d,%d gust=%v", v, r, c, gust) // airflow is drawn exactly in the zone
			}
		}
		if vis.FanT%4 == 0 {
			var b [4]int
			for dr := 0; dr < 2; dr++ {
				for dc := 0; dc < 2; dc++ {
					b[dr*2+dc] = vis.Scr[7+dr][7+dc]
				}
			}
			poses[gust][b] = true
		}
	}
	for _, g := range []bool{true, false} {
		ensure(t, len(poses[g]) == 2 && poses[g][L.FanA] && poses[g][L.FanB], "blade poses %v %v", g, poses[g]) // blades visibly turn in both phases
	}
	sp := M.New(c3, 0)
	turns := map[bool]int{}
	last := -1
	for i := 0; i < 256; i++ {
		sp.Step(0)
		b := sp.Scr[7][7]
		if sp.FanT%4 == 0 {
			if last != -1 && b != last {
				turns[sp.FanT&64 != 0]++
			}
			last = b
		}
	}
	ensure(t, turns[true] > 3*turns[false] && 3*turns[false] > 0, "fan spin turns %v", turns) // the fan spins up during the gust
}

func TestLevel02Room4RoofLift(t *testing.T) {
	c4 := P.Rooms[3]
	start := M.Node{0, 80}
	par4 := lv2Plan(c4, start, nil)
	for n := range par4 {
		ensure(t, !(n[1] == 88 && n[0] >= 58) && !(n[1] <= 64), "lift bypass %v", n) // lift required
	}
	for n := range lv2Plan(c4, M.Node{60, 88}, nil) {
		ensure(t, n[1] != 56, "catwalk reachable from console ledge %v", n) // catwalk is 4 rows above the console ledge
	}
	for _, col := range append(M.Range(18, 22, 1), M.Range(25, 29, 1)...) { // every catwalk miss is caught by the console ledge
		first := -1
		for r := 7; r < 23; r++ {
			if (r == 10 && col >= 17 && col < 32) || r == 22 {
				first = r
				break
			}
		}
		ensure(t, first == 10, "catch under col %d is row %d", col, first)
	}
	s := M.New(c4, 0)
	M.Drive(s, M.PathTo(par4, M.Node{46, 184}), 0)
	lv2Need(t, M.HoldUntil(s, 0, func(t *M.Sim) bool { return t.Y == 88 }, 0), "ride up") // riding up past the console ledge
	onCl := func(t *M.Sim) bool { return t.Y == 88 && t.Col() >= 17 && t.Air == 0 }
	var res []M.JumpPlan
	var ok []bool
	for w := 0; w < 20; w++ {
		r, f := M.SearchJump(s, onCl, []int{w}, nil, 0)
		res, ok = append(res, r), append(ok, f)
	}
	cnt, first := 0, -1
	for i, f := range ok {
		if f {
			cnt++
			if first < 0 {
				first = i
			}
		}
	}
	lv2Need(t, cnt >= 10, "transfer window only %d frames", cnt) // readable transfer: many consecutive frames work
	M.ApplyJump(s, res[first], 0)
	g4 := M.New(c4, 0)
	g4.X, g4.Y = 140, 56 // no remote/early unlock: gate shut, no L03 hand-off
	g4.Step(M.Right)
	g4.Step(M.Right)
	ensure(t, !g4.Exited, "gate open early")
	M.HoldUntil(s, M.Right, func(t *M.Sim) bool { return t.Col() == 26 }, 0)
	far := s.Clone()
	far.X = 60
	far.Step(M.Fire)
	ensure(t, far.SwOn == 0, "remote switch")
	lv2SettlePress(s, M.Fire)
	ensure(t, s.SwOn != 0 && s.Scr[3][39] == L.Hatch && s.Scr[4][39] == L.Hatch && s.Scr[5][39] == L.Hatch && s.Scr[10][26] == L.Ctrl, "switch result")
	M.HoldUntil(s, M.Left, func(t *M.Sim) bool { return t.X <= 58 }, 0)
	onLift := func(t *M.Sim) bool { return t.Air == 0 && t.Y == (t.LiftRow+1)*8 && t.Col()-13 >= 0 && t.Col()-13 < 3 }
	r, f := M.SearchJump(s, onLift, M.Range(0, 330, 2), M.Range(0, 16, 2), 0)
	lv2Need(t, f, "reboard")
	M.ApplyJump(s, r, 0)
	lv2Need(t, M.HoldUntil(s, 0, func(t *M.Sim) bool { return t.Y == 56 }, 0), "ride to roof")
	onTop := func(t *M.Sim) bool { return t.Y == 56 && t.Air == 0 && t.Col() >= 18 }
	r, f = M.SearchJump(s, onTop, M.Range(0, 12, 1), nil, 0)
	lv2Need(t, f, "lift top to catwalk")
	M.ApplyJump(s, r, 0)
	final := lv2Plan(lv2Blank(c4, lv2Span{22, 0, 39}), M.Node{s.X, 56}, nil)
	lv2Need(t, lv2Reach(final, M.Node{140, 56}) && M.Drive(s, M.PathTo(final, M.Node{140, 56}), 0), "catwalk route") // crumbling catwalk, real timers
	s.Step(M.Right)
	s.Step(M.Right)
	ensure(t, s.Exited && s.SwOn != 0 && s.Deaths == 0, "roof exit")
}

func TestLevel02SoftlockClosure(t *testing.T) {
	metas := lv2Metas()
	starts := []M.Node{{0, 184}, {0, 112}, {0, 96}, {0, 80}}
	for n, start := range starts {
		reach := lv2Plan(P.Rooms[n], start, &M.PlanOpts{XStep: 4})
		ensure(t, lv2Reach(reach, M.Node{140, metas[n]["exit_y"]}) || n == 0 || n == 3, "room %d exit unreachable", n) // lift rooms need the dynamic lift
		nodes := make([]M.Node, 0, len(reach))
		for node := range reach {
			nodes = append(nodes, node)
		}
		sort.Slice(nodes, func(i, j int) bool {
			if nodes[i][0] != nodes[j][0] {
				return nodes[i][0] < nodes[j][0]
			}
			return nodes[i][1] < nodes[j][1]
		})
		for _, node := range nodes {
			if node[1] == 184 {
				continue
			}
			lower := false // every ledge position can leave by walking off an edge, to the floor or lower
			for k := range lv2Plan(P.Rooms[n], node, &M.PlanOpts{XStep: 4}) {
				lower = lower || k[1] > node[1]
			}
			ensure(t, lower, "room %d: stuck at %v", n, node)
		}
	}
	for _, x := range []int{0, 70, 140} {
		ensure(t, lv2Reach(lv2Plan(P.Rooms[0], M.Node{x, 184}, nil), M.Node{x, 184}), "floor %d", x)
	}
}
