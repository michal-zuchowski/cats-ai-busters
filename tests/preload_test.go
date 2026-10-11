package tests

import (
	"testing"

	L "github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
)

func TestLevel2PreloadStaging(t *testing.T) {
	pre := gpStripComments(gpBody("preload_level2"))
	for _, s := range []string{"&<StageRoom", "a=5\n  load_checked_chunk", "&<StageMusic", "a=19\n  load_checked_chunk", "preload_ready=a=1"} {
		gpHas(t, pre, s)
	}
	ensure(t, mustIndex(t, pre, "a=5") < mustIndex(t, pre, "a=19") && mustIndex(t, pre, "a=19") < mustIndex(t, pre, "preload_ready=a=1"), "chunk order, flag last")
	gpHasNot(t, pre, "disk_read")
	gpHas(t, src(), "var preload_ready = 0x06AA")
	gpHas(t, src(), "var StageRoom = 0xA800")
	gpHas(t, src(), "var StageMusic = 0xAC00")
	ensure(t, len(readAsset("music-level-02.bin")) <= 512, "staged track fits 512 B up to $ADFF")

	init := gpBody("init_gameplay")
	ensure(t, mustIndex(t, init, "preload_ready=a=0") < mustIndex(t, init, "disk_selftest"), "flag cleared before any load")
	guard := gpStripComments(gpBody("l01_preload"))
	gpHas(t, guard, "a?2")
	gpHas(t, guard, "call preload_level2")
	gpHas(t, gpBody("enter_room"), "call l01_preload")

	// cached fast path vs uncached fallback
	tm := gpStripComments(gpBody("take_staged_music"))
	gpHas(t, tm, "a?19")
	gpHas(t, tm, "preload_ready")
	gpHas(t, tm, "Music+256,x=a")
	sm := gpStripComments(gpBody("start_music"))
	ensure(t, mustIndex(t, sm, "call take_staged_music") < mustIndex(t, sm, "load_checked_chunk") &&
		mustIndex(t, sm, "load_checked_chunk") < mustIndex(t, sm, "a=Music+432") &&
		mustIndex(t, sm, "a=Music+432") < mustIndex(t, sm, "music_ready=a=1"), "validation/timers unchanged after staged-or-disk load")
	tr := gpStripComments(gpBody("take_staged_room"))
	gpHas(t, tr, "a?5")
	gpHas(t, tr, "RoomArtBuf+768,x=a") // 1024 B: tiles + metadata
	lr := gpStripComments(gpBody("load_room_art"))
	ensure(t, mustIndex(t, lr, "call take_staged_room") < mustIndex(t, lr, "call load_room_chunk") &&
		mustIndex(t, lr, "call load_room_chunk") < mustIndex(t, lr, "call paint_room_art"), "room: staged else disk, then paint")
	ip := gpBody("init_plat")
	ensure(t, mustIndex(t, ip, "stop_music") < mustIndex(t, ip, "level_finished=a=0") &&
		mustIndex(t, ip, "level_finished=a=0") < mustIndex(t, ip, "enter_plat_room") &&
		mustIndex(t, ip, "enter_plat_room") < mustIndex(t, ip, "level_music") &&
		mustIndex(t, ip, "level_music") < mustIndex(t, ip, "preload_ready=a=0"), "init_plat: silence, load room, start music only when ready, then clear stage")
	gpHas(t, gpBody("enter_plat_room"), "load_room_art")

	// no tall right wall in room 3; the hatch marker stays on the floor row
	r := L.Room(2)
	for y := 13; y < 22; y++ {
		ensure(t, r[y*40+38] != L.Wall && r[y*40+39] != L.Wall, "no wall at row %d", y)
	}
	ensure(t, r[22*40+38] == L.Wall, "hatch marker")
}
