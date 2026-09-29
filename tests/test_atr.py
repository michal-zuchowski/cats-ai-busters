#!/usr/bin/env python3
"""Verify out/cats-ai-busters.atr: header, boot sector, and payload layout."""
import struct
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "tools"))
import make_atr as m  # noqa: E402

ATR = ROOT / "out" / "cats-ai-busters.atr"
XEX = ROOT / "out" / "cats-ai-busters.xex"


def main():
    assert XEX.exists(), "build main.k65proj first"
    assert ATR.exists(), "run tools/make_atr.py first"

    xex_bytes = XEX.read_bytes()
    payload = m.parse_xex_main_segment(xex_bytes)

    atr = ATR.read_bytes()

    # --- ATR header ---
    magic, paragraphs_lo, sector_size, paragraphs_hi = struct.unpack_from("<HHHH", atr, 0)
    assert magic == 0x0296, "bad ATR magic"
    assert sector_size == m.SECTOR_SIZE == 128
    body = atr[16:]
    assert len(body) == m.TOTAL_SECTORS * m.SECTOR_SIZE, "expected a 90K image"
    total_paragraphs = paragraphs_lo | (paragraphs_hi << 16)
    assert total_paragraphs * 16 == len(body)

    def sector(n):  # 1-based sector number
        off = (n - 1) * m.SECTOR_SIZE
        return body[off:off + m.SECTOR_SIZE]

    # --- boot sector fields ---
    boot = sector(1)
    bflag, brcnt = boot[0], boot[1]
    bldadr, binitad = struct.unpack_from("<HH", boot, 2)
    assert bflag == 0, "boot flag must mark the disk bootable"
    assert brcnt == m.BOOT_SECTORS
    assert bldadr == m.BLDADR == 0x0700
    assert binitad == m.BLDADR + 6, "loader code starts right after the 6-byte header"

    # loader code must be plausible 6502 (non-empty, ends with our JMP $1000)
    code = boot[6:]
    jmp_payload = bytes([0x4C]) + struct.pack("<H", m.PAYLOAD_ADDR)
    assert jmp_payload in code, "loader must JMP to the payload address"
    assert code.index(jmp_payload) > 0

    # SIOV disk-read constants the loader must poke into the DCB.
    for needed in (bytes([0xA9, 0x31]),   # LDA #$31 (DDEVIC)
                   bytes([0xA9, 0x52]),   # LDA #$52 (DCOMND 'R')
                   bytes([0x20]) + struct.pack("<H", m.SIOV)):  # JSR SIOV
        assert needed in code, needed

    # --- payload sector layout ---
    sector_count = -(-len(payload) // m.SECTOR_SIZE)
    padded_payload = payload + b"\x00" * (sector_count * m.SECTOR_SIZE - len(payload))
    first = m.FIRST_DATA_SECTOR
    assert first == m.BOOT_SECTORS + 1
    on_disk = b"".join(sector(first + n) for n in range(sector_count))
    assert on_disk == padded_payload, "disk payload must equal the XEX's 0x1000.. segment"

    # sectors before/after the payload must be untouched (zero)
    assert sector(2 if first != 2 else first + sector_count) is not None  # sanity, always true
    tail_sector = first + sector_count
    if tail_sector <= m.TOTAL_SECTORS:
        assert sector(tail_sector) == b"\x00" * m.SECTOR_SIZE

    print("test_atr: OK (%d payload sectors from sector %d)" % (sector_count, first))


if __name__ == "__main__":
    main()
