"""Ginger agent cat as four single-line PMG players (16x24, two colours).

Walk rotoscoped from a treadmill cat-walk profile video: one stride is twelve
drawings, one per colour clock of travel, the same distance a planted paw
slides back, so paws stay planted.  Player pixels are 2:1 like the playfield.
Frame layout (128 bytes each): P0 dark-left, P1 dark-right, P2 fur-left, P3 fur-right,
32 bytes per player (24 used).  Frames 0-11 walk right, 12-13 idle right,
14-27 the same mirrored for walking left; 28-35 leap right, 36-43 leap left;
44-45 raised forepaw / hind kick right, 46-47 mirrored left;
48-49 downward forepaw strike right / left; 50-51 seated on a vacuum roof right / left;
52-53 seated hind-leg shove right / left (toes at (0,23),(1,23), mirrored 15-x).
Logical frames are packed: byte-identical drawings share one 128-byte physical
frame and assets/cat-frame-map.bin maps logical -> physical (main.k65 FrameMap).
"""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
W, H = 16, 24
FILL, DARK = 1, 2

UPPER = (  # rows 0..14 facing right: round head with ears, tabby stripes
    "................",
    "................",
    "................",
    "................",
    "...........F..F.",
    "...........FF.FF",
    "...........FFFFF",
    "..........FFF.FF",  # eye: a see-through slit, the dark room shows through
    "...FFFFFF.FFF.FF",
    "..FdFFdFFdFFFFFF",
    "..FFdFFdFFdFFFFd",
    "..FFFFFFFFFFFFF.",
    "..FFFFFFFFFFFd..",
    "..FFFFFFFFFFF...",
    "...FFFFFFFFFF...",
)
TAIL = (  # rows 0..7, cols 0..4: long tail up with a hooked tip; flicks only at rest
    (".Fd..", "F....", "F....", "F....", "F....", ".F...", ".F...", "..F.."),
    ("..Fd.", ".F...", ".F...", ".F...", "F....", "F....", ".F...", "..F.."),
    (".....", "Fd...", "F....", "F....", "F....", ".F...", ".F...", "..F.."),
)
# Paw track over one stride (12 drawings, 1 px = 1 colour clock of travel
# each), traced from the video: (x, lifted) for a hind and a fore paw.  The far legs run the
# same track half a stride later; the near fore lags the near hind by a
# quarter stride, the cat's lateral-sequence walk.
HIND = ((3.5, 0), (3, 0), (2, 0), (1, 0), (1, 1), (2, 2), (5, 2), (6.5, 1),
        (7.5, 0), (7, 0), (6, 0), (5, 0))
FORE = ((12.5, 0), (12, 0), (11, 0), (10, 0), (9, 0), (8, 0), (8, 0), (9, 1),
        (11, 2), (13, 2), (14, 1), (14, 0))
DROP = 2  # rows the body sits lower: shorter legs
HIP, SHOULDER, TOP, FLOOR = 4, 11, 15 + DROP, 23


def put(img, x, y, c):
    if 0 <= x < W and 0 <= y < H:
        img[y][x] = c


def line(img, x0, y0, x1, y1, c):
    steps = max(abs(x1 - x0), abs(y1 - y0), 1)
    for i in range(steps + 1):
        put(img, round(x0 + (x1 - x0) * i / steps), round(y0 + (y1 - y0) * i / steps), c)


def hind(img, x, lift, c, o):
    x = round(x) + o
    hip = HIP + o
    paw_y = FLOOR - lift
    hock = (round((hip + x) / 2) - 1, 19 + DROP // 2)  # digitigrade: hock sits behind
    for dx in (-1, 0, 1):  # haunch
        put(img, hip + dx, TOP, c)
    put(img, hip - 1, TOP + 1, c)
    line(img, hip, TOP, *hock, c)
    line(img, *hock, x, paw_y, c)
    put(img, x + 1, paw_y, c)


def fore(img, x, lift, c, o):
    x = round(x) + o
    paw_y = FLOOR - lift
    if lift:  # wrist folds, paw tucked back under
        line(img, SHOULDER + o, TOP, x, paw_y - 1, c)
        put(img, x - 1, paw_y, c)
    else:
        line(img, SHOULDER + o, TOP, x, paw_y, c)
        put(img, x + 1, paw_y, c)


def drawing(tail, k=None):
    rows = [t + u[5:] for t, u in zip(TAIL[tail], UPPER)] + list(UPPER[8:])
    img = [[{".": 0, "F": FILL, "d": DARK}[ch] for ch in row] for row in rows]
    img = [[0] * W for _ in range(DROP)] + img
    img += [[0] * W for _ in range(H - len(img))]
    if k is None:
        legs = (((5, 0), (12, 0), FILL), ((4, 0), (11, 0), FILL))
    else:
        legs = ((HIND[(k + 6) % 12], FORE[(k + 6) % 12], FILL), (HIND[k], FORE[k], FILL))
    for (h, f, c), o in zip(legs, (0, 0)):
        hind(img, *h, c, o)
        fore(img, *f, c, o)
    return img


def leap(kind):
    """Crouch, push off, stretch, reach, contact, compress, recover, stand."""
    if kind == 7:
        return drawing(0)
    img = drawing(0)
    for y in range(TOP, H):
        img[y] = [0] * W
    for y in range(10):
        img[y][:5] = [0] * 5
    line(img, 0, 9, 2, 10, FILL)
    line(img, 2, 10, 3, 12, FILL)
    if kind in (1, 2):
        flight = [[0] * W for _ in range(H)]
        for y in range(TOP):
            for x, colour in enumerate(img[y]):
                if colour:
                    # Leave room ahead of the muzzle for straight forelegs.
                    dx = -2 if x >= 10 else (-1 if x >= 2 else 0)
                    dy = -2 if x >= 9 else (-1 if x >= 6 else 0)
                    put(flight, x + dx, y + dy, colour)
        img = flight
        line(img, 4, 14, 9, 13, FILL)
        line(img, 4, 15, 9, 14, FILL)
        hind(img, 1 if kind == 1 else 3, 2 if kind == 1 else 3, FILL, 0)
        line(img, 9, 13, 15, 12, FILL)
        line(img, 9, 15, 14, 14, FILL)
        put(img, 15, 14, FILL)
        return img
    if kind in (3, 4):
        tilted = [[0] * W for _ in range(H)]
        for y in range(TOP):
            for x, colour in enumerate(img[y]):
                if colour:
                    dy = 2 if x >= 9 else (1 if x >= 6 else (-2 if kind == 3 else -1))
                    put(tilted, x, y + dy, colour)
        img = tilted
        line(img, HIP, 15 if kind == 3 else 16, 3, 18, FILL)
        line(img, 3, 18, 5, 20, FILL)
        put(img, 6, 20, FILL)
        line(img, 11, 19, 12, 22, FILL)
        line(img, 12, 23, 13, 23, FILL)
        return img
    if kind in (5, 6):
        drop = 2 if kind == 5 else 1
        img = [[0] * W for _ in range(drop)] + img[:-drop]
        # Keep planted paws fixed while the elbows and hocks absorb the load.
        line(img, HIP, TOP + drop, 2 if kind == 5 else 3, 21, FILL)
        line(img, 2 if kind == 5 else 3, 21, 5, FLOOR, FILL)
        put(img, 6, FLOOR, FILL)
        line(img, SHOULDER, TOP + drop, 10 if kind == 5 else 11, 21, FILL)
        line(img, 10 if kind == 5 else 11, 21, 12, FLOOR, FILL)
        put(img, 13, FLOOR, FILL)
        if kind == 6:
            for y in range(11):
                img[y][:5] = [0] * 5
            line(img, 0, 6, 1, 7, FILL)
            line(img, 1, 7, 1, 10, FILL)
            line(img, 1, 10, 3, 12, FILL)
        return img
    hind(img, 5, 1, FILL, 0)
    fore(img, 11, 1, FILL, 0)
    return [[0] * W] + img[:-1]


def paw_attack(back, strike=False):
    img = [[0] * W] + drawing(0)[:-1]
    if not back:
        # Pull the head back so the raised foreleg does not merge with the face.
        for y in range(6, TOP):
            head = img[y][10:]
            img[y][10:] = [0] * 6
            for x, colour in enumerate(head, 8):
                if colour:
                    img[y][x] = colour
    for y in range(TOP, H):
        img[y] = [0] * W
    hind(img, 5, 0, FILL, 0)
    fore(img, 12, 0, FILL, 0)
    if back:
        line(img, HIP, TOP, 2, 19, FILL)
        line(img, 2, 19, 0, 19, FILL)
        put(img, 0, 20, FILL)
    else:
        if strike:
            line(img, SHOULDER, TOP, 13, 14, FILL)
            line(img, 13, 14, 15, 15, FILL)
            line(img, 15, 15, 15, 18, FILL)
            line(img, 15, 18, 14, 19, FILL)
            put(img, 13, 19, FILL)
        else:
            line(img, SHOULDER, TOP, 15, 10, FILL)
            line(img, 15, 10, 15, 4, FILL)
            put(img, 14, 3, FILL)
            put(img, 13, 4, FILL)
    return img


def seated(shove=False):
    """Upright chest, rounded seated haunch and low curled tail; roof contact unchanged."""
    rows = (
        ".........F..F...",
        ".........FF.FF..",
        ".........FFFFF..",
        "........FFF.FFF.",
        "........FFF.FFF.",
        "........FFFFFd..",
        ".........FFFF...",
        ".........FFF....",
        "........FFFF....",
        ".......FdFFF....",
        "......FFFFdF....",
        ".....FFFFFFF....",
        "....FdFFFFFF....",
        "...FFFFFFdFF....",
        "..FFFFFFFFFF....",
        ".FFFdFFFFFFF....",
        ".FFFFFFF.FFF....",
        "F.FFFFFF..FFF...",
        "F.FFdFFF....F...",
        "F.FFFFFF....F...",
        "FF.FdFFF....F...",
        ".FFFFFFFF...F...",
        "FFFFFFFFF...F...",
        "FFFFFFFFF..FFF..",
    )
    img = [[{".": 0, "F": FILL, "d": DARK}[ch] for ch in row] for row in rows]
    if shove:
        for y in range(19, 24):
            for x in range(0, 4):
                img[y][x] = 0
        line(img, 3, 19, 1, 22, FILL)
        line(img, 2, 20, 0, 23, FILL)
        put(img, 1, 23, FILL)
        put(img, 2, 22, FILL)
    return img


def frames():
    right = [drawing(0, k) for k in range(12)]  # tail held still while walking
    right += [drawing(0), drawing(2)]
    left = [[row[::-1] for row in f] for f in right]
    jr = [leap(k) for k in range(8)]
    jl = [[row[::-1] for row in f] for f in jr]
    attacks = [paw_attack(False), paw_attack(True)]
    strike = paw_attack(False, strike=True)
    sit, shove = seated(), seated(True)
    return (right + left + jr + jl + attacks
            + [[row[::-1] for row in f] for f in attacks]
            + [strike, [row[::-1] for row in strike]]
            + [sit, [row[::-1] for row in sit], shove, [row[::-1] for row in shove]])


def packed_atlas(imgs):
    """(physical bytes, logical -> physical index list); identical drawings share a frame."""
    physical, order, mapping = [], {}, []
    for img in imgs:
        data = bytes(pack(img))
        if data not in order:
            order[data] = len(physical)
            physical.append(data)
        mapping.append(order[data])
    return b"".join(physical), bytes(mapping)


def pack(img):
    out = bytearray()
    for colour in (DARK, FILL):
        for half in (0, 8):
            player = bytearray(32)
            for y in range(H):
                player[y] = sum(0x80 >> x for x in range(8) if img[y][half + x] == colour)
            out += player
    return out


def preview(path, imgs, scale=10):
    from PIL import Image
    pal = {0: (30, 40, 110), FILL: (236, 150, 60), DARK: (110, 45, 10)}
    sheet = Image.new("RGB", (len(imgs) * (W + 1) * 2 * scale, H * scale))
    for i, img in enumerate(imgs):
        for y in range(H):
            for x in range(W):
                x0 = (i * (W + 1) + x) * 2 * scale
                sheet.paste(pal[img[y][x]], (x0, y * scale, x0 + 2 * scale, (y + 1) * scale))
    sheet.save(path)


if __name__ == "__main__":
    imgs = frames()
    assert all(len(r) == W for f in imgs for r in f) and all(len(f) == H for f in imgs)
    assert len(imgs) == 54
    data, mapping = packed_atlas(imgs)
    (ROOT / "assets" / "cat-sprites.bin").write_bytes(data)
    (ROOT / "assets" / "cat-frame-map.bin").write_bytes(mapping)
    import sys
    if len(sys.argv) > 1:
        preview(Path(sys.argv[1]), imgs)
