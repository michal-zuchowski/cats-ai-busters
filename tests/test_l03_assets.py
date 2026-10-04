"""Portable L03 asset checks: generator output == committed asset, plus meaningful L03 content."""
import sys
from pathlib import Path
R = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(R / "tools"))
import make_plat_levels as G

new = (R / "assets/plat-rooms.bin").read_bytes()
assert new == b"".join(G.ROOMS) and len(new) == 10240
ch = lambda i: new[i * 1024:(i + 1) * 1024]
M = lambda i: ch(i)[960:]
A = lambda i, r, c: ch(i)[r * 40 + c]
for i in range(4, 8):
    assert M(i)[18] == (184 if i in (5, 7) else 168) and M(i)[8] == 21  # exit_y, vacuum row
assert all(A(5, 20, c) & 127 != 70 for c in range(35, 40))  # no head-height exit shelf
assert all(A(5, 22, c) == 70 for c in range(35, 40))
assert all(A(5, r, 39) == 85 for r in range(19, 22))  # floor-height hatch
import hashlib
assert hashlib.sha256(new[:7168] + new[8192:]).hexdigest() == "93d290a72a3dc2bed518097e5b4af8b0c176c255c6ad87dea3746a8f90c43024"
assert M(5)[4] == 22 and M(5)[6] == 6 and M(5)[7] == 33 and M(5)[3] == 28 and M(5)[30] == 1
assert M(7)[30] == 2 and M(7)[6] == M(7)[7] == 12 and M(7)[3] == 19
assert M(6)[29] and not any(M(i)[29] for i in (4, 5, 7))  # pursuer only in R3
assert M(6)[28] == 26 and M(4)[30] == M(6)[30] == 0
# charger bay marks and no baked beam; reader marks; R4 has one grounded lane,
# a clear first pit, and a balcony with no cosmetic lower-middle support.
assert all(A(5, 21, c) == 12 and A(5, 20, c) == 88 for c in (22, 23, 24))
assert all(A(5, 19, c) & 0x7F != 79 for c in range(40))
assert all(A(7, 21, c) == 12 for c in (12, 13, 14))
assert all(A(7, 20, c) == 83 for c in (12, 13, 14))
assert all(A(6, 22, c) & 127 != 70 for c in range(17, 33))
assert all(A(6, 22, c) == 70 for c in (*range(17), *range(33, 40)))
assert all(A(7, r, c) == 64 for r in (22, 23) for c in range(23, 33))
assert all(A(7, 22, c) == 70 for c in (*range(23), *range(33, 40)))
assert all(A(7, 22, c) != 70 for c in range(23, 33))
assert M(7)[9] == 8 and M(7)[10] == 10 and M(7)[12] == 8  # amin, amax, acol: patrol 8..10, starts at 8
assert all(A(7, 18, c) == 70 for c in range(23, 28))
assert all(A(7, 22, c) == 70 for c in range(8, 22))  # every normal robot footprint has real floor support
assert sum(A(7, r, c) != 64 for r in range(5, 18) for c in range(40)) >= 200
assert bytes(A(7, 18, c) for c in range(11, 15)) == bytes(ord(ch) - 32 for ch in "SCAN")
assert bytes(A(7, 18, c) for c in range(34, 38)) == bytes(ord(ch) - 32 for ch in "EXIT")
print("test_l03_assets.py: all checks passed")
