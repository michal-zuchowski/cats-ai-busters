"""Regression checks for the compiled checked-loader contract when artifacts exist."""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
from sim import SIOV, Sim


def invoke(s, name, a):
    m = s.mpu
    caller_pc, caller_sp = m.pc, m.sp
    sentinel = 0x0800
    ret = sentinel - 1
    s.mem[0x100 + caller_sp] = ret >> 8
    s.mem[0x100 + ((caller_sp - 1) & 255)] = ret & 255
    m.sp, m.pc, m.a = (caller_sp - 2) & 255, s.sym[name], a
    for _ in range(100_000):
        if m.pc == s.sym["music_failure"]:
            return "failure"
        if m.pc == sentinel:
            m.pc, m.sp = caller_pc, caller_sp
            return "success"
        if m.pc == SIOV:
            s._siov()
        else:
            m.step()
    raise AssertionError("checked loader did not terminate")


class ReadFailureSim(Sim):
    def _siov(self):
        super()._siov()
        self.mpu.y = 0


def main():
    source = (ROOT / "main.k65").read_text()
    assert len(re.findall(r"a=25\s+load_checked_chunk", source)) == 1
    scene = re.search(r"func ai_cutscene \{(.*?)\n\}", source, re.S).group(1)
    assert scene.count("load_checked_chunk") == 3
    assert "call music_failure" not in scene

    try:
        good = Sim()
    except FileNotFoundError:
        print("test_l04_loader_runtime.py: source contract passed; compiled artifacts unavailable")
        return

    # The existing compiled loader is the only executable under test here.
    for chunk, dest in ((14, 0xA000), (15, 0xA600), (25, 0xA000)):
        entry = good.mem[good.sym["ChunkTable"] + chunk * 5:good.sym["ChunkTable"] + chunk * 5 + 5]
        sector = int.from_bytes(entry[:2], "little")
        size = int.from_bytes(entry[2:4], "little") * 128
        good.mem[good.sym["disk_dest"]] = dest & 255
        good.mem[good.sym["disk_dest"] + 1] = dest >> 8
        assert invoke(good, "load_checked_chunk", chunk) == "success", chunk
        assert good.mpu.a == entry[4] != 0 and good["disk_failed"] == 0, chunk
        assert bytes(good.mem[dest:dest + size]) == good.atr[(sector - 1) * 128:(sector - 1) * 128 + size], chunk

    bad_atr = bytearray(good.atr)
    entry = good.mem[good.sym["ChunkTable"] + 25 * 5:good.sym["ChunkTable"] + 25 * 5 + 5]
    sector, count = int.from_bytes(entry[:2], "little"), int.from_bytes(entry[2:4], "little")
    bad_atr[(sector - 1) * 128] ^= 1
    checksum = Sim()
    checksum.atr = bytes(bad_atr)
    checksum.mem[checksum.sym["disk_dest"]] = 0
    checksum.mem[checksum.sym["disk_dest"] + 1] = 0xA0
    assert invoke(checksum, "load_checked_chunk", 25) == "failure"

    read_error = ReadFailureSim()
    read_error.mem[read_error.sym["disk_dest"]] = 0
    read_error.mem[read_error.sym["disk_dest"] + 1] = 0xA0
    assert invoke(read_error, "load_checked_chunk", 25) == "failure"
    print("test_l04_loader_runtime.py: compiled valid-checksum and checksum/SIO failure paths passed")


if __name__ == "__main__":
    main()
