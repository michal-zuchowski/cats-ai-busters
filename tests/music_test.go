package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/atr"
	"github.com/michal-zuchowski/cats-ai-busters/tools/music"
)

func musicTracks() [][]byte {
	tracks := [][]byte{music.Music()}
	for i := 0; i < 4; i++ {
		tracks = append(tracks, music.LevelMusic(i))
	}
	return tracks
}

func sumBytes(b []byte) int {
	n := 0
	for _, v := range b {
		n += int(v)
	}
	return n
}

// Per-track encoding: distinct scores, valid tables, tuning within two cents.
func TestMusicTrackEncoding(t *testing.T) {
	tracks := musicTracks()
	paths := []string{"music.bin"}
	for i := 1; i <= 4; i++ {
		paths = append(paths, fmt.Sprintf("music-level-%02d.bin", i))
	}
	h := sha256.Sum256(tracks[0][:432])
	ensure(t, hex.EncodeToString(h[:]) == "3a1f4f4b160b74845c26daceaca22c2f289d4ec3dd9c26865c0ccdbfe7652716",
		"title score hash changed: %x", h)
	whole, leads, sums := map[string]bool{}, map[string]bool{}, map[int]bool{}
	for _, tr := range tracks {
		whole[string(tr)], leads[string(tr[:256])], sums[sumBytes(tr)&255] = true, true, true
	}
	ensure(t, len(whole) == 5 && len(leads) == 5, "tracks/leads not all distinct")
	ensure(t, len(sums) == 5, "a stale whole song could pass a new song's checksum")

	for i, tr := range tracks {
		ensure(t, bytes.Equal(readAsset(paths[i]), tr) && len(tr) == 435, "%s differs from generated track", paths[i])
		ensure(t, 1 <= tr[432] && tr[432] < 16, "track %d tempo %d", i, tr[432])
		for _, v := range tr[433:435] {
			ensure(t, 0 < v && v <= 15, "track %d attack volume %d", i, v)
		}
		for _, v := range tr[256:272] {
			ensure(t, v <= 2, "track %d bass rhythm %d", i, v)
		}
		for _, v := range tr[272:304] {
			ensure(t, v <= 3, "track %d drum %d", i, v)
		}
		for _, v := range tr[304:336] {
			ensure(t, v > 1, "track %d bass note %d", i, v)
		}
		lead := music.Lead
		if i > 0 {
			lead = music.DoubledLead(i - 1)
		}
		tokens := strings.Fields(lead)
		notes := music.Notes(lead)
		if len(tokens) != 256 || len(notes) > 16 {
			t.Fatalf("track %d: %d tokens, %d notes", i, len(tokens), len(notes))
		}
		for n, token := range tokens {
			index := int(tr[n])
			expected := 0
			switch token {
			case "-":
			case ".":
				expected = 1
			default:
				for k, name := range notes {
					if name == token {
						expected = k + 2
					}
				}
			}
			if index != expected {
				t.Fatalf("track %d step %d: index %d, want %d", i, n, index, expected)
			}
			if index >= 2 {
				if index > 17 {
					t.Fatalf("track %d: note index %d > 17", i, index)
				}
				divisor := int(tr[334+index]) + 256*int(tr[350+index])
				ensure(t, divisor == music.Audf16(token), "track %d %s divisor %d", i, token, divisor)
				f := 1789790 / (2 * float64(divisor+7))
				midi := 69 + 12*math.Log2(f/440)
				ensure(t, math.Abs(midi-math.RoundToEven(midi))*100 < 2, "track %d %s off pitch by %.2f cents", i, token, math.Abs(midi-math.RoundToEven(midi))*100)
			}
		}
		for step := 0; step < 256; step++ {
			bar, beat := step>>4, step&15
			ensure(t, tr[304+bar] != 0 && tr[320+bar] != 0, "track %d bar %d bass root", i, bar)
			off := 0
			if bar&3 == 3 {
				off = 16
			}
			drum := int(tr[272+beat+off])
			if drum > 3 {
				t.Fatalf("track %d drum %d", i, drum)
			}
			for frame := 0; frame < 8; frame++ {
				if 400+drum*8+frame >= 432 {
					t.Fatalf("percussion fetch enters the tempo footer")
				}
			}
		}
	}
}

// Relative character of the four level scores.
func TestMusicLevelCharacter(t *testing.T) {
	tracks := musicTracks()
	stealth, machinery, chase, finale := tracks[1], tracks[2], tracks[3], tracks[4]
	onsets := func(tr []byte) int {
		n := 0
		for _, v := range tr[:256] {
			if v >= 2 {
				n++
			}
		}
		return n
	}
	nonzero := func(b []byte) int {
		n := 0
		for _, v := range b {
			if v != 0 {
				n++
			}
		}
		return n
	}
	maxVol := func(b []byte) byte {
		var m byte
		for _, v := range b {
			m = max(m, v&15)
		}
		return m
	}
	ensure(t, onsets(stealth) == 32 && 32 < onsets(finale) && onsets(finale) < onsets(machinery) && onsets(machinery) < onsets(chase),
		"onset ordering: %d %d %d %d", onsets(stealth), onsets(finale), onsets(machinery), onsets(chase))
	for i, want := range []byte{5, 8, 5, 4, 6} {
		ensure(t, tracks[i][432] == want, "track %d tempo %d, want %d", i, tracks[i][432], want)
	}
	ensure(t, stealth[433] < machinery[433] && machinery[433] < chase[433], "lead attack volumes not increasing")
	ensure(t, nonzero(stealth[272:304]) < nonzero(machinery[272:304]), "stealth drums should be sparser")
	ensure(t, maxVol(stealth[400:432]) < maxVol(chase[400:432]), "stealth drums should be quieter than chase")
	rests := 0
	for _, v := range finale[:256] {
		if v == 1 {
			rests++
		}
	}
	ensure(t, rests >= 64, "finale has %d rests", rests)
	ensure(t, strings.Contains(music.LevelLeads[3], "Eb4") && strings.Contains(music.LevelLeads[3], "C#4"),
		"level 4 lead lacks chromatic neighbours of D")
}

// RAM map and disk chunk layout of the scores.
func TestMusicRAMAndDisk(t *testing.T) {
	source := src()
	addr := func(n string) int { return varAddress(source, n) }
	ensure(t, addr("TermScreen")+25*40 <= addr("Music") && addr("Music") == 0xB400, "TermScreen/Music layout")
	ensure(t, addr("Music")+512 <= 0xB800 && 0xB800 < addr("PmMem"), "Music buffer overlaps PMG")
	ensure(t, addr("CryBuf") < addr("TermScreen") && addr("TermScreen") < addr("Music"), "CryBuf/TermScreen/Music order")
	others := regexp.MustCompile(`var (\w+)(?:\[\d+\])? = (0x[0-9A-Fa-f]+)`).FindAllStringSubmatch(source, -1)
	for _, name := range []string{"music_ready", "title_phase", "music_chunk", "disk_failed"} {
		a := addr(name)
		var sharing []string
		for _, m := range others {
			var v int
			fmt.Sscanf(m[2], "0x%x", &v)
			if v == a {
				sharing = append(sharing, m[1])
			}
		}
		ensure(t, len(sharing) == 1 && sharing[0] == name, "%s shares its address with %v", name, sharing)
	}
	for _, p := range []string{"music.bin", "music-level-01.bin", "music-level-02.bin", "music-level-03.bin", "music-level-04.bin"} {
		ensure(t, !strings.Contains(source, `binary "assets/`+p+`"`), "%s must not be linked", p)
	}

	image, _, table, err := atr.BuildATR(make([]byte, 128), atr.ExtraChunks())
	if err != nil {
		t.Fatal(err)
	}
	ensure(t, len(table) == 130 && bytes.Equal(table, readAsset("disk-chunks.bin")), "chunk table differs from assets/disk-chunks.bin")
	memory := bytes.Repeat([]byte{0xAA}, 0x10000)
	beforeScreen := append([]byte{}, memory[0xB000:0xB400]...)
	beforePCM := append([]byte{}, memory[0xA600:0xB000]...)
	beforePMG := append([]byte{}, memory[0xB800:0xC000]...)
	for i, track := range musicTracks() {
		e := (17 + i) * 5
		start := int(binary.LittleEndian.Uint16(table[e:]))
		count := int(binary.LittleEndian.Uint16(table[e+2:]))
		checksum := table[e+4]
		ensure(t, count == 4, "track %d chunk has %d sectors", i, count)
		disk := image[16+(start-1)*128 : 16+(start-1+count)*128]
		padded := append(append([]byte{}, track...), make([]byte, 512-len(track))...)
		ensure(t, bytes.Equal(disk, padded) && byte(sumBytes(disk)) == checksum, "track %d disk bytes/checksum", i)
		copy(memory[0xB400:0xB600], disk)
		ensure(t, bytes.Equal(memory[0xB400:0xB5B3], track), "track %d not in RAM buffer", i)
		ensure(t, bytes.Equal(memory[0xB000:0xB400], beforeScreen) && bytes.Equal(memory[0xA600:0xB000], beforePCM), "track %d load clobbered screen/PCM", i)
		ensure(t, bytes.Equal(memory[0xB800:0xC000], beforePMG) && bytes.Equal(memory[0xB600:0xB800], bytes.Repeat([]byte{0xAA}, 512)), "track %d load clobbered PMG/gap", i)
	}
}

// K65 source contracts of the audio/music code paths.
func TestMusicSourceContracts(t *testing.T) {
	source := src()
	body := func(n string) string { return funcBody(source, n) }
	has := func(s string, subs ...string) {
		t.Helper()
		for _, sub := range subs {
			ensure(t, strings.Contains(s, sub), "missing %q", sub)
		}
	}
	lt := func(s string, a ...string) {
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

	lt(body("tick_audio"), "sfx_timer--", "music_ready", "call tick_music")
	has(body("music_vbi_imm"), "level_finished")
	stop, start := body("stop_music"), body("start_music")
	lt(stop, "music_ready=a=0", "AUDC2=a")
	for n := 1; n <= 4; n++ {
		has(stop, fmt.Sprintf("AUDC%d=a", n))
	}
	has(stop, "sfx_timer=a")
	lt(start, "stop_music", "load_checked_chunk", "mus_timer=a=1")
	lt(start, "mus_timer=a=1", "music_ready=a=1")
	has(start, "&<Music", ">= { music_failure }")
	checked := body("load_checked_chunk")
	lt(checked, "load_room_chunk", "disk_failed", "disk_checksum")
	lt(checked, "disk_checksum", "a?ChunkTable+4,x")
	has(checked, "disk_ptr=a=disk_dest", "a=ChunkTable+2,x")
	for _, reset := range []string{"mus_step=a=0", "mus_bar=a", "lead_vol=a", "bass_vol=a", "bass_freq=a", "drum_type=a", "drum_t=a"} {
		has(start, reset)
	}
	has(body("disk_read"), "disk_failed=a=0", "y?1", "disk_failed=a=1")
	has(body("music_failure"), "stop_music", "MsgDiskErr", "{ x=1 wait_frames } always")
	step := body("music_step")
	has(step, "mus_timer=a=Music+432", "lead_vol=a=Music+433")
	ensure(t, strings.Count(step, "bass_vol=a=Music+434") == 2, "bass_vol=Music+434 count %d", strings.Count(step, "bass_vol=a=Music+434"))
	lt(body("tick_music"), "a=disk_busy", "AUDF3=a")
	lt(body("tick_music"), "a=sfx_timer", "AUDC4=a")
	has(body("init_gameplay"), "stop_music", "init_audio")
	lt(body("init_gameplay"), "cur_level=a=0", "level_music")
	lt(body("init_plat"), "stop_music", "enter_plat_room", "level_music")
	has(body("level_music"), "c- a+18")
	has(body("next_level"), "init_plat")
	has(body("dbg_level"), "init_gameplay", "init_plat")
	for _, name := range []string{"enter_room", "enter_plat_room", "respawn", "plat_respawn"} {
		if regexp.MustCompile(`func ` + name + ` \{`).MatchString(source) {
			b := body(name)
			ensure(t, !strings.Contains(b, "level_music") && !strings.Contains(b, "start_music"), "%s must not touch the score", name)
		}
	}
	has(body("final_scene"), "level_finished=a=1", "&<CryBuf")

	title := body("title_screen")
	lt(title, "cur_room=a=0", "wait_frames") // no recursive intro-skip branch
	has(title, "a=17\n  start_music", "title_phase=a=1", "title_phase=a=2", "title_phase=a=0", "a?0x0C", "a?0x21",
		"dl_title", "a=22", "a=23", "a=24")
	scene := source[mustIndex(t, source, "intro_skip:\n"):]
	lt(scene, "cut_to_black", "title_screen", "init_gameplay")
	mainAt := mustIndex(t, source, "main {")
	end := mainAt + mustIndex(t, source[mainAt:], "build_storm_list\n")
	ensure(t, !strings.Contains(source[mainAt:end], "goto intro_skip"), "main jumps to intro_skip before the storm")
}
