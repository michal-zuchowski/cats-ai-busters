package tests

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Source-level checks for the nine cat lives and the game-over restart.
func TestLivesSource(t *testing.T) {
	s := src()
	lives := varAddress(s, "lives")
	ensure(t, lives >= 0x0600 && lives < 0x0700, "lives outside page 6: %#x", lives)
	re := regexp.MustCompile(`var (\w+)(?:\[\d\])? *= *(0x[0-9A-Fa-f]+)`)
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		if m[1] != "lives" && strings.EqualFold(m[2], "0x06A9") {
			t.Errorf("lives address shared with %s", m[1])
		}
	}

	scene := regexp.MustCompile(`(?s)main \{(.*?)\n\}`).FindStringSubmatch(s)[1]
	tail := scene[mustIndex(t, scene, "title_screen"):]
	ensure(t, mustIndex(t, tail, "lives=a=9") < mustIndex(t, tail, "init_gameplay"), "lives=9 before init_gameplay")

	alert := funcBodyNorm(s, "trigger_alert")
	ensure(t, mustIndex(t, alert, "a?0 != { return }") < mustIndex(t, alert, "lose_life"), "lose_life after alert early return")
	ensure(t, mustIndex(t, alert, "lose_life") < mustIndex(t, alert, "alert_timer=a=90"), "lose_life before alert starts")
	ensure(t, strings.Contains(funcBodyNorm(s, "respawn"), "lose_life"), "respawn loses a life")

	lose := funcBodyNorm(s, "lose_life")
	for _, want := range []string{"lives--", "game_over", "draw_lives"} {
		ensure(t, strings.Contains(lose, want), "lose_life lacks %s", want)
	}
	over := funcBodyNorm(s, "game_over")
	for _, want := range []string{"stop_music", "MsgGameOver", "wait_frames", "goto intro_skip"} {
		ensure(t, strings.Contains(over, want), "game_over lacks %s", want)
	}
	ensure(t, strings.Contains(funcBodyNorm(s, "draw_lives"), "TermScreen+970=a"), "draw_lives digit cell")
	ensure(t, strings.Contains(funcBodyNorm(s, "draw_hud_static"), "draw_lives"), "draw_hud_static draws lives")
	// 'GAME OVER' in ascii-32 encoding
	ensure(t, strings.Contains(s, "39 33 45 37 0 47 54 37 50"), "MsgGameOver bytes")
	// intro_skip restart path resets stack, then title -> new game
	skip := scene[mustIndex(t, scene, "intro_skip:"):]
	ensure(t, mustIndex(t, skip, "s=x=0xFF") < mustIndex(t, skip, "title_screen"), "restart resets stack before title")
	ensure(t, strings.Contains(funcBodyNorm(s, "dbg_level"), "lives=a=9"), "debug keys refill lives")
}

// draw_lives copies its glyphs into TermFont slots 29 ('=') and 32 ('@'),
// because the font's '5'..'9' slots hold level tiles.
func TestLivesGlyphs(t *testing.T) {
	s := src()
	var digits []int
	for _, f := range strings.Fields(lineComment.ReplaceAllString(funcBody(s, "LifeDigits"), "")) {
		v, err := strconv.ParseInt(f, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		digits = append(digits, int(v))
	}
	if len(digits) != 80 {
		t.Fatalf("LifeDigits has %d bytes, want 80", len(digits))
	}
	font := readAsset("term-font.pic")
	for d := 0; d <= 4; d++ { // intact font digits must match the table
		for r := 0; r < 8; r++ {
			ensure(t, int(font[(16+d)*8+r]) == digits[d*8+r], "digit %d row %d differs from font", d, r)
		}
	}
	body := funcBodyNorm(s, "draw_lives")
	for _, want := range []string{"TermFont+232,y=a", "TermFont+256,y=a", "TermScreen+969=a=29", "TermScreen+970=a=32"} {
		ensure(t, strings.Contains(body, want), "draw_lives lacks %s", want)
	}
	text := regexp.MustCompile(`(?s)data (?:Msg|Line)\w* \{(.*?)\n\}`)
	for _, m := range text.FindAllStringSubmatch(s, -1) {
		for _, f := range strings.Fields(lineComment.ReplaceAllString(m[1], "")) {
			ensure(t, f != "29" && f != "32", "text uses glyph %s reserved for lives", f)
		}
	}
}
