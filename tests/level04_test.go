package tests

// L04 source/data checks and frame-level route replay; does not execute a K65 build.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	A "github.com/michal-zuchowski/cats-ai-busters/tools/aiscene"
	ATR "github.com/michal-zuchowski/cats-ai-busters/tools/atr"
	L "github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
	P "github.com/michal-zuchowski/cats-ai-busters/tools/platlevels"
	M "github.com/michal-zuchowski/cats-ai-busters/tools/platmodel"
)

// lv4Finale is plain jump/lift physics plus L04 floor contact and final-action gates.
type lv4Finale struct {
	*M.Sim
	DlgDone, Scenes, Facing, PawT int
	Stages                        [4]int
}

func lv4NewFinale(chunk []byte) *lv4Finale {
	f := &lv4Finale{Sim: M.New(chunk, 0)}
	f.OverrideUnder = f.under
	f.OverrideSwitch = f.switchHook
	f.OverrideStep = f.step
	return f
}

func (f *lv4Finale) glassReady() bool {
	delta := f.M["glass"]*4 - f.X
	return f.DlgDone != 0 && f.M["glass"] != 0 && f.Y == f.M["desk"] &&
		f.Air == 0 && f.Jprep == 0 && f.Lnd == 0 && f.PawT == 0 && f.Facing == 0 &&
		12 <= delta && delta < 16
}

func (f *lv4Finale) under(*M.Sim) bool {
	if f.Y&7 == 0 {
		r := f.Y/8 - 1
		if f.Scr[r][f.Col()]&127 == L.Beam && f.Art[r][f.Col()]&127 == L.Floor {
			return true
		}
	}
	return f.BaseUnder()
}

// switchHook models the shared pixel-edge gate, not the old six-column shortcut.
func (f *lv4Finale) switchHook(*M.Sim) {
	if f.glassReady() {
		f.Done = true
		f.Stages = [4]int{54, 55, 56, 57}
	}
}

func (f *lv4Finale) step(_ *M.Sim, keys int) {
	f.Input(keys)
	if f.Done {
		return
	}
	f.Phys()
	f.Lift()
	on := f.T&32 == 0
	for c := f.M["h0"]; c <= f.M["h1"]; c++ {
		if on {
			f.Scr[22][c] = L.Beam | 128
		} else {
			f.Scr[22][c] = f.Art[22][c]
		}
	}
	if on && f.Y == 184 && f.M["h0"] <= f.Col() && f.Col() <= f.M["h1"] {
		f.Respawn()
	}
	if f.M["dlg"] != 0 && f.DlgDone == 0 && f.Air == 0 && f.Y < 137 && f.Col() >= 20 {
		f.DlgDone = 1
		f.Scenes++
		f.FirePrev = 0
	}
	if f.Lnd != 0 {
		f.Lnd--
	}
	f.T++
}

func lv4Walk(t *testing.T, s *lv4Finale, col int) {
	t.Helper()
	dir := M.Left
	if s.Col() < col {
		dir = M.Right
	}
	if !M.HoldUntil(s.Sim, dir, func(g *M.Sim) bool { return g.Col() == col }, 0) {
		t.Fatalf("walk to col %d failed at x=%d y=%d", col, s.X, s.Y)
	}
	if err := M.Settle(s.Sim, 40); err != nil {
		t.Fatal(err)
	}
}

func lv4Jump(t *testing.T, s *lv4Finale) {
	t.Helper()
	if err := M.DoJump(s.Sim, M.Right, 100, M.Right); err != nil {
		t.Fatal(err)
	}
}

func lv4Body(name string) string { return funcBodyBraced(src(), name) }

func lv4Has(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("missing %q", sub)
	}
}

func TestLevel04RoomsAsset(t *testing.T) {
	rooms := readAsset("plat-rooms.bin")
	var all []byte
	for _, r := range P.Rooms {
		all = append(all, r...)
	}
	ensure(t, bytes.Equal(rooms, all), "assets/plat-rooms.bin is stale")
	h := sha256.Sum256(append(append([]byte(nil), rooms[:5120]...), rooms[6144:7168]...))
	ensure(t, hex.EncodeToString(h[:]) == "72914d2e88474c393a75bc7febdd697330bc0f7734ee5add073f84745344a568", "L03/L04 room hash %x", h)
}

func TestLevel04Room1Route(t *testing.T) {
	// Room 1: three rising steps, transfer to boarding dock, mandatory lift, roof hatch.
	r1 := lv4NewFinale(P.Rooms[8])
	for _, st := range [][2]int{{2, 168}, {8, 152}, {15, 136}, {22, 152}} {
		lv4Walk(t, r1, st[0])
		lv4Jump(t, r1)
		if r1.Y != st[1] || r1.Deaths != 0 {
			t.Fatalf("step col %d: x=%d y=%d want %d deaths=%d", st[0], r1.X, r1.Y, st[1], r1.Deaths)
		}
	}
	lv4Walk(t, r1, 31)
	if !M.HoldUntil(r1.Sim, 0, func(g *M.Sim) bool { return g.LiftRow == 18 && g.LiftT == 0 }, 0) {
		t.Fatal("lift never at row 18")
	}
	lv4Walk(t, r1, 33)
	if r1.Y != (r1.LiftRow+1)*8 || r1.Deaths != 0 {
		t.Fatalf("not boarded: y=%d row=%d deaths=%d", r1.Y, r1.LiftRow, r1.Deaths)
	}
	if !M.HoldUntil(r1.Sim, 0, func(g *M.Sim) bool { return g.LiftRow == 10 && g.LiftT == 0 }, 0) {
		t.Fatal("lift never at row 10")
	}
	lv4Jump(t, r1)
	if r1.Y != 80 || r1.Deaths != 0 {
		t.Fatalf("roof: y=%d deaths=%d", r1.Y, r1.Deaths)
	}
	if !M.HoldUntil(r1.Sim, M.Right, func(g *M.Sim) bool { return g.Exited }, 0) {
		t.Fatal("no exit")
	}
	ensure(t, r1.Scenes == 0, "room 1 played a scene")

	art1, m1 := M.Unpack(P.Rooms[8])
	supports := map[[2]int]bool{}
	for r := 0; r < 24; r++ {
		for c := 0; c < 40; c++ {
			if M.IsSupport(art1[r][c]) {
				supports[[2]int{r, c}] = true
			}
		}
	}
	expected := map[[2]int]bool{}
	for _, s := range [][3]int{{22, 0, 39}, {20, 4, 8}, {18, 11, 15}, {16, 18, 22}, {18, 27, 31}, {9, 35, 39}} {
		for c := s[1]; c <= s[2]; c++ {
			expected[[2]int{s[0], c}] = true
		}
	}
	ensure(t, len(supports) == len(expected), "support count %d != %d", len(supports), len(expected))
	for k := range expected {
		ensure(t, supports[k], "missing support %v", k)
	}
	ensure(t, 136-m1["exit_y"] > 26, "exit hatch within jump apex") // highest other fixed support exceeds the measured jump apex
	for _, h := range []int{184, 168, 152, 136, 88} {
		r1.Y, r1.Air = h, 0
		ensure(t, !r1.ExitAllowed(), "exit allowed at y=%d", h)
	}
	r1.Y, r1.Air = 80, 1
	ensure(t, !r1.ExitAllowed(), "exit allowed airborne")
}

func TestLevel04ElectricPhases(t *testing.T) {
	// Electric state and contact agree at every phase and boundary, with no phantom support in L03.
	lv4Has(t, lv4Body("under"), "a?79 == { a=cur_level a?3 == { goto under_base } a=0 return }")
	haz := lv4Body("plat_haz")
	for _, n := range []string{"a&0x20", "a?184", "a?m_h0", "a?player_col", "respawn"} {
		lv4Has(t, haz, n)
	}
	for tt := 0; tt < 64; tt++ {
		for _, col := range []int{6, 7, 20, 34, 35} {
			s := lv4NewFinale(P.Rooms[8])
			s.T, s.X = tt, (col-2)*4-2
			s.Step(0)
			want := 0
			if tt < 32 && 7 <= col && col <= 34 {
				want = 1
			}
			ensure(t, s.Deaths == want, "t=%d col=%d deaths=%d", tt, col, s.Deaths)
			for c := 7; c < 35; c++ {
				w := L.Floor
				if tt < 32 {
					w = L.Beam | 128
				}
				ensure(t, s.Scr[22][c] == w, "t=%d col=%d tile[%d]=%d", tt, col, c, s.Scr[22][c])
			}
		}
		s := lv4NewFinale(P.Rooms[8])
		s.T, s.X, s.Y, s.Air, s.Vy = tt, 70, 180, 1, -1
		s.Step(0)
		ensure(t, s.Deaths == 0, "airborne death t=%d", tt) // airborne cat is above the electrified surface
		s = lv4NewFinale(P.Rooms[8])
		s.T, s.X, s.Y = tt, 22, 168
		s.Step(0)
		ensure(t, s.Deaths == 0, "ledge death t=%d", tt) // same columns are safe on an elevated ledge
	}
}

func TestLevel04Room2Finale(t *testing.T) {
	_, m1 := M.Unpack(P.Rooms[8])
	// Room 2: jump across descending ledges, confront AI, then jump to the glass yourself.
	r2 := lv4NewFinale(P.Rooms[9])
	ensure(t, r2.Y == 80 && r2.Y == m1["exit_y"], "room 2 entry y=%d", r2.Y)
	ensure(t, r2.M["desk"] == 120 && r2.M["glass"] == 33, "desk/glass metas")
	ensure(t, r2.M["lw"] == 0 && r2.M["arow"] == 0, "live lift/actor in cinematic room") // cinematic cleanup has no live lift/actor to repaint
	lv4Walk(t, r2, 6)
	lv4Jump(t, r2)
	if r2.Y != 96 || r2.Deaths != 0 {
		t.Fatalf("ledge 1: y=%d deaths=%d", r2.Y, r2.Deaths)
	}
	lv4Walk(t, r2, 16)
	lv4Jump(t, r2)
	if r2.Y != 112 || r2.Deaths != 0 || r2.Scenes != 1 || r2.Done {
		t.Fatalf("ledge 2: y=%d deaths=%d scenes=%d done=%v", r2.Y, r2.Deaths, r2.Scenes, r2.Done)
	}
	lv4Walk(t, r2, 24)
	lv4Jump(t, r2)
	if r2.Y != 120 || r2.Deaths != 0 || r2.Scenes != 1 || r2.Done {
		t.Fatalf("ledge 3: y=%d deaths=%d scenes=%d done=%v", r2.Y, r2.Deaths, r2.Scenes, r2.Done)
	}
	lv4Walk(t, r2, 32)
	r2.Step(M.Fire)
	ensure(t, r2.Done, "final push did not fire")

	// Preserve the original dialogue/action edge coverage: the first FIRE starts
	// dialogue, held FIRE is ignored, and only a fresh release/new press pushes.
	s := lv4NewFinale(P.Rooms[9])
	s.X, s.Y = 118, 120 // direct/debug approach cannot bypass the scene with FIRE
	s.Step(M.Fire)
	ensure(t, !s.Done && s.DlgDone != 0, "first FIRE")
	s.Step(M.Fire)
	ensure(t, !s.Done, "held FIRE leaked into push") // held skip/action cannot leak into the final push
	s.Step(0)
	s.Step(M.Fire)
	ensure(t, s.Done, "fresh FIRE did not push")
	s.Respawn()
	ensure(t, s.DlgDone != 0 && s.Scenes == 1, "respawn repeated the monologue") // falling does not force another monologue
	s.X, s.Y = 90, 136
	s.Step(0)
	ensure(t, s.Scenes == 1, "scene repeated")
	for x := 117; x < 121; x++ {
		s = lv4NewFinale(P.Rooms[9])
		s.DlgDone, s.X, s.Y, s.FirePrev = 1, x, 120, 1
		s.Step(M.Fire) // each case starts with a fresh released FIRE edge
		ensure(t, s.Done && s.Stages == [4]int{54, 55, 56, 57}, "push at x=%d", x)
	}
	for _, attr := range []string{"air", "jprep", "lnd", "paw_t"} {
		s = lv4NewFinale(P.Rooms[9])
		s.DlgDone, s.X, s.Y, s.FirePrev = 1, 118, 120, 1
		switch attr {
		case "air":
			s.Air = 1
		case "jprep":
			s.Jprep = 1
		case "lnd":
			s.Lnd = 1
		case "paw_t":
			s.PawT = 1
		}
		s.Step(M.Fire)
		ensure(t, !s.Done, "pushed with %s set", attr)
	}
	s = lv4NewFinale(P.Rooms[9])
	s.DlgDone, s.X, s.Y, s.Facing, s.FirePrev = 1, 118, 120, 1, 1
	s.Step(M.Fire)
	ensure(t, !s.Done, "wrong-facing push") // wrong-facing FIRE cannot trigger the push
	s = lv4NewFinale(P.Rooms[9])
	s.X, s.Y, s.T = 74, 184, 32
	s.Step(0)
	ensure(t, s.DlgDone == 0, "floor perch started the scene") // floor is not a safe cinematic perch
}

func TestLevel04FinaleSource(t *testing.T) {
	main := src()
	lv4Has(t, lv4Body("plat_push"), "glass_ready")
	ready := lv4Body("glass_ready")
	for _, n := range []string{"a=dlg_done", "a=m_glass", "a=player_y", "a=air", "a=jprep",
		"a=lnd", "a=paw_t", "a=facing", "a<< a<<", "c+ a-player_x", "a?12", "a?16"} {
		lv4Has(t, ready, n)
	}
	fs := lv4Body("final_scene")
	for _, n := range []string{"a=54", "a=55", "a=56", "a=57"} {
		lv4Has(t, fs, n)
	}
	ensure(t, mustIndex(t, fs, "a=56") < mustIndex(t, fs, "gcol++"), "a=56 after gcol++")
	ensure(t, mustIndex(t, fs, "put_run") < mustIndex(t, fs, "AUDC1=a=0"), "put_run after AUDC1=0")
	events := []string{
		"wm=a=13 gcol=a=m_glass clear_glass_scene kk=a=0 glass_glyph gcol++ draw_glass_scene",
		"wm=a=13 clear_glass_scene kk=a=0 glass_glyph gcol++ draw_glass_scene",
		"wm=a=13 clear_glass_scene wm=a=14 kk=a=1 glass_glyph draw_glass_scene",
		"wm=a=14 clear_glass_scene wm=a=15 gcol=a=36 kk=a=2 glass_glyph draw_glass_scene",
		"wm=a=15 clear_glass_scene wm=a=17 gcol=a=36 kk=a=3 glass_glyph draw_glass_scene",
	}
	last := -1
	for _, e := range events {
		pos := mustIndex(t, fs, e)
		ensure(t, pos > last, "event out of order: %s", e)
		last = pos
	}
	ensure(t, mustIndex(t, fs, "tmp=a=12 x=18 y=35 cnt=a=4 put_run") > last, "closing text before events")
	pmgOff := mustRIndex(t, fs, "GRACTL=a=0")
	pcmEnd := mustRIndex(t, fs, "AUDC1=a=0")
	textSwitch := mustIndex(t, fs, "clear_term_screen")
	ensure(t, pcmEnd < pmgOff && pmgOff < textSwitch, "ending teardown order")
	for _, reg := range []string{"SDMCTL=a=0x22", "SDLSTL=a=&<dl_term", "SDLSTH=a=&>dl_term", "CHBAS=a=&>TermFont"} {
		i := strings.Index(fs[pmgOff:], reg)
		ensure(t, i >= 0 && pmgOff+i < textSwitch, "%s not before text switch", reg)
	}
	dlg := lv4Body("plat_dlg")
	lv4Has(t, dlg, "a?3 != { return }")
	lv4Has(t, dlg, "a?137 >= { return }")
	lv4Has(t, dlg, "a?20 < { return }")
	lv4Has(t, dlg, "ai_cutscene")
	lv4Has(t, lv4Body("init_plat"), "dlg_done=a=0")
	ensure(t, !strings.Contains(lv4Body("respawn"), "dlg_done"), "respawn resets dlg_done")
	lv4Has(t, main, "0x4D &<ZoomBuf &>ZoomBuf")
	lv4Has(t, main, "for x=0..78 eval [0x0D]")
	lv4Has(t, fs, "level_finished=a=1")
	lv4Has(t, fs, "CryBuf")
}

func TestLevel04AICutsceneData(t *testing.T) {
	// Streamed cinematic, mouth packing and all three overwritten memory regions.
	image, closed, opened := A.Portrait()
	portrait := readAsset("ai-portrait.pic")
	want := append(append(A.Pack(image), A.Pack(closed)...), A.Pack(opened)...)
	ensure(t, bytes.Equal(portrait, want) && len(portrait) == 3260, "ai-portrait.pic stale or wrong size %d", len(portrait))
	ensure(t, len(A.Pack(closed)) == 30 && len(A.Pack(opened)) == 30 && !bytes.Equal(closed.Pix, opened.Pix), "mouth packing")
	m := A.Mouth
	ensure(t, bytes.Equal(A.Pack(image.Crop(m[0], m[1], m[2], m[3])), A.Pack(closed)), "mouth crop")
	chunks := ATR.ExtraChunks()
	ensure(t, chunks[25].Name == "ai-portrait" && len(chunks) == 26, "chunk table")
	disk, _, table, err := ATR.BuildATR(make([]byte, 128), chunks)
	if err != nil {
		t.Fatal(err)
	}
	ensure(t, bytes.Equal(table, readAsset("disk-chunks.bin")) && len(table) == 130, "disk-chunks.bin stale")
	entry := table[125:130]
	ensure(t, entry[2] == 26 && entry[3] == 0, "portrait chunk sector count")
	padded := make([]byte, 3328)
	copy(padded, portrait)
	ensure(t, int(entry[4]) == sumBytes(padded)&255, "portrait checksum")
	ensure(t, 0xA000+3328 <= 0xB000 && 0xB000 < 0xB400 && 0xB400 < 0xBC00, "memory layout")
	memory := bytes.Repeat([]byte{0xAA}, 65536)
	load := func(chunk, dest int) {
		e := table[chunk*5 : chunk*5+5]
		start, count := int(e[0])|int(e[1])<<8, int(e[2])|int(e[3])<<8
		copy(memory[dest:dest+count*128], disk[16+(start-1)*128:16+(start-1+count)*128])
	}
	load(14, 0xA000)
	load(15, 0xA600)
	load(21, 0xB400)
	savedCry := append([]byte(nil), memory[0xA600:0xB000]...)
	savedMusic := append([]byte(nil), memory[0xB400:0xB600]...)
	load(25, 0xA000)
	ensure(t, bytes.Equal(memory[0xA000:0xA000+3260], portrait) && !bytes.Equal(memory[0xA600:0xB000], savedCry), "portrait load")
	ensure(t, bytes.Equal(memory[0xB400:0xB600], savedMusic), "music clobbered")
	load(14, 0xA000)
	copy(memory[0xB000:0xB000+960], memory[0xA000:0xA000+960])
	load(15, 0xA600)
	ensure(t, bytes.Equal(memory[0xA000:0xA400], P.Rooms[9]), "room 2 not restored")
	ensure(t, bytes.Equal(memory[0xB000:0xB000+960], P.Rooms[9][:960]), "room 2 screen copy")
	ensure(t, bytes.Equal(memory[0xA600:0xB000], savedCry) && bytes.Equal(memory[0xB400:0xB600], savedMusic), "cry/music not restored")
	ensure(t, bytes.Equal(memory[0x1000:0xA000], bytes.Repeat([]byte{0xAA}, 0x9000)), "resident atlas/code touched")
	ensure(t, bytes.Equal(memory[0xBC00:0xC000], bytes.Repeat([]byte{0xAA}, 1024)), "cat pages touched") // only init_pmg, not SIO, owns the cat pages

	mouth := lv4Body("ai_mouth")
	for _, p := range [][2]string{{"3200", "2256"}, {"3206", "2296"}, {"3212", "2336"}, {"3218", "2376"}, {"3224", "2416"}} {
		lv4Has(t, mouth, "ZoomBuf+"+p[0]+",x")
		lv4Has(t, mouth, "ZoomBuf+"+p[1]+",y")
	}
	lv4Has(t, mouth, "y?6")
	lv4Has(t, mouth, "x=ai_pose")
	scene := lv4Body("ai_cutscene")
	lv4Has(t, scene, "a=25 load_checked_chunk")
	lv4Has(t, scene, "a?180")
	lv4Has(t, scene, "a?5")
	ensure(t, strings.Count(scene, "load_checked_chunk") == 3 && strings.Count(scene, "call music_failure") == 0, "load_checked_chunk uses")
	checked := lv4Body("load_checked_chunk")
	lv4Has(t, checked, "a?0")
	lv4Has(t, checked, "!= { music_failure }")
	ensure(t, mustIndex(t, checked, "disk_failed") < mustIndex(t, checked, "disk_checksum"), "disk_failed after checksum")
	parts := strings.SplitN(scene, "ai_restore:", 2)
	lv4Need(t, len(parts) == 2, "no ai_restore label")
	for _, req := range []string{"paint_room_art", "a=15 load_checked_chunk", "draw_glass", "draw_hud_static",
		"clear_row0", "clear_row1", "&<dl_term", "LvC1,x", "LvC2,x", "init_pmg",
		"fire_prev=a=0", "NMIEN=a=0xC0"} {
		lv4Has(t, parts[1], req)
	}
	for _, n := range []string{"enter_plat_room", "init_plat", "RevealImg", "stop_music", "level_music"} {
		ensure(t, !strings.Contains(scene, n), "ai_cutscene references %s", n)
	}
	tick := lv4Body("ai_tick")
	for _, n := range []string{"a^30", "a?2", "a?4", "sfx_switch", "input_pressed x=a CH=a=0xFF a=x", "fire_prev=a=1"} {
		lv4Has(t, tick, n)
	}
	ensure(t, !strings.Contains(tick, "goto"), "ai_tick uses goto") // skip returns through the call stack before parent cleanup
}

func lv4Need(t *testing.T, ok bool, format string, args ...any) {
	t.Helper()
	if !ok {
		t.Fatalf(format, args...)
	}
}
