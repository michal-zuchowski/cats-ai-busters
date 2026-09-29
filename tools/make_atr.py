#!/usr/bin/env python3
"""Build out/cats-ai-busters.atr: a DOS-less, self-booting Atari disk image.

Layout (90K single-density, 128-byte sectors, 720 sectors total):
  sector 1        hand-assembled 6502 boot loader (see `assemble_loader`)
  sectors 2..3    unused, zero
  sectors 4..N    the K65 XEX's resident segment (0x1000..end), copied
                  verbatim, read straight into place by the boot loader
  remaining       zero

The loader is the standard Atari cold-boot stub: BFLAG/BRCNT/BLDADR/BINITAD
at the front of sector 1, then code that repeatedly calls SIOV to read the
payload sectors directly to $1000 and finally JMPs there (the XEX's RUNAD
segment always points at $1000, a `JMP main`, so we don't need to load or
honor RUNAD at runtime -- see doc/tech/gameplay.md).

After the resident program's sectors, extra named "chunks" are appended:
room 0/1/2 art (960-byte slices of assets/level-rooms.pic), CamCfg
(assets/level-cams.bin) and ConeStep (assets/cone-step.bin) -- level 01's
disk-streamed assets (a3e.3) -- plus a next-level-placeholder chunk
standing in for level 2's not-yet-authored data (a3e.5). assets/disk-chunks.bin
records, per chunk, a 5-byte entry: word start_sector, word sector_count,
byte checksum (sum of the padded on-disk bytes, mod 256). main.k65's
`disk_read` func loads each chunk with SIOV; `disk_selftest` re-checks
chunk 0's (room 0 art) checksum as a boot-time smoke test, and
`prefetch_next_level` re-checks the placeholder chunk's checksum during
the level-ending pause -- see doc/tech/gameplay.md.
"""
import struct
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
XEX_PATH = ROOT / "out" / "cats-ai-busters.xex"
ATR_PATH = ROOT / "out" / "cats-ai-busters.atr"
CHUNKS_PATH = ROOT / "assets" / "disk-chunks.bin"
# Proof-of-concept extra chunks appended after the resident program. Each is
# (name, source file); a3e.3 will replace/extend this with real level assets.
EXTRA_CHUNKS = [
    ("room0-art", ROOT / "assets" / "level-rooms.pic", 0, 960),
    ("room1-art", ROOT / "assets" / "level-rooms.pic", 960, 960),
    ("room2-art", ROOT / "assets" / "level-rooms.pic", 1920, 960),
    ("cam-cfg", ROOT / "assets" / "level-cams.bin"),
    ("cone-step", ROOT / "assets" / "cone-step.bin"),
    # a3e.5: level 2 doesn't exist yet, so this stands in for its data; the
    # level-ending prefetch (main.k65's prefetch_next_level) reads and
    # checksums this chunk to prove the continuous-loading trick works.
    ("next-level-placeholder", ROOT / "assets" / "next-level-placeholder.bin"),
]

SECTOR_SIZE = 128
TOTAL_SECTORS = 720          # 90K single-density disk
BOOT_SECTORS = 1             # BRCNT: our loader fits in sector 1 alone
BLDADR = 0x0700              # where the OS loads the boot sector(s)
PAYLOAD_ADDR = 0x1000        # -lowAddr from main.k65proj
FIRST_DATA_SECTOR = BOOT_SECTORS + 1   # sector 2 is the first free sector

# Fixed OS entry points / DCB, always present without DOS.
DDEVIC, DUNIT, DCOMND, DSTATS = 0x0300, 0x0301, 0x0302, 0x0303
DBUFLO, DBUFHI, DTIMLO = 0x0304, 0x0305, 0x0306
DBYTLO, DBYTHI, DAUX1, DAUX2 = 0x0308, 0x0309, 0x030A, 0x030B
SIOV = 0xE459
# Zero-page scratch for the sector counter: 0xC8-0xCF are free at boot time
# (nothing from main.k65's `var` table is alive yet).
COUNTLO, COUNTHI = 0xCB, 0xCC


class Asm:
    """Minimal two-pass 6502 assembler: just enough for the boot loader."""

    def __init__(self, origin):
        self.origin = origin
        self.ops = []  # list of callables(pc) -> bytes, for pass 2
        self.labels = {}
        self.pc = origin

    def _emit(self, size, make_bytes):
        addr = self.pc
        self.ops.append((addr, size, make_bytes))
        self.pc += size

    def label(self, name):
        self.labels[name] = self.pc

    def lda_imm(self, v):
        self._emit(2, lambda pc: bytes([0xA9, v & 0xFF]))

    def sta_abs(self, addr):
        self._emit(3, lambda pc: bytes([0x8D, addr & 0xFF, addr >> 8]))

    def lda_abs(self, addr):
        self._emit(3, lambda pc: bytes([0xAD, addr & 0xFF, addr >> 8]))

    def adc_imm(self, v):
        self._emit(2, lambda pc: bytes([0x69, v & 0xFF]))

    def inc_abs(self, addr):
        self._emit(3, lambda pc: bytes([0xEE, addr & 0xFF, addr >> 8]))

    def sta_zp(self, addr):
        self._emit(2, lambda pc: bytes([0x85, addr & 0xFF]))

    def lda_zp(self, addr):
        self._emit(2, lambda pc: bytes([0xA5, addr & 0xFF]))

    def dec_zp(self, addr):
        self._emit(2, lambda pc: bytes([0xC6, addr & 0xFF]))

    def ora_zp(self, addr):
        self._emit(2, lambda pc: bytes([0x05, addr & 0xFF]))

    def clc(self):
        self._emit(1, lambda pc: bytes([0x18]))

    def jsr(self, addr):
        self._emit(3, lambda pc: bytes([0x20, addr & 0xFF, addr >> 8]))

    def jmp(self, addr):
        self._emit(3, lambda pc: bytes([0x4C, addr & 0xFF, addr >> 8]))

    def bne(self, label_name):
        def make(pc):
            target = self.labels[label_name]
            offset = target - (pc + 2)
            if not -128 <= offset <= 127:
                raise ValueError("branch out of range: %s" % label_name)
            return bytes([0xD0, offset & 0xFF])
        self._emit(2, make)

    def assemble(self):
        out = bytearray()
        for addr, size, make in self.ops:
            b = make(addr)
            assert len(b) == size
            out += b
        assert len(out) == self.pc - self.origin
        return bytes(out)


def assemble_loader(start_sector, sector_count, code_origin):
    a = Asm(code_origin)
    # One-time setup.
    a.lda_imm(start_sector & 0xFF)
    a.sta_abs(DAUX1)
    a.lda_imm(start_sector >> 8)
    a.sta_abs(DAUX2)
    a.lda_imm(PAYLOAD_ADDR & 0xFF)
    a.sta_abs(DBUFLO)
    a.lda_imm(PAYLOAD_ADDR >> 8)
    a.sta_abs(DBUFHI)
    a.lda_imm(sector_count & 0xFF)
    a.sta_zp(COUNTLO)
    a.lda_imm(sector_count >> 8)
    a.sta_zp(COUNTHI)
    # Read one sector per iteration.
    a.label("loop")
    a.lda_imm(0x31)          # DDEVIC: disk
    a.sta_abs(DDEVIC)
    a.lda_imm(0x01)          # DUNIT: drive 1
    a.sta_abs(DUNIT)
    a.lda_imm(0x52)          # DCOMND: read sector ('R')
    a.sta_abs(DCOMND)
    a.lda_imm(0x40)          # DSTATS: peripheral -> memory
    a.sta_abs(DSTATS)
    a.lda_imm(0x0F)          # DTIMLO: 15s timeout
    a.sta_abs(DTIMLO)
    a.lda_imm(SECTOR_SIZE & 0xFF)
    a.sta_abs(DBYTLO)
    a.lda_imm(SECTOR_SIZE >> 8)
    a.sta_abs(DBYTHI)
    a.jsr(SIOV)
    # dest += 128 (DBUFLO/DBUFHI)
    a.clc()
    a.lda_abs(DBUFLO)
    a.adc_imm(SECTOR_SIZE)
    a.sta_abs(DBUFLO)
    a.lda_abs(DBUFHI)
    a.adc_imm(0)
    a.sta_abs(DBUFHI)
    # sector += 1 (DAUX1/DAUX2)
    a.inc_abs(DAUX1)
    a.bne("nowrap")
    a.inc_abs(DAUX2)
    a.label("nowrap")
    # counter -= 1 (COUNTLO/COUNTHI, 16-bit)
    a.lda_zp(COUNTLO)
    a.bne("decl")
    a.dec_zp(COUNTHI)
    a.label("decl")
    a.dec_zp(COUNTLO)
    a.lda_zp(COUNTLO)
    a.ora_zp(COUNTHI)
    a.bne("loop")
    a.jmp(PAYLOAD_ADDR)
    return a.assemble()


def parse_xex_main_segment(xex_bytes):
    """Return the bytes of the 0x1000.. segment (the resident program)."""
    i = 0
    segments = []
    while i < len(xex_bytes):
        if xex_bytes[i:i + 2] == b"\xff\xff":
            i += 2
            continue
        start, end = struct.unpack_from("<HH", xex_bytes, i)
        i += 4
        length = end - start + 1
        segments.append((start, xex_bytes[i:i + length]))
        i += length
    for start, data in segments:
        if start == PAYLOAD_ADDR:
            return data
    raise SystemExit("no segment starting at 0x%04X found in %s" % (PAYLOAD_ADDR, XEX_PATH))


def build_atr(payload, extra_chunks):
    padded_len = -(-len(payload) // SECTOR_SIZE) * SECTOR_SIZE
    payload = payload + b"\x00" * (padded_len - len(payload))
    sector_count = padded_len // SECTOR_SIZE

    code_origin = BLDADR + 6
    code = assemble_loader(FIRST_DATA_SECTOR, sector_count, code_origin)
    if len(code) > SECTOR_SIZE - 6:
        raise SystemExit("boot loader (%d bytes) no longer fits in one sector" % len(code))
    binitad = code_origin

    boot_sector = bytearray(SECTOR_SIZE)
    boot_sector[0] = 0x00                       # BFLAG: bootable
    boot_sector[1] = BOOT_SECTORS                # BRCNT
    boot_sector[2:4] = struct.pack("<H", BLDADR)  # BLDADR
    boot_sector[4:6] = struct.pack("<H", binitad)  # BINITAD
    boot_sector[6:6 + len(code)] = code

    sectors = [bytes(SECTOR_SIZE)] * TOTAL_SECTORS
    sectors[0] = bytes(boot_sector)
    for n in range(sector_count):
        sectors[FIRST_DATA_SECTOR - 1 + n] = payload[n * SECTOR_SIZE:(n + 1) * SECTOR_SIZE]

    chunk_table = bytearray()
    next_sector = FIRST_DATA_SECTOR + sector_count
    for entry in extra_chunks:
        if len(entry) == 4:
            name, path, offset, length = entry
            data = path.read_bytes()[offset:offset + length]
        else:
            name, path = entry
            data = path.read_bytes()
        padded = -(-len(data) // SECTOR_SIZE) * SECTOR_SIZE
        data = data + b"\x00" * (padded - len(data))
        count = padded // SECTOR_SIZE
        if next_sector + count - 1 > TOTAL_SECTORS:
            raise SystemExit("chunk %r does not fit on a 90K disk" % name)
        for n in range(count):
            sectors[next_sector - 1 + n] = data[n * SECTOR_SIZE:(n + 1) * SECTOR_SIZE]
        checksum = sum(data) & 0xFF
        chunk_table += struct.pack("<HHB", next_sector, count, checksum)
        next_sector += count

    body = b"".join(sectors)
    paragraphs = len(body) // 16
    header = struct.pack(
        "<HHHH", 0x0296, paragraphs & 0xFFFF, SECTOR_SIZE, paragraphs >> 16
    ) + b"\x00" * 8
    return header + body, sector_count, bytes(chunk_table)


def main():
    if not XEX_PATH.exists():
        raise SystemExit("missing %s; build main.k65proj first" % XEX_PATH)
    xex = XEX_PATH.read_bytes()
    payload = parse_xex_main_segment(xex)
    atr, sector_count, chunk_table = build_atr(payload, EXTRA_CHUNKS)
    ATR_PATH.parent.mkdir(parents=True, exist_ok=True)
    ATR_PATH.write_bytes(atr)
    CHUNKS_PATH.write_bytes(chunk_table)
    print("wrote %s (%d bytes, %d payload sectors from sector %d, %d extra chunk(s))"
          % (ATR_PATH, len(atr), sector_count, FIRST_DATA_SECTOR, len(EXTRA_CHUNKS)))


if __name__ == "__main__":
    main()
