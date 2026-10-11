package tests

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
	"github.com/michal-zuchowski/cats-ai-busters/tools/meow"
)

// Meow PCM sample: size, dynamic range, play loop timing, source layout.
func TestMeowSample(t *testing.T) {
	sample := string(mustRead(k65src.Path("src", "meow_sample.k65")))
	main := k65src.Read("meow_sample.k65")
	project := string(mustRead("main.k65proj"))
	pcm := readAsset("meow.pcm")

	m := regexp.MustCompile(`a\?&>Meow\+(\d+) == break`).FindStringSubmatch(main)
	if m == nil {
		t.Fatal("no Meow page-break compare in the source")
	}
	n, _ := strconv.Atoi(m[1])
	if pages := n / 256; len(pcm) != pages*256 {
		t.Errorf("pcm is %d bytes, want %d pages", len(pcm), pages)
	}
	lo, hi := byte(255), byte(0)
	for _, v := range pcm {
		if v > 15 {
			t.Fatalf("sample value %d exceeds 15", v)
		}
		lo, hi = min(lo, v), max(hi, v)
	}
	if hi-lo < 12 {
		t.Errorf("dynamic range %d < 12", hi-lo)
	}
	wav, err := meow.ReadWAV(mustRead(k65src.Path("assets", "meow-source.wav")))
	if err != nil {
		t.Fatal(err)
	}
	if wav.Rate == 0 || float64(wav.Frames())/float64(wav.Rate) <= 1 {
		t.Errorf("source wav shorter than a second")
	}
	if !strings.Contains(sample, "align 256") {
		t.Errorf("sample must be page-aligned")
	}
	if !strings.Contains(sample, `binary "assets/meow.pcm"`) {
		t.Errorf("sample must embed meow.pcm")
	}
	body := regexp.MustCompile(`(?s)func play_meow \{(.*?)\n\}`).FindStringSubmatch(main)
	if body == nil {
		t.Fatal("no play_meow")
	}
	perSample := strings.Count(body[1], "WSYNC=a")
	if perSample != 3 { // face_kernel also plays one sample per 3 lines
		t.Errorf("scanlines per sample = %d, want 3", perSample)
	}
	if secs := float64(len(pcm)*perSample) / 15625; !(0.55 < secs && secs < 0.65) {
		t.Errorf("meow duration %.3fs outside 0.55..0.65", secs)
	}
	for _, s := range []string{"a|0x10", "AUDC1=a=0"} {
		if !strings.Contains(main, s) {
			t.Errorf("main source missing %q", s)
		}
	}
	if !strings.Contains(main, "meow_ptr+1++") || !strings.Contains(sample, "for x=0..63 eval [7]") {
		t.Errorf("kernel overrun pad missing")
	}
	if !strings.Contains(main, "play_meow") || !strings.Contains(project, "src/meow_sample.k65 main") {
		t.Errorf("play_meow / project entry missing")
	}
	if strings.Contains(sample, "func play_meow") {
		t.Errorf("play_meow must not live in the sample file")
	}
	if strings.Index(project, "src/meow_sample.k65 main") >= strings.Index(project, "src/main_scene.k65 main") {
		t.Errorf("meow_sample must precede main_scene in the project")
	}
}
