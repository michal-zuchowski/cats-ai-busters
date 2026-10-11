// Package solver is a breadth-first "bot" proving levels 02-04 are completable in the headless sim.
// It explores joystick macros from each room start and returns the winning macro list.
package solver

import (
	"errors"

	"github.com/michal-zuchowski/cats-ai-busters/tools/sim"
)

// Joystick bits and the FIRE pseudo-direction used by macros.
const (
	R = 8
	L = 4
	U = 1
	F = 0x100
)

type Macro struct {
	Name string
	Joy  int
	N    int
}

var Macros = []Macro{
	{"R4", R, 4}, {"L4", L, 4}, {"R16", R, 16}, {"L16", L, 16}, {"R40", R, 40}, {"L40", L, 40},
	{"jumpR", R | U, 1}, {"jumpL", L | U, 1}, {"jumpUp", U, 1},
	{"wait", 0, 40}, {"fire", F, 3},
}

// Machine is the part of the simulator the solver drives; tests substitute a fake.
type Machine interface {
	Get(name string) int
	Run(frames, joy int, fire bool)
}

// KeyMachine additionally exposes bytes at symbol+offset for Key.
type KeyMachine interface {
	Machine
	Byte(name string, off int) int
}

// Every byte that can change future behaviour; no bucketing. Page-6 crumble slots alias
// other levels' actors, so they are only meaningful in level 02 (cur_level == 1).
var KeyVars = []string{"cur_level", "cur_room", "player_x", "player_y", "air", "vy", "gt", "jprep", "lnd",
	"fire_prev", "facing", "walk_timer", "paw_t", "paw_kind", "paw_x", "paw_y", "paw_hit",
	"act_col", "act_dir", "act_t", "act_wob", "lift_row", "lift_dir", "lift_t", "dlg_done",
	"sw_on", "fan_t", "ride", "level_finished"}

var L03KeyVars = []string{"bcol", "bdir", "bt", "bwob", "carrier", "dk", "gate", "bw", "bk",
	"e_mode", "e_bspd"}

// Where returns cur_level, cur_room and level_finished.
func Where(s Machine) (level, room, finished int) {
	return s.Get("cur_level"), s.Get("cur_room"), s.Get("level_finished")
}

// RunMacro runs one macro; it returns true at the first room/level/level_finished transition (stops there).
func RunMacro(s Machine, name string, joy, n int) bool {
	_, startRoom, _ := Where(s)
	startLevel, _, _ := Where(s)

	frame := func(j int, fire bool) bool {
		s.Run(1, j, fire)
		lv, rm, fin := Where(s)
		return lv != startLevel || rm != startRoom || fin != 0
	}

	switch {
	case len(name) >= 4 && name[:4] == "jump": // anticipation, flight and landing must all finish
		h := joy &^ U
		if frame(joy, false) {
			return true
		}
		for i := 0; i < 200; i++ {
			if s.Get("air") == 0 && s.Get("jprep") == 0 && s.Get("lnd") == 0 {
				break
			}
			j := 0
			if s.Get("jprep") != 0 {
				j = joy
			} else if s.Get("air") != 0 {
				j = h
			}
			if frame(j, false) {
				return true
			}
		}
	case joy == F:
		for i := 0; i < n; i++ {
			if frame(0, true) {
				return true
			}
		}
		for i := 0; i < 2; i++ {
			if frame(0, false) {
				return true
			}
		}
	default:
		for i := 0; i < n; i++ {
			if frame(joy, false) {
				return true
			}
		}
	}
	return false
}

// Key identifies the machine state for the visited set; equal keys mean equal future behaviour.
func Key(s KeyMachine) string {
	k := make([]byte, 0, 64)
	for _, v := range KeyVars {
		k = append(k, byte(s.Get(v)))
	}
	switch s.Get("cur_level") {
	case 1:
		for _, v := range []string{"cr_t", "cr_r", "cr_c"} {
			for i := 0; i < 8; i++ {
				k = append(k, byte(s.Byte(v, i)))
			}
		}
	case 2:
		for _, v := range L03KeyVars {
			k = append(k, byte(s.Get(v)))
		}
	}
	// ground steps use RTCLOK&1; the L04 floor hazard uses RTCLOK&0x20 (64-frame period)
	mask := 1
	if s.Get("m_h0") != 0 {
		mask = 0x3F
	}
	return string(append(k, byte(s.Get("RTCLOK")&mask)))
}

// Boot skips the intro, releases FIRE on the title, then starts and waits for level 01 music.
func Boot(s Machine, limit int) error {
	for i := 0; i < limit; i++ {
		if s.Get("player_y") == 184 && s.Get("level_finished") == 0 && s.Get("title_phase") == 0 &&
			s.Get("music_ready") != 0 && s.Get("music_chunk") == 18 {
			return nil
		}
		s.Run(1, 0, s.Get("cur_room") == 0xFF || s.Get("title_phase") == 2)
	}
	return errors.New("Game did not reach level 01 after the title; check matching XEX/ATR and DISKERR.")
}

type node struct {
	snap *sim.Snapshot
	path []string
}

// Solve returns (solved, states tried, macro path per room, room where it got stuck).
func Solve(level, limit int) (bool, int, [][]string, int, error) {
	s, err := sim.New("", "")
	if err != nil {
		return false, 0, nil, 0, err
	}
	if err := Boot(s, 600); err != nil {
		return false, 0, nil, 0, err
	}
	for i := 0; i < level; i++ {
		s.Set("level_finished", 1)
		s.Run(20, 0, false)
	}
	s.Run(5, 0, false)
	target := s.Get("cur_level")
	total := 0
	var paths [][]string
	for { // one search per room: goal = next room / end of the selected level
		room := s.Get("cur_room")
		seen := map[string]bool{}
		q := []node{{s.Snapshot(), nil}}
		var found *node
		n := 0
		for len(q) > 0 && n < limit && found == nil {
			cur := q[0]
			q = q[1:]
			for _, m := range Macros {
				s.Restore(cur.snap)
				n++
				if RunMacro(s, m.Name, m.Joy, m.N) || s.Get("cur_level") != target {
					found = &node{s.Snapshot(), append(append([]string(nil), cur.path...), m.Name)}
					break
				}
				k := Key(s)
				if !seen[k] {
					seen[k] = true
					q = append(q, node{s.Snapshot(), append(append([]string(nil), cur.path...), m.Name)})
				}
			}
		}
		total += n
		if found == nil {
			return false, total, paths, room, nil
		}
		s.Restore(found.snap)
		paths = append(paths, found.path)
		if s.Get("level_finished") != 0 || s.Get("cur_level") != target {
			return true, total, paths, room, nil
		}
	}
}
