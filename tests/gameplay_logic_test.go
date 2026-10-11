package tests

// Structural + behavioral checks for Level 01 ("The Blind Spot") gameplay in the src/*.k65
// sources. The K65 source is parsed as text and asserts invariants the compiler cannot check;
// several checks pin down the meaning of specific branches (which literal a= value each
// condition selects) so that reintroducing the switch-polarity bug, the redraw/erase-cat bug or
// the frozen switch_timer bug fails a test, not just a search. Compiling is out of scope.
//
// The Level 02 / Level 04 route proofs that the python file pulled in with runpy are the
// separate TestLevel02* / TestLevel04* tests.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	L "github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
)

func gpBody(name string) string { return funcBody(src(), name) }

func gpHas(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("missing %q", sub)
	}
}

func gpHasNot(t *testing.T, s, sub string) {
	t.Helper()
	if strings.Contains(s, sub) {
		t.Errorf("unexpected %q", sub)
	}
}

func gpRe(t *testing.T, pattern, text string) []string {
	t.Helper()
	m := regexp.MustCompile(pattern).FindStringSubmatch(text)
	if m == nil {
		t.Errorf("no match for %q", pattern)
	}
	return m
}

func gpDataBytes(name string) []int {
	m := regexp.MustCompile(`(?s)data ` + name + ` \{([^}]*)\}`).FindStringSubmatch(src())
	if m == nil {
		panic("gpDataBytes: " + name)
	}
	b := regexp.MustCompile(`//[^\n]*`).ReplaceAllString(m[1], "") // line comments may contain digits
	var out []int
	for _, n := range regexp.MustCompile(`\d+`).FindAllString(b, -1) {
		v, _ := strconv.Atoi(n)
		out = append(out, v)
	}
	return out
}

func gpAscii32(text string) []int {
	out := make([]int, len(text))
	for i, c := range []byte(text) {
		out[i] = int(c) - 32
	}
	return out
}

func gpStripComments(s string) string { return regexp.MustCompile(`//[^\n]*`).ReplaceAllString(s, "") }

func gpHasInt(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

func gpMax(a []int, def int) int {
	if len(a) == 0 {
		return def
	}
	m := a[0]
	for _, v := range a {
		m = max(m, v)
	}
	return m
}

func gpMin(a []int) int {
	m := a[0]
	for _, v := range a {
		m = min(m, v)
	}
	return m
}

func gpMirror(a []int) []int {
	out := make([]int, len(a))
	for i, x := range a {
		out[i] = 15 - x
	}
	sort.Ints(out)
	return out
}

func gpRange(a, b int) []int {
	var out []int
	for i := a; i < b; i++ {
		out = append(out, i)
	}
	return out
}

func gpTail3(a []int) []int {
	if len(a) > 3 {
		return a[len(a)-3:]
	}
	return a
}

// gpFrames returns the 58 logical sprite frames (physical atlas indexed by FrameMap).
func gpFrames() [][]byte {
	sprites := readAsset("cat-sprites.bin")
	fm := readAsset("cat-frame-map.bin")
	out := make([][]byte, len(fm))
	for i, p := range fm {
		out[i] = sprites[int(p)*128 : (int(p)+1)*128]
	}
	return out
}

func gpFur(frame []byte, row int) []int {
	out := []int{}
	for x := 0; x < 16; x++ {
		if frame[64+(x/8)*32+row]&(0x80>>(x%8)) != 0 {
			out = append(out, x)
		}
	}
	return out
}

func gpSprites(t *testing.T) [][]byte {
	t.Helper()
	main := src()
	sprites := readAsset("cat-sprites.bin")
	fm := readAsset("cat-frame-map.bin") // packed atlas: identical drawings share a physical frame
	mx := 0
	for _, v := range fm {
		mx = max(mx, int(v))
	}
	if len(fm) != 58 || len(sprites) != (mx+1)*128 || len(sprites) != 43*128 {
		t.Fatalf("atlas sizes: map %d sprites %d", len(fm), len(sprites))
	}
	gpHas(t, main, `binary "assets/cat-frame-map.bin"`)
	seen := map[byte]bool{}
	for _, v := range fm {
		seen[v] = true
	}
	for i := 0; i < 43; i++ {
		if !seen[byte(i)] {
			t.Fatalf("frame map lacks physical frame %d", i)
		}
	}
	if len(seen) != 43 {
		t.Fatalf("frame map has %d distinct values", len(seen))
	}
	return gpFrames()
}

func gpPyMod(a, b float64) float64 {
	r := math.Mod(a, b)
	if r < 0 {
		r += b
	}
	return r
}

func TestGameplayCameras(t *testing.T) {
	gpHas(t, gpBody("repaint_current"), "cam_sees")
	gpHas(t, gpBody("check_detection"), "cam_sees")
	cams := readAsset("level-cams.bin")
	ensure(t, len(cams) == 3*2*8, "level-cams size %d", len(cams))
	ensure(t, len(readAsset("cone-step.bin")) == 160, "cone-step size")
	for r := 0; r < 3; r++ {
		for s := 0; s < 2; s++ {
			e := cams[(r*2+s)*8 : (r*2+s+1)*8]
			col, lo, hi, speed, hw := int(e[0]), int(e[1]), int(e[2]), int(e[3]), int(e[5])
			if col != 0 {
				ensure(t, lo <= col && col <= hi && hi <= 39 && speed != 0 && hw != 0, "camera %d/%d bounds", r, s)
				ensure(t, lo-hw > 2, "camera %d/%d watches the entry column", r, s) // room entry column is never watched
				// The widest cone row (cone_hwh starts at 1, gains a pixel per carry out of the
				// fraction) must end exactly on the detected footprint, or the cat walks
				// through a lit column that never alarms.
				hwstep := int(e[7])
				ensure(t, 1+L.ConeRows*hwstep/256 == hw*4+1, "camera %d/%d cone width", r, s)
			}
		}
	}
	// Only the rack shadow hides the cat: a tile that is drawn (relay terminal, hatch) is
	// watched like any other floor, and drawn inverted while watched.
	gpHas(t, gpBody("cam_sees"), "a?78")
	gpHas(t, gpBody("repaint_current"), "rp_fixed")
	gpHas(t, gpBody("repaint_current"), "0x80")
	// check_detection tests both ends of the 16-pixel cat, not just its middle column.
	gpHas(t, gpBody("check_detection"), "c- a+3")
	gpHas(t, gpBody("check_detection"), "c- a+12")

	cam2 := L.Cameras[2][1]
	ensure(t, !gpRoom3Playable(t, 30, 7, 100), "old room-3 sweep is playable")
	ensure(t, gpRoom3Playable(t, cam2.Lo, cam2.Speed, cam2.Pause), "room-3 sweep is not playable")
	// Only room 1's first camera is gated by the switch.
	var gates []int
	for r := 0; r < 3; r++ {
		for s := 0; s < 2; s++ {
			gates = append(gates, int(cams[(r*2+s)*8+6]))
		}
	}
	ensure(t, intsEq(gates, []int{0, 0, 1, 0, 0, 0}), "camera gates %v", gates)
	// Switch polarity: switch_timer counts frames the gated camera stays dark.
	if m := regexp.MustCompile(`var switch_timer.*`).FindString(src()); m == "" || !strings.Contains(m, "dark") {
		t.Errorf("switch_timer var lacks dark comment")
	}
	gpRe(t, `a=switch_timer\s*\n\s*a\?0\s*\n\s*==\s*\{\s*a=1\s*\}\s*else\s*\{\s*a=0\s*\}`, gpBody("tick_cam"))
	// switch_timer must count down every gameplay frame; cameras tick too.
	ensure(t, regexp.MustCompile(`switch_timer\s*--`).MatchString(gpBody("tick_switch")), "tick_switch lacks switch_timer--")
	gf := gpBody("game_frame")
	gpHas(t, gf, "tick_switch")
	gpHas(t, gf, "tick_cams")
}

// The cat is four columns wide and walks one column in 12 frames while the room-3 exit camera
// sweeps one in 9, so the floor past the shelter can only be crossed while the camera waits at
// an end. Replay the sweep pixel by pixel and check the terminal can be held for the full
// recording and the hatch reached afterwards.
func gpRoom3Playable(t *testing.T, lo, speed, pause int) bool {
	t.Helper()
	relayFrames := func() int {
		m := gpRe(t, `a=relay_progress\s*\n\s*a\?(\d+)`, gpBody("handle_relay"))
		if m == nil {
			t.FailNow()
		}
		v, _ := strconv.Atoi(m[1])
		return v
	}()
	cam := L.Cameras[2][1]
	hi, hw := cam.Hi, cam.HalfWidth
	floor := L.Room(2)[22*40 : 23*40]
	centre, direction, wait := lo, 0, speed
	var sweep []int
	for i := 0; i < 2*(hi-lo)*speed+2*pause; i++ {
		sweep = append(sweep, centre)
		if wait != 0 {
			wait--
			continue
		}
		wait = speed
		if direction == 0 {
			centre++
		} else {
			centre--
		}
		if centre == hi {
			direction, wait = 1, pause
		}
		if centre == lo {
			direction, wait = 0, pause
		}
	}
	cycle := len(sweep)
	safe := func(x, tt int) bool { // both end columns of the cat, as check_detection tests
		for _, col := range []int{(x + 3) >> 2, (x + 12) >> 2} {
			d := col - sweep[tt%cycle]
			if floor[col] != L.Shadow && d >= -hw && d <= hw {
				return false
			}
		}
		return true
	}
	type st struct {
		x, t int
		done bool
	}
	var queue []st
	seen := map[st]bool{}
	for tt := 0; tt < cycle; tt++ {
		if safe(0, tt) { // room entrance
			s := st{0, tt, false}
			seen[s] = true
			queue = append(queue, s)
		}
	}
	for len(queue) > 0 {
		cur := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if cur.x >= 140 && cur.done {
			return true
		}
		type mv struct{ target, frames int }
		moves := []mv{{cur.x + 1, 3}, {cur.x - 1, 3}, {cur.x, 1}} // 1 pixel per 3 frames
		if !cur.done && ((cur.x+2)>>2)+2 == 33 {
			moves = append(moves, mv{cur.x, relayFrames})
		}
		for _, m := range moves {
			if m.target < 0 || m.target > 140 {
				continue
			}
			ok := true
			for k := 1; k <= m.frames && ok; k++ {
				ok = safe(m.target, cur.t+k)
			}
			if ok {
				n := st{m.target, (cur.t + m.frames) % cycle, cur.done || m.frames == relayFrames}
				if !seen[n] {
					seen[n] = true
					queue = append(queue, n)
				}
			}
		}
	}
	return false
}

func TestGameplayRedrawFollowedByDrawPlayer(t *testing.T) {
	// redraw_zone repaints monitored floor and sprite rows; after every pass the four-column
	// cat must be restored, including during alert recovery.
	main := src()
	var sites []int
	for _, m := range regexp.MustCompile(`\bredraw_zone\b`).FindAllStringIndex(main, -1) {
		if main[max(0, m[0]-5):m[0]] != "func " {
			sites = append(sites, m[0])
		}
	}
	ensure(t, len(sites) >= 2, "expected redraw_zone called from game_frame's two branches")
	for _, pos := range sites {
		tail := main[pos+len("redraw_zone") : min(len(main), pos+len("redraw_zone")+200)]
		next := regexp.MustCompile(`\S+`).FindString(gpStripComments(tail))
		ensure(t, next == "draw_player", "redraw_zone call not immediately followed by draw_player: %q", tail)
	}
}

func TestGameplaySpriteAtlas(t *testing.T) {
	frames := gpSprites(t)
	var packed []byte
	for _, f := range frames[:50] {
		packed = append(packed, f...)
	}
	h := sha256.Sum256(packed)
	ensure(t, hex.EncodeToString(h[:]) == "015127fe60b2317ff0b6e85f8503c8a8d122624b32cfdaa93b57af2003c159b6", "frames[:50] hash %x", h)
	uniq := func(fs [][]byte) int {
		m := map[string]bool{}
		for _, f := range fs {
			m[string(f)] = true
		}
		return len(m)
	}
	ensure(t, len(frames) == 58 && uniq(frames) == 43 && len(readAsset("cat-sprites.bin")) < 50*128, "packed atlas")
	ensure(t, regexp.MustCompile(`(?s)func draw_frame \{[^}]*?x=a\s*a=FrameMap,x\s*x=a`).MatchString(src()), "draw_frame FrameMap")
	ensure(t, uniq(frames[:12]) == 6 && !bytes.Equal(frames[12], frames[13]), "stride frames")
	for i, f := range frames[:14] {
		ensure(t, f[23]|f[32+23]|f[22]|f[32+22] == 0, "dark legs in frame %d", i) // no dark legs
	}
	for i, f := range frames {
		for chunk := 0; chunk < 4; chunk++ {
			for _, b := range f[chunk*32+24 : chunk*32+32] {
				ensure(t, b == 0, "frame %d has pixels below row 24", i)
			}
		}
	}
	n := 0
	for _, f := range frames[:12] {
		if f[64+23]|f[96+23] != 0 {
			n++
		}
	}
	ensure(t, n >= 8, "only %d stride frames touch the floor", n)
	fur := func(k, row int) []int { return gpFur(frames[k], row) }

	for k := 29; k < 32; k++ {
		var paws []int
		for row := 18; row < 24; row++ {
			paws = append(paws, fur(k, row)...)
		}
		ensure(t, len(paws) > 0 && gpMin(paws) >= 1 && gpMax(paws, 0) <= 14, "paws of frame %d", k)
	}
	for k := 29; k < 31; k++ {
		var rows []int
		for row := 4; row < 12; row++ {
			rows = append(rows, fur(k, row)...)
		}
		muzzle := gpMax(rows, 0)
		ensure(t, gpMax(fur(k, 12), -1) > muzzle && gpMax(fur(k, 14), -1) > muzzle, "ears frame %d", k)
		ensure(t, intsEq(gpTail3(fur(k, 12)), []int{13, 14, 15}) && intsEq(gpTail3(fur(k, 14)), []int{13, 14, 15}), "ear tips frame %d", k)
	}
	differs := false
	for row := 10; row < 17; row++ {
		differs = differs || !intsEq(fur(30, row), fur(12, row))
	}
	ensure(t, differs, "frame 30 equals frame 12 over rows 10..16")
	ensure(t, uniq(frames[28:36]) == 8, "frames 28..35 distinct")
	// Fore paws touch the floor before the hind paws (frame 32).
	ensure(t, len(fur(32, 23)) > 0 && gpMin(fur(32, 23)) >= 11, "frame 32 fore paws")
	for k := 31; k < 33; k++ {
		hind, fore := -1, -1
		for row := 17; row < 24; row++ {
			for _, x := range fur(k, row) {
				if x < 8 {
					hind = row
				}
				if x >= 10 {
					fore = row
				}
			}
		}
		ensure(t, fore == 23 && hind <= 20, "frame %d: fore %d hind %d", k, fore, hind)
	}
	hindDown := false
	for _, x := range fur(33, 23) {
		hindDown = hindDown || x < 8
	}
	ensure(t, hindDown, "frame 33 hind paws not down")
	ensure(t, intsEq(fur(33, 23), fur(34, 23)), "planted paws slide")             // planted paws do not slide
	ensure(t, bytes.Equal(frames[35], frames[12]), "recovery does not join idle") // recovery joins the idle pose without a snap
	var head []int
	for _, k := range []int{33, 34, 35} {
		first := -1
		for row := 0; row < 24 && first < 0; row++ {
			for _, x := range fur(k, row) {
				if x >= 10 {
					first = row
					break
				}
			}
		}
		head = append(head, first)
	}
	ensure(t, head[0] > head[1] && head[1] > head[2], "head rows %v", head)
	for k := 0; k < 8; k++ {
		for row := 0; row < 24; row++ {
			ensure(t, intsEq(fur(36+k, row), gpMirror(fur(28+k, row))), "mirror frame %d row %d", 36+k, row)
		}
	}
	ensure(t, gpHasInt(fur(44, 3), 14) && gpHasInt(fur(45, 19), 0), "swat frames 44/45")
	ensure(t, gpHasInt(fur(48, 18), 15) && gpHasInt(fur(48, 19), 13) && gpHasInt(fur(48, 19), 14), "frame 48")
	ensure(t, !bytes.Equal(frames[44], frames[45]) && !bytes.Equal(frames[44], frames[12]), "frames 44/45/12")
	for k := 0; k < 2; k++ {
		for row := 0; row < 24; row++ {
			ensure(t, intsEq(fur(46+k, row), gpMirror(fur(44+k, row))), "mirror frame %d row %d", 46+k, row)
		}
	}
	for row := 0; row < 24; row++ {
		ensure(t, intsEq(fur(49, row), gpMirror(fur(48, row))), "mirror frame 49 row %d", row)
	}
	ensure(t, gpHasInt(fur(56, 12), 15) && 11 >= gpMax(fur(55, 12), 0), "frames 55/56")
	ensure(t, gpMax(fur(54, 12), 0) <= 11, "frame 54")
	glass := map[[2]int]bool{}
	for py, row := range L.TileGlyphs[L.Glass] {
		for px, v := range row {
			if v != '0' {
				glass[[2]int{33*4 + px, 128 + py}] = true
			}
		}
	}
	for _, logical := range []int{54, 55} {
		for origin := 117; origin < 121; origin++ {
			for y := 0; y < 24; y++ {
				for x := 0; x < 16; x++ {
					lit := false
					for colour := 0; colour < 2; colour++ {
						lit = lit || frames[logical][(colour*2+x/8)*32+y]&(0x80>>(x%8)) != 0
					}
					ensure(t, !(lit && glass[[2]int{origin + x, 120 + y}]), "frame %d origin %d overlaps glass at %d,%d", logical, origin, x, y)
				}
			}
		}
	}
	for origin := 117; origin < 121; origin++ {
		ensure(t, glass[[2]int{origin + 15, 132}], "glass pixel for origin %d", origin)
		ensure(t, (frames[56][(0*2+1)*32+12]|frames[56][(1*2+1)*32+12])&1 != 0, "frame 56 reach")
	}
}

func TestGameplayDrawAndPhysicsSource(t *testing.T) {
	draw := gpBody("draw_frame")
	for _, p := range []string{"PmMem,x", "PmMem+256,x", "PmMem+512,x", "PmMem+768,x", "HPOSP0", "HPOSP3"} {
		gpHas(t, draw, p)
	}
	dp := gpBody("draw_player")
	gpHas(t, dp, "a=player_x")
	gpHas(t, dp, "a?12")
	gpHas(t, dp, "c- a+14")
	gpHas(t, gpBody("plat_input"), "jprep--")
	gpHas(t, gpBody("plat_input"), "jprep=a=3")
	gpHas(t, dp, "a=jprep")
	gpHas(t, dp, "a=28 jump_facing return")
	gpRe(t, `a\?1\s+>= \{ a=31 \} else \{ a=30 \}`, dp)
	gpHas(t, gpBody("plat_phys"), "lnd=a=10")
	gpRe(t, `a\?9\s+>= \{ a=32 \}`, dp)
	gpRe(t, `a\?6\s+>= \{ a=33 \}`, dp)
	gpRe(t, `a\?3\s+>= \{ a=34 \} else \{ a=35 \}`, dp)
	gpRe(t, `a=lnd\s+a\?0\s+!= \{ return \}`, gpBody("plat_input"))
	gpHas(t, gpBody("jump_facing"), "c- a+8")
	gpRe(t, `a=vy\s+a\?0\s+== \{ gt=a=2 \}`, gpBody("plat_phys"))
	gpHas(t, dp, "paw_t--")
	gpHas(t, dp, "c- a+44")
	gpHas(t, dp, "c- a+2")
	gpRe(t, `paw_t--\s+a=paw_t\s+a\?3\s+< \{ a=28 jump_facing return \}`, dp)
	gpHas(t, dp, "a=48")
	gpHas(t, dp, "a=49")
	gpHas(t, dp, "a?7")
}

func TestGameplaySwatAndPawContact(t *testing.T) {
	frames := gpSprites(t)
	fur := func(k, row int) []int { return gpFur(frames[k], row) }
	swat := gpBody("swat")
	ensure(t, mustIndex(t, swat, "paw_t=a=10") < mustIndex(t, swat, "on_vacuum"), "swat order")
	gpHas(t, swat, "paw_kind=a=2") // seated: hind shove; grounded stays the forepaw swat
	gpHasNot(t, swat, "act_wob=")  // FIRE alone cannot frighten the vacuum
	gpHasNot(t, swat, "act_dir=")
	contact := gpBody("paw_contact")
	gpRe(t, `a=paw_t\s+a\?4\s+< \{ return \}\s+a\?8\s+>= \{ return \}`, contact)
	gpHas(t, contact, "a=paw_hit")
	gpHas(t, contact, "paw_hit=a=1")
	gpHas(t, swat, "paw_hit=a")
	gpRe(t, `a\?kk\s+== \{ return \}\s+\} always`, contact) // loop needs JMP, not a relative branch
	gpHas(t, contact, "a=(sp),y")
	gpHas(t, contact, "a&PixelMask,x")
	gpHas(t, contact, "a^1")
	pf := gpBody("plat_frame")
	ensure(t, mustIndex(t, pf, "plat_actor") < mustIndex(t, pf, "paw_both") && mustIndex(t, pf, "paw_both") < mustIndex(t, pf, "plat_hit"), "plat_frame order")
	gpHas(t, gpBody("paw_both"), "paw_contact")
	ensure(t, strings.Count(gpBody("paw_both"), "paw_contact") == 2, "paw_both contact count")
	px, py := gpDataBytes("PawX"), gpDataBytes("PawY")
	ensure(t, len(px) == 7 && len(py) == 7, "paw tips")
	ensure(t, intsEq(gpDataBytes("PawK"), []int{0, 3, 5}) && intsEq(gpDataBytes("PawE"), []int{3, 5, 7}), "PawK/PawE")
	type tip struct{ x, y int }
	tips := make([]tip, 7)
	for i := range tips {
		tips[i] = tip{px[i], py[i]}
	}
	for _, c := range []struct {
		frame  int
		mirror bool
		group  []tip
	}{{48, false, tips[:3]}, {49, true, tips[:3]}, {45, false, tips[3:5]}, {47, true, tips[3:5]}, {52, false, tips[5:]}, {53, true, tips[5:]}} {
		for _, p := range c.group {
			x := p.x
			if c.mirror {
				x = 15 - p.x
			}
			ensure(t, gpHasInt(fur(c.frame, p.y), x), "paw tip %v frame %d", p, c.frame)
		}
	}
	for _, frame := range []int{52, 53} { // seated shove: toes on the floor row, haunch folded above them
		ensure(t, len(fur(frame, 23)) > 0 && len(fur(frame-2, 23)) > 0, "toes frame %d", frame)
		low := -1
		for row := 0; row < 24; row++ {
			if len(fur(frame, row)) > 0 {
				low = row
			}
		}
		ensure(t, low == 23, "lowest row of frame %d is %d", frame, low)
	}
	ensure(t, !bytes.Equal(frames[50], frames[52]) && !bytes.Equal(frames[51], frames[53]) && !bytes.Equal(frames[50], frames[12]), "seated frames")
	ensure(t, intsEq(fur(50, 0), []int{9, 12}), "seated head/ears") // not the standing back and raised tail
	for y := 0; y < 9; y++ {
		for _, x := range fur(50, y) {
			ensure(t, x >= 5, "seated frame 50 row %d x=%d", y, x)
		}
	}
	for y := 0; y < 8; y++ {
		ensure(t, len(fur(50, y)) <= 7, "seated frame 50 row %d wide", y)
	}
	for y := 18; y < 24; y++ {
		ensure(t, gpHasInt(fur(50, y), 12), "seated frame 50 row %d lacks tail", y)
	}
	ensure(t, intsEq(fur(50, 21), append(gpRange(1, 9), 12)), "frame 50 row 21")
	ensure(t, intsEq(fur(50, 22), append(gpRange(0, 9), 12)), "frame 50 row 22")
	ensure(t, intsEq(fur(50, 23), append(gpRange(0, 9), 11, 12, 13)), "frame 50 row 23") // original roof/catch footprint
	for row := 0; row < 24; row++ {
		ensure(t, intsEq(fur(51, row), gpMirror(fur(50, row))), "mirror 51 row %d", row)
		ensure(t, intsEq(fur(53, row), gpMirror(fur(52, row))), "mirror 53 row %d", row)
	}
	font := readAsset("term-font.pic")
	actor := gpDataBytes("ActT")
	pawTouches := func(px, py, facing, col, row, timer int) bool {
		if !(4 <= timer && timer < 8) {
			return false
		}
		for _, p := range tips[:3] {
			x := p.x
			if facing != 0 {
				x = 15 - p.x
			}
			x = px + x - col*4
			y := py + p.y - (24 + row*8)
			if 0 <= x && x < 12 && 0 <= y && y < 16 {
				tile := actor[(y/8)*3+x/4]
				if int(font[tile*8+y%8])&(192>>(2*(x%4))) != 0 {
					return true
				}
			}
		}
		return false
	}
	pt := func(px, py, facing int) bool { return pawTouches(px, py, facing, 10, 21, 7) }
	ensure(t, !pt(24, 184, 0) && pt(25, 184, 0), "paw gap/contact") // one-pixel gap vs contact
	ensure(t, !pt(52, 184, 1) && pt(51, 184, 1), "mirrored paw gap/contact")
	ensure(t, !pt(25, 185, 0) && pt(26, 185, 0), "paw silhouette corner") // transparent silhouette corner
	ensure(t, !pawTouches(25, 184, 0, 10, 21, 10), "raised paw hits")     // raised paw does not hit
	ensure(t, !pawTouches(25, 184, 0, 10, 21, 3), "recoil hits")          // recoil cannot hit
	ensure(t, !pt(25, 160, 0), "wrong height hits")                       // wrong height cannot hit
	seatedTouch := func(px, facing, bcol, timer int) bool {
		// seated rider origin 171 on a row-21 carrier; pursuer dome tiles drawn with the chase lamp glyphs
		if !(4 <= timer && timer < 8) {
			return false
		}
		for _, p := range tips[5:] {
			x := p.x
			if facing != 0 {
				x = 15 - p.x
			}
			wx := px + x - bcol*4
			wy := 171 + p.y - (24 + 21*8)
			if 0 <= wx && wx < 12 && 0 <= wy && wy < 16 {
				tile := actor[6+(wy/8)*3+wx/4]
				if int(font[tile*8+wy%8])&(192>>(2*(wx%4))) != 0 {
					return true
				}
			}
		}
		return false
	}
	st := func(px, facing, bcol int) bool { return seatedTouch(px, facing, bcol, 7) }
	maxY := 0
	for _, p := range tips[5:] {
		maxY = max(maxY, p.y)
	}
	ensure(t, maxY == 23, "seated toes row %d", maxY) // toes reach world Y 194, the dome's first visible row
	catX := func(col int) int { return col*4 - 8 }    // player_col = ((x+2)//4)+2 -> col -2 gives the sprite origin
	// right-facing rider (hind paw trailing left): pursuer dome ends just behind the toes
	ensure(t, st(catX(20), 0, 16) && !st(catX(20), 0, 14), "right-facing rider")
	ensure(t, !seatedTouch(catX(20), 0, 16, 3) && !seatedTouch(catX(20), 0, 16, 10), "rider timers")
	reach := func(facing int) []int {
		var out []int
		for b := 10; b < 40; b++ {
			if st(catX(20), facing, b) {
				out = append(out, b)
			}
		}
		return out
	}
	// mirrored left-facing rider reaches a pursuer on the right only, never the one behind
	ensure(t, intsEq(reach(1), []int{20, 21}), "left-facing reach %v", reach(1))
	ensure(t, intsEq(reach(0), []int{16, 17}), "right-facing reach %v", reach(0))
	// pursuer spacing 4 behind a carrier at col 19: reachable from cat cols 18..19, not from the far end of the roof
	ensure(t, st(catX(18), 0, 15) && st(catX(19), 0, 15) && !st(catX(21), 0, 15), "pursuer spacing")

	brushes := readAsset("vacuum-brushes.bin")
	b3 := map[string]bool{}
	for i := 0; i < 12; i += 3 {
		b3[string(brushes[i:i+3])] = true
	}
	ensure(t, len(brushes) == 24 && len(b3) == 4, "vacuum brushes")
	gpHas(t, gpBody("spin_brushes"), "TermFont+765,y=a")
	gpHas(t, gpBody("spin_brushes"), "TermFont+1005,y=a")
	gpHas(t, gpBody("plat_actor"), "spin_brushes")
	gpHas(t, gpBody("vacuum_roof"), "c- a+3")
	for _, name := range []string{"under", "swat", "plat_actor"} {
		gpHas(t, gpBody(name), "on_vacuum")
	}
	gpHas(t, gpBody("hit_check"), "vacuum_roof")
	gpHasNot(t, gpBody("under"), "a?91")
	gpHas(t, gpBody("under"), "a?94")
	gpRe(t, `player_y\+\+\s+under\s+a\?0`, gpBody("plat_phys"))
	pa := gpBody("plat_actor")
	ensure(t, mustIndex(t, pa, "ride=a=0") < mustIndex(t, pa, "a?m_aspd"), "plat_actor ride order")
	// The paws rest on the first visible roof row, not five pixels into its dome.
	for _, row := range []int{18, 21} {
		roof := row*8 + 3
		ensure(t, roof+23 == 24+row*8+2, "roof %d", row)
		for speed := 1; speed < 6; speed++ {
			y := roof - 1
			for i := 0; i < speed; i++ {
				y++
				if y == roof {
					break
				}
			}
			ensure(t, y == roof, "roof skipped at speed %d", speed) // a non-tile-aligned roof cannot be skipped while falling
		}
	}
	gpRe(t, `a=act_wob,x\s+a\?0\s+!= \{ return \}`, gpBody("hit_check"))
}

func TestGameplayExitAndMetadata(t *testing.T) {
	main := src()
	// Exit gates precede both room advancement and the level-ending hand-off.
	exit := gpBody("plat_exit")
	ensure(t, regexp.MustCompile(`^\s*plat_exit_allowed\s+a\?0\s+== \{ return \}`).MatchString(gpStripComments(exit)), "plat_exit gate not first")
	ensure(t, mustIndex(t, exit, "plat_exit_allowed") < mustIndex(t, exit, "cur_room++"), "gate after cur_room++")
	ensure(t, mustIndex(t, exit, "plat_exit_allowed") < mustIndex(t, exit, "call level_ending"), "gate after level_ending")
	gate := strings.TrimSpace(gpStripComments(gpBody("plat_exit_allowed")))
	// unrestricted -> controller gate (sw_on) -> grounded -> exact exit height
	ensure(t, regexp.MustCompile(`a=m_exit\s+a\?0\s+== \{ a=1 return \}\s+a=m_swy\s+a\?0\s+!= \{\s+a=sw_on\s+`+
		`a\?0\s+== \{ return \}\s+\}\s+a=air\s+a\?0\s+!= \{ a=0 return \}\s+`+
		`a=player_y\s+a\?m_exit\s+== \{ a=1 return \}\s+a=0$`).MatchString(gate), "plat_exit_allowed chain")

	// Metadata layout: 32 bytes copied, new fields directly after m_exit and below lift_row.
	ensure(t, len(readAsset("plat-rooms.bin")) == 10*1024, "plat-rooms size")
	addr := map[string]int{}
	for _, m := range regexp.MustCompile(`var (m_\w+) = (0x[0-9A-Fa-f]+)`).FindAllStringSubmatch(main, -1) {
		v, _ := strconv.ParseInt(m[2][2:], 16, 64)
		addr[m[1]] = int(v)
	}
	ensure(t, addr["m_exit"]-addr["m_sx"] == 18, "m_exit offset")
	var offs []int
	for _, n := range []string{"m_swy", "m_swc", "m_lpow", "m_fc1", "m_fr0", "m_fr1", "m_lspd"} {
		offs = append(offs, addr[n]-addr["m_sx"])
	}
	ensure(t, intsEq(offs, []int{19, 20, 21, 22, 23, 24, 25}), "metadata offsets %v", offs)
	gpHas(t, gpBody("enter_plat_room"), "y?32")
	ensure(t, 0x06B0+32 <= varAddress(main, "lift_row"), "lift_row overlaps metadata")

	pmg := gpBody("init_pmg")
	gpHas(t, pmg, "PMBASE=a=0xB8")
	gpHas(t, pmg, "SDMCTL=a=0x3A")
	gpHas(t, pmg, "GRACTL=a=2")
	gpHas(t, gpBody("init_gameplay"), "init_pmg")
	gpHas(t, gpBody("game_frame"), "tick_cat")
	ensure(t, strings.Count(gpBody("read_and_move"), "cat_stepped") == 3, "cat_stepped count")
	gpHas(t, gpBody("cat_stepped"), "set_col")
	gpHas(t, gpBody("enter_room"), "set_col")
	gpHas(t, gpBody("read_and_move"), "a?140")
	gpHas(t, gpBody("read_and_move"), "player_x=a=144")
	gpHas(t, gpBody("trigger_alert"), "player_x=a=0")
	roomArt := readAsset("level-rooms.pic")
	ensure(t, len(roomArt) == 3*40*24, "level-rooms size")
	for r := 0; r < 3; r++ {
		row := roomArt[r*960+880 : r*960+920]
		has70 := false
		for _, v := range row {
			ensure(t, v == 70 || v == 78 || v == 81 || v == 82 || v == 83, "room %d floor tile %d", r, v)
			has70 = has70 || v == 70
		}
		ensure(t, has70 && row[2] != 70, "room %d floor row", r)
	}
}

func TestGameplayRelayHudAndMain(t *testing.T) {
	main := src()
	// relay_done is permanent progress and must never be reset by trigger_alert.
	ensure(t, !regexp.MustCompile(`relay_done\s*(=|\+\+|--)`).MatchString(gpBody("trigger_alert")), "trigger_alert touches relay_done")
	// handle_relay must not touch relay_done unless relay_progress has reached the completion threshold.
	hr := gpBody("handle_relay")
	gpRe(t, `a=relay_progress\s*\n\s*a\?90`, hr)
	ensure(t, mustIndex(t, hr, "a?90") < mustIndex(t, hr, "relay_done=a=1"), "relay_done before threshold")

	// The objective stays visible and is restored after an alert.
	gf := gpBody("game_frame")
	gpHasNot(t, gf, "hint_timer")
	branch := gf[mustIndex(t, gf, "alert_timer--"):]
	gpHas(t, branch, "clear_row0")
	gpHas(t, branch, "redraw_zone")
	gpHas(t, branch, "&<MsgStop")
	gpHas(t, branch, "draw_row0_msg")

	// HUD strings: the ascii-32 data tables must match the documented literals, and the msg_len
	// used at each call site must match the table's actual length.
	hud := []struct{ name, text string }{
		{"MsgStop", "PLAY RECORD. STAY UNSEEN."},
		{"MsgAlert", "UNRECOGNIZED DEVICE"},
		{"MsgRoute", "ROUTE: TEST / ISOLATED"},
	}
	gpHas(t, gpBody("Txt"), `"PHASE 02 DELAYED. AGENCY UNDETECTED."`)
	gpHas(t, main, "TxtOff { 0 36")
	for _, h := range hud {
		ensure(t, intsEq(gpDataBytes(h.name), gpAscii32(h.text)), "%s data", h.name)
		for _, cs := range regexp.MustCompile(`&<`+h.name+`\b`).FindAllStringIndex(main, -1) {
			tail := main[cs[1]:min(len(main), cs[1]+200)]
			m := regexp.MustCompile(`msg_len=a=(\d+)`).FindStringSubmatch(tail)
			if m == nil {
				t.Errorf("%s: no msg_len after call site", h.name)
				continue
			}
			n, _ := strconv.Atoi(m[1])
			ensure(t, n == len(h.text), "%s msg_len %d != %d", h.name, n, len(h.text))
		}
	}
	// "CONTROL INFRASTRUCTURE" is reused verbatim from the intro's Line4 data.
	header := gpBody("draw_room_header")
	gpHas(t, header, "&<Line4+10")
	gpHas(t, header, "msg_len=a=22")
	line4 := gpDataBytes("Line4")
	ensure(t, len(line4) >= 32 && intsEq(line4[10:32], gpAscii32("CONTROL INFRASTRUCTURE")), "Line4 substring")

	// The map loader must initialize both background row pointers after its three-page copy,
	// and room entry must repaint the dynamic play row.
	er := gpBody("enter_room")
	gpHas(t, er, "redraw_zone")
	gpHas(t, gpBody("redraw_zone"), "repaint_current")
	ensure(t, mustIndex(t, er, "load_room_art") < mustIndex(t, er, "load_cams"), "enter_room order")
	loader := gpBody("load_room_art")
	gpHas(t, loader, "call load_room_chunk") // a3e.3: per-room disk chunk
	gpHas(t, loader, "&<RoomArtBuf")
	gpHas(t, gpBody("load_room_chunk"), "call disk_read")
	gpHas(t, gpBody("load_room_chunk"), "a=ChunkTable,x")
	gpHas(t, gpBody("read_and_move"), "cur_room--")
	gpHas(t, gpBody("read_and_move"), "cur_room++")

	// The gameplay hand-off must happen after cut_to_black, and level_finished must gate the
	// main loop so the ending freezes the game once it plays.
	sm := regexp.MustCompile(`(?s)main \{(.*)\}\s*$`).FindStringSubmatch(main)
	if sm == nil {
		t.Fatal("no main block")
	}
	scene := sm[1]
	ensure(t, mustIndex(t, scene, "init_system") < mustIndex(t, scene, "build_storm_list") && mustIndex(t, scene, "build_storm_list") < mustIndex(t, scene, "video_call"), "main order")
	gpHasNot(t, scene[:mustIndex(t, scene, "build_storm_list")], "goto intro_skip") // intro enabled
	tail := scene[mustIndex(t, scene, "cut_to_black"):]
	ensure(t, mustIndex(t, tail, "cut_to_black") < mustIndex(t, tail, "init_gameplay") && mustIndex(t, tail, "init_gameplay") < mustIndex(t, tail, "frame_any"), "hand-off order")
	gpHas(t, tail, "level_finished")

	// HUD row 24 sits below the floor, recoloured by its own DLI, refreshed every frame.
	dl := gpBody("dl_term")
	ensure(t, strings.Contains(dl, "0x90") && strings.Index(dl, "0x90") < strings.LastIndex(dl, "0x04"), "dl_term")
	gpHas(t, gpBody("init_gameplay"), "hud_dli")
	gpHas(t, gpBody("init_gameplay"), "NMIEN=a=0xC0")
	gpHas(t, gpBody("game_frame"), "draw_hud")
	gpHas(t, er, "draw_hud_static")
	gpHas(t, gpBody("clear_term_screen"), "y?232") // 25 rows = 1000 bytes

	gpHas(t, gpBody("fire_prompt"), "MsgFireHud")
	dh := gpBody("draw_hud")
	ensure(t, strings.Count(dh, "fire_prompt") == 2, "draw_hud fire_prompt count") // switch + relay prompts
	gpHas(t, dh, "a?6")
	gpHas(t, dh, "a?33")
}

func TestGameplayAudioAndEnding(t *testing.T) {
	main := src()
	music := readAsset("music.bin")
	ensure(t, len(music) == 435, "music size %d", len(music))
	gpHasNot(t, main, `binary "assets/music.bin"`)
	gpHas(t, gpBody("init_gameplay"), "init_audio")
	gpHas(t, gpBody("init_audio"), "AUDCTL=a=0x50")
	notes := map[int]bool{}
	for _, v := range music[:256] {
		if v != 0 && v != 1 {
			notes[int(v)] = true
		}
	}
	for k := range notes { // every lead note in tune within 2 cents
		f := 1789790 / (2 * float64(int(music[336+k-2])+256*int(music[352+k-2])+7))
		ensure(t, math.Abs(gpPyMod(1200*math.Log2(f/440), 100)-50) > 48, "note %d out of tune (f=%v)", k, f)
	}
	for _, v := range music[256:272] {
		ensure(t, v <= 2, "music[256:272] value %d", v)
	}
	for _, v := range music[272:304] {
		ensure(t, v <= 3, "music[272:304] value %d", v)
	}
	gpHasNot(t, gpBody("music_step"), "a&127")
	gpHas(t, gpBody("tick_music"), "Music+400,x") // 256-step song
	gpHas(t, gpBody("tick_music"), "sfx_timer")
	gpHas(t, gpBody("sfx_alert"), "AUDC4") // drums yield to SFX
	gpHas(t, gpBody("music_vbi_imm"), "tick_audio")
	gpHasNot(t, gpBody("game_frame"), "tick_audio") // steady tempo
	gpHas(t, gpBody("init_audio"), "VVBLKI")
	audio := gpStripComments(gpBody("tick_audio"))
	gpHas(t, audio, "call tick_music")
	gpHas(t, audio, "music_ready")
	gpHas(t, audio, "sfx_timer--")
	gpHas(t, gpBody("level_ending"), "AUDC3=a=0")

	font := readAsset("term-font.pic")
	ensure(t, font[73*8+2] == 0xAE && font[75*8+4] == 0xBA, "LED bytes") // LED bytes blink_leds toggles
	gpHas(t, gpBody("blink_leds"), "TermFont+586")
	gpHas(t, gpBody("blink_leds"), "TermFont+604")
	gpHas(t, gpBody("game_frame"), "blink_leds")

	// level_ending shows the per-level message from Txt/TxtOff (a3e.5 prefetch placeholder is gone)
	gpHas(t, gpBody("level_ending"), "TxtOff,x")
	gpHasNot(t, main, "prefetch_next_level")

	// level 01 hop: UP jumps, plat_phys lands the cat back on the single flat floor
	hop := gpBody("l01_hop")
	gpHas(t, hop, "vy=a=0xFC")
	gpHas(t, hop, "air=a=1")
	gpHas(t, hop, "plat_phys")
	gpHas(t, hop, "alert_timer") // no hopping while caught or finishing
	gpHas(t, hop, "level_finished")
	gpHas(t, gpBody("frame_any"), "l01_hop")
	under := gpBody("under")
	gpHas(t, under, "a?184")
	ensure(t, mustIndex(t, under, "a?184") < mustIndex(t, under, "on_vacuum"), "under order") // level 01 never reads L02 metadata
}
