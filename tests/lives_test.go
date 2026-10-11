package tests

import (
	"regexp"
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
	livesBody := funcBodyNorm(s, "draw_lives")
	for _, want := range []string{"TermFont+232,y=a", "x?lives < { a=157 }", "TermScreen+979,x=a", "x?9"} {
		ensure(t, strings.Contains(livesBody, want), "draw_lives lacks %s", want)
	}
	ensure(t, strings.Contains(funcBody(s, "LifeHeart"), "0x00 0xCC 0xFC 0xFC 0x30"), "heart glyph uses colour 3")
	ensure(t, !strings.Contains(s, "hud_bar"), "progress bar replaced by hearts")
	// glyph slot 29 (=) must not appear in any text
	for _, m := range regexp.MustCompile(`(?s)data (?:Msg|Line)\w* \{(.*?)\n\}`).FindAllStringSubmatch(s, -1) {
		for _, f := range strings.Fields(lineComment.ReplaceAllString(m[1], "")) {
			ensure(t, f != "29", "text uses glyph 29 reserved for the heart")
		}
	}
	ensure(t, strings.Contains(funcBodyNorm(s, "draw_hud_static"), "draw_lives"), "draw_hud_static draws lives")
	// 'GAME OVER' in ascii-32 encoding
	ensure(t, strings.Contains(s, "39 33 45 37 0 47 54 37 50"), "MsgGameOver bytes")
	// intro_skip restart path resets stack, then title -> new game
	skip := scene[mustIndex(t, scene, "intro_skip:"):]
	ensure(t, mustIndex(t, skip, "s=x=0xFF") < mustIndex(t, skip, "title_screen"), "restart resets stack before title")
	ensure(t, strings.Contains(funcBodyNorm(s, "dbg_level"), "lives=a=9"), "debug keys refill lives")
}
