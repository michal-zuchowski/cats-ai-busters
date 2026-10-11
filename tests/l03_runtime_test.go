package tests

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/sim"
	"github.com/michal-zuchowski/cats-ai-busters/tools/solver"
)

// R4 joystick route on the user's XEX. Setting L03_ASSET_OVERLAY=1 (the Python test's
// --asset-overlay flag) tests new room data on that engine; the default is off.
//
// No compilation, executable patching or file writes. The overlay changes only the in-memory
// ATR room sector bytes and corresponding chunk checksum; it does not test uncompiled source changes.

var l03rtRespawns int

var l03rtCaptions = []string{"SWAT THE VACUUM THROUGH THE READER. ", "RIDE RIGHT. JUMP TO THE BALCONY.    ",
	"JUMP THE GAP TO THE SERVICE EXIT.   ", "SERVICE ACCESS OPEN. GO RIGHT.      "}

// l03rtInvoke calls an existing compiled routine, preserving the suspended main-loop call stack.
func l03rtInvoke(t *testing.T, s *sim.Sim, name string) {
	t.Helper()
	m := s.Mpu
	pc, sp := m.PC, m.SP
	const ret = 0x07FF
	s.Mem[0x100+int(sp)] = ret >> 8
	s.Mem[0x100+int(sp-1)] = ret & 255
	m.SP, m.PC = sp-2, uint16(s.Sym[name])
	respawn := s.Sym["respawn"]
	for i := 0; i < 2_000_000; i++ {
		if int(m.PC) == respawn {
			l03rtRespawns++
		}
		if m.PC == 0x0800 {
			m.PC, m.SP = pc, sp
			return
		}
		if m.PC == sim.SIOV {
			s.SiovFn()
		} else {
			m.Step()
		}
		if m.Cycles >= s.NextTick {
			s.NextTick += sim.FrameCycles
			s.Tick()
		}
	}
	t.Fatalf("Compiled routine did not return: %s", name)
}

func l03rtFrame(t *testing.T, s *sim.Sim, joy int, fire bool) {
	t.Helper()
	s.Mem[0xD300] = 0xFF ^ byte(joy)
	s.Mem[0xD010] = 1
	if fire {
		s.Mem[0xD010] = 0
	}
	s.Mem[0x14]++
	s.NextTick = s.Mpu.Cycles + sim.FrameCycles
	l03rtInvoke(t, s, "plat_frame")
	if s.Get("cur_room") == 3 && s.Has("RoomArtBuf") {
		base := s.Sym["RoomArtBuf"] + 22*40
		col := s.Get("act_col")
		if !bytes.Equal(s.Mem[base+col:base+col+3], []byte{70, 70, 70}) {
			t.Fatalf("R4 robot footprint is not grounded: col %d", col)
		}
	}
}

func l03rtEntry(t *testing.T, s *sim.Sim) {
	s.Set("cur_room", 3)
	s.Set("entry_col", 0)
	s.Set("level_finished", 0)
	s.Mem[0xD300], s.Mem[0xD010] = 0xFF, 1
	l03rtInvoke(t, s, "enter_plat_room")
}

// l03rtReader returns the sorted distinct tile values of the reader.
func l03rtReader(s *sim.Sim) []int {
	var set []int
	for _, b := range s.Screen(20, 12, 3) {
		if !slices.Contains(set, int(b)) {
			set = append(set, int(b))
		}
	}
	slices.Sort(set)
	return set
}

func l03rtDoor(s *sim.Sim) [3]int {
	var d [3]int
	for i, r := range []int{19, 20, 21} {
		d[i] = int(s.Screen(r, 39, 1)[0])
	}
	return d
}

type l03rtProbe struct {
	got    bool
	reader []int
	door   [3]int
}

// l03rtRoute uses joystick/FIRE only after room entry. It reads game state like a player reads
// the screen: approach the front robot, swing again after a miss, jump to its roof when it is
// near, ride, leave at the balcony and cross the second gap. Returns frames, swings, respawns.
func l03rtRoute(t *testing.T, s *sim.Sim, stand int, board [2]int, rear, delay, edge int, probe *l03rtProbe, withProbe bool) (int, int, int) {
	t.Helper()
	swings, cooldown := 0, 0
	boarded := false
	seen := map[int]bool{}
	if withProbe {
		if r := l03rtReader(s); !slices.Equal(r, []int{83}) || l03rtDoor(s) != [3]int{81, 81, 81} {
			t.Fatalf("initial reader/door: %v %v", r, l03rtDoor(s))
		}
	}
	for i := 0; i < 1500; i++ {
		joy, fire := 0, false
		if i >= delay {
			col, robot, x := s.Get("player_col"), s.Get("act_col"), s.Get("player_x")
			air := s.Get("air") != 0
			switch {
			case s.Get("gate") == 0:
				switch {
				case cooldown != 0:
					cooldown--
				case s.Get("paw_hit") != 0 && s.Get("act_wob") != 0:
					// struck: wait while it is carried; a hit at col 7 falls one column short
				case x >= stand && s.Get("paw_t") == 0:
					swings++
					fire, cooldown = true, 11+5*(swings%3) // unsynchronised retries
				case x < stand:
					joy = 8
				}
			case col >= 33 && !air:
				joy = 8
			case s.Get("player_y") == 152 && !air:
				joy = 8
				if col >= 27 {
					joy = 9
				}
			case air:
				joy = 8
			case s.Get("ride") != 0:
				boarded = true
				switch {
				case robot >= 19:
					joy = 9
				case col-robot < rear:
					joy = 8
				case col-robot > rear:
					joy = 4
				}
			case s.Get("player_y") == 184:
				d := robot - col
				if s.Get("act_dir") == 1 && board[0]-l03rtRespawns <= d && d <= board[1]-l03rtRespawns {
					joy = 9
				} else if col < edge {
					joy = 8 // walk to the pit edge, then wait for the robot to pass by
				}
			}
		}
		l03rtFrame(t, s, joy, fire)
		if withProbe && s.Get("gate") != 0 && !probe.got {
			probe.got, probe.reader, probe.door = true, l03rtReader(s), l03rtDoor(s)
		}
		if s.Get("air") == 0 {
			seen[s.Get("player_y")] = true
		}
		if s.Get("level_finished") != 0 {
			if swings == 0 || !boarded || s.Get("gate") == 0 || !seen[171] || !seen[152] || !seen[184] {
				t.Fatalf("route incomplete: swings=%d boarded=%v seen=%v", swings, boarded, seen)
			}
			if s.Get("player_y") != 184 || s.Get("player_col") < 33 {
				t.Fatalf("bad finish: y=%d col=%d", s.Get("player_y"), s.Get("player_col"))
			}
			return i + 1, swings, l03rtRespawns
		}
	}
	state := map[string]int{}
	for _, k := range []string{"player_x", "player_y", "ride", "carrier", "act_col", "gate", "air"} {
		state[k] = s.Get(k)
	}
	t.Fatalf("Compiled R4 route stalled: %v", state)
	return 0, 0, 0
}

func TestL03RuntimeR4Route(t *testing.T) {
	requireBuilt(t)
	overlay := os.Getenv("L03_ASSET_OVERLAY") == "1"
	s := l04NewSim(t)
	if !overlay && !s.Has("R4Goal") {
		t.Fatal("Build predates R4 changes. Build matching XEX/ATR, or set L03_ASSET_OVERLAY=1 for room-data QA only.")
	}
	if err := solver.Boot(s, 600); err != nil {
		t.Fatal(err)
	}
	s.Mem[s.Sym["CH"]] = 0x1A // use the real debug entry to L03, not a fabricated physics state
	s.Run(3, 0, false)
	if s.Get("cur_level") != 2 {
		t.Fatalf("cur_level = %d, want 2", s.Get("cur_level"))
	}
	if overlay {
		table := readAsset("disk-chunks.bin")
		meta := table[60:65] // chunk 12: L03 R4
		sector, count := int(meta[0])|int(meta[1])<<8, int(meta[2])|int(meta[3])<<8
		if count != 8 {
			t.Fatalf("chunk 12 sector count = %d, want 8", count)
		}
		disk := append([]byte(nil), s.Atr...)
		copy(disk[(sector-1)*128:(sector-1+count)*128], readAsset("plat-rooms.bin")[7168:8192])
		s.Atr = disk
		copy(s.Mem[s.Sym["ChunkTable"]+60:], meta)
	}
	l03rtEntry(t, s)
	if s.Get("m_exit") != 184 || s.Get("m_xmax") != 19 || s.Get("e_bspd") != 0 {
		t.Fatalf("entry state: m_exit=%d m_xmax=%d e_bspd=%d", s.Get("m_exit"), s.Get("m_xmax"), s.Get("e_bspd"))
	}
	start := s.Snapshot()
	l03rtFrame(t, s, 1, false)
	for i := 0; i < 60; i++ {
		l03rtFrame(t, s, 0, false)
	}
	if s.Get("player_y") != 184 || s.Get("ride") != 0 || s.Get("gate") != 0 {
		t.Fatalf("idle R4: y=%d ride=%d gate=%d", s.Get("player_y"), s.Get("ride"), s.Get("gate"))
	}
	// (stand x, boarding window, rear offset, input delay, pit-edge column): different approach spots, early/late
	// roof jumps, riding positions and start timings; failed swings/boardings simply retry.
	variants := []struct {
		stand       int
		board       [2]int
		rear, delay int
		edge        int
	}{{17, [2]int{5, 7}, 0, 0, 11}, {19, [2]int{5, 6}, 1, 7, 11}, {18, [2]int{6, 7}, 2, 19, 10},
		{21, [2]int{5, 5}, 0, 11, 12}, {17, [2]int{7, 7}, 1, 31, 12}, {20, [2]int{6, 6}, 2, 3, 11},
		{19, [2]int{8, 8}, 0, 5, 11}}
	for _, v := range variants {
		s.Restore(start)
		l03rtRespawns = 0
		var probe l03rtProbe
		n, swings, deaths := l03rtRoute(t, s, v.stand, v.board, v.rear, v.delay, v.edge, &probe, !overlay)
		if !overlay { // compiled reader update: RELAY 83 until the genuine gate, then DONE 84, door open
			if !slices.Equal(probe.reader, []int{84}) || probe.door == [3]int{81, 81, 81} {
				t.Errorf("gate probe: reader=%v door=%v", probe.reader, probe.door)
			}
		}
		want := 0
		if v.board == [2]int{8, 8} {
			want = 1 // the late jump misses once, then retries
		}
		if deaths != want {
			t.Errorf("variant %+v: respawns = %d, want %d", v, deaths, want)
		}
		t.Log(fmt.Sprintf("R4 compiled-engine route: stand=%d board=%v rear=%d delay=%d edge=%d: %d frames, %d swings, %d respawns, complete",
			v.stand, v.board, v.rear, v.delay, v.edge, n, swings, deaths))
	}
	if overlay {
		t.Log("Asset overlay only: new room data ran in memory, not from a freshly built ATR.")
		return
	}
	s.Restore(start)
	h := s.Sym["Hints"] // encoding check against old compiled text: ASCII - 32
	if want := []byte{'J' - 32, 'U' - 32, 'M' - 32, 'P' - 32}; !bytes.Equal(s.Mem[h+7:h+11], want) {
		t.Errorf("Hints encoding: % x", s.Mem[h+7:h+11])
	}
	for i, text := range l03rtCaptions {
		s.Set("gate", 1)
		s.Set("player_col", []int{0, 10, 23, 33}[i])
		if i == 0 {
			s.Set("gate", 0)
		}
		s.Set("ride", 0)
		l03rtInvoke(t, s, "plat_hud")
		want := make([]byte, len(text))
		for j := range text {
			want[j] = text[j] - 32
		}
		if got := s.Screen(0, 0, 36); !bytes.Equal(got, want) {
			t.Errorf("caption %d: got % x, want % x", i, got, want)
		}
	}
	s.Restore(start) // the caption loop left gate=1; reset to the start fixture first
	s.Set("gate", 0)
	s.Set("e_bspd", 2)
	s.Set("bcol", 0)
	s.Set("player_x", 0)
	s.Set("player_y", 171)
	l03rtInvoke(t, s, "set_col")
	l03rtInvoke(t, s, "on_vacuum")
	if s.Mpu.A != 0 { // compiled root guard: dormant pursuer cannot be a carrier before the gate
		t.Errorf("on_vacuum before gate: A=%d", s.Mpu.A)
	}
	s.Set("gate", 1)
	l03rtInvoke(t, s, "on_vacuum")
	if s.Mpu.A != 21 || s.Get("carrier") != 21 { // gate open: B may carry
		t.Errorf("on_vacuum after gate: A=%d carrier=%d", s.Mpu.A, s.Get("carrier"))
	}
	s.Set("gate", 1)
	l03rtInvoke(t, s, "act_init")
	if s.Get("gate") != 0 || !slices.Equal(l03rtReader(s), []int{83}) || l03rtDoor(s) != [3]int{81, 81, 81} {
		t.Errorf("act_init reset: gate=%d reader=%v door=%v", s.Get("gate"), l03rtReader(s), l03rtDoor(s))
	}
}
