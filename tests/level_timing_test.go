package tests

import (
	"regexp"
	"strconv"
	"testing"

	L "github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
)

// Every floor crossing between safe spots (rack shadow, switch, relay) must
// leave a human-sized timing window: the cat walks 1 px per 3 frames and the
// cameras sweep per levelart.Cameras (same stepping as tick_cam).
func TestLevelTiming(t *testing.T) {
	art := readAsset("level-rooms.pic")
	const minWindow = 60
	m := regexp.MustCompile(`switch_timer=a=(\d+) call sfx_switch`).FindStringSubmatch(src())
	if m == nil {
		t.Fatal("switch_timer not found in source")
	}
	switchFrames, _ := strconv.Atoi(m[1])

	col := func(px int) int { return (px+2)/4 + 2 }
	trace := func(cfg L.Camera, frames int) []int {
		c, d, w := cfg.Lo, 0, cfg.Speed
		out := make([]int, 0, frames)
		for i := 0; i < frames; i++ {
			if w == 0 {
				w = cfg.Speed
				if d == 0 {
					c++
				} else {
					c--
				}
				if c == cfg.Hi {
					d, w = 1, cfg.Pause
				}
				if c == cfg.Lo {
					d, w = 0, cfg.Pause
				}
			} else {
				w--
			}
			out = append(out, c)
		}
		return out
	}
	type cam struct {
		tr    []int
		hw    int
		gated bool
	}

	for room := 0; room < 3; room++ {
		floor := art[room*960+880 : room*960+920]
		var cams []cam
		for _, k := range L.Cameras[room] {
			if k.Col != 0 {
				cams = append(cams, cam{trace(k, 4000), k.HalfWidth, k.Gate != 0})
			}
		}
		p := 0
		for p <= 140 {
			if floor[col(p)] != 70 {
				p++
				continue
			}
			a := p - 1
			for p <= 140 && floor[col(p)] == 70 {
				p++
			}
			steps := (p - a) * 3
			sw := room == 1 && floor[col(a)] == 82
			run, best := 0, 0
			for t0 := 1000; t0 < 3000; t0++ {
				ok := true
				for k := 0; k < steps && ok; k++ {
					for _, c := range cams {
						d := col(a+1+k/3) - c.tr[t0+k]
						if d < 0 {
							d = -d
						}
						if d <= c.hw && !(c.gated && sw && k < switchFrames) {
							ok = false
							break
						}
					}
				}
				if ok {
					run++
				} else {
					run = 0
				}
				best = max(best, run)
			}
			if best < minWindow {
				t.Errorf("room %d crossing cols %d..%d: window %d < %d", room, col(a), col(min(p, 140)), best, minWindow)
			}
		}
	}
}
