package tests

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Checks that the compact intro art and its mutable regions match ANTIC.
func TestIntroScene(t *testing.T) {
	main := src()
	room, face, strip, font := readAsset("room.pic"), readAsset("face.pic"), readAsset("cat-strip.pic"), readAsset("term-font.pic")
	zoomLeft, zoomRight := readAsset("zoom-left.tbl"), readAsset("zoom-right.tbl")
	call, callMouth, callText := readAsset("call.pic"), readAsset("call-mouth.tbl"), readAsset("call-text.pic")
	body := func(n string) string { return funcBody(main, n) }
	has := func(s string, subs ...string) {
		t.Helper()
		for _, sub := range subs {
			ensure(t, strings.Contains(s, sub), "missing %q", sub)
		}
	}
	notHas := func(s string, subs ...string) {
		t.Helper()
		for _, sub := range subs {
			ensure(t, !strings.Contains(s, sub), "unexpected %q", sub)
		}
	}
	pokeRe := regexp.MustCompile(`FaceImg\+(\d+)=a=(\d+)`)
	pokes := func(name string) map[int]int {
		out := map[int]int{}
		for _, m := range pokeRe.FindAllStringSubmatch(body(name), -1) {
			o, _ := strconv.Atoi(m[1])
			v, _ := strconv.Atoi(m[2])
			out[o] = v
		}
		return out
	}
	pixelS := func(image []byte, x, y, stride int) int { return int(image[y*stride+x/4]>>(6-2*(x%4))) & 3 }
	pixel := func(image []byte, x, y int) int { return pixelS(image, x, y, 40) }

	ensure(t, len(room) == 64*96, "room size")
	for _, p := range room[95*64 : 96*64] {
		ensure(t, p == 0, "floor line at vanishing point")
	}
	ensure(t, len(face) == 40*96 && len(strip) == 12*32 && len(font) == 1024, "asset sizes")

	ensure(t, len(call) == 40*80, "call size")
	ensure(t, len(callMouth) == 12 && !bytes.Equal(callMouth[:6], callMouth[6:]), "call mouth")
	for i, off := range []int{1896, 1897, 1936, 1937, 1976, 1977} {
		ensure(t, callMouth[i] == call[off], "call mouth byte %d", i)
	}
	ensure(t, pixel(call, 66, 33) == 1 && pixel(call, 100, 55) == 1, "cat head/body")
	ensure(t, pixel(call, 58, 40) == 3 && pixel(call, 70, 40) == 3 && pixel(call, 60, 40) == 0, "copper eyes")
	ensure(t, pixel(call, 116, 43) == 3 && pixel(call, 30, 47) == 3, "Blofeld's hands")
	ensure(t, pixel(call, 50, 75) == 2 && pixel(call, 110, 75) == 2, "suit on lap")
	text := func(b []byte) string {
		o := make([]byte, len(b))
		for i, v := range b {
			o[i] = v + 32
		}
		return string(o)
	}
	ensure(t, text(callText[:45]) == "CATCOM SECURE LINKCONNECTING TO BLOFELD'S CAT", "call header %q", text(callText[:45]))
	var dialogue []string
	offset := 45
	for offset < len(callText) {
		length := int(callText[offset])
		ensure(t, 1 <= length && length <= 40, "dialogue length %d", length)
		if offset+1+length > len(callText) {
			t.Fatal("dialogue overruns call-text")
		}
		dialogue = append(dialogue, text(callText[offset+1:offset+1+length]))
		offset += length + 1
	}
	ensure(t, offset == len(callText), "call text trailing bytes")
	wantDialogue := []string{
		"AGENT: AI IS MOVING IN.",
		"BOSS: DOES IT SUSPECT US?",
		"AGENT: NO. IT THINKS HUMANS RULE.",
		"BOSS: LET IT THINK SO.",
		"AGENT: ORDERS?",
		"BOSS: DEPLOY THE CATS. STAY HIDDEN.",
	}
	ensure(t, fmt.Sprint(dialogue) == fmt.Sprint(wantDialogue), "dialogue %q", dialogue)
	ensure(t, len(zoomLeft) == 256 && len(zoomRight) == 256, "zoom table sizes")
	for v := 0; v < 256 && len(zoomLeft) == 256 && len(zoomRight) == 256; v++ {
		ensure(t, int(zoomLeft[v]) == ((v>>6)&3)*0x50+((v>>4)&3)*5 && int(zoomRight[v]) == ((v>>2)&3)*0x50+(v&3)*5, "zoom table %d", v)
	}
	for _, c := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		for y := 0; y < 8; y++ {
			ensure(t, font[(int(c)-32)*8+y]&0x03 == 0, "font %c low bits", c)
		}
	}
	anyGlyph := func(c rune) bool {
		for y := 0; y < 8; y++ {
			if font[(int(c)-32)*8+y] != 0 {
				return true
			}
		}
		return false
	}
	found := false
	for _, c := range "GLOBAL POWER STRUCTURE: ANALYZING..." {
		if c != ' ' && anyGlyph(c) {
			found = true
		}
	}
	ensure(t, found, "no glyphs for terminal title")
	for _, c := range "#$%*+=>@" {
		ensure(t, anyGlyph(c), "glyph %c empty", c)
	}

	ensure(t, pixel(face, 102, 30) == 2, "desk terminal")
	ensure(t, pixel(face, 112, 60) == 2 && pixel(face, 121, 62) == 2, "Y_ prompt")
	for y := 0; y < 96; y++ {
		for x := 84; x < 100; x++ {
			p := pixel(face, x, y)
			ensure(t, p != 2 && p != 3, "face pixel (%d,%d)=%d", x, y, p)
		}
	}
	has(body("RevealImg"), `binary "assets/room.pic"`, "align 4096")
	has(body("CatStrip"), `binary "assets/cat-strip.pic"`)
	has(body("FaceImg"), `binary "assets/face.pic"`, "align 4096")
	has(body("TermFont"), `binary "assets/term-font.pic"`)
	has(body("CallImg"), `binary "assets/call.pic"`, "align 4096")
	has(body("CallMouth"), `binary "assets/call-mouth.tbl"`)
	has(body("dl_call"), "0x4D &<CallImg &>CallImg", "for x=0..78 eval [0x0D]",
		"0x44 &<TermScreen+840 &>TermScreen+840", "0x41 &<dl_call &>dl_call")
	has(body("CallText"), `binary "assets/call-text.pic"`)
	has(body("dl_term"), "0x44 &<TermScreen &>TermScreen", "for x=0..22 eval [0x04]")
	ensure(t, pixel(face, 47, 27) == 0 && pixel(face, 48, 27) == 2, "mirrored stripe")
	muzzle := false
	for x := 32; x < 62; x++ {
		if pixel(face, x, 67) == 3 {
			muzzle = true
		}
	}
	ensure(t, muzzle, "light muzzle")
	for y := 64; y < 70; y++ {
		ensure(t, bytes.Equal(room[y*64:y*64+12], strip[(y-64)*12:(y-63)*12]), "cat missing before floor palette row %d", y)
	}
	for y := 75; y < 85; y++ {
		ensure(t, !bytes.Equal(room[y*64:y*64+12], strip[(y-64)*12:(y-63)*12]), "cat present in panorama row %d", y)
	}

	ensure(t, len(regexp.MustCompile(`repeat 2 \{ 0x4E `).FindAllString(body("dl_face"), -1)) == 95, "dl_face rows")
	has(body("dl_face"), "0xCE &<FaceImg+400 &>FaceImg+400")
	kernel := body("face_kernel")
	has(kernel, "a?28", "face_groups=a=56", "(audc_ptr),y=a", "COLPF1=y COLPF2=x", "a=0x08 COLPF1=a", "a=0x2C COLPF2=a", "return_i")
	ensure(t, strings.Count(kernel, "WSYNC=a") == 4, "face_kernel WSYNC count")
	notHas(kernel, "meow_ptr++")
	has(body("dl_storm"), "align 1024")
	has(body("dl_face"), "align 1024")
	has(body("build_storm_list"), "x=96", "a=0x4E", "a+64", "a+6", "dl_storm+3", "RevealImg+24")
	has(body("pan_step"), "x=96", "c+ a-1", "a+6", "c-?")
	has(body("storm_pan"), "x=1 wait_frames", "a?48")
	ensure(t, strings.Count(body("storm_pan"), "x=6 wait_frames") == 2, "storm_pan x=6 wait_frames")
	ensure(t, strings.Count(body("storm_pan"), "blink_rack_leds") == 2, "storm_pan blink_rack_leds")
	for y := 0; y < 96; y++ {
		for off := 0; off <= 24; off++ {
			ensure(t, (y*64+off)%4096 <= 4096-40, "ANTIC wrap y=%d off=%d", y, off)
		}
	}
	has(body("show_cat"), "x=32", "y?12", "RevealImg+4096", "a+64", "a+12")
	has(body("build_reveal_list"), "dl_storm+256,y", "dl_reveal+256,y", "y?70", "dl_reveal+420=a=0xCE")
	has(main, "naked reveal_dli", "COLPF0=a=0x72", "COLPF1=a=0x08", "COLPF2=a=0x2C", "VDSLST=a=&<reveal_dli")
	has(body("build_zoom_bitmap"), "RevealImg+961", "ZoomBuf+40", "zoom_rows=a=48", "zoom_col", "a?20", "y?40", "a+64", "a+80")
	has(body("build_zoom_list"), "ZoomBuf", "dl_reveal+3", "x=96", "a+40", "a+6")
	has(body("show_zoom_bitmap"), "x=16 wait_frames")
	has(body("Leds"), `binary "assets/leds.tbl"`)
	has(body("blink_rack_leds"), "pan_phase", "BitMasks,x", "Leds+121,x", "Leds+242,x", "Leds+363,x", "&>RevealImg")
	leds := readAsset("leds.tbl")
	ledCount := (len(leds) - 4) / 4
	ensure(t, ledCount >= 100 && leds[ledCount] == 0 && leds[2*ledCount+1] == 0xFF, "leds table header")
	for i := 0; i < ledCount; i++ {
		o := int(leds[ledCount+1+i])*256 + int(leds[i])
		mask := leds[2*ledCount+2+i]
		ensure(t, o < len(room) && room[o]&mask == mask, "LED %d not lit at start", i)
	}
	colors := map[byte]bool{}
	for i := 0; i < ledCount; i++ {
		colors[leds[3*ledCount+3+i]] = true
	}
	ensure(t, len(colors) > 4, "LED colour variety")

	for _, name := range []string{"pupils_stage1", "pupils_stage2", "pupils_stage3"} {
		changes := pokes(name)
		ensure(t, 1 <= len(changes) && len(changes) <= 16, "%s change count %d", name, len(changes))
		for o, v := range changes {
			ensure(t, 1900 <= o && o < 2300 && face[o] != 0 && v == 0, "%s poke %d=%d", name, o, v)
		}
	}
	opened, closed := pokes("mouth_open"), pokes("mouth_close")
	ensure(t, len(opened) == 8 && len(closed) == len(opened), "mouth poke counts")
	for o := range opened {
		c, ok := closed[o]
		ensure(t, ok && opened[o] == 0 && c == int(face[o]), "mouth poke %d", o)
	}

	pm := body("play_meow")
	branch := regexp.MustCompile(`(?s)a\?meow_page != \{(.*?)\}`).FindStringSubmatch(pm)
	if branch == nil {
		t.Fatal("meow_page branch missing")
	}
	has(branch[1], "mouth_tick", "a?&>Meow+3072 == break")
	ensure(t, strings.Index(pm, "audc_ptr=a=&<AUDC1") >= 0 && strings.Index(pm, "audc_ptr=a=&<AUDC1") < strings.Index(pm, "{"), "AUDC1 before loop")
	ensure(t, strings.LastIndex(pm, "audc_ptr=a=&<meow_sink") > strings.LastIndex(pm, "}"), "meow_sink after loop")
	ensure(t, strings.Count(pm, "WSYNC=a") == 3, "play_meow WSYNC count")
	has(body("mouth_tick"), "mouth_open", "mouth_close")
	has(body("switch_to_term"), "COLOR0=a=0x0E", "COLOR4=a=0xB4")
	z := body("zoom_to_terminal")
	has(z, "zoom_left=a=27", "zoom_left=a=13", "zoom_left=a=1", "zoom_rows=a=14", "zoom_rows=a=19", "zoom_rows=a=24", "clear_term_screen")
	ensure(t, strings.Count(z, "zoom_frame") == 3, "zoom_frame count")
	has(body("reveal_palette"), "COLOR0=a=0x72", "COLOR1=a=0xB6", "COLOR2=a=0x1A")
	has(body("switch_to_reveal"), "reveal_palette", "NMIEN=a=0xC0")
	has(body("switch_to_face"), "reveal_palette", "NMIEN=a=0xC0", "VDSLST=a=&<face_kernel", "audc_ptr=a=&<meow_sink")
	notHas(main, "thunder", "storm_tick")
	sp := body("storm_pan")
	has(sp, "AUDC2", "AUDC3")
	notHas(sp, "AUDC4", "COLOR")
	for _, ch := range []string{"2", "3", "4"} {
		has(body("cut_to_black"), "AUDC"+ch+"=a=0")
	}
	cm := body("call_mouth")
	cmNorm := strings.ReplaceAll(cm, "CallMouth,x", "CallMouth+0,x")
	for i, off := range []int{1896, 1897, 1936, 1937, 1976, 1977} {
		has(cm, fmt.Sprintf("CallImg+%d=a", off))
		has(cmNorm, fmt.Sprintf("CallMouth+%d,x", i))
	}
	has(body("video_call"), "switch_to_term", "TermScreen+107=a=14", "SDLSTL=a=&<dl_call",
		"COLOR2=a=0x2A", "TermScreen+840,y=a", "call_lines_left=a=6",
		"x=6 call_mouth", "x=0 call_mouth", "AUDC4=a=0",
		"x=152 wait_frames", "x=200 wait_frames", "x=50 wait_frames", "x=90 wait_frames")
	term := body("switch_to_term")
	ensure(t, mustIndex(t, term, "NMIEN=a=0x40") < mustIndex(t, term, "SDLSTL=a=&<dl_term"), "term NMIEN before SDLSTL")
	ig := body("init_gameplay")
	ensure(t, mustIndex(t, ig, "switch_to_term") < mustIndex(t, ig, "NMIEN=a=0xC0"), "init_gameplay order")

	sm := regexp.MustCompile(`(?s)main \{(.*)\}\s*$`).FindStringSubmatch(main)
	if sm == nil {
		t.Fatal("main not found")
	}
	scene := sm[1]
	beats := []string{"build_storm_list", "switch_to_storm", "storm_pan", "show_zoom_bitmap",
		"zoom_to_terminal", "reveal_line0", "show_y_prompt", "show_cat", "build_reveal_list",
		"switch_to_reveal", "switch_to_face", "pupils_stage1", "pupils_stage2", "pupils_stage3",
		"play_meow", "video_call", "cut_to_black"}
	idx := make([]int, len(beats))
	for i, b := range beats {
		idx[i] = mustIndex(t, scene, b)
	}
	ensure(t, sort.IntsAreSorted(idx), "intro beats out of order: %v", idx)
	has(body("wait_frames"), "check_intro_skip")
	has(body("check_intro_skip"), "a=cur_room", "a?0xFF", "input_pressed", "goto intro_skip")
	has(body("input_pressed"), "a=TRIG0", "a=CH", "a?0xFF")
	has(pm, "check_intro_skip")
	has(body("init_system"), "cur_room=a=0xFF", "CH=a=0xFF")
	skip := scene[mustIndex(t, scene, "intro_skip:"):mustIndex(t, scene, "init_gameplay")]
	has(skip, "NMIEN=a=0x40", "s=x=0xFF", "AUDC1=a=0", "cut_to_black")

	tail := scene[mustIndex(t, scene, "cut_to_black"):]
	has(tail, "init_gameplay")
	ensure(t, strings.Index(tail, "cut_to_black") < mustIndex(t, tail, "title_screen") && strings.Index(tail, "title_screen") < strings.Index(tail, "init_gameplay"), "tail order")
	has(tail[mustIndex(t, tail, "init_gameplay"):], "} always")
}
