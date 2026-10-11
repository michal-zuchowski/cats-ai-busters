package tests

import (
	"strings"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/solver"
)

// tools/solver macro/key logic against a fake simulator (no compiler, no XEX).

var solverNames = func() []string {
	n := append(append([]string{}, solver.KeyVars...), solver.L03KeyVars...)
	return append(n, "cr_t", "cr_r", "cr_c", "RTCLOK", "m_h0")
}()

type bootFake struct {
	t      *testing.T
	state  map[string]int
	frames int
}

func newBootFake(t *testing.T, intro bool) *bootFake {
	b := &bootFake{t: t, state: map[string]int{"player_y": 0, "level_finished": 0, "title_phase": 1,
		"music_ready": 1, "music_chunk": 17, "cur_room": 0}}
	if intro {
		b.state["cur_room"], b.state["title_phase"], b.state["music_ready"] = 0xFF, 0, 0
	}
	return b
}

func (b *bootFake) Get(n string) int { return b.state[n] }

func (b *bootFake) Run(frames, joy int, fire bool) {
	b.t.Helper()
	if frames != 1 || joy != 0 {
		b.t.Fatalf("boot ran frames=%d joy=%d", frames, joy)
	}
	b.frames++
	st := b.state
	switch {
	case st["cur_room"] == 0xFF:
		if !fire {
			b.t.Fatal("intro needs FIRE")
		}
		st["cur_room"], st["title_phase"], st["music_ready"] = 0, 1, 1
	case st["title_phase"] == 1:
		if fire {
			b.t.Fatal("don't hold FIRE forever in the release loop")
		}
		st["title_phase"] = 2
	case st["title_phase"] == 2:
		if !fire {
			b.t.Fatal("title needs FIRE")
		}
		st["title_phase"], st["player_y"], st["music_ready"] = 0, 184, 0
	default:
		if fire {
			b.t.Fatal("release immediately, don't trigger a level-01 action")
		}
		st["music_ready"], st["music_chunk"] = 1, 18
	}
}

func TestSolverBoot(t *testing.T) {
	s := newBootFake(t, false)
	if err := solver.Boot(s, 600); err != nil {
		t.Fatal(err)
	}
	if s.frames != 3 || s.Get("music_ready") == 0 || s.Get("music_chunk") != 18 {
		t.Errorf("boot: frames=%d state=%v", s.frames, s.state)
	}
	s = newBootFake(t, true)
	if err := solver.Boot(s, 600); err != nil {
		t.Fatal(err)
	}
	if s.frames != 4 || s.Get("music_ready") == 0 || s.Get("music_chunk") != 18 {
		t.Errorf("intro boot: frames=%d state=%v", s.frames, s.state)
	}
	err := solver.Boot(newBootFake(t, false), 2)
	if err == nil {
		t.Fatal("boot silently accepted unfinished music loading")
	}
	if !strings.Contains(err.Error(), "XEX/ATR") {
		t.Errorf("boot error lacks XEX/ATR hint: %v", err)
	}
}

// solverFake: jump is 3 prep frames, 6 air frames, 4 landing frames. x >= 10 changes room (or level).
type solverFake struct {
	addr   map[string]int
	mem    [0x400]int
	frames int
}

func newSolverFake(kv ...any) *solverFake {
	f := &solverFake{addr: map[string]int{}}
	for i, n := range solverNames {
		f.addr[n] = 0x100 + i*8
	}
	for i := 0; i < len(kv); i += 2 {
		f.set(kv[i].(string), kv[i+1].(int))
	}
	return f
}

func (f *solverFake) Get(n string) int         { return f.mem[f.addr[n]] }
func (f *solverFake) set(n string, v int)      { f.mem[f.addr[n]] = v }
func (f *solverFake) Byte(n string, o int) int { return f.mem[f.addr[n]+o] }

func (f *solverFake) Run(frames, joy int, fire bool) {
	for ; frames > 0; frames-- {
		f.frames++
		switch {
		case f.Get("jprep") != 0:
			f.set("jprep", f.Get("jprep")-1)
			if f.Get("jprep") == 0 {
				f.set("air", 6)
				f.set("vy", 1)
			}
		case f.Get("air") != 0:
			f.set("air", f.Get("air")-1)
			if f.Get("air") == 0 {
				f.set("lnd", 4)
			}
		case f.Get("lnd") != 0:
			f.set("lnd", f.Get("lnd")-1)
		case joy&solver.U != 0:
			f.set("jprep", 3)
		}
		if joy&solver.R != 0 && f.Get("air") == 0 {
			f.set("player_x", f.Get("player_x")+1)
		}
		if f.Get("player_x") >= 10 && f.Get("cur_room") == 0 {
			f.set("cur_room", 1)
		}
		if f.Get("player_x") >= 20 && f.Get("cur_room") == 1 {
			f.set("level_finished", 1)
		}
		f.set("RTCLOK", (f.Get("RTCLOK")+1)&255)
	}
}

func TestSolverRunMacro(t *testing.T) {
	// anticipation: the jump macro must not stop at air==0 during jprep; it ends fully landed
	s := newSolverFake()
	if solver.RunMacro(s, "jumpUp", solver.U, 1) {
		t.Error("jumpUp reported a transition")
	}
	if s.frames != 1+3+6+4 { // press + prep + flight + landing, no early stop on air == 0
		t.Errorf("jumpUp frames=%d", s.frames)
	}
	if s.Get("lnd") != 0 || s.Get("air") != 0 || s.Get("jprep") != 0 {
		t.Error("jumpUp did not finish landed")
	}

	// stop at the first transition, not after the whole walk in the next room
	s = newSolverFake("player_x", 7)
	if !solver.RunMacro(s, "R40", solver.R, 40) {
		t.Error("R40 did not report the room change")
	}
	if s.Get("cur_room") != 1 || s.Get("player_x") != 10 || s.frames != 3 {
		t.Errorf("R40: room=%d x=%d frames=%d", s.Get("cur_room"), s.Get("player_x"), s.frames)
	}
	s = newSolverFake("player_x", 19, "cur_room", 1)
	if !solver.RunMacro(s, "R16", solver.R, 16) || s.Get("level_finished") == 0 || s.frames != 1 {
		t.Errorf("R16 level end: frames=%d", s.frames)
	}
	s = newSolverFake("player_x", 0)
	if solver.RunMacro(s, "wait", 0, 40) || s.frames != 40 {
		t.Errorf("wait: frames=%d", s.frames)
	}
}

func TestSolverKey(t *testing.T) {
	key := func(kv ...any) string { return solver.Key(newSolverFake(kv...)) }
	base := key()
	for _, n := range solver.KeyVars {
		if key(n, 1) == base {
			t.Errorf("key ignores %s", n)
		}
	}
	level2 := key("cur_level", 1)
	for _, n := range []string{"cr_t", "cr_r", "cr_c"} {
		for i := 0; i < 8; i++ {
			g := newSolverFake("cur_level", 1)
			g.mem[g.addr[n]+i] = 1
			if solver.Key(g) == level2 {
				t.Errorf("key ignores %s[%d] in level 02", n, i)
			}
		}
	}
	g := newSolverFake("cur_level", 2)
	h := newSolverFake("cur_level", 2)
	h.set("cr_t", 5) // aliased actor bytes outside L02 are not crumble state
	if solver.Key(g) != solver.Key(h) {
		t.Error("crumble bytes leaked into level 03 key")
	}
	level3 := key("cur_level", 2)
	for _, n := range solver.L03KeyVars {
		if key("cur_level", 2, n, 1) == level3 {
			t.Errorf("key ignores %s in level 03", n)
		}
		if key("cur_level", 1, n, 1) != key("cur_level", 1) {
			t.Errorf("not L03 state in L02: %s", n)
		}
	}
	ne := func(a, b string, msg string) {
		t.Helper()
		if a == b {
			t.Error(msg)
		}
	}
	ne(key("player_x", 3), key("player_x", 2), "x//2 bucketing")
	ne(key("fan_t", 64), key("fan_t", 65), "fan_t bucketed")
	ne(key("act_col", 3), key("act_col", 4), "act_col bucketed")
	ne(key("RTCLOK", 1), key("RTCLOK", 0), "RTCLOK parity ignored")
	if key("RTCLOK", 2) != key("RTCLOK", 0) { // parity only unless the hazard row exists
		t.Error("RTCLOK 2 vs 0 differ without hazard row")
	}
	ne(key("RTCLOK", 2, "m_h0", 5), key("RTCLOK", 0, "m_h0", 5), "hazard row RTCLOK&0x20 ignored")
	ne(key("cur_level", 3, "dlg_done", 0), key("cur_level", 3, "dlg_done", 1), "dlg_done ignored")
}
