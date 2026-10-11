// Package sim is a headless 6502 harness that runs out/cats-ai-busters.xex frame by frame.
//
// It fakes only what gameplay touches: PORTA/TRIG0 (joystick), RTCLOK (one tick
// per ~PAL frame of CPU cycles) and SIOV (reads sectors from the ATR). No
// VBI/music/DLI -- game logic and zero-page/screen state only.
package sim

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

const (
	FrameCycles = 29000
	SIOV        = 0xE459
)

var symHex = regexp.MustCompile(`^[0-9a-f]{1,4}$`)

// LoadSymbols reads a .sym file ("name hexaddr ..." per line, first line skipped).
// An empty path means out/cats-ai-busters.sym.
func LoadSymbols(path string) (map[string]int, error) {
	if path == "" {
		path = k65src.Path("out", "cats-ai-busters.sym")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	syms := map[string]int{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[min(1, len(lines)):] {
		parts := strings.Fields(line)
		if len(parts) >= 2 && symHex.MatchString(parts[1]) {
			v, _ := strconv.ParseInt(parts[1], 16, 32)
			syms[parts[0]] = int(v)
		}
	}
	return syms, nil
}

// Snapshot is a full machine state copy (see Sim.Snapshot / Sim.Restore).
type Snapshot struct {
	mem              [0x10000]byte
	pc               uint16
	a, x, y, sp, p   byte
	cycles           int64
	frames, nextTick int64
}

type Sim struct {
	Sym      map[string]int
	Mpu      *CPU
	Mem      *[0x10000]byte
	Atr      []byte // ATR image without its 16-byte header
	Frames   int64
	NextTick int64
	// SiovFn handles a JSR to SIOV; defaults to (*Sim).Siov. Tests override it to inject errors.
	SiovFn func()
}

// New boots a Sim from an XEX and ATR; "" selects the default out/cats-ai-busters.* file.
func New(xex, atr string) (*Sim, error) {
	if xex == "" {
		xex = k65src.Path("out", "cats-ai-busters.xex")
	}
	if atr == "" {
		atr = k65src.Path("out", "cats-ai-busters.atr")
	}
	sym, err := LoadSymbols("")
	if err != nil {
		return nil, err
	}
	s := &Sim{Sym: sym, Mpu: NewCPU()}
	s.Mem = &s.Mpu.Mem
	s.SiovFn = s.Siov
	data, err := os.ReadFile(xex)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(data); {
		if i+1 < len(data) && data[i] == 0xFF && data[i+1] == 0xFF {
			i += 2
		}
		if i+3 >= len(data) {
			return nil, fmt.Errorf("%s: truncated XEX segment header", xex)
		}
		start := int(data[i]) | int(data[i+1])<<8
		end := int(data[i+2]) | int(data[i+3])<<8
		n := end - start + 1
		if n < 0 || i+4+n > len(data) {
			return nil, fmt.Errorf("%s: bad XEX segment %04x-%04x", xex, start, end)
		}
		copy(s.Mem[start:], data[i+4:i+4+n])
		i += 4 + n
	}
	raw, err := os.ReadFile(atr)
	if err != nil {
		return nil, err
	}
	if len(raw) < 16 {
		return nil, fmt.Errorf("%s: too short for an ATR", atr)
	}
	s.Atr = raw[16:]
	s.Mpu.PC = uint16(s.Mem[0x2E0]) | uint16(s.Mem[0x2E1])<<8
	s.Mpu.SP = 0xFF
	s.Mem[0xD300] = 0xFF
	s.Mem[0xD010] = 1
	s.Mem[0xD20F] = 0xFF // SKSTAT: keyboard and Shift released
	s.Mem[0xD01F] = 7    // CONSOL: Start/Select/Option released
	s.NextTick = FrameCycles
	return s, nil
}

func (s *Sim) Snapshot() *Snapshot {
	m := s.Mpu
	return &Snapshot{mem: m.Mem, pc: m.PC, a: m.A, x: m.X, y: m.Y, sp: m.SP, p: m.P,
		cycles: m.Cycles, frames: s.Frames, nextTick: s.NextTick}
}

func (s *Sim) Restore(sn *Snapshot) {
	m := s.Mpu
	m.Mem = sn.mem
	m.PC, m.A, m.X, m.Y, m.SP, m.P, m.Cycles = sn.pc, sn.a, sn.x, sn.y, sn.sp, sn.p, sn.cycles
	s.Frames, s.NextTick = sn.frames, sn.nextTick
}

func (s *Sim) addr(name string) int {
	a, ok := s.Sym[name]
	if !ok {
		panic("sim: unknown symbol " + name)
	}
	return a
}

// Get returns the byte at symbol name (Python s[name]).
func (s *Sim) Get(name string) int { return int(s.Mem[s.addr(name)]) }

// Set stores a byte at symbol name (Python s[name] = v).
func (s *Sim) Set(name string, v int) { s.Mem[s.addr(name)] = byte(v) }

// Has reports whether the symbol exists.
func (s *Sim) Has(name string) bool { _, ok := s.Sym[name]; return ok }

// Word returns the little-endian word at symbol name.
func (s *Sim) Word(name string) int {
	a := s.addr(name)
	return int(s.Mem[a]) | int(s.Mem[a+1])<<8
}

// Siov emulates SIOV: read DAUX sector into DBUF, status 1, then RTS.
func (s *Sim) Siov() {
	m := s.Mem
	buf := int(m[0x304]) | int(m[0x305])<<8
	sec := int(m[0x30A]) | int(m[0x30B])<<8
	off := (sec - 1) * 128
	if off >= 0 {
		for i := 0; i < 128; i++ {
			if off+i < len(s.Atr) {
				m[(buf+i)&0xFFFF] = s.Atr[off+i]
			}
		}
	}
	m[0x303] = 1
	c := s.Mpu
	c.Y = 1
	sp := c.SP
	lo := m[0x100+int(sp+1)]
	hi := m[0x100+int(sp+2)]
	c.SP = sp + 2
	c.PC = uint16(int(hi)<<8|int(lo)) + 1
}

// Tick advances the 24-bit RTCLOK (low byte at $14) once.
func (s *Sim) Tick() {
	for _, i := range []int{0x14, 0x13, 0x12} {
		s.Mem[i]++
		if s.Mem[i] != 0 {
			break
		}
	}
}

// RunFrame runs one frame; joy bits (1 up, 2 down, 4 left, 8 right) are pressed directions.
func (s *Sim) RunFrame(joy int, fire bool) {
	s.Mem[0xD300] = 0xFF ^ byte(joy)
	if fire {
		s.Mem[0xD010] = 0
	} else {
		s.Mem[0xD010] = 1
	}
	c := s.Mpu
	for c.Cycles < s.NextTick {
		if c.PC == SIOV {
			s.SiovFn()
		} else {
			c.Step()
		}
	}
	s.NextTick += FrameCycles
	s.Tick()
	s.Frames++
}

func (s *Sim) Run(frames, joy int, fire bool) {
	for i := 0; i < frames; i++ {
		s.RunFrame(joy, fire)
	}
}

// Screen returns n bytes of TermScreen starting at row/col.
func (s *Sim) Screen(row, col, n int) []byte {
	base := s.addr("TermScreen") + row*40 + col
	return append([]byte(nil), s.Mem[base:base+n]...)
}

// Byte returns the byte at symbol name plus offset.
func (s *Sim) Byte(name string, off int) int { return int(s.Mem[s.addr(name)+off]) }
