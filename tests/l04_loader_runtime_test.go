package tests

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/sim"
)

// Regression checks for the compiled checked-loader contract when artifacts exist.

// l04Invoke calls a compiled routine with A=a and reports "success" on return or "failure"
// when execution reaches music_failure.
func l04Invoke(t *testing.T, s *sim.Sim, name string, a byte) string {
	t.Helper()
	m := s.Mpu
	callerPC, callerSP := m.PC, m.SP
	const sentinel = 0x0800
	const ret = sentinel - 1
	s.Mem[0x100+int(callerSP)] = ret >> 8
	s.Mem[0x100+int(callerSP-1)] = ret & 255
	m.SP, m.PC, m.A = callerSP-2, uint16(s.Sym[name]), a
	for i := 0; i < 100_000; i++ {
		if int(m.PC) == s.Sym["music_failure"] {
			return "failure"
		}
		if m.PC == sentinel {
			m.PC, m.SP = callerPC, callerSP
			return "success"
		}
		if m.PC == sim.SIOV {
			s.SiovFn()
		} else {
			m.Step()
		}
	}
	t.Fatal("checked loader did not terminate")
	return ""
}

// l04ReadFailureSim makes every SIO read report an error status after performing it.
func l04ReadFailureSim(t *testing.T) *sim.Sim {
	s := l04NewSim(t)
	real := s.Siov
	s.SiovFn = func() {
		real()
		s.Mpu.Y = 0
	}
	return s
}

func l04NewSim(t *testing.T) *sim.Sim {
	t.Helper()
	s, err := sim.New("", "")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("compiled artifacts unavailable: " + err.Error())
	}
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func l04ChunkEntry(s *sim.Sim, chunk int) []byte {
	base := s.Sym["ChunkTable"] + chunk*5
	return append([]byte(nil), s.Mem[base:base+5]...)
}

func l04SetDest(s *sim.Sim, dest int) {
	s.Mem[s.Sym["disk_dest"]] = byte(dest)
	s.Mem[s.Sym["disk_dest"]+1] = byte(dest >> 8)
}

func TestL04LoaderSourceContract(t *testing.T) {
	source := src()
	if n := len(regexp.MustCompile(`a=25\s+load_checked_chunk`).FindAllString(source, -1)); n != 1 {
		t.Errorf("a=25 load_checked_chunk occurrences = %d, want 1", n)
	}
	scene := regexp.MustCompile(`(?s)func ai_cutscene \{(.*?)\n\}`).FindStringSubmatch(source)
	if scene == nil {
		t.Fatal("func ai_cutscene not found")
	}
	if n := strings.Count(scene[1], "load_checked_chunk"); n != 3 {
		t.Errorf("ai_cutscene load_checked_chunk calls = %d, want 3", n)
	}
	if strings.Contains(scene[1], "call music_failure") {
		t.Error("ai_cutscene calls music_failure directly")
	}
}

func TestL04LoaderRuntime(t *testing.T) {
	requireBuilt(t)
	good := l04NewSim(t)

	// The existing compiled loader is the only executable under test here.
	for _, c := range []struct{ chunk, dest int }{{14, 0xA000}, {15, 0xA600}, {25, 0xA000}} {
		entry := l04ChunkEntry(good, c.chunk)
		sector := int(entry[0]) | int(entry[1])<<8
		size := (int(entry[2]) | int(entry[3])<<8) * 128
		l04SetDest(good, c.dest)
		if got := l04Invoke(t, good, "load_checked_chunk", byte(c.chunk)); got != "success" {
			t.Fatalf("chunk %d: %s", c.chunk, got)
		}
		if good.Mpu.A != entry[4] || entry[4] == 0 || good.Get("disk_failed") != 0 {
			t.Errorf("chunk %d: A=%02x checksum=%02x disk_failed=%d", c.chunk, good.Mpu.A, entry[4], good.Get("disk_failed"))
		}
		want := good.Atr[(sector-1)*128 : (sector-1)*128+size]
		if !bytes.Equal(good.Mem[c.dest:c.dest+size], want) {
			t.Errorf("chunk %d: loaded data differs from ATR", c.chunk)
		}
	}

	entry := l04ChunkEntry(good, 25)
	sector := int(entry[0]) | int(entry[1])<<8
	bad := append([]byte(nil), good.Atr...)
	bad[(sector-1)*128] ^= 1
	checksum := l04NewSim(t)
	checksum.Atr = bad
	l04SetDest(checksum, 0xA000)
	if got := l04Invoke(t, checksum, "load_checked_chunk", 25); got != "failure" {
		t.Errorf("corrupt sector: %s, want failure", got)
	}

	readErr := l04ReadFailureSim(t)
	l04SetDest(readErr, 0xA000)
	if got := l04Invoke(t, readErr, "load_checked_chunk", 25); got != "failure" {
		t.Errorf("SIO error: %s, want failure", got)
	}
}
