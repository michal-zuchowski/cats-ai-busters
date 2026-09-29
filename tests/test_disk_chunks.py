#!/usr/bin/env python3
"""Verify assets/disk-chunks.bin matches the ATR's on-disk chunk layout."""
import struct
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "tools"))
import make_atr as m  # noqa: E402

ATR = ROOT / "out" / "cats-ai-busters.atr"
CHUNKS = ROOT / "assets" / "disk-chunks.bin"


def main():
    assert ATR.exists(), "run tools/make_atr.py first"
    assert CHUNKS.exists(), "run tools/make_atr.py first"

    chunk_table = CHUNKS.read_bytes()
    assert len(chunk_table) == 5 * len(m.EXTRA_CHUNKS), "5 bytes (2+2+1) per chunk entry"

    atr = ATR.read_bytes()
    body = atr[16:]

    def sector(n):  # 1-based sector number
        off = (n - 1) * m.SECTOR_SIZE
        return body[off:off + m.SECTOR_SIZE]

    for i, entry in enumerate(m.EXTRA_CHUNKS):
        if len(entry) == 4:
            name, path, offset, length = entry
            data = path.read_bytes()[offset:offset + length]
        else:
            name, path = entry
            data = path.read_bytes()
        start, count, checksum = struct.unpack_from("<HHB", chunk_table, i * 5)
        padded_len = -(-len(data) // m.SECTOR_SIZE) * m.SECTOR_SIZE
        padded = data + b"\x00" * (padded_len - len(data))
        assert count == padded_len // m.SECTOR_SIZE, name
        on_disk = b"".join(sector(start + n) for n in range(count))
        assert on_disk == padded, "chunk %r bytes on disk must match %s" % (name, path)
        assert checksum == sum(padded) & 0xFF, "chunk %r checksum mismatch" % name

    print("test_disk_chunks: OK (%d chunk(s))" % len(m.EXTRA_CHUNKS))


if __name__ == "__main__":
    main()
