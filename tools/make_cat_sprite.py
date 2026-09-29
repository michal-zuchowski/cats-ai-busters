"""Ginger agent cat as four single-line PMG players (16x24, two colours).

Walk rotoscoped from a treadmill cat-walk profile video: one stride is twelve
drawings, one per colour clock of travel, the same distance a planted paw
slides back, so paws stay planted.  Player pixels are 2:1 like the playfield.
Frame layout (128 bytes each): P0 dark-left, P1 dark-right, P2 fur-left, P3 fur-right,
32 bytes per player (24 used).  Frames 0-11 walk right, 12-13 idle right,
14-27 the same mirrored for walking left.
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


def frames():
    right = [drawing(0, k) for k in range(12)]  # tail held still while walking
    right += [drawing(0), drawing(2)]
    left = [[row[::-1] for row in f] for f in right]
    return right + left


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
    data = b"".join(pack(f) for f in imgs)
    assert len(data) == 28 * 128
    (ROOT / "assets" / "cat-sprites.bin").write_bytes(data)
    import sys
    if len(sys.argv) > 1:
        preview(Path(sys.argv[1]), imgs)
