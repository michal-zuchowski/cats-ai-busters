// Package platmodel is a frame-faithful Go model of the platform-level runtime for Level 02
// (port of tools/plat_model.py).
//
// It mirrors src/plat_*.k65 plat_input / plat_phys / plat_fan / plat_lift / plat_crumble /
// plat_switch / plat_exit (same frame order as plat_frame) so room routes can be replayed and
// searched without a compiler. It is a source-level model: it does not run the XEX.
package platmodel

import (
	"container/heap"
	"errors"

	L "github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
	P "github.com/michal-zuchowski/cats-ai-busters/tools/platlevels"
)

// Key bits for Step.
const (
	Jump  = 1
	Left  = 4
	Right = 8
	Fire  = 16
)

const (
	CrWarn   = 40 // crumble: erase at 40 ...
	CrGone   = 120
	CrSlots  = 8
	GustBit  = 64 // fan_t & 64: gust half of a 128-frame cycle
	NoT0     = -1 // "no fan phase" for plan macros (python None)
	rowCount = 24
	colCount = 40
)

// Dirs are the steering keys tried by the route search.
var Dirs = [3]int{0, Left, Right}

// Screen is the 24x40 tile screen.
type Screen [rowCount][colCount]int

func isSupport(t int) bool {
	switch t & 0x7F {
	case L.Floor, L.Lift, L.Crumb, L.Ctrl:
		return true
	}
	return false
}

// IsSupport reports whether a tile (inverse bit ignored) is solid ground.
func IsSupport(t int) bool { return isSupport(t) }

// Unpack splits a room chunk into its screen and metadata fields.
func Unpack(chunk []byte) (Screen, map[string]int) {
	var scr Screen
	for r := range scr {
		for c := range scr[r] {
			scr[r][c] = int(chunk[r*colCount+c])
		}
	}
	meta := map[string]int{}
	for i, n := range P.Meta {
		meta[n] = int(chunk[960+i])
	}
	return scr, meta
}

// row maps a (possibly negative) row index like python list indexing.
func row(r int) int {
	if r < 0 {
		return r + rowCount
	}
	return r
}

// Sim is the dynamic model. Fields mirror the python attributes.
type Sim struct {
	Art, Scr Screen
	M        map[string]int

	X, Y, Air, Vy, Gt, Jprep, Lnd, T int
	FirePrev                         int
	LiftRow, LiftDir, LiftT          int
	SwOn, FanT                       int
	Cr                               [CrSlots][3]int // t, row, col
	Done, Exited                     bool
	Deaths                           int

	// OverrideUnder / OverrideSwitch replace under()/switch() (python subclass overrides);
	// they may call BaseUnder / BaseSwitch.
	OverrideUnder  func(s *Sim) bool
	OverrideSwitch func(s *Sim)
	// OverrideStep replaces the whole frame (python subclass step()); Drive/DoJump/HoldUntil
	// then run the subclass frame like the python methods do.
	OverrideStep func(s *Sim, keys int)
}

// New builds the model for a room chunk; the cat enters at x = entryX.
func New(chunk []byte, entryX int) *Sim {
	s := &Sim{}
	s.Art, s.M = Unpack(chunk)
	s.Scr = s.Art
	s.X, s.Y = entryX, s.M["sy"]
	s.LiftRow, s.LiftDir = s.M["lrow"], -1
	s.FirePrev = 1
	if s.M["lw"] != 0 {
		s.DrawLift(s.LiftTile())
	}
	return s
}

// Clone returns an independent copy (python copy.deepcopy).
func (s *Sim) Clone() *Sim { c := *s; return &c }

// Col is the tile column under the cat.
func (s *Sim) Col() int { return ((s.X + 2) >> 2) + 2 }

func (s *Sim) LiftTile() int {
	if s.M["lpow"] != 0 && s.SwOn == 0 {
		return L.Lift | 0x80
	}
	return L.Lift
}

func (s *Sim) DrawLift(tile int) {
	m := s.M
	for c := m["lcol"]; c < m["lcol"]+m["lw"]; c++ {
		if tile != 0 {
			s.Scr[s.LiftRow][c] = tile
		} else {
			s.Scr[s.LiftRow][c] = s.Art[s.LiftRow][c]
		}
	}
}

func (s *Sim) TileUnder() int { return s.Scr[row((s.Y>>3)-1)][s.Col()] }

// BaseUnder is the unmodified under().
func (s *Sim) BaseUnder() bool { return s.Y&7 == 0 && isSupport(s.TileUnder()) }

func (s *Sim) Under() bool {
	if s.OverrideUnder != nil {
		return s.OverrideUnder(s)
	}
	return s.BaseUnder()
}

func (s *Sim) Respawn() {
	s.X, s.Y = s.M["sx"], s.M["sy"]
	s.Air, s.Vy, s.Jprep, s.Lnd = 0, 0, 0, 0
	s.Deaths++
	s.CrumbleReset()
}

func (s *Sim) CrumbleReset() {
	for i := range s.Cr {
		c := &s.Cr[i]
		if c[0] != 0 {
			s.Scr[c[1]][c[2]] = s.Art[c[1]][c[2]]
		}
		c[0] = 0
	}
}

// Step runs one frame.
func (s *Sim) Step(keys int) {
	if s.OverrideStep != nil {
		s.OverrideStep(s, keys)
		return
	}
	s.BaseStep(keys)
}

// BaseStep is the unmodified frame.
func (s *Sim) BaseStep(keys int) {
	s.Input(keys)
	s.Phys()
	s.Fan()
	s.Lift()
	s.Crumble()
	if s.Lnd != 0 {
		s.Lnd--
	}
	s.T++
}

func (s *Sim) Input(keys int) {
	if s.Jprep != 0 {
		s.Jprep--
		if s.Jprep != 0 {
			return
		}
		s.Vy, s.Air, s.Gt = -4, 1, 0
	} else if keys&Jump != 0 && s.Air == 0 && s.Lnd == 0 {
		s.Jprep = 3
		return
	}
	// plat_fire (edge triggered, grounded)
	if keys&Fire == 0 {
		s.FirePrev = 1
	} else if s.FirePrev != 0 {
		s.FirePrev = 0
		if s.Air == 0 {
			s.Switch()
		}
	}
	if s.Lnd != 0 {
		return
	}
	if s.Air == 0 && s.T&1 != 0 {
		return
	}
	if keys&Left != 0 {
		if s.X != 0 {
			s.X--
		}
		return
	}
	if keys&Right != 0 {
		if s.X == 140 {
			s.TryExit()
			return
		}
		s.X++
	}
}

// BaseSwitch is the unmodified plat_switch.
func (s *Sim) BaseSwitch() {
	m := s.M
	if m["swy"] != 0 && s.Y == m["swy"] && s.SwOn == 0 && abs(s.Col()-m["swc"]) <= 2 {
		s.SwOn = 1
		s.Scr[m["swy"]/8-1][m["swc"]] = L.Ctrl
		r := m["exit_y"]/8 - 1
		for rr := r - 3; rr < r; rr++ {
			s.Scr[row(rr)][39] = L.Hatch
		}
		if m["lpow"] != 0 {
			s.DrawLift(s.LiftTile())
		}
	}
}

func (s *Sim) Switch() {
	if s.OverrideSwitch != nil {
		s.OverrideSwitch(s)
		return
	}
	s.BaseSwitch()
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (s *Sim) ExitAllowed() bool {
	m := s.M
	if m["exit_y"] == 0 {
		return true
	}
	if m["swy"] != 0 && s.SwOn == 0 {
		return false
	}
	return s.Air == 0 && s.Y == m["exit_y"]
}

func (s *Sim) TryExit() {
	if s.ExitAllowed() {
		s.Exited = true
	}
}

func (s *Sim) Phys() {
	if s.Air == 0 {
		if s.Under() {
			return
		}
		s.Air, s.Vy, s.Jprep, s.Lnd = 1, 0, 0, 0
	}
	s.Gt++
	if s.Gt == 3 {
		s.Gt = 0
		if s.Vy != 5 {
			s.Vy++
		}
		if s.Vy == 0 {
			s.Gt = 2
		}
	}
	if s.Vy < 0 {
		s.Y += s.Vy
	} else if s.Vy != 0 {
		for i := 0; i < s.Vy; i++ {
			s.Y++
			if s.Under() {
				s.Air, s.Vy = 0, 0
				s.Lnd = 10
				break
			}
		}
	}
	if s.Y >= 196 {
		s.Respawn()
	}
}

// FanDraw draws blades and airflow streaks (fan_draw in src/plat_fan_lift.k65).
func (s *Sim) FanDraw() {
	m := s.M
	gust := s.FanT&GustBit != 0
	pose := (s.FanT >> 4) & 1
	if gust {
		pose = (s.FanT >> 2) & 1
	}
	blades := append(append([]int{}, L.FanA[:]...), L.FanB[:]...)
	for dr := 0; dr < 2; dr++ {
		for dc := 0; dc < 2; dc++ {
			s.Scr[m["fr0"]+2+dr][m["fan"]-3+dc] = blades[pose*4+dr*2+dc]
		}
	}
	for r := m["fr0"]; r < m["fr1"]; r++ {
		for c := m["fan"]; c < m["fc1"]; c++ {
			if gust {
				s.Scr[r][c] = L.AirA + (((s.FanT >> 2) + r) & 1)
			} else {
				s.Scr[r][c] = s.Art[r][c]
			}
		}
	}
}

func (s *Sim) Fan() {
	s.FanT = (s.FanT + 1) & 255
	m := s.M
	if m["fan"] != 0 && s.FanT&3 == 0 {
		s.FanDraw()
	}
	if m["fan"] == 0 || s.Air == 0 || s.FanT&GustBit == 0 {
		return
	}
	if !(m["fan"] <= s.Col() && s.Col() < m["fc1"]) {
		return
	}
	cb := (s.Y >> 3) - 2
	if !(m["fr0"] <= cb && cb < m["fr1"]) || s.X >= 140 {
		return
	}
	s.X++
}

func (s *Sim) Lift() {
	m := s.M
	if m["lw"] == 0 || (m["lpow"] != 0 && s.SwOn == 0) {
		return
	}
	s.LiftT++
	if s.LiftT < m["lspd"] {
		return
	}
	s.LiftT = 0
	rel := s.Col() - m["lcol"]
	ride := s.Air == 0 && s.Y == (s.LiftRow+1)*8 && rel >= 0 && rel < m["lw"]
	s.DrawLift(0)
	kk := s.LiftDir
	s.LiftRow += kk
	if s.LiftRow == m["lmin"] {
		s.LiftDir = 1
	}
	if s.LiftRow == m["lmax"] {
		s.LiftDir = -1
	}
	s.DrawLift(s.LiftTile())
	if ride {
		if kk == -1 {
			s.Y -= 8
		} else {
			s.Y += 8
		}
	}
}

func (s *Sim) Crumble() {
	for i := range s.Cr {
		c := &s.Cr[i]
		if c[0] != 0 {
			c[0]++
			if c[0] == CrWarn {
				s.Scr[c[1]][c[2]] = L.Void
			} else if c[0] == CrGone {
				s.Scr[c[1]][c[2]] = s.Art[c[1]][c[2]]
				c[0] = 0
			}
		}
	}
	if s.Air != 0 || s.Y&7 != 0 || s.TileUnder() != L.Crumb {
		return
	}
	for i := range s.Cr {
		c := &s.Cr[i]
		if c[0] == 0 {
			*c = [3]int{1, (s.Y >> 3) - 1, s.Col()}
			s.Scr[c[1]][c[2]] = L.Crumb | 0x80
			return
		}
	}
}

// Key is the state key for searches.
type Key struct {
	X, Y, Air, Vy, Gt, Jprep, Lnd, T1, LiftRow, LiftDir, LiftT, SwOn, FanT, FirePrev int
	Cr                                                                               [CrSlots][3]int
}

func (s *Sim) Key() Key {
	return Key{s.X, s.Y, s.Air, s.Vy, s.Gt, s.Jprep, s.Lnd, s.T & 1, s.LiftRow, s.LiftDir,
		s.LiftT, s.SwOn, s.FanT, s.FirePrev, s.Cr}
}

// Script is a (keys, frames) step of Run.
type Script struct{ Keys, Frames int }

// Run plays a script; it stops as soon as the cat exits.
func Run(s *Sim, script []Script) *Sim {
	for _, st := range script {
		for i := 0; i < st.Frames; i++ {
			s.Step(st.Keys)
			if s.Exited {
				return s
			}
		}
	}
	return s
}

// HoldUntil steps with keys until cond holds (up to limit frames, default 600 when limit <= 0).
func HoldUntil(s *Sim, keys int, cond func(*Sim) bool, limit int) bool {
	if limit <= 0 {
		limit = 600
	}
	for i := 0; i < limit; i++ {
		if cond(s) {
			return true
		}
		s.Step(keys)
		if s.Exited {
			return cond(s)
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Static route planner: stand/jump/fall graph over the art (no lift, crumble
// timing or switch).
// ---------------------------------------------------------------------------

// Static is the static stand/jump/fall map.
type Static struct {
	Art Screen
	M   map[string]int
	sup [rowCount][colCount]bool
}

func NewStatic(chunk []byte) *Static {
	st := &Static{}
	st.Art, st.M = Unpack(chunk)
	for r := range st.Art {
		for c := range st.Art[r] {
			st.sup[r][c] = isSupport(st.Art[r][c])
		}
	}
	return st
}

func (st *Static) Stand(x, y int) bool {
	return y&7 == 0 && (y>>3)-1 < rowCount && st.sup[row((y>>3)-1)][((x+2)>>2)+2]
}

// FlightResult is the outcome of Flight.
type FlightResult struct {
	Land   bool // false: dead
	X, Y   int
	Frames int
}

// Flight simulates a jump (takeoff) or a fall; sched(f) is the key for frame f. t0 is the fan
// phase at the start (NoT0: no fan).
func (st *Static) Flight(x, y int, sched func(f int) int, takeoff bool, t0 int) FlightResult {
	m := st.M
	vy, gt := 0, 0
	if takeoff {
		vy = -4
	}
	f := 0
	for {
		key := sched(f)
		if takeoff || f != 0 {
			if key&Left != 0 {
				if x-1 > 0 {
					x--
				} else {
					x = 0
				}
			} else if key&Right != 0 && x != 140 {
				x++
			}
		}
		gt++
		if gt == 3 {
			gt = 0
			if vy != 5 {
				vy++
			}
			if vy == 0 {
				gt = 2
			}
		}
		if vy < 0 {
			y += vy
		} else {
			for i := 0; i < vy; i++ {
				y++
				if st.Stand(x, y) {
					return FlightResult{true, x, y, f + 1}
				}
			}
		}
		if y >= 196 {
			return FlightResult{false, x, y, f + 1}
		}
		if t0 != NoT0 && m["fan"] != 0 {
			ft := (t0 + f + 1) & 255
			col := ((x + 2) >> 2) + 2
			cb := (y >> 3) - 2
			if ft&GustBit != 0 && m["fan"] <= col && col < m["fc1"] && m["fr0"] <= cb && cb < m["fr1"] && x < 140 {
				x++
			}
		}
		f++
	}
}

// Schedule is a steering schedule: d1 for f < k, then d2.
type Schedule struct{ D1, K, D2 int }

func (sc Schedule) key(f int) int {
	if f < sc.K {
		return sc.D1
	}
	return sc.D2
}

// Schedules enumerates (d1, k, d2) in the python generator order.
func Schedules(ks []int) []Schedule {
	var out []Schedule
	for _, d1 := range Dirs {
		for _, k := range ks {
			for _, d2 := range Dirs {
				out = append(out, Schedule{d1, k, d2})
			}
		}
	}
	return out
}

// Range is python range(start, stop, step) for positive step.
func Range(start, stop, step int) []int {
	var out []int
	for v := start; v < stop; v += step {
		out = append(out, v)
	}
	return out
}

// Node is a standing position.
type Node [2]int

// Macro is one planned move: Kind "walk" (X), "jump" (D1, K, D2, T0) or "fall" (Dx, D1, K, D2, T0).
type Macro struct {
	Kind          string
	X, Dx         int
	D1, K, D2, T0 int
}

// Parent is a planner back-pointer.
type Parent struct {
	Prev  Node
	Macro Macro
	Cost  int
}

// Parents is the planner result; a missing key is unreachable, a nil value is the start.
type Parents map[Node]*Parent

// PlanOpts are the optional planner parameters (nil / zero values give the python defaults:
// ks=range(0,28,2), xstep=1, t0s=(None,), no forbidden nodes).
type PlanOpts struct {
	Ks     []int
	XStep  int
	T0s    []int
	Forbid map[Node]bool
}

type item struct {
	cost int
	node Node
}
type pq []item

func (h pq) Len() int { return len(h) }
func (h pq) Less(i, j int) bool {
	if h[i].cost != h[j].cost {
		return h[i].cost < h[j].cost
	}
	if h[i].node[0] != h[j].node[0] {
		return h[i].node[0] < h[j].node[0]
	}
	return h[i].node[1] < h[j].node[1]
}
func (h pq) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *pq) Push(x any)   { *h = append(*h, x.(item)) }
func (h *pq) Pop() any {
	o := *h
	n := len(o)
	it := o[n-1]
	*h = o[:n-1]
	return it
}

// Plan is a Dijkstra (cost = frames) over standing positions of the static map.
func Plan(chunk []byte, start Node, opts *PlanOpts) Parents {
	var o PlanOpts
	if opts != nil {
		o = *opts
	}
	if o.Ks == nil {
		o.Ks = Range(0, 28, 2)
	}
	if o.XStep == 0 {
		o.XStep = 1
	}
	if o.T0s == nil {
		o.T0s = []int{NoT0}
	}
	st := NewStatic(chunk)
	parents := Parents{start: nil}
	dist := map[Node]int{start: 0}
	h := &pq{{0, start}}
	done := map[Node]bool{}
	jumpScheds := Schedules(o.Ks)
	fallScheds := Schedules(Range(0, 20, 2))

	relax := func(node, prev Node, mac Macro, cost int) {
		if o.Forbid[node] {
			return
		}
		if d, ok := dist[node]; !ok || cost < d {
			dist[node] = cost
			parents[node] = &Parent{Prev: prev, Macro: mac, Cost: cost}
			heap.Push(h, item{cost, node})
		}
	}

	for h.Len() > 0 {
		it := heap.Pop(h).(item)
		c, x, y := it.cost, it.node[0], it.node[1]
		if done[it.node] {
			continue
		}
		done[it.node] = true
		cur := Node{x, y}
		for _, dx := range [2]int{-1, 1} {
			nx := x + dx
			if 0 <= nx && nx <= 140 && st.Stand(nx, y) {
				relax(Node{nx, y}, cur, Macro{Kind: "walk", X: nx}, c+2)
			}
		}
		if x%o.XStep == 0 && st.Stand(x, y) {
			for _, t0 := range o.T0s {
				for _, sc := range jumpScheds {
					res := st.Flight(x, y, sc.key, true, t0)
					if res.Land {
						relax(Node{res.X, res.Y}, cur, Macro{Kind: "jump", D1: sc.D1, K: sc.K, D2: sc.D2, T0: t0}, c+4+res.Frames+10)
					}
				}
			}
		}
		for _, d := range [2]struct{ dx, key int }{{-1, Left}, {1, Right}} {
			nx := x + d.dx
			if 0 <= nx && nx <= 140 && st.Stand(x, y) && !st.Stand(nx, y) && (nx+2)>>2 != (x+2)>>2 {
				for _, t0 := range o.T0s {
					for _, sc := range fallScheds {
						res := st.Flight(nx, y, sc.key, false, t0)
						if res.Land {
							relax(Node{res.X, res.Y}, cur, Macro{Kind: "fall", Dx: d.dx, D1: sc.D1, K: sc.K, D2: sc.D2, T0: t0}, c+2+res.Frames+10)
						}
					}
				}
			}
		}
	}
	return parents
}

// CostOf is the planned cost of reaching node (0 for the start).
func CostOf(p Parents, node Node) int {
	if q := p[node]; q != nil {
		return q.Cost
	}
	return 0
}

// Hop is one step of a planned path.
type Hop struct {
	Prev  Node
	Macro Macro
	Node  Node
}

// PathTo walks the parents back from node; node must be reachable.
func PathTo(p Parents, node Node) []Hop {
	var out []Hop
	for p[node] != nil {
		q := p[node]
		out = append(out, Hop{q.Prev, q.Macro, node})
		node = q.Prev
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

var errNoSettle = errors.New("no settle")

// Settle steps with no keys until the cat is on the ground and idle.
func Settle(s *Sim, n int) error {
	for s.Air != 0 || s.Lnd != 0 || s.Jprep != 0 {
		s.Step(0)
		n--
		if n < 0 {
			return errNoSettle
		}
	}
	return nil
}

func mustSettle(s *Sim) {
	if err := Settle(s, 40); err != nil {
		panic(err)
	}
}

// Drive replays a planned path in the dynamic model. It returns false when the cat leaves the
// plan (and panics, like the python AssertionError, when it cannot settle).
func Drive(s *Sim, path []Hop, limit int) bool {
	if limit <= 0 {
		limit = 2000
	}
	for _, h := range path {
		mustSettle(s)
		if (Node{s.X, s.Y}) != h.Prev {
			return false
		}
		mac := h.Macro
		switch mac.Kind {
		case "walk":
			key := Left
			if mac.X > s.X {
				key = Right
			}
			n := 0
			for s.X != mac.X {
				s.Step(key)
				n++
				if n > limit {
					return false
				}
			}
		case "jump":
			if mac.T0 != NoT0 {
				for (s.FanT+3)&255 != mac.T0 {
					s.Step(0)
				}
			}
			s.Step(Jump)
			s.Step(Jump)
			s.Step(Jump)
			sc := Schedule{mac.D1, mac.K, mac.D2}
			f := 0
			s.Step(Jump | sc.key(f))
			f++
			for s.Air != 0 {
				s.Step(sc.key(f))
				f++
			}
		default: // fall: step off the edge, then steer
			key := Left
			if mac.Dx > 0 {
				key = Right
			}
			if mac.T0 != NoT0 {
				for (s.FanT+1)&255 != mac.T0 {
					s.Step(0)
				}
			}
			for s.Air == 0 {
				s.Step(key)
			}
			sc := Schedule{mac.D1, mac.K, mac.D2}
			f := 0
			for s.Air != 0 {
				s.Step(sc.key(f))
				f++
			}
		}
		mustSettle(s)
		if (Node{s.X, s.Y}) != h.Node {
			return false
		}
	}
	return true
}

// DoJump presses JUMP, then steers d1 for k air frames and d2 afterwards until landing.
func DoJump(s *Sim, d1, k, d2 int) error {
	for i := 0; i < 3; i++ {
		s.Step(Jump)
	}
	sc := Schedule{d1, k, d2}
	f := 0
	s.Step(Jump | sc.key(f))
	f++
	for s.Air != 0 {
		s.Step(sc.key(f))
		f++
	}
	return Settle(s, 40)
}

// JumpPlan is a found (wait, d1, k, d2) jump.
type JumpPlan struct{ W, D1, K, D2 int }

// SearchJump finds a jump from a copy of s that reaches goal without a death. nil waits means
// range(0, 40), nil ks means range(0, 30, 2).
func SearchJump(s *Sim, goal func(*Sim) bool, waits, ks []int, key int) (JumpPlan, bool) {
	if waits == nil {
		waits = Range(0, 40, 1)
	}
	if ks == nil {
		ks = Range(0, 30, 2)
	}
	scheds := Schedules(ks)
	for _, w := range waits {
		base := s.Clone()
		for i := 0; i < w; i++ {
			base.Step(key)
		}
		for _, sc := range scheds {
			t := base.Clone()
			deaths := t.Deaths
			if err := DoJump(t, sc.D1, sc.K, sc.D2); err != nil {
				continue
			}
			if t.Deaths == deaths && goal(t) {
				return JumpPlan{w, sc.D1, sc.K, sc.D2}, true
			}
		}
	}
	return JumpPlan{}, false
}

// ApplyJump replays a SearchJump result on s.
func ApplyJump(s *Sim, p JumpPlan, key int) {
	for i := 0; i < p.W; i++ {
		s.Step(key)
	}
	if err := DoJump(s, p.D1, p.K, p.D2); err != nil {
		panic(err)
	}
}
