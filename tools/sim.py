"""Headless 6502 harness: runs out/cats-ai-busters.xex frame by frame.

Needs `py65`. Fakes only what gameplay touches: PORTA/TRIG0 (joystick),
RTCLOK (one tick per ~PAL frame of CPU cycles) and SIOV (reads sectors from
the ATR). No VBI/music/DLI -- game logic and zero-page/screen state only.
"""
import re
from pathlib import Path

from py65.devices.mpu6502 import MPU

ROOT = Path(__file__).resolve().parents[1]
FRAME_CYCLES = 29000
SIOV = 0xE459


def load_symbols(path=ROOT / "out" / "cats-ai-busters.sym"):
    syms = {}
    for line in path.read_text().splitlines()[1:]:
        parts = line.split()
        if len(parts) >= 2 and re.fullmatch(r"[0-9a-f]{1,4}", parts[1]):
            syms[parts[0]] = int(parts[1], 16)
    return syms


class Sim:
    def __init__(self, xex=None, atr=None):
        self.sym = load_symbols()
        self.mpu = MPU()
        self.mem = self.mpu.memory
        data = (xex or ROOT / "out" / "cats-ai-busters.xex").read_bytes()
        i = 0
        while i < len(data):
            if data[i:i + 2] == b"\xff\xff":
                i += 2
            start = data[i] | data[i + 1] << 8
            end = data[i + 2] | data[i + 3] << 8
            self.mem[start:end + 1] = list(data[i + 4:i + 5 + end - start])
            i += 5 + end - start
        self.atr = (atr or ROOT / "out" / "cats-ai-busters.atr").read_bytes()[16:]
        self.mpu.pc = self.mem[0x2E0] | self.mem[0x2E1] << 8
        self.mpu.sp = 0xFF
        self.mem[0xD300] = 0xFF
        self.mem[0xD010] = 1
        self.mem[0xD20F] = 0xFF  # SKSTAT: keyboard and Shift released
        self.mem[0xD01F] = 7     # CONSOL: Start/Select/Option released
        self.frames = 0
        self.next_tick = FRAME_CYCLES

    def snapshot(self):
        m = self.mpu
        return (list(self.mem), m.pc, m.a, m.x, m.y, m.sp, m.p, m.processorCycles,
                self.frames, self.next_tick)

    def restore(self, snap):
        m = self.mpu
        self.mem[:] = snap[0]
        m.pc, m.a, m.x, m.y, m.sp, m.p, m.processorCycles, self.frames, self.next_tick = snap[1:]

    def __getitem__(self, name):
        return self.mem[self.sym[name]]

    def __setitem__(self, name, value):
        self.mem[self.sym[name]] = value

    def word(self, name):
        a = self.sym[name]
        return self.mem[a] | self.mem[a + 1] << 8

    def _siov(self):
        m = self.mem
        buf = m[0x304] | m[0x305] << 8
        sec = m[0x30A] | m[0x30B] << 8
        off = (sec - 1) * 128
        m[buf:buf + 128] = list(self.atr[off:off + 128])
        m[0x303] = 1
        self.mpu.y = 1
        sp = self.mpu.sp
        lo = m[0x100 + ((sp + 1) & 0xFF)]
        hi = m[0x100 + ((sp + 2) & 0xFF)]
        self.mpu.sp = (sp + 2) & 0xFF
        self.mpu.pc = ((hi << 8 | lo) + 1) & 0xFFFF

    def run_frame(self, joy=0, fire=False):
        """joy bits (1 up, 2 down, 4 left, 8 right) are pressed directions."""
        self.mem[0xD300] = 0xFF ^ joy
        self.mem[0xD010] = 0 if fire else 1
        mpu = self.mpu
        while mpu.processorCycles < self.next_tick:
            if mpu.pc == SIOV:
                self._siov()
            else:
                mpu.step()
        self.next_tick += FRAME_CYCLES
        for i in (0x14, 0x13, 0x12):  # 24-bit RTCLOK, low byte at $14
            self.mem[i] = (self.mem[i] + 1) & 0xFF
            if self.mem[i]:
                break
        self.frames += 1

    def run(self, frames, joy=0, fire=False):
        for _ in range(frames):
            self.run_frame(joy, fire)

    def screen(self, row, col=0, n=40):
        base = self.sym["TermScreen"] + row * 40 + col
        return bytes(self.mem[base:base + n])
