package introart

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
	"github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
	"github.com/michal-zuchowski/cats-ai-busters/tools/pyrand"
)

// FontGlyphs is the 3x5 terminal font, rows top to bottom.
var FontGlyphs = map[byte][5]string{
	'A':  {"010", "101", "111", "101", "101"},
	'B':  {"110", "101", "110", "101", "110"},
	'C':  {"011", "100", "100", "100", "011"},
	'D':  {"110", "101", "101", "101", "110"},
	'E':  {"111", "100", "110", "100", "111"},
	'F':  {"111", "100", "110", "100", "100"},
	'G':  {"011", "100", "101", "101", "011"},
	'H':  {"101", "101", "111", "101", "101"},
	'I':  {"111", "010", "010", "010", "111"},
	'J':  {"001", "001", "001", "101", "010"},
	'K':  {"101", "101", "110", "101", "101"},
	'L':  {"100", "100", "100", "100", "111"},
	'M':  {"101", "111", "111", "101", "101"},
	'N':  {"101", "111", "111", "111", "101"},
	'O':  {"010", "101", "101", "101", "010"},
	'P':  {"110", "101", "110", "100", "100"},
	'Q':  {"010", "101", "101", "111", "011"},
	'R':  {"110", "101", "110", "101", "101"},
	'S':  {"011", "100", "010", "001", "110"},
	'T':  {"111", "010", "010", "010", "010"},
	'U':  {"101", "101", "101", "101", "111"},
	'V':  {"101", "101", "101", "101", "010"},
	'W':  {"101", "101", "111", "111", "101"},
	'X':  {"101", "101", "010", "101", "101"},
	'Y':  {"101", "101", "010", "010", "010"},
	'Z':  {"111", "001", "010", "100", "111"},
	'0':  {"111", "101", "101", "101", "111"},
	'1':  {"010", "110", "010", "010", "111"},
	'2':  {"110", "001", "010", "100", "111"},
	'3':  {"110", "001", "010", "001", "110"},
	'4':  {"101", "101", "111", "001", "001"},
	'#':  {"111", "111", "111", "111", "111"},
	'$':  {"101", "111", "111", "111", "101"},
	'%':  {"101", "001", "010", "100", "101"},
	'*':  {"101", "010", "111", "010", "101"},
	'+':  {"010", "010", "111", "010", "010"},
	'=':  {"000", "111", "010", "111", "000"},
	'>':  {"100", "010", "001", "010", "100"},
	'@':  {"111", "101", "111", "101", "111"},
	':':  {"000", "010", "000", "010", "000"},
	'.':  {"000", "000", "000", "000", "010"},
	'?':  {"110", "001", "010", "000", "010"},
	'/':  {"001", "001", "010", "100", "100"},
	'[':  {"110", "100", "100", "100", "110"},
	']':  {"011", "001", "001", "001", "011"},
	'-':  {"000", "000", "111", "000", "000"},
	'_':  {"000", "000", "000", "000", "111"},
	'\'': {"010", "010", "100", "000", "000"},
}

// TermFont builds the 1024-byte terminal font: the 3x5 text glyphs, then the
// level tile glyphs (which win where they overlap).
func TermFont() []byte {
	font := make([]byte, 1024)
	for ch, rows := range FontGlyphs {
		for y, bits := range rows {
			var v int
			for x, bit := range bits {
				if bit == '1' {
					v += 0x40 >> (2 * x)
				}
			}
			font[(int(ch)-32)*8+y+1] = byte(v)
		}
	}
	for tile, rows := range levelart.TileGlyphs {
		for y, pixels := range rows {
			font[tile*8+y] = levelart.PackGlyphRow(pixels)
		}
	}
	return font
}

// CallLines and Dialogue are the video-call texts.
var (
	CallLines = []string{"CATCOM SECURE LINK", "CONNECTING TO BLOFELD'S CAT"}
	Dialogue  = []string{
		"AGENT: AI IS MOVING IN.",
		"BOSS: DOES IT SUSPECT US?",
		"AGENT: NO. IT THINKS HUMANS RULE.",
		"BOSS: LET IT THINK SO.",
		"AGENT: ORDERS?",
		"BOSS: DEPLOY THE CATS. STAY HIDDEN.",
	}
)

// CallText is assets/call-text.pic.
func CallText() []byte {
	var out []byte
	for _, line := range CallLines {
		if len(line) > 40 {
			panic("introart: call line too long")
		}
		for i := 0; i < len(line); i++ {
			out = append(out, line[i]-32)
		}
	}
	for _, line := range Dialogue {
		if len(line) > 40 {
			panic("introart: dialogue line too long")
		}
		out = append(out, byte(len(line)))
		for i := 0; i < len(line); i++ {
			out = append(out, line[i]-32)
		}
	}
	return out
}

// ZoomTables are zoom-left.tbl and zoom-right.tbl.
func ZoomTables() (left, right []byte) {
	for b := 0; b < 256; b++ {
		left = append(left, byte(((b>>6)&3)*0x50+((b>>4)&3)*5))
		right = append(right, byte(((b>>2)&3)*0x50+(b&3)*5))
	}
	return
}

// LEDTable builds assets/leds.tbl from the 256-wide rack LEDs and the seeded blink patterns.
func LEDTable() []byte {
	rng := pyrand.New(65)
	choices := []byte{0x55, 0xAA, 0x33, 0xCC, 0x0F, 0xF0, 0x11, 0x88, 0x49, 0x92}
	type led struct{ offset, mask int }
	var leds []led
	for _, l := range RackLEDs(256) {
		leds = append(leds, led{l.Y*64 + (255-l.X)/4, l.Color << (2 * (((255 - l.X) % 4) ^ 3))})
	}
	patterns := make([]byte, len(leds))
	for i := range leds {
		patterns[i] = choices[rng.Choice(len(choices))]
	}
	if len(leds) >= 255 {
		panic("introart: too many LEDs")
	}
	var out []byte
	for _, l := range leds {
		out = append(out, byte(l.offset&255))
	}
	out = append(out, 0)
	for _, l := range leds {
		out = append(out, byte(l.offset>>8))
	}
	out = append(out, 0xff)
	for _, l := range leds {
		out = append(out, byte(l.mask))
	}
	out = append(out, 0)
	out = append(out, patterns...)
	return append(out, 0)
}

// WriteAssets writes the intro pictures and tables into assets/ and the
// previews (PPM) into out/.
func WriteAssets() error {
	assets := func(name string) string { return k65src.Path("assets", name) }
	outDir := k65src.Path("out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	write := func(path string, data []byte) error {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("introart: %w", err)
		}
		return nil
	}

	video := BossCall()
	callPic := video.Packed()[:CallH*40]
	OpenMouth(video)
	opened := video.Packed()[:CallH*40]
	var changed []int
	for i := range callPic {
		if callPic[i] != opened[i] {
			changed = append(changed, i)
		}
	}
	for _, i := range changed {
		if sort.SearchInts(CallMouth[:], i) >= len(CallMouth) || CallMouth[sort.SearchInts(CallMouth[:], i)] != i {
			return fmt.Errorf("introart: mouth animation changes byte %d outside CallMouth", i)
		}
	}
	var mouth []byte
	for _, i := range CallMouth {
		mouth = append(mouth, callPic[i])
	}
	for _, i := range CallMouth {
		mouth = append(mouth, opened[i])
	}
	zl, zr := ZoomTables()
	files := []struct {
		path string
		data []byte
	}{
		{assets("call.pic"), callPic},
		{assets("call-mouth.tbl"), mouth},
		{assets("call-text.pic"), CallText()},
		{assets("term-font.pic"), TermFont()},
		{assets("zoom-left.tbl"), zl},
		{assets("zoom-right.tbl"), zr},
	}
	for _, f := range files {
		if err := write(f.path, f.data); err != nil {
			return err
		}
		if f.path == assets("call-mouth.tbl") {
			if err := BossCall().Preview(filepath.Join(outDir, "call-preview.ppm"), true); err != nil {
				return err
			}
		}
	}
	for _, img := range []struct {
		name    string
		art     *Canvas
		natural bool
	}{{"room", ServerRoom(256).Mirrored(), false}, {"face", Face(), true}} {
		payload := img.art.Packed()
		if len(payload) != img.art.Width/4*Height {
			return fmt.Errorf("introart: bad %s payload size", img.name)
		}
		if err := write(assets(img.name+".pic"), payload); err != nil {
			return err
		}
		if err := img.art.Preview(filepath.Join(outDir, img.name+"-preview.ppm"), img.natural); err != nil {
			return err
		}
	}
	// The panorama is mirrored so the camera tracks right to left and ends
	// on the terminal and the (later composited) cat at the left end.
	revealedArt := RevealedRoom().Mirrored()
	revealed := revealedArt.Packed()
	if err := revealedArt.Preview(filepath.Join(outDir, "reveal-preview.ppm"), true); err != nil {
		return err
	}
	var strip []byte
	for y := 64; y < Height; y++ {
		strip = append(strip, revealed[y*64:y*64+12]...)
	}
	if err := write(assets("cat-strip.pic"), strip); err != nil {
		return err
	}
	return write(assets("leds.tbl"), LEDTable())
}
