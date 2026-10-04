"""Approved title pixels, streamed DMA layout and skippable-intro source contracts."""
import hashlib
import re
import struct
import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
import make_atr as ATR
import make_title as TITLE

source = (ROOT / "main.k65").read_text()


def body(name):
    match = re.search(r"(?:func|data) " + name + r" \{(.*?)\n\}", source, re.S)
    assert match, name
    return match.group(1)


def address(name):
    return int(re.search(r"var " + name + r" = (0x[0-9A-Fa-f]+)", source).group(1), 16)


with Image.open(ROOT / "assets/title-screen.png") as original:
    original_rgb = original.convert("RGB").tobytes()
assert hashlib.sha256(original_rgb).hexdigest() == (
    "d8ae50f77ee9f9794df5cfd64896f59b9e86b31cac6c4e8527ca73e892a0ab0e")
bitmap = (ROOT / "assets/title-screen.pic").read_bytes()
assert bitmap == TITLE.title_bitmap() and len(bitmap) == 188 * 40
assert 'binary "assets/title-screen.pic"' not in source

dl = body("dl_title")
assert "nocross" in dl and "0x70 0x70 0x70" in dl
regions = [("ZoomBuf", 100), ("TermScreen", 24), ("TitleBottom", 64)]
loops = [int(n) + 2 for n in re.findall(r"for x=0\.\.(\d+) eval \[0x0E\]", dl)]
assert loops == [rows for _, rows in regions]
assert all("0x4E &<%s &>%s" % (name, name) in dl for name, _ in regions)
assert "0x30" in dl and "0x41 &<dl_title &>dl_title" in dl
assert 3 + sum(3 + rows - 1 for _, rows in regions) + 1 + 3 <= 256
assert address("ZoomBuf") == 0xA000 and address("TermScreen") == 0xB000
assert address("TitleBottom") == 0xB600

image, _, table = ATR.build_atr(bytes(128), ATR.EXTRA_CHUNKS)
assert len(ATR.EXTRA_CHUNKS) == 26 and len(table) == 130
assert table == (ROOT / "assets/disk-chunks.bin").read_bytes()
memory = bytearray([0xAA] * 65536)
music = (ROOT / "assets/music.bin").read_bytes().ljust(512, b"\0")
memory[0xB400:0xB600] = music
for chunk, (name, rows), sectors in zip(range(22, 25), regions, (32, 8, 20)):
    start, count, checksum = struct.unpack_from("<HHB", table, chunk * 5)
    assert count == sectors
    data = image[16 + (start - 1) * 128:16 + (start - 1 + count) * 128]
    assert len(data) == count * 128 and sum(data) & 255 == checksum
    dest = address(name)
    memory[dest:dest + len(data)] = data
    assert memory[0xB400:0xB600] == music, (chunk, "title overwrote active score")
    assert memory[0x9FFF] == memory[0xC000] == 0xAA
    for row in range(rows):
        ptr = dest + row * 40
        assert (ptr & 0xFFF) + 40 <= 4096, (chunk, row, "ANTIC wraps this scanline")
    assert not any(data[rows * 40:])  # sector padding cannot draw into the next DMA region

displayed = b"".join(bytes(memory[address(name):address(name) + rows * 40]) for name, rows in regions)
assert displayed == bitmap
rgb = bytearray()
for packed in displayed:
    for shift in (6, 4, 2, 0):
        rgb.extend(TITLE.PALETTE[(packed >> shift) & 3])
rgb.extend(bytes(TITLE.PALETTE[0]) * 160 * 4)
assert bytes(rgb) == original_rgb  # every displayed pixel including the blank bottom is preserved

scene = source[source.index("main {"):]
assert "goto intro_skip" not in scene[:scene.index("build_storm_list")]
assert scene.index("video_call") < scene.index("intro_skip:") < scene.index("title_screen") < scene.index("init_gameplay")
assert "s=x=0xFF" in scene[scene.index("intro_skip:"):]
skip = body("check_intro_skip")
assert skip.index("a=cur_room") < skip.index("input_pressed") < skip.index("goto intro_skip")
assert "a?0xFF" in skip and "check_intro_skip" in body("wait_frames")
assert "check_intro_skip" in body("play_meow")  # PCM is skippable too

pressed = body("input_pressed")
for token in ("a=TRIG0", "a&1", "a=SKSTAT", "a&12", "a?12", "a=CONSOL", "a&7", "a?7",
              "a=CH", "a?0xFF", "!= { a=1 return }"):
    assert token in pressed, token


def input_pressed(trig=1, skstat=0xFF, console=7, ch=0xFF):
    return not (trig & 1) or (skstat & 12) != 12 or (console & 7) != 7 or ch != 0xFF


assert not input_pressed()
assert input_pressed(trig=0) and input_pressed(skstat=0xFB) and input_pressed(skstat=0xF7)
assert all(input_pressed(ch=k) for k in range(255))  # not just Space and Return
assert all(input_pressed(console=k) for k in range(7))

title = body("title_screen")
assert title.index("SDMCTL=a=0") < title.index("DMACTL=a") < title.index("a=22")
assert title.index("cur_room=a=0") < title.index("a=22")
assert title.count("load_checked_chunk") == 3
for chunk, name in zip(range(22, 25), ("ZoomBuf", "TermScreen", "TitleBottom")):
    assert re.search(r"disk_dest=a=&<" + name + r"\s+disk_dest\+1=a=&>" + name +
                     r"\s+a=" + str(chunk) + r"\s+load_checked_chunk", title)
assert title.index("a=24") < title.index("init_audio") < title.index("a=17") < title.index("SDLSTL")
assert "NMIEN=a=0x40" in title and "GRACTL=a=0" in title and "SDMCTL=a=0x22" in title
release = title[title.index("title_release:"):title.index("title_wait:")]
assert release.index("CH=a=0xFF") < release.index("wait_frames") < release.index("input_pressed")
assert "!= { goto title_release }" in release
leave = title[title.index("title_start:"):]
assert "SDMCTL=a" in leave and "DMACTL=a" in leave and "music_ready=a" in leave
gameplay = body("init_gameplay")
assert gameplay.index("switch_to_term") < gameplay.index("clear_term_screen") < gameplay.index("SDMCTL=a=0x22")
assert gameplay.index("load_level_assets") < gameplay.index("init_pmg")
failure = body("music_failure")
assert failure.index("switch_to_term") < failure.index("clear_term_screen") < failure.index("draw_row0_msg")
assert "SDMCTL=a=0x22" in failure and "MsgDiskErr" in failure
print("test_title_screen.py: all checks passed (exact mockup pixels, safe DMA/music, intro/title transitions)")
