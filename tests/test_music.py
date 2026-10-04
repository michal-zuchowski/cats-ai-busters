"""Per-scene score, RAM/disk and source-contract checks; no K65 build or audio emulation."""
import hashlib
import math
import re
import struct
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
import make_atr as ATR
import make_music as M

source = (ROOT / "main.k65").read_text()


def body(name):
    match = re.search(r"(?:func|data|naked) " + name + r" \{(.*?)\n\}", source, re.S)
    assert match, name
    return match.group(1)


def address(name):
    return int(re.search(r"var " + name + r" = (0x[0-9A-Fa-f]+)", source).group(1), 16)


tracks = [M.music()] + [M.level_music(i) for i in range(4)]
paths = ["music.bin"] + ["music-level-%02d.bin" % i for i in range(1, 5)]
assert hashlib.sha256(tracks[0][:432]).hexdigest() == (
    "3a1f4f4b160b74845c26daceaca22c2f289d4ec3dd9c26865c0ccdbfe7652716")
assert len(set(tracks)) == 5 and len({t[:256] for t in tracks}) == 5
assert len({sum(t) & 255 for t in tracks}) == 5  # a stale whole song cannot pass a new song's checksum
for i, (track, name) in enumerate(zip(tracks, paths)):
    assert (ROOT / "assets" / name).read_bytes() == track and len(track) == 435
    assert 1 <= track[432] < 16 and all(0 < v <= 15 for v in track[433:435])
    assert set(track[256:272]) <= {0, 1, 2} and set(track[272:304]) <= {0, 1, 2, 3}
    assert all(v > 1 for v in track[304:336])
    tokens = M.LEAD.split() if not i else (M.LEVEL_LEADS[i - 1] * 2).split()
    notes = sorted({t for t in tokens if t not in "-."}, key=M.audf16, reverse=True)
    assert len(tokens) == 256 and len(notes) <= 16
    for token, index in zip(tokens, track[:256]):
        expected = 0 if token == "-" else 1 if token == "." else notes.index(token) + 2
        assert index == expected
        if index >= 2:
            assert index <= 17
            divisor = track[334 + index] + 256 * track[350 + index]
            assert divisor == M.audf16(token)
            f = 1789790 / (2 * (divisor + 7))
            midi = 69 + 12 * math.log2(f / 440)
            assert abs(midi - round(midi)) * 100 < 2  # two cents, not just a rounded note name
    for step in range(256):
        bar, beat = step >> 4, step & 15
        assert track[304 + bar] and track[320 + bar]
        drum = track[272 + beat + (16 if bar & 3 == 3 else 0)]
        assert 0 <= drum <= 3
        for frame in range(8):
            assert 400 + drum * 8 + frame < 432  # no percussion fetch enters the tempo footer

stealth, machinery, chase, finale = tracks[1:]
onsets = lambda t: sum(v >= 2 for v in t[:256])
assert onsets(stealth) == 32 < onsets(finale) < onsets(machinery) < onsets(chase)
assert tuple(t[432] for t in tracks) == (5, 8, 5, 4, 6)
assert stealth[433] < machinery[433] < chase[433]
assert sum(v != 0 for v in stealth[272:304]) < sum(v != 0 for v in machinery[272:304])
assert max(v & 15 for v in stealth[400:432]) < max(v & 15 for v in chase[400:432])
assert finale[:256].count(1) >= 64
assert "Eb4" in M.LEVEL_LEADS[3] and "C#4" in M.LEVEL_LEADS[3]  # tense chromatic neighbours of D

assert address("TermScreen") + 25 * 40 <= address("Music") == 0xB400
assert address("Music") + 512 <= 0xB800 < address("PmMem")
assert address("CryBuf") < address("TermScreen") < address("Music")
for name in ("music_ready", "title_phase", "music_chunk", "disk_failed"):
    addr = address(name)
    other = re.findall(r"var (\w+)(?:\[\d+\])? = (0x[0-9A-Fa-f]+)", source)
    assert [n for n, a in other if int(a, 16) == addr] == [name]
assert all('binary "assets/%s"' % p not in source for p in paths)

image, _, table = ATR.build_atr(bytes(128), ATR.EXTRA_CHUNKS)
assert len(table) == 130 and table == (ROOT / "assets/disk-chunks.bin").read_bytes()
memory = bytearray([0xAA] * 0x10000)
before_screen = memory[0xB000:0xB400]
before_pcm = memory[0xA600:0xB000]
before_pmg = memory[0xB800:0xC000]
for i, track in enumerate(tracks, 17):
    start, count, checksum = struct.unpack_from("<HHB", table, i * 5)
    assert count == 4
    disk = image[16 + (start - 1) * 128:16 + (start - 1 + count) * 128]
    assert disk == track.ljust(512, b"\0") and sum(disk) & 255 == checksum
    memory[0xB400:0xB600] = disk
    assert memory[0xB400:0xB5B3] == track
    assert memory[0xB000:0xB400] == before_screen and memory[0xA600:0xB000] == before_pcm
    assert memory[0xB800:0xC000] == before_pmg and memory[0xB600:0xB800] == bytes([0xAA] * 512)

audio = body("tick_audio")
assert audio.index("sfx_timer--") < audio.index("music_ready") < audio.index("call tick_music")
assert "level_finished" in body("music_vbi_imm")
stop, start = body("stop_music"), body("start_music")
assert stop.index("music_ready=a=0") < stop.index("AUDC2=a")
assert all("AUDC%d=a" % n in stop for n in range(1, 5)) and "sfx_timer=a" in stop
assert start.index("stop_music") < start.index("load_checked_chunk") < start.index("mus_timer=a=1")
assert start.index("mus_timer=a=1") < start.index("music_ready=a=1")
assert "&<Music" in start and ">= { music_failure }" in start
checked = body("load_checked_chunk")
assert checked.index("load_room_chunk") < checked.index("disk_failed") < checked.index("disk_checksum")
assert checked.index("disk_checksum") < checked.index("a?ChunkTable+4,x")
assert "disk_ptr=a=disk_dest" in checked and "a=ChunkTable+2,x" in checked
for reset in ("mus_step=a=0", "mus_bar=a", "lead_vol=a", "bass_vol=a", "bass_freq=a", "drum_type=a", "drum_t=a"):
    assert reset in start, reset
assert "disk_failed=a=0" in body("disk_read") and "y?1" in body("disk_read")
assert "disk_failed=a=1" in body("disk_read")
assert "stop_music" in body("music_failure") and "MsgDiskErr" in body("music_failure")
assert "{ x=1 wait_frames } always" in body("music_failure")
assert "mus_timer=a=Music+432" in body("music_step")
assert "lead_vol=a=Music+433" in body("music_step")
assert body("music_step").count("bass_vol=a=Music+434") == 2
assert body("tick_music").index("a=disk_busy") < body("tick_music").index("AUDF3=a")
assert body("tick_music").index("a=sfx_timer") < body("tick_music").index("AUDC4=a")
assert "stop_music" in body("init_gameplay") and "init_audio" in body("init_gameplay")
assert body("init_gameplay").index("cur_level=a=0") < body("init_gameplay").index("level_music")
assert body("init_plat").index("level_music") < body("init_plat").index("enter_plat_room")
assert "c- a+18" in body("level_music") and "init_plat" in body("next_level")
assert "init_gameplay" in body("dbg_level") and "init_plat" in body("dbg_level")
for name in ("enter_room", "enter_plat_room", "respawn", "plat_respawn"):
    match = re.search(r"func " + name + r" \{", source)
    if match:
        assert "level_music" not in body(name) and "start_music" not in body(name), name
assert "level_finished=a=1" in body("final_scene") and "&<CryBuf" in body("final_scene")

title = body("title_screen")
assert title.index("cur_room=a=0") < title.index("wait_frames")  # no recursive intro-skip branch
assert "a=17\n  start_music" in title and "title_phase=a=1" in title and "title_phase=a=2" in title
assert "title_phase=a=0" in title and "a?0x0C" in title and "a?0x21" in title
assert "dl_title" in title and "a=22" in title and "a=23" in title and "a=24" in title
scene = source[source.index("intro_skip:\n"):]
assert scene.index("cut_to_black") < scene.index("title_screen") < scene.index("init_gameplay")
assert "goto intro_skip" not in source[source.index("main {"):source.index("build_storm_list\n", source.index("main {"))]
print("test_music.py: all checks passed (original title and four distinct streamed scores)")
