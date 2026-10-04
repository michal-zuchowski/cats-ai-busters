"""Cat-atlas disk/overlay checks using synthetic payloads, without compiling K65."""
import re
import struct
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
import make_atr as M

source = (ROOT / "main.k65").read_text()
atlas = (ROOT / "assets/cat-sprites.bin").read_bytes()
panorama = (ROOT / "assets/room.pic").read_bytes()
mapping = (ROOT / "assets/cat-frame-map.bin").read_bytes()


def body(name):
    return re.search(r"(?:data|func) " + name + r" \{(.*?)\n\}", source, re.S).group(1)


assert 'binary "assets/cat-sprites.bin"' not in source and "data CatSprites {" not in source
assert "align 4096" in body("RevealImg") and 'binary "assets/room.pic"' in body("RevealImg")
assert len(atlas) == 5504 <= len(panorama) == 6144
assert "&>RevealImg" in body("draw_frame") and "a=FrameMap,x" in body("draw_frame")
assert re.search(r"disk_dest=a=&<RevealImg\s+disk_dest\+1=a=&>RevealImg\s+a=16\s+load_room_chunk",
                 body("load_level_assets"))
init = body("init_gameplay")
assert init.index("switch_to_term") < init.index("load_level_assets") < init.index("init_pmg")
assert source.index("intro_skip:\n") < source.index("  init_gameplay\n", source.index("intro_skip:\n"))
assert M.EXTRA_CHUNKS[15][0] == "death-cry" and M.EXTRA_CHUNKS[16][0] == "cat-atlas"
assert len(M.EXTRA_CHUNKS) == 26
assert "-lowAddr 0x1000" in (ROOT / "main.k65proj").read_text()
assert "-hiaddr 0x9FFF" in (ROOT / "main.k65proj").read_text()
assert M.FIRST_CHUNK_SECTOR == 290

# Program size changes must not invalidate the table embedded by the compiler.
image, _, table = M.build_atr(bytes(128), M.EXTRA_CHUNKS)
assert table == M.build_atr(bytes(1024), M.EXTRA_CHUNKS)[2]
assert table == M.build_atr(bytes(M.RESIDENT_CAPACITY), M.EXTRA_CHUNKS)[2]
assert (ROOT / "assets/disk-chunks.bin").read_bytes() == table
try:
    M.build_atr(bytes(M.RESIDENT_CAPACITY + 1), M.EXTRA_CHUNKS)
except SystemExit as exc:
    assert "reserved" in str(exc)
else:
    raise AssertionError("oversized resident program accepted")

for i, entry in enumerate(M.EXTRA_CHUNKS):
    data = entry[1].read_bytes()
    if len(entry) == 4:
        data = data[entry[2]:entry[2] + entry[3]]
    start, count, checksum = struct.unpack_from("<HHB", table, i * 5)
    padded = data.ljust(count * M.SECTOR_SIZE, b"\0")
    disk = image[16 + (start - 1) * M.SECTOR_SIZE:16 + (start - 1 + count) * M.SECTOR_SIZE]
    assert disk == padded and checksum == sum(padded) & 255, entry[0]

start, count, _ = struct.unpack_from("<HHB", table, 16 * 5)
assert count == 43
streamed = image[16 + (start - 1) * 128:16 + (start - 1 + count) * 128]
memory = bytearray(b"\xAA" + panorama + b"\xBB")
memory[1:1 + len(streamed)] = streamed
assert memory[0] == 0xAA and memory[-1] == 0xBB
assert memory[1 + len(atlas):-1] == panorama[len(atlas):]
for physical in mapping:
    offset = physical * 128
    assert memory[1 + offset:1 + offset + 128] == atlas[offset:offset + 128]

assert len(mapping) == 58 and max(mapping) == 42
assert mapping[:54] == bytes.fromhex(
    "000102030405000102030405060708090a0b0c0d08090a0b0c0d0e0f101112131415"
    "16061718191a1b1c1d0e1e1f2021222324252627"
)  # old logical prefix remains mapped byte-for-byte
print("test_cat_streaming.py: all checks passed (5504-byte atlas, 26 stable disk chunks)")
