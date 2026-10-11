// Package music generates the original title spy-funk and four level-specific
// POKEY scores (assets/music*.bin).
//
// 8-bit POKEY pure tones on the 64 kHz clock drift up to ~20 cents above C4,
// so the lead plays on channels 1+2 joined into one 16-bit voice clocked at
// 1.79 MHz (AUDCTL $50): AUDF = 1789790 / (2 * f) - 7. The bass stays 8-bit on
// ch3; ch4 plays table drums and yields to sound effects.
//
// Layout of a 435-byte track:
//
//	0 lead[256] (0 nothing, 1 cut, k >= 2: note k)  256 bass rhythm[16]
//	272 drums[2*16]  304 bass low[16]  320 bass high[16]
//	336 note AUDF1[16]  352 note AUDF2[16]  368 drum AUDF[4*8]  400 drum AUDC[4*8]
//	432 frames/step  433 lead attack volume  434 bass attack volume
package music

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

// N maps bass note names to their 8-bit AUDF values.
var N = map[string]int{"C3": 243, "D3": 217, "G3": 162, "Bb3": 136, "F3": 182, "A3": 144, "C4": 121, "C#4": 114, "D4": 108,
	"E4": 96, "F4": 91, "G4": 81, "A4": 72, "Bb4": 68, "C5": 60, "C#5": 57, "D5": 53,
	"E5": 47, "F5": 45, "G5": 40, "A5": 35}

const (
	Hold = 0
	Rest = 1
)

// Audf16 is the joined-channel divisor for a note name such as "C#4".
func Audf16(name string) int {
	step := map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}[name[0]]
	step += strings.Count(name, "#") - strings.Count(name, "b")
	octave := int(name[len(name)-1] - '0')
	midi := 12*(octave+1) + step
	return int(math.RoundToEven(1789790/(2*440*math.Pow(2, float64(midi-69)/12)) - 7))
}

// Lead is the title's 16-bar lead; "-" = nothing new, "." = cut.
const Lead = `
D4 - D4 F4 - A4 - D4 - - C5 - A4 - F4 -
F4 - F4 A4 - C5 - F4 - - A4 - C5 A4 - -
E4 - E4 G4 - C5 - E4 - - G4 - E4 - C4 -
C#4 - E4 - A4 - E4 - C#4 E4 A4 - G4 - E4 -
A4 - A4 - G4 A4 - F4 - G4 - E4 - F4 D4 -
C5 - C5 - A4 - F4 - A4 C5 - A4 - F4 - -
G4 - G4 E4 - C4 - E4 G4 - C5 - Bb4 - G4 -
A4 - - C#4 - E4 - A4 - G4 - E4 - C#4 - -
G4 - Bb4 - D5 - - . D5 C5 Bb4 - A4 - G4 -
F4 - Bb4 - D5 - F5 - - D5 - Bb4 - C5 - -
C5 - A4 - F4 - A4 C5 - F5 - E5 - C5 - -
E5 - C#5 - A4 - E5 - C#5 - A4 - G4 - E4 -
D5 - - D5 - C5 - A4 - - F4 - A4 - C5 -
A4 - - A4 - F4 - C5 - - A4 - F4 - E4 -
G4 - - G4 - E4 - C5 - - E5 - D5 - C5 -
C#5 - E5 - A4 - - E4 - C#4 - E4 - A4 . .
`

type Pair struct{ Lo, Hi string }

var (
	BassRhythm = []int{1, 0, 0, 2, 0, 0, 1, 0, 1, 0, 0, 2, 0, 0, 1, 2} // 1 root, 2 octave up
	Drums      = []int{1, 0, 3, 0, 2, 0, 3, 0, 1, 0, 1, 3, 2, 0, 3, 3, // kick, snare, hat
		1, 0, 1, 3, 2, 0, 2, 3, 1, 0, 2, 2, 2, 3, 2, 2} // fill: every 4th bar
	A     = []Pair{{"D3", "D4"}, {"F3", "F4"}, {"C3", "C4"}, {"A3", "A4"}}
	Roots = cat(A, A, []Pair{{"G3", "G4"}, {"Bb3", "Bb4"}, {"F3", "F4"}, {"A3", "A4"}}, A)
	DrumF = [4][8]int{
		{},
		{40, 70, 110, 160, 220, 250, 255, 255}, // kick: pitch drops fast
		{6, 6, 8, 8, 10, 10, 12, 12},
		{1, 1, 1, 1, 1, 1, 1, 1},
	}
	DrumC = [4][8]int{
		{},
		{0xAE, 0xAC, 0xAA, 0xA7, 0xA4, 0xA2, 0xA1, 0},
		{0x8C, 0x8A, 0x88, 0x86, 0x84, 0x82, 0x81, 0},
		{0x85, 0x83, 0x81, 0, 0, 0, 0, 0},
	}
)

func cat(ps ...[]Pair) []Pair {
	var out []Pair
	for _, p := range ps {
		out = append(out, p...)
	}
	return out
}

func rep(p Pair, n int) []Pair {
	out := make([]Pair, n)
	for i := range out {
		out[i] = p
	}
	return out
}

// Encode builds one 435-byte track.
func Encode(lead string, bass []int, drums []int, roots []Pair, settings [3]int, drumGain int) []byte {
	tokens := strings.Fields(lead)
	if len(tokens) != 256 || len(bass) != 16 || len(roots) != 16 || len(drums) != 32 {
		panic("music.Encode: bad track dimensions")
	}
	notes := Notes(lead)
	if len(notes) > 16 {
		panic("music.Encode: more than 16 distinct notes")
	}
	out := make([]byte, 0, 435)
	for _, t := range tokens {
		switch t {
		case "-":
			out = append(out, Hold)
		case ".":
			out = append(out, Rest)
		default:
			out = append(out, byte(indexOf(notes, t)+2))
		}
	}
	for _, v := range bass {
		out = append(out, byte(v))
	}
	for _, v := range drums {
		out = append(out, byte(v))
	}
	for _, r := range roots {
		out = append(out, byte(N[r.Lo]))
	}
	for _, r := range roots {
		out = append(out, byte(N[r.Hi]))
	}
	freq := make([]int, 16)
	for i, n := range notes {
		freq[i] = Audf16(n)
	}
	for _, f := range freq {
		out = append(out, byte(f&255))
	}
	for _, f := range freq {
		out = append(out, byte(f>>8))
	}
	for _, row := range DrumF {
		for _, v := range row {
			out = append(out, byte(v))
		}
	}
	for _, row := range DrumC {
		for _, v := range row {
			out = append(out, byte(v&0xF0|(v&15)*drumGain/15))
		}
	}
	return append(out, byte(settings[0]), byte(settings[1]), byte(settings[2]))
}

// Notes reproduces the distinct note list of a lead as Encode orders it (high AUDF first).
func Notes(lead string) []string {
	seen := map[string]bool{}
	var notes []string
	for _, t := range strings.Fields(lead) {
		if t != "-" && t != "." && !seen[t] {
			seen[t] = true
			notes = append(notes, t)
		}
	}
	sort.SliceStable(notes, func(i, j int) bool { return Audf16(notes[i]) > Audf16(notes[j]) })
	return notes
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// Music is the title track.
func Music() []byte { return Encode(Lead, BassRhythm, Drums, Roots, [3]int{5, 12, 12}, 15) }

// LevelLeads holds the eight-bar phrases (repeated twice by LevelMusic).
var LevelLeads = [4]string{
	`
    D4 - - . - - - - A4 - - . - - - -
    F4 - - . - - - - D4 - - - . - - -
    E4 - - . - - - - C4 - - . - - - -
    G4 - - - . - - - E4 - - . - - - -
    F4 - - . - - - - Bb4 - - . - - - -
    D4 - - . - - - - F4 - - - . - - -
    C#4 - - . - - - - E4 - - . - - - -
    A4 - - - . - - - C#4 - - . - - - -
    `,
	`
    D4 - A4 - D5 - A4 - F4 - A4 - D5 - A4 -
    D4 - F4 - A4 - F4 - D5 - A4 - F4 - A4 -
    C4 - G4 - C5 - G4 - E4 - G4 - C5 - G4 -
    C4 - E4 - G4 - E4 - C5 - G4 - E4 - G4 -
    Bb4 - F4 - D4 - F4 - Bb4 - F4 - D5 - F4 -
    F4 - C5 - A4 - C5 - F4 - A4 - C5 - A4 -
    A4 - E4 - C#5 - E4 - A4 - E4 - C#5 - E4 -
    A4 - C#5 - E5 - C#5 - A4 - E4 - C#4 . . -
    `,
	`
    D5 - A4 D5 F5 - E5 D5 C5 - A4 C5 D5 F5 E5 -
    E5 - C5 E5 G5 - E5 D5 C5 - G4 C5 E5 D5 C5 -
    D5 - Bb4 D5 F5 - D5 C5 Bb4 - F4 Bb4 D5 C5 Bb4 -
    C#5 - A4 C#5 E5 - C#5 A4 G4 - E4 G4 A4 C#5 E5 -
    D5 F5 - A4 D5 - F5 E5 D5 C5 - A4 F4 A4 D5 -
    F5 - C5 A4 F5 - E5 C5 A4 F4 - A4 C5 E5 F5 -
    G5 - D5 Bb4 G5 - F5 D5 Bb4 G4 - Bb4 D5 F5 G5 -
    E5 C#5 - A4 E5 - C#5 A4 G4 E4 - C#4 E4 A4 . .
    `,
	`
    D4 . - - Eb4 . - - D4 . - - C#4 - . -
    D4 - . - A4 - . - Eb4 - . - D4 . - -
    E4 . - - F4 . - - E4 . - - Eb4 - . -
    C4 - . - G4 - . - Eb4 - . - C4 . - -
    Bb4 . - - A4 . - - F4 . - - E4 - . -
    F4 - . - Eb4 - . - D4 - . - C#4 . - -
    C#4 . - - E4 . - - A4 . - - Bb4 - . -
    A4 - . - G4 - . - E4 - C#4 . D4 - . -
    `,
}

var (
	LevelRoots = [4][]Pair{
		cat(rep(Pair{"D3", "D4"}, 2), rep(Pair{"C3", "C4"}, 2), rep(Pair{"Bb3", "Bb4"}, 2), rep(Pair{"A3", "A4"}, 2)),
		cat(rep(Pair{"D3", "D4"}, 2), rep(Pair{"C3", "C4"}, 2), []Pair{{"Bb3", "Bb4"}, {"F3", "F4"}}, rep(Pair{"A3", "A4"}, 2)),
		cat(A[:1], []Pair{{"C3", "C4"}, {"Bb3", "Bb4"}, {"A3", "A4"}}, A[:2], []Pair{{"G3", "G4"}, {"A3", "A4"}}),
		cat(rep(Pair{"D3", "D4"}, 2), rep(Pair{"C3", "C4"}, 2), []Pair{{"Bb3", "Bb4"}, {"F3", "F4"}}, rep(Pair{"A3", "A4"}, 2)),
	}
	LevelBass = [4][]int{
		{1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0},
		{1, 0, 2, 0, 1, 0, 2, 0, 1, 0, 2, 0, 1, 0, 2, 0},
		{1, 0, 2, 1, 0, 2, 1, 0, 1, 2, 0, 1, 2, 0, 1, 2},
		{1, 0, 0, 0, 0, 0, 2, 0, 1, 0, 0, 0, 0, 0, 2, 0},
	}
	LevelDrums = [4][]int{
		{3, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0,
			3, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 3, 0},
		{1, 0, 3, 0, 2, 0, 3, 0, 1, 0, 3, 0, 2, 0, 3, 0,
			1, 0, 3, 0, 2, 0, 3, 0, 1, 3, 1, 3, 2, 0, 2, 3},
		Drums,
		{1, 0, 0, 0, 0, 0, 3, 0, 1, 0, 0, 0, 0, 0, 3, 0,
			1, 0, 0, 0, 0, 3, 0, 0, 1, 0, 3, 0, 2, 0, 3, 0},
	}
	LevelSettings = [4][3]int{{8, 6, 6}, {5, 9, 10}, {4, 12, 12}, {6, 8, 8}}
	levelGain     = [4]int{6, 10, 15, 8}
)

// DoubledLead is a level's lead as encoded (the eight-bar phrase twice).
func DoubledLead(level int) string { return LevelLeads[level] + "\n" + LevelLeads[level] }

// LevelMusic builds the score for level 0..3.
func LevelMusic(level int) []byte {
	if level < 0 || level >= 4 {
		panic("music.LevelMusic: level out of range")
	}
	return Encode(DoubledLead(level), LevelBass[level], LevelDrums[level],
		append(append([]Pair{}, LevelRoots[level]...), LevelRoots[level]...),
		LevelSettings[level], levelGain[level])
}

// Run writes assets/music.bin and assets/music-level-0N.bin.
func Run() error {
	data := Music()
	hold := 0
	for _, v := range data[:256] {
		if v == Hold {
			hold++
		}
	}
	if len(data) != 435 || hold >= 256 {
		return fmt.Errorf("unexpected title track")
	}
	for _, v := range data[304:336] {
		if v <= Rest {
			return fmt.Errorf("bass note missing")
		}
	}
	if math.Abs(1789790/(2*float64(Audf16("A4")+7))-440) >= 0.5 {
		return fmt.Errorf("A4 out of tune")
	}
	if err := writeFile(k65src.Path("assets", "music.bin"), data); err != nil {
		return err
	}
	for level := 0; level < 4; level++ {
		if err := writeFile(k65src.Path("assets", fmt.Sprintf("music-level-%02d.bin", level+1)), LevelMusic(level)); err != nil {
			return err
		}
	}
	return nil
}

func writeFile(path string, b []byte) error { return os.WriteFile(path, b, 0o644) }
