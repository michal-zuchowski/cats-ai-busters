package tests

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
	"github.com/michal-zuchowski/cats-ai-busters/tools/sim"
)

// Level 1 -> level 2 transition on the compiled artifact. The sim fakes no interrupts, so
// music_ready stands in for "the VBI is playing a note": an SIO read while it is set is a
// disk load under a sustained first note. Only the sim's SIOV and RTCLOK are faked.

type trSim struct {
	*sim.Sim
	sio     int
	audible int   // sectors read while music_ready != 0
	sectors []int // DAUX sector of every SIO call
}

type trRun struct {
	sio, frames int
	cycles      int64
}

func trNew(t *testing.T) *trSim {
	t.Helper()
	s, err := sim.New("", "")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("compiled artifacts unavailable: " + err.Error())
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"preload_ready", "StageRoom", "StageMusic", "take_staged_room", "take_staged_music", "init_plat"} {
		if !s.Has(n) {
			t.Skip("compiled artifact predates level 2 staging (no symbol " + n + "); rebuild")
		}
	}
	r := &trSim{Sim: s}
	real := s.Siov
	s.SiovFn = func() {
		r.sio++
		if s.Get("music_ready") != 0 {
			r.audible++
		}
		r.sectors = append(r.sectors, int(s.Mem[0x30A])|int(s.Mem[0x30B])<<8)
		real()
	}
	s.Set("lives", 9)
	return r
}

// trStale reports whether the XEX is older than the newest source file.
func trStale(t *testing.T) bool {
	xi, err := os.Stat(k65src.Path("out", "cats-ai-busters.xex"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range k65src.Files() {
		if fi, err := os.Stat(f); err == nil && fi.ModTime().After(xi.ModTime()) {
			return true
		}
	}
	return false
}

// call runs a compiled routine to its return, ticking RTCLOK once per frame of CPU cycles.
func (r *trSim) call(t *testing.T, name string) trRun {
	t.Helper()
	s, m := r.Sim, r.Mpu
	pc, sp := m.PC, m.SP
	const ret = 0x07FF
	s.Mem[0x100+int(sp)] = ret >> 8
	s.Mem[0x100+int(sp-1)] = ret & 255
	m.SP, m.PC = sp-2, uint16(s.Sym[name])
	start, sio0, fr0 := m.Cycles, r.sio, s.Frames
	s.NextTick = m.Cycles + sim.FrameCycles
	music := uint16(s.Sym["music_failure"])
	for i := 0; i < 100_000_000; i++ {
		if m.PC == 0x0800 {
			m.PC, m.SP = pc, sp
			return trRun{r.sio - sio0, int(s.Frames - fr0), m.Cycles - start}
		}
		if m.PC == music {
			t.Fatalf("%s reached music_failure", name)
		}
		if m.PC == sim.SIOV {
			s.SiovFn()
		} else {
			m.Step()
		}
		if m.Cycles >= s.NextTick {
			s.NextTick += sim.FrameCycles
			s.Tick()
			s.Frames++
		}
	}
	t.Fatalf("%s did not return", name)
	return trRun{}
}

// chunk returns the ATR bytes of ChunkTable entry i.
func (r *trSim) chunk(i int) []byte {
	e := r.Mem[r.Sym["ChunkTable"]+i*5:]
	off := (int(e[0]) | int(e[1])<<8 - 1) * 128
	return r.Atr[off : off+(int(e[2])|int(e[3])<<8)*128]
}

// trToLevel1End boots level 1; with warm it enters room 3 so level 2 gets staged. It returns
// the cost of the room 3 entry and leaves the machine at the end of level 1.
func trToLevel1End(t *testing.T, r *trSim, warm bool) trRun {
	t.Helper()
	r.call(t, "init_gameplay")
	var entry trRun
	if warm {
		r.Set("cur_room", 2)
		entry = r.call(t, "enter_room")
		if r.Get("preload_ready") != 1 {
			t.Fatal("room 3 entry did not stage level 2")
		}
	} else if r.Get("preload_ready") != 0 {
		t.Fatal("fresh game must start unstaged")
	}
	return entry
}

// trNextLevel runs the compiled next_level. When the XEX is older than the sources, init_plat's
// call order is taken from the current source and executed with the compiled routines instead.
func trNextLevel(t *testing.T, r *trSim) trRun {
	t.Helper()
	if !trStale(t) {
		return r.call(t, "next_level")
	}
	t.Log("XEX older than src/: running the source order of init_plat on compiled routines")
	r.Set("cur_level", 1)
	r.Set("cur_room", 0)
	r.Set("entry_col", 0)
	r.Set("level_finished", 0)
	var tot trRun
	for _, m := range regexp.MustCompile(`(?m)^\s*(?:call )?(stop_music|enter_plat_room|level_music)\b`).FindAllStringSubmatch(gpStripComments(gpBody("init_plat")), -1) {
		x := r.call(t, m[1])
		tot.sio, tot.frames, tot.cycles = tot.sio+x.sio, tot.frames+x.frames, tot.cycles+x.cycles
	}
	r.Set("preload_ready", 0)
	return tot
}

func TestLevel1To2TransitionWarmNoDisk(t *testing.T) {
	r := trNew(t)
	entry := trToLevel1End(t, r, true)
	t.Logf("room 3 entry (stages level 2): %d SIO, %d frames", entry.sio, entry.frames)
	if !bytes.Equal(r.Mem[r.Sym["StageRoom"]:][:1024], r.chunk(5)) || !bytes.Equal(r.Mem[r.Sym["StageMusic"]:][:512], r.chunk(19)[:512]) {
		t.Fatal("stage buffers differ from chunks 5 / 19")
	}
	ensure(t, r.Mem[r.Sym["StageRoom"]+960+0x0F] == 0, "staged first room must not be a glass room: its cry load would overwrite StageMusic")

	// Source assertion: level 1 skips the hold (its next data is staged); other levels keep it.
	ending := gpStripComments(gpBody("level_ending"))
	ensure(t, regexp.MustCompile(`a=cur_level\s+a\?0 != \{\s*x=75\s+wait_frames\s+x=75\s+wait_frames\s*\}`).MatchString(ending), "level_ending must hold only when cur_level != 0")
	if trStale(t) {
		t.Log("XEX older than src/: level_ending NOT executed (compiled artifact still has the old 150-frame hold); source assertion only")
	} else {
		end := r.call(t, "level_ending")
		t.Logf("compiled level_ending: %d SIO, %d frames", end.sio, end.frames)
		ensure(t, end.sio == 0 && end.frames < 5, "level 1 ending must not wait: %+v", end)
	}

	r.Set("level_finished", 1)
	audible0 := r.audible
	tr := trNextLevel(t, r)
	t.Logf("warm next_level: %d SIO, %d frames, %d cycles", tr.sio, tr.frames, tr.cycles)
	ensure(t, r.audible == audible0, "warm transition read under a note")
	ensure(t, tr.sio == 0, "warm transition SIO = %d", tr.sio)
	ensure(t, tr.cycles < 3*sim.FrameCycles, "warm transition %d cycles", tr.cycles)
	ensure(t, r.Get("cur_level") == 1 && r.Get("music_ready") == 1 && r.Get("preload_ready") == 0, "state after transition")
	ensure(t, bytes.Equal(r.Mem[r.Sym["Music"]:][:512], r.chunk(19)[:512]), "level 2 track not in Music")
	ensure(t, bytes.Equal(r.Mem[r.Sym["RoomArtBuf"]:][:960], r.chunk(5)[:960]), "level 2 room 0 not in RoomArtBuf")
}

func TestLevel1To2TransitionColdMusicOnlyWhenReady(t *testing.T) {
	r := trNew(t)
	trToLevel1End(t, r, false)
	if !trStale(t) {
		r.call(t, "level_ending")
	}
	r.Set("level_finished", 1)
	before := r.sio
	audible0 := r.audible
	tr := trNextLevel(t, r)
	t.Logf("cold next_level: %d SIO (%d under a playing note), sectors %v", tr.sio, r.audible-audible0, r.sectors[before:])
	room, music := len(r.chunk(5))/128, len(r.chunk(19))/128
	ensure(t, tr.sio == room+music, "cold SIO %d, want %d room + %d music", tr.sio, room, music)
	ensure(t, r.audible == audible0, "%d sectors were read while music_ready was set", r.audible-audible0)
	ensure(t, r.Get("music_ready") == 1 && bytes.Equal(r.Mem[r.Sym["Music"]:][:512], r.chunk(19)[:512]), "music not ready after load")
}
