package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/atr"
	"github.com/michal-zuchowski/cats-ai-busters/tools/title"
)

// Approved title pixels, streamed DMA layout and skippable-intro source contracts.
func TestTitleScreen(t *testing.T) {
	source := src()
	body := func(n string) string { return funcBody(source, n) }
	has := func(s string, subs ...string) {
		t.Helper()
		for _, sub := range subs {
			ensure(t, strings.Contains(s, sub), "missing %q", sub)
		}
	}
	order := func(s string, a ...string) {
		t.Helper()
		prev := -1
		for _, sub := range a {
			i := strings.Index(s, sub)
			if i < 0 {
				t.Errorf("missing %q", sub)
				return
			}
			ensure(t, prev < i, "%q out of order", sub)
			prev = i
		}
	}

	originalRGB, err := title.TitleRGB()
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(originalRGB)
	ensure(t, hex.EncodeToString(h[:]) == "d8ae50f77ee9f9794df5cfd64896f59b9e86b31cac6c4e8527ca73e892a0ab0e", "title RGB hash %x", h)
	bitmap := readAsset("title-screen.pic")
	want, err := title.TitleBitmap()
	if err != nil {
		t.Fatal(err)
	}
	ensure(t, bytes.Equal(bitmap, want) && len(bitmap) == 188*40, "title-screen.pic differs")
	ensure(t, !strings.Contains(source, `binary "assets/title-screen.pic"`), "title-screen.pic linked")

	dl := body("dl_title")
	has(dl, "nocross", "0x70 0x70 0x70", "0x30", "0x41 &<dl_title &>dl_title")
	type region struct {
		name string
		rows int
	}
	regions := []region{{"ZoomBuf", 100}, {"TermScreen", 24}, {"TitleBottom", 64}}
	var loops []int
	for _, m := range regexp.MustCompile(`for x=0\.\.(\d+) eval \[0x0E\]`).FindAllStringSubmatch(dl, -1) {
		n, _ := strconv.Atoi(m[1])
		loops = append(loops, n+2)
	}
	ensure(t, len(loops) == 3 && loops[0] == 100 && loops[1] == 24 && loops[2] == 64, "dl loops %v", loops)
	total := 3
	for _, r := range regions {
		has(dl, "0x4E &<"+r.name+" &>"+r.name)
		total += 3 + r.rows - 1
	}
	ensure(t, total+1+3 <= 256, "display list too long")
	ensure(t, varAddress(source, "ZoomBuf") == 0xA000 && varAddress(source, "TermScreen") == 0xB000 && varAddress(source, "TitleBottom") == 0xB600, "region addresses")

	image, _, table, err := atr.BuildATR(make([]byte, 128), atr.ExtraChunks())
	if err != nil {
		t.Fatal(err)
	}
	ensure(t, len(atr.ExtraChunks()) == 26 && len(table) == 130 && bytes.Equal(table, readAsset("disk-chunks.bin")), "chunk table")
	memory := bytes.Repeat([]byte{0xAA}, 65536)
	music := append(readAsset("music.bin"), make([]byte, 512-len(readAsset("music.bin")))...)
	copy(memory[0xB400:], music)
	sectors := []int{32, 8, 20}
	for i, r := range regions {
		chunk := 22 + i
		e := chunk * 5
		start := int(binary.LittleEndian.Uint16(table[e:]))
		count := int(binary.LittleEndian.Uint16(table[e+2:]))
		checksum := table[e+4]
		ensure(t, count == sectors[i], "chunk %d count %d", chunk, count)
		data := image[16+(start-1)*128 : 16+(start-1+count)*128]
		ensure(t, len(data) == count*128 && byte(sumBytes(data)) == checksum, "chunk %d checksum", chunk)
		dest := varAddress(source, r.name)
		copy(memory[dest:], data)
		ensure(t, bytes.Equal(memory[0xB400:0xB600], music), "chunk %d title overwrote active score", chunk)
		ensure(t, memory[0x9FFF] == 0xAA && memory[0xC000] == 0xAA, "chunk %d spilled", chunk)
		for row := 0; row < r.rows; row++ {
			ensure(t, ((dest+row*40)&0xFFF)+40 <= 4096, "chunk %d row %d: ANTIC wraps", chunk, row)
		}
		ensure(t, sumBytes(data[r.rows*40:]) == 0, "chunk %d padding not zero", chunk)
	}
	var displayed []byte
	for _, r := range regions {
		a := varAddress(source, r.name)
		displayed = append(displayed, memory[a:a+r.rows*40]...)
	}
	ensure(t, bytes.Equal(displayed, bitmap), "displayed != bitmap")
	var rgb []byte
	for _, packed := range displayed {
		for _, shift := range []int{6, 4, 2, 0} {
			c := title.Palette[(packed>>shift)&3]
			rgb = append(rgb, c[:]...)
		}
	}
	for i := 0; i < 160*4; i++ {
		rgb = append(rgb, title.Palette[0][:]...)
	}
	ensure(t, bytes.Equal(rgb, originalRGB), "displayed pixels differ from original")

	scene := source[mustIndex(t, source, "main {"):]
	ensure(t, !strings.Contains(scene[:mustIndex(t, scene, "build_storm_list")], "goto intro_skip"), "goto intro_skip before storm")
	order(scene, "video_call", "intro_skip:", "title_screen", "init_gameplay")
	has(scene[mustIndex(t, scene, "intro_skip:"):], "s=x=0xFF")
	skip := body("check_intro_skip")
	order(skip, "a=cur_room", "input_pressed", "goto intro_skip")
	has(skip, "a?0xFF")
	has(body("wait_frames"), "check_intro_skip")
	has(body("play_meow"), "check_intro_skip")

	has(body("input_pressed"), "a=TRIG0", "a&1", "a=SKSTAT", "a&12", "a?12", "a=CONSOL", "a&7", "a?7",
		"a=CH", "a?0xFF", "!= { a=1 return }")
	pressed := func(trig, skstat, console, ch int) bool {
		return trig&1 == 0 || skstat&12 != 12 || console&7 != 7 || ch != 0xFF
	}
	ensure(t, !pressed(1, 0xFF, 7, 0xFF), "idle reads as pressed")
	ensure(t, pressed(0, 0xFF, 7, 0xFF) && pressed(1, 0xFB, 7, 0xFF) && pressed(1, 0xF7, 7, 0xFF), "trig/skstat")
	for k := 0; k < 255; k++ {
		ensure(t, pressed(1, 0xFF, 7, k), "key %d", k)
	}
	for k := 0; k < 7; k++ {
		ensure(t, pressed(1, 0xFF, k, 0xFF), "console %d", k)
	}

	ti := body("title_screen")
	order(ti, "SDMCTL=a=0", "DMACTL=a", "a=22")
	order(ti, "cur_room=a=0", "a=22")
	ensure(t, strings.Count(ti, "load_checked_chunk") == 3, "load_checked_chunk count")
	for i, name := range []string{"ZoomBuf", "TermScreen", "TitleBottom"} {
		ensure(t, regexp.MustCompile(`disk_dest=a=&<`+name+`\s+disk_dest\+1=a=&>`+name+`\s+a=`+strconv.Itoa(22+i)+`\s+load_checked_chunk`).MatchString(ti), "chunk load for %s", name)
	}
	order(ti, "a=24", "init_audio", "a=17", "SDLSTL")
	has(ti, "NMIEN=a=0x40", "GRACTL=a=0", "SDMCTL=a=0x22")
	release := ti[mustIndex(t, ti, "title_release:"):mustIndex(t, ti, "title_wait:")]
	order(release, "CH=a=0xFF", "wait_frames", "input_pressed")
	has(release, "!= { goto title_release }")
	has(ti[mustIndex(t, ti, "title_start:"):], "SDMCTL=a", "DMACTL=a", "music_ready=a")
	gp := body("init_gameplay")
	order(gp, "switch_to_term", "clear_term_screen", "SDMCTL=a=0x22")
	order(gp, "load_level_assets", "init_pmg")
	f := body("music_failure")
	order(f, "switch_to_term", "clear_term_screen", "draw_row0_msg")
	has(f, "SDMCTL=a=0x22", "MsgDiskErr")
}
