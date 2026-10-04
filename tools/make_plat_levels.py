"""Rooms for the platform levels 02-04 and the PCM computer-death cry.

Each room is a 1024-byte disk chunk: 960 tile bytes (platforms are FLOOR tiles,
one-way) + 64 bytes of metadata read by main.k65 into page 6 (see META).
Run: python3 tools/make_plat_levels.py
"""

import random
from pathlib import Path

import make_level_art as L

ROOT = Path(__file__).resolve().parents[1]
META = ("sx", "sy", "fan", "lrow", "lcol", "lw", "lmin", "lmax", "arow", "amin",
        "amax", "aspd", "acol", "h0", "h1", "glass", "dlg", "desk", "exit_y",
        "swy", "swc", "lpow", "fc1", "fr0", "fr1", "lspd")  # keep <= 32; main.k65 copies 32 at m_sx
# L02 additions: swy/swc controller standing y + column (0 = none; it also gates the exit and,
# with lpow, the lift); fan = first column of the gust zone, fc1 = end column (exclusive),
# fr0/fr1 = body-row band [fr0, fr1); lspd = frames the lift dwells per row.
# Level 03 only (cur_level == 2): offsets inside the 32 copied bytes.  3/4/6/7 reuse the lift
# fields (m_lw stays 0 there, so no lift exists); 26..30 are free (L02 owns 19..25).
L03_META = {"xmax": 3, "dock": 4, "z0": 6, "z1": 7,
            "bcol": 26, "bmin": 27, "bmax": 28, "bspd": 29, "mode": 30}


def ground(row):  # standing y (PM byte) on a platform in this tile row
    return (row + 1) * 8


def build(conduits, platforms, racks=(), extra=None, decor=None, l03=None, **meta):
    screen = L.shell((), conduits)
    for left, top, width, bottom in racks:
        L.rack(screen, left, top, width=width, bottom=bottom, seed=left)
    if decor:
        decor(screen)
    for row, c0, c1 in platforms:
        for c in range(c0, c1 + 1):
            screen[row][c] = L.FLOOR
    if extra:
        extra(screen)
    if meta.get("exit_y"):
        exit_row = meta["exit_y"] // 8 - 1
        for row in range(exit_row - 3, exit_row):
            screen[row][39] = L.WALL if meta.get("swy") else L.HATCH
    art = bytes(t for r in screen for t in r)
    m = {k: 0 for k in META}
    m.update(sx=8, sy=ground(22))
    m.update(meta)
    raw = bytearray(bytes(m[k] for k in META).ljust(64, b"\0"))
    for name, value in (l03 or {}).items():
        assert raw[L03_META[name]] == 0 and meta.get("lw", 0) == 0
        raw[L03_META[name]] = value
    return art + bytes(raw)


def fan_frame(screen):
    L.frame(screen, 18, 5, 22, 12, lambda x, y: L.TXT_A if (x + y) % 2 else L.TXT_B | L.INV)


def computer(screen):
    L.frame(screen, 35, 7, 39, 13, lambda x, y: L.TXT_B if (x * y) % 3 else L.TXT_A)


FLOOR_ALL = [(22, 0, 39)]
PIT = (17, 32)  # nonwalkable section of room 3 of level 03 (wider than any jump)


def charger(screen):  # marked alcove: robot base cols 22..24
    for c in range(22, 25):
        screen[20][c], screen[21][c] = L.DOCK_PLATE, L.DOCK_BAY


def reader(screen):  # service reader over the vacuum's lane
    for c in range(12, 15):
        screen[20][c], screen[21][c] = L.RELAY, L.READ_STRIP
    for c in range(23, 33):
        screen[22][c] = screen[23][c] = L.VOID


def pit(screen):
    for c in range(PIT[0], PIT[1] + 1):
        screen[22][c] = L.BEAM | L.INV

def hang(screen, left, bottom, width=3):
    """Machinery hung from the ceiling (decorative, never supporting)."""
    L.rack(screen, left, 4, width=width, bottom=bottom, seed=left + bottom)


def crumbs(*runs):
    """Crumbling catwalk runs (row, c0, c1): CRUMB tiles drawn over everything."""
    def draw(screen):
        for row, c0, c1 in runs:
            for c in range(c0, c1 + 1):
                screen[row][c] = L.CRUMB
    return draw


def room2_1(screen):  # service shaft: zigzag ledges hang on cabinets, controller on the top-left ledge
    hang(screen, 2, 8, 4)
    hang(screen, 13, 7)
    hang(screen, 21, 9, 4)
    L.rack(screen, 28, 5, width=5, bottom=21, seed=3)
    L.rack(screen, 7, 13, width=6, bottom=21, seed=5)
    L.rack(screen, 17, 15, width=3, bottom=17, seed=7)
    L.rack(screen, 24, 17, width=3, bottom=19, seed=9)
    L.rack(screen, 17, 19, width=3, bottom=21, seed=11)
    for r in range(10, 22):
        screen[r][32], screen[r][35] = L.CONDUIT, L.CONDUIT  # lift guide rails
    for r in range(5, 13):
        screen[r][36] = L.CONDUIT


def room2_1_late(screen):
    screen[12][9] = L.CTRL | L.INV  # controller console, unpowered (red) until FIRE


def room2_2(screen):  # catwalks: crumbling shortcut above, stable stair below
    hang(screen, 1, 6, 4)
    hang(screen, 14, 8)
    hang(screen, 22, 7, 4)
    hang(screen, 31, 8)
    L.rack(screen, 3, 14, width=4, bottom=21, seed=2)
    L.rack(screen, 12, 16, width=4, bottom=21, seed=6)
    L.rack(screen, 16, 14, width=5, bottom=21, seed=4)
    L.rack(screen, 34, 12, width=5, bottom=21, seed=8)
    L.rack(screen, 24, 20, width=3, bottom=21, seed=9)
    for r in range(15, 22):
        screen[r][10] = L.CONDUIT
        screen[r][22] = L.CONDUIT


def room2_3(screen):  # cooling channel: fan on the left, airflow band over the gap
    hang(screen, 1, 9, 4)
    hang(screen, 20, 7, 4)
    hang(screen, 31, 6, 4)
    L.rack(screen, 1, 13, width=6, bottom=21, seed=2)
    L.rack(screen, 29, 11, width=5, bottom=21, seed=4)
    L.rack(screen, 36, 11, width=4, bottom=21, seed=8)
    L.frame(screen, 6, 6, 9, 9, lambda x, y: L.PANEL)  # fan housing
    for tile, (r, c) in zip(L.FAN_A, ((7, 7), (7, 8), (8, 7), (8, 8))):
        screen[r][c] = tile


def room2_4(screen):  # roof lift: shaft, controller ledge, catwalk to the roof hatch
    hang(screen, 1, 5, 4)
    hang(screen, 8, 5)
    L.rack(screen, 2, 11, width=5, bottom=21, seed=2)
    L.rack(screen, 20, 12, width=5, bottom=21, seed=6)
    L.rack(screen, 27, 13, width=4, bottom=21, seed=3)
    L.rack(screen, 34, 10, width=5, bottom=21, seed=9)
    for r in range(8, 22):
        screen[r][12] = L.CONDUIT
        screen[r][16] = L.CONDUIT


def room2_4_late(screen):
    screen[10][26] = L.CTRL | L.INV


def room4_1(screen):
    hang(screen, 3, 13, 4)
    hang(screen, 12, 10, 4)
    L.rack(screen, 18, 17, width=5, bottom=21, seed=65)
    L.rack(screen, 27, 19, width=4, bottom=21, seed=66)
    for r in range(7, 22):
        screen[r][31], screen[r][35] = L.CONDUIT, L.CONDUIT
    L.frame(screen, 18, 5, 26, 10, lambda x, y: L.TXT_B | L.INV if (x + y) % 3 else L.TXT_A)


def room4_2(screen):
    hang(screen, 2, 7, 4)
    hang(screen, 12, 9, 4)
    L.rack(screen, 18, 15, width=4, bottom=21, seed=67)
    L.rack(screen, 26, 18, width=4, bottom=21, seed=68)
    computer(screen)


def service_room(screen):
    hang(screen, 1, 14, 6)
    L.frame(screen, 11, 5, 18, 10, lambda x, y: L.TXT_A if (x + y) % 3 else L.TXT_B)
    L.rack(screen, 20, 4, width=7, bottom=12, seed=71)
    L.rack(screen, 30, 7, width=7, bottom=17, seed=73)
    L.rack(screen, 8, 13, width=3, bottom=18, seed=74)
    for c, ch in enumerate("SCAN", 11):
        screen[18][c] = ord(ch) - 32
    for c, ch in enumerate("EXIT", 34):
        screen[18][c] = ord(ch) - 32
    for r in range(4, 17):
        screen[r][28] = L.CONDUIT


def two(builder, late=None, crumb=(), **kw):
    def extra(screen):
        crumbs(*crumb)(screen)
        if late:
            late(screen)
    return build((), kw.pop("platforms"), decor=builder, extra=extra, **kw)


ROOMS = [
    # ---- Level 02 ----
    two(room2_1, room2_1_late, exit_y=ground(13), swy=ground(12), swc=9, lrow=22, lcol=33, lw=2,
        lmin=13, lmax=22, lpow=1, lspd=18,
        platforms=FLOOR_ALL + [(20, 22, 25), (18, 16, 19), (16, 23, 26), (14, 16, 19),
                               (12, 6, 13), (13, 35, 39)]),
    two(room2_2, exit_y=ground(11), sy=ground(13), crumb=((13, 10, 13), (12, 17, 20), (11, 25, 28)),
        platforms=FLOOR_ALL + [(13, 0, 6), (20, 26, 29), (18, 32, 35), (16, 26, 29), (14, 32, 35), (11, 33, 39)]),
    two(room2_3, exit_y=ground(9), sy=ground(11), fan=10, fc1=18, fr0=5, fr1=11,
        platforms=FLOOR_ALL + [(11, 0, 9), (11, 18, 25), (9, 29, 39), (19, 12, 15), (16, 17, 20),
                               (13, 22, 25)]),
    two(room2_4, room2_4_late, crumb=((6, 18, 21), (6, 25, 28)), exit_y=ground(6), sy=ground(9),
        swy=ground(10), swc=26, lrow=22, lcol=13, lw=3, lmin=6, lmax=22, lspd=12,
        platforms=FLOOR_ALL + [(9, 0, 7), (10, 17, 31), (6, 33, 39)]),
    # ---- Level 03 ----
    # 1 cleaning corridor: niches at row 20, podium exit; one patrolling vacuum
    build((5, 20), FLOOR_ALL + [(20, 10, 14), (20, 18, 22), (20, 35, 39)], racks=((28, 6, 5, 18),),
          arow=21, amin=16, amax=22, aspd=6, acol=16, exit_y=ground(20)),
    # 2 charger puzzle: lasers over the aisle until the struck vacuum docks in the bay
    build((12, 38), FLOOR_ALL, extra=charger,
          arow=21, amin=14, amax=20, aspd=5, acol=14, exit_y=ground(22),
          l03=dict(xmax=28, dock=22, z0=6, z1=33, mode=1)),
    # 3 signature ride over the pit with a separate pursuer
    build((3, 36), [(22, 0, PIT[0] - 1), (22, PIT[1] + 1, 39), (20, 34, 39)], extra=pit,
          arow=21, amin=6, amax=29, aspd=12, acol=8, exit_y=ground(20),
          l03=dict(bcol=0, bmin=0, bmax=26, bspd=6)),
    # 4 scan -> one clear carrier -> service balcony -> jump to the floor-height exit
    build((10, 33), [(22, 0, 22), (22, 33, 39), (18, 23, 27)],
          decor=service_room, extra=reader,
          arow=21, amin=8, amax=10, aspd=8, acol=8, exit_y=ground(22),
          l03=dict(xmax=19, z0=12, z1=12, mode=2)),
    # ---- Level 04 ----
    build((4, 36), FLOOR_ALL + [(20, 4, 8), (18, 11, 15), (16, 18, 22),
                               (18, 27, 31), (9, 35, 39)], decor=room4_1,
          sx=0, h0=7, h1=34, lrow=18, lcol=32, lw=3, lmin=9, lmax=18, lspd=12,
          exit_y=ground(9)),
    build((3, 25), FLOOR_ALL + [(9, 0, 6), (11, 11, 16), (13, 20, 24),
                              (16, 24, 28), (14, 30, 39)], decor=room4_2,
          sx=0, sy=ground(9), h0=7, h1=34, glass=33, dlg=ground(13), desk=ground(14)),
]
assert len(ROOMS) == 10 and all(len(r) == 1024 for r in ROOMS)


def cry(n=2560, rate=5200):
    """Strained synthetic shriek, then irregular sputters that drop and die (4-bit volumes)."""
    rnd, out, phase = random.Random(65), [], 0.0
    for i in range(n):
        t = i / n
        if t < 0.30:  # shriek: wavering, slightly falling pitch
            f = 1500 - 700 * t + 160 * __import__("math").sin(i / 25)
            amp = 7
        else:  # sputters: noise bursts and gaps, pitch collapses, fades out
            f = 800 * (1 - t) ** 2 + 60
            amp = 7 * (1 - t) ** 0.6 * (1 if (i // 60 + rnd.randint(0, 1)) % 3 else 0.2)
        phase = (phase + f / rate) % 1
        v = 1 if phase < 0.5 else -1
        if t >= 0.30 and rnd.random() < 0.35 * t:
            v = rnd.choice((-1, 1))
        out.append(max(0, min(15, round(7 + amp * v))) if t < 0.97 else 0)
    return bytes(out)


if __name__ == "__main__":
    (ROOT / "assets" / "plat-rooms.bin").write_bytes(b"".join(ROOMS))
    (ROOT / "assets" / "plat-cry.pcm").write_bytes(cry())
