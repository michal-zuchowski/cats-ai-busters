"""Druidarium-style ANTIC 4 tiles and three data-center room screens.

Pixel values: 0 black, 1 light grey (PF0, also HUD text), 2 steel blue (PF1),
3 rust orange (PF2); tiles with bit 7 set draw their 3s in red (PF3).
The cat is a PMG sprite (tools/make_cat_sprite.py), so no tile is reserved
for it and the backdrop can be dense.
"""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
INV = 128

VOID = 64
CEIL = 65
CEIL_EDGE = 66
TRAY = 67
CAM = 68
RAIL_L = 69
FLOOR = 70            # play row: open floor (fixed ID, used by main.k65)
RAIL_R = 71
SRV_A = 72
SRV_B = 73
SRV_C = 74
SRV_D = 75
CAP_L = 76
CAP = 77
SHADOW = 78           # play row: rack shelter
BEAM = 79             # play row: watched floor, drawn inverse (red)
SAFE = 80             # play row: camera dark
WALL = 81             # hatch closed
SWITCH = 82
RELAY = 83
DONE = 84
HATCH = 85
CAP_R = 86
PLINTH_L = 87
PLINTH = 88
PLINTH_R = 89
EDGE = 90
FR_TL = 96
FR_T = 97
FR_TR = 98
FR_L = 99
FR_R = 100
FR_BL = 101
FR_B = 102
FR_BR = 103
TXT_A = 104
TXT_B = 105
CONDUIT = 106
CEIL_JOINT = 107
PANEL = 108
CONE = 109            # 109..112: light cone, lit from pixel k (0 = full cell)
CONE_R = 113          # 113..116: lit up to pixel k (116 = full cell)
FLOOR_LIT = 117       # play row inside a camera footprint
CAM_L = 118
CAM_R = 119
HUD_FULL = 120        # HUD bar cell, filled (COLPF1: state colour)
HUD_EMPTY = 121       # HUD bar cell, empty (COLPF2)
HUD_L = 122           # HUD bar brackets
HUD_R = 123

TILE_GLYPHS = {
    VOID: ("0000",) * 8,
    CEIL: ("2222", "2212", "2222", "1222", "2222", "2221", "2222", "2122"),
    CEIL_EDGE: ("2222", "2212", "2222", "1111", "2222", "0000", "0000", "0000"),
    CEIL_JOINT: ("2112", "2112", "2112", "1111", "2112", "0220", "0000", "0000"),
    TRAY: ("0000", "2002", "2332", "3333", "3133", "2222", "0000", "0000"),
    CAM: ("0220", "0220", "2222", "2112", "2222", "2232", "0220", "0000"),
    CAP_L: ("0111", "1122", "1222", "1222", "1222", "1222", "1200", "1211"),
    CAP: ("1111", "2222", "2212", "2222", "2222", "2222", "0000", "1111"),
    CAP_R: ("1110", "2220", "2220", "2220", "2220", "2220", "0020", "1020"),
    RAIL_L: ("1222", "1232", "1222", "1200", "1211", "1222", "1222", "1200"),
    RAIL_R: ("2220", "2220", "2020", "0020", "1120", "2220", "2220", "0020"),
    # calm 1U fronts; SRV_B/SRV_D LEDs are blinked at runtime by main.k65 blink_leds
    SRV_A: ("2222", "2222", "2222", "2222", "2222", "2222", "2222", "0000"),
    SRV_B: ("2222", "2222", "2232", "2222", "2222", "2222", "2222", "0000"),
    SRV_C: ("2222", "2222", "2222", "2112", "2222", "2222", "2222", "0000"),
    SRV_D: ("2222", "2222", "2222", "2222", "2322", "2222", "2222", "0000"),
    PLINTH_L: ("1111", "1222", "1202", "1222", "0000", "0000", "0000", "0000"),
    PLINTH: ("1111", "2222", "0202", "2222", "0000", "0000", "0000", "0000"),
    PLINTH_R: ("1110", "2220", "0220", "2220", "0000", "0000", "0000", "0000"),
    # play row (row 18): black space above a thin floor surface
    FLOOR: ("0000", "0000", "0000", "0000", "0000", "0000", "2121", "2222"),
    SHADOW: ("0000", "0000", "0000", "0000", "0000", "0000", "0202", "2222"),
    SAFE: ("0000", "0000", "0000", "0000", "0000", "0000", "0200", "2222"),
    BEAM: ("0300", "0000", "0003", "3000", "0030", "0303", "3333", "3333"),
    WALL: ("1222", "1222", "1232", "1222", "1222", "1222", "1222", "2222"),
    SWITCH: ("0000", "0110", "0130", "0110", "0110", "0020", "2121", "2222"),
    RELAY: ("1111", "1331", "1311", "1331", "1111", "0220", "2121", "2222"),
    DONE: ("1111", "1221", "1211", "1221", "1111", "0220", "2121", "2222"),
    HATCH: ("1000", "1000", "1000", "1000", "1000", "1000", "1121", "2222"),
    EDGE: ("1111", "1111", "2222", "2212", "2222", "2122", "2222", "0000"),
    FR_TL: ("0111", "1122", "1200", "1200", "1200", "1200", "1200", "1200"),
    FR_T: ("1111", "2222", "0000", "0000", "0000", "0000", "0000", "0000"),
    FR_TR: ("1110", "2220", "0020", "0020", "0020", "0020", "0020", "0020"),
    FR_L: ("1200",) * 8,
    FR_R: ("0020",) * 8,
    FR_BL: ("1200", "1200", "1200", "1200", "1200", "1222", "1222", "0000"),
    FR_B: ("0000", "0000", "0000", "0000", "0000", "2222", "2222", "0000"),
    FR_BR: ("0020", "0020", "0020", "0020", "0020", "2220", "2220", "0000"),
    TXT_A: ("0000", "3303", "0000", "3033", "0000", "3330", "0000", "0000"),
    TXT_B: ("0000", "0333", "0000", "3300", "0000", "3033", "0000", "0000"),
    CONDUIT: ("0210", "0210", "0220", "0210", "0210", "0210", "0220", "0210"),
    **{CONE + k: tuple(("".join(p if i >= k else "0" for i, p in enumerate(r))) for r in (
        "1010", "0000", "0101", "0000") * 2) for k in range(4)},
    **{CONE_R + k: tuple(("".join(p if i <= k else "0" for i, p in enumerate(r))) for r in (
        "1010", "0000", "0101", "0000") * 2) for k in range(4)},
    FLOOR_LIT: ("1010", "0000", "0101", "0000", "1010", "0101", "1111", "2222"),
    CAM_L: ("0220", "0220", "2222", "2112", "2222", "3222", "0220", "0000"),
    CAM_R: ("0220", "0220", "2222", "2112", "2222", "2223", "0220", "0000"),
    PANEL: ("2222", "2112", "2332", "2112", "2222", "2002", "2222", "0000"),
    HUD_FULL: ("0000", "2220", "2220", "2220", "2220", "2220", "0000", "0000"),
    HUD_EMPTY: ("0000", "3330", "3030", "3030", "3030", "3330", "0000", "0000"),
    HUD_L: ("0000", "0033", "0030", "0030", "0030", "0033", "0000", "0000"),
    HUD_R: ("0000", "3300", "0300", "0300", "0300", "3300", "0000", "0000"),
}

SERVERS = (SRV_A, SRV_B | INV, SRV_A, SRV_C, SRV_A, SRV_D, SRV_A, SRV_A, SRV_D | INV)


def rack(screen, left, top, width=6, bottom=21, seed=0):
    screen[top][left] = CAP_L
    screen[top][left + width - 1] = CAP_R
    for x in range(left + 1, left + width - 1):
        screen[top][x] = CAP
    for y in range(top + 1, bottom):
        screen[y][left] = RAIL_L
        screen[y][left + width - 1] = RAIL_R
        for x in range(left + 1, left + width - 1):
            screen[y][x] = SERVERS[(x * 7 + y * 5 + seed * 3 + x * y) % len(SERVERS)]
    screen[bottom][left] = PLINTH_L
    screen[bottom][left + width - 1] = PLINTH_R
    for x in range(left + 1, left + width - 1):
        screen[bottom][x] = PLINTH


def frame(screen, left, top, right, bottom, fill):
    for x in range(left + 1, right):
        screen[top][x] = FR_T
        screen[bottom][x] = FR_B
    for y in range(top + 1, bottom):
        screen[y][left] = FR_L
        screen[y][right] = FR_R
        for x in range(left + 1, right):
            screen[y][x] = fill(x, y)
    screen[top][left], screen[top][right] = FR_TL, FR_TR
    screen[bottom][left], screen[bottom][right] = FR_BL, FR_BR


def shell(camera_cols, conduits):
    screen = [[VOID for _ in range(40)] for _ in range(24)]
    for x in range(40):
        screen[2][x] = CEIL_JOINT if x % 8 == 3 else CEIL
        screen[3][x] = CEIL_JOINT if x % 8 == 3 else CEIL_EDGE
        screen[23][x] = EDGE
    for x in conduits:
        for y in range(4, 22):
            screen[y][x] = CONDUIT
    for x in camera_cols:
        screen[4][x] = CAM
    return screen


def floor(screen, specials=()):
    """Row 22: open FLOOR where a camera can see it, SHADOW under racks."""
    for x in range(40):
        screen[22][x] = FLOOR if screen[21][x] in (VOID, CONDUIT) else SHADOW
    for x, tile in specials:
        screen[22][x] = tile


# Per room, two camera slots: (col, sweep min, sweep max, frames per column,
# pause at each end, footprint half-width, gated by the room-1 switch).
# Sweep limits are the footprint centre column; a slot with col 0 is unused.
CAMERAS = (
    ((20, 11, 28, 10, 90, 2, 0), (0, 0, 0, 1, 1, 0, 0)),
    ((14, 10, 19, 5, 10, 2, 1), (31, 23, 38, 9, 35, 2, 0)),
    ((20, 9, 26, 8, 30, 2, 0), (35, 30, 38, 7, 100, 2, 0)),
)
CONE_TOP, CONE_ROWS = 5, 17   # rows 5..21, apex under the camera tile


def camera_table():
    out = bytearray()
    for slots in CAMERAS:
        for col, lo, hi, speed, pause, hw, gate in slots:
            hwstep = round((hw * 4 + 1) * 256 / CONE_ROWS)
            assert hwstep < 256 and (not col or lo <= hi)
            out += bytes((col, lo, hi, speed, pause, hw, gate, hwstep))
    step = [round(d * 4 * 256 / CONE_ROWS) & 0xFFFF for d in range(-40, 40)]
    return bytes(out), bytes(v & 255 for v in step) + bytes(v >> 8 for v in step)


def room(number):
    if number == 0:  # shelter racks 1..13, camera strip 14..25, racks 26..37
        screen = shell((20,), (15, 25))
        rack(screen, 1, 6, width=7, seed=0)
        rack(screen, 8, 5, seed=1)
        frame(screen, 16, 7, 23, 12, lambda x, y: TXT_A if (x + y) % 3 else TXT_B)
        rack(screen, 26, 7, seed=2)
        rack(screen, 32, 5, seed=3)
        floor(screen)
    elif number == 1:  # switch col 6, open segment A 9..19, shelter 20..23, segment B
        screen = shell((14, 31), (9, 25))
        rack(screen, 1, 5, width=4, seed=4)
        for y in range(16, 19):
            screen[y][6] = PANEL
        frame(screen, 11, 8, 17, 13, lambda x, y: TXT_B if (x + y) % 3 else TXT_A)
        rack(screen, 20, 5, width=4, seed=6)
        rack(screen, 28, 7, width=5, seed=7)
        rack(screen, 36, 6, width=4, seed=8)
        floor(screen, ((6, SWITCH),))
    else:  # zone 1 7..25 with cover 12..16, shelter 26..31, zone 2, relay 33, hatch 38
        screen = shell((20, 35), (8, 24))
        rack(screen, 1, 6, seed=9)
        rack(screen, 12, 8, width=5, seed=10)
        frame(screen, 17, 6, 23, 11, lambda x, y: TXT_B if (x * y) % 3 else TXT_A)
        rack(screen, 26, 5, seed=11)
        frame(screen, 32, 10, 36, 15, lambda x, y: TXT_A if y % 2 else TXT_B | INV)
        for y in range(13, 22):
            screen[y][38] = WALL
            screen[y][39] = WALL
        floor(screen, ((33, RELAY), (38, WALL)))
    for slots in CAMERAS[number]:
        if slots[0]:
            assert screen[4][slots[0]] == CAM

    assert all(len(row) == 40 for row in screen) and len(screen) == 24
    return bytes(tile for row in screen for tile in row)


PALETTE = {0: (0, 0, 0), 1: (170, 170, 170), 2: (48, 72, 168), 3: (160, 88, 24), 4: (200, 56, 40)}


def preview(path, number, cat=None):
    """PNG at roughly the real ANTIC 4 pixel aspect (8:5)."""
    from PIL import Image
    data = room(number)
    image = Image.new("RGB", (160 * 8, 192 * 5))
    for row in range(24):
        for col in range(40):
            tile = data[row * 40 + col]
            glyph = TILE_GLYPHS[tile & 127]
            for y, line in enumerate(glyph):
                for x, value in enumerate(line):
                    value = int(value)
                    colour = PALETTE[4 if value == 3 and tile & INV else value]
                    px, py = (col * 4 + x) * 8, (row * 8 + y) * 5
                    image.paste(colour, (px, py, px + 8, py + 5))
    if cat is not None:
        x0, img = cat
        for y, line in enumerate(img):
            for x, value in enumerate(line):
                if value:
                    colour = (240, 168, 80) if value == 1 else (70, 30, 8)
                    px, py = (x0 + x) * 8, (184 - 24 + y) * 5
                    image.paste(colour, (px, py, px + 8, py + 5))
    image.save(path)


if __name__ == "__main__":
    (ROOT / "assets" / "level-rooms.pic").write_bytes(
        b"".join(room(number) for number in range(3))
    )
    cams, steps = camera_table()
    (ROOT / "assets" / "level-cams.bin").write_bytes(cams)
    (ROOT / "assets" / "cone-step.bin").write_bytes(steps)
    import sys
    if len(sys.argv) > 1:
        sys.path.insert(0, str(ROOT / "tools"))
        from make_cat_sprite import drawing
        for n in range(3):
            preview(Path(sys.argv[1]) / f"room{n}.png", n, (40 + 30 * n, drawing(0, 3)))
