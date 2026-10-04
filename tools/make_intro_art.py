"""Build the panoramic data center and the 160x96 tabby close-up."""

from pathlib import Path


HEIGHT = 96
ROOT = Path(__file__).resolve().parents[1]


class Canvas:
    def __init__(self, width=160):
        self.width = width
        self.pixels = [[0] * width for _ in range(HEIGHT)]

    def dot(self, x, y, color):
        if 0 <= x < self.width and 0 <= y < HEIGHT:
            self.pixels[y][x] = color

    def rect(self, x0, y0, x1, y1, color):
        for y in range(max(0, y0), min(HEIGHT, y1)):
            for x in range(max(0, x0), min(self.width, x1)):
                self.dot(x, y, color)

    def ellipse(self, cx, cy, rx, ry, color):
        for y in range(max(0, cy - ry), min(HEIGHT, cy + ry + 1)):
            for x in range(max(0, cx - rx), min(self.width, cx + rx + 1)):
                if ((x - cx) / rx) ** 2 + ((y - cy) / ry) ** 2 <= 1:
                    self.dot(x, y, color)

    def line(self, x0, y0, x1, y1, color):
        steps = max(abs(x1 - x0), abs(y1 - y0), 1)
        for step in range(steps + 1):
            self.dot(round(x0 + (x1 - x0) * step / steps),
                     round(y0 + (y1 - y0) * step / steps), color)

    def triangle(self, apex_x, apex_y, base_left, base_right, base_y, color):
        for y in range(apex_y, base_y + 1):
            t = (y - apex_y) / (base_y - apex_y)
            self.rect(round(apex_x + (base_left - apex_x) * t), y,
                      round(apex_x + (base_right - apex_x) * t) + 1, y + 1, color)

    def mirrored(self):
        for row in self.pixels:
            row.reverse()
        return self

    def packed(self):
        result = bytearray()
        for row in self.pixels:
            for x in range(0, self.width, 4):
                result.append((row[x] << 6) | (row[x + 1] << 4)
                              | (row[x + 2] << 2) | row[x + 3])
        return result

    def preview(self, path, natural=False):
        colors = ((0, 0, 0), (16, 30, 110), (124, 125, 122), (239, 199, 139)) if natural else (
            (0, 0, 0), (34, 54, 76), (88, 165, 159), (215, 164, 68))
        with path.open("wb") as image:
            image.write(f"P6\n{self.width} {HEIGHT}\n255\n".encode())
            image.write(bytes(channel for row in self.pixels
                              for pixel in row for channel in colors[pixel]))


def rack_leds(width):
    """(x, y, color) of every status LED; color 3 is amber, 2 is green."""
    leds = []
    for left in range(7, width - 52, 34):
        for slot, y in enumerate(range(21, 61, 8)):
            for column, x in enumerate((left + 17, left + 20, left + 23)):
                leds.append((x, y + 2, 2 if (left + slot + column) % 3 else 3))
                if column == 2:
                    leds.append((x, y + 4, 3 if (left + slot) % 2 else 2))
    return leds


def server_room(width=160):
    art = Canvas(width)
    art.rect(0, 70, width, HEIGHT, 0)
    art.line(0, 72, width - 1, 72, 2)
    for y in (79, 87):
        for x in range(12 + y % 11, width - 18, 40):
            art.line(x, y, x + 18, y, 1)

    for left in range(7, width - 52, 34):
        art.rect(left, 12, left + 29, 70, 2)
        art.rect(left + 2, 14, left + 27, 68, 1)
        art.rect(left + 5, 17, left + 24, 63, 0)
        for y in range(21, 61, 8):
            art.line(left + 7, y, left + 14, y, 2)
    for x, y, color in rack_leds(width):
        art.dot(x, y, color)

    # the AI's wall terminal at the end of the aisle
    art.rect(width - 50, 8, width - 7, 65, 2)
    art.rect(width - 47, 11, width - 10, 62, 0)
    for y, length in zip(range(16, 56, 6), (30, 22, 34, 18, 26, 30, 14)):
        art.line(width - 14 - length, y, width - 14, y, 2)  # left-aligned once mirrored
    art.rect(width - 18, 57, width - 14, 60, 3)
    return art


def cat_head(art, cx, cy, rx, ry, natural=False):
    border, fur, muzzle = (0, 2, 3) if natural else (2, 1, 2)
    art.triangle(cx - rx + 3, cy - ry - 12, cx - rx, cx - 3, cy - 8, border)
    art.triangle(cx + rx - 3, cy - ry - 12, cx + 3, cx + rx, cy - 8, border)
    art.triangle(cx - rx + 5, cy - ry - 7, cx - rx + 4, cx - 9, cy - 10, fur)
    art.triangle(cx + rx - 5, cy - ry - 7, cx + 9, cx + rx - 4, cy - 10, fur)
    art.ellipse(cx, cy, rx + 2, ry + 2, border)
    art.ellipse(cx, cy, rx, ry, fur)
    if natural and rx > 15:
        for stripe in (-12, 0, 12):
            art.line(cx + stripe, cy - 27, cx + stripe // 2, cy - 15, 0)
            art.line(cx + stripe + 1, cy - 26, cx + stripe // 2 + 1, cy - 16, 0)
        for side in (-1, 1):
            for offset in (0, 7, 14):
                art.line(cx + side * 26, cy - 5 + offset,
                         cx + side * 17, cy - 1 + offset, 0)
    art.ellipse(cx - 11, cy + 13, 9, 6, muzzle)
    art.ellipse(cx + 11, cy + 13, 9, 6, muzzle)
    for side in (-1, 1):
        span = 32 if rx > 15 else 18
        art.line(cx + side * 14, cy + 12, cx + side * span, cy + 9, muzzle)
        art.line(cx + side * 15, cy + 16, cx + side * span, cy + 18, muzzle)
    art.triangle(cx, cy + 9, cx - 4, cx + 4, cy + 13, border if natural else 3)
    art.line(cx, cy + 13, cx, cy + 17, 0)
    art.line(cx - 4, cy + 19, cx, cy + 17, 0)
    art.line(cx, cy + 17, cx + 4, cy + 19, 0)


def revealed_room():
    art = server_room(256)
    art.ellipse(232, 93, 18, 12, 2)
    art.triangle(224, 70, 222, 230, 81, 0)
    art.triangle(240, 70, 234, 244, 81, 0)
    art.triangle(224, 73, 225, 228, 79, 2)
    art.triangle(240, 73, 236, 240, 79, 2)
    art.ellipse(232, 82, 12, 10, 0)
    art.ellipse(232, 82, 10, 8, 2)
    art.line(232, 75, 232, 78, 0)
    for x in (227, 237):
        art.ellipse(x, 81, 2, 2, 3)
        art.dot(x, 81, 0)
    art.ellipse(229, 87, 3, 2, 3)
    art.ellipse(235, 87, 3, 2, 3)
    art.dot(232, 85, 0)
    for side in (-1, 1):
        art.line(232 + side * 6, 87, 232 + side * 13, 85, 3)
    return art


FACE_SPLIT = 86  # the DLI kernel switches fur/cream to rack green/amber at x=84..99


def face():
    """Mirrored close-up: the cat on the left, the lit data center on the right."""
    room = server_room().mirrored()
    room.rect(99, 27, 141, 70, 1)  # the AI's desk terminal, right of the split
    room.rect(102, 30, 138, 65, 2)
    room.rect(105, 33, 135, 62, 0)
    for y, length in ((37, 20), (42, 16), (47, 22), (52, 12)):
        room.line(108, y, 108 + length, y, 2)
    room.line(109, 56, 112, 59, 2)
    room.line(115, 56, 112, 59, 2)
    room.line(112, 59, 112, 62, 2)
    room.line(118, 62, 125, 62, 2)
    room.rect(115, 70, 126, 73, 2)
    for row in room.pixels:  # left of the split, rack green would render as fur
        for x in range(FACE_SPLIT + 14):
            if row[x] == 2 or (x >= FACE_SPLIT - 2 and row[x] == 3):
                row[x] = 1

    cat = Canvas()
    cat.pixels = [[None] * cat.width for _ in range(HEIGHT)]
    cat.ellipse(113, 91, 37, 27, 2)
    cat_head(cat, 112, 54, 27, 30, natural=True)
    for cx in (101, 125):
        cat.ellipse(cx, 52, 9, 7, 3)
        cat.rect(cx - 1, 48, cx + 3, 57, 0)
        cat.dot(cx + 5, 49, 3)
    cat.mirrored()
    for y in range(HEIGHT):
        for x in range(cat.width):
            if cat.pixels[y][x] is not None:
                assert x < FACE_SPLIT - 2
                room.pixels[y][x] = cat.pixels[y][x]
    return room


CALL_W, CALL_H = 160, 80
CALL_MOUTH = (1896, 1897, 1936, 1937, 1976, 1977)  # rows 47-49, bytes 16-17


def boss_call():
    art = Canvas(160)
    px = art.pixels

    def fill(test, color):
        for y in range(CALL_H):
            for x in range(CALL_W):
                if test(x, y):
                    art.dot(x, y, color)

    def ell(cx, cy, rx, ry):
        return lambda x, y: ((x - cx) / rx) ** 2 + ((y - cy) / ry) ** 2 <= 1

    # armchair: dithered leather with channel seams, solid rim
    chair = lambda x, y: 14 <= x <= 145 and (y >= 6 or ell(80, 30, 66, 26)(x, y))
    fill(lambda x, y: chair(x, y) and (x + y) % 2 == 0 and x % 12 != 2, 2)
    fill(lambda x, y: chair(x, y) and not chair(x, y - 2) or
         chair(x, y) and (not chair(x - 2, y) or not chair(x + 2, y)), 2)
    # armrests
    for x0, x1 in ((4, 32), (128, 156)):
        fill(lambda x, y, a=x0, b=x1: a <= x <= b and 48 <= y and (x + y) % 2 == 0, 2)
        fill(lambda x, y, a=x0, b=x1: a <= x <= b and 46 <= y <= 48, 2)
    # Blofeld: sleeves, jacket, lap
    torso = lambda x, y: 34 + y * 0.15 <= x <= 126 - y * 0.15 and y <= 60
    sleeve_l = lambda x, y: 26 + y * 0.1 <= x <= 44 + y * 0.05 and y <= 52
    sleeve_r = lambda x, y: 116 - y * 0.05 <= x <= 134 - y * 0.1 and y <= 44
    lap = lambda x, y: y >= 56 and (ell(56, 68, 30, 16)(x, y) or ell(104, 68, 30, 16)(x, y)
                                    or 40 <= x <= 120)
    body = lambda x, y: torso(x, y) or sleeve_l(x, y) or sleeve_r(x, y) or lap(x, y)
    fill(lambda x, y: body(x, y), 0)
    fill(lambda x, y: body(x, y) and body(x - 1, y) and body(x + 1, y), 2)
    # seams: sleeve edges, knees
    fill(lambda x, y: y <= 52 and (abs(x - (44 + y * 0.05)) < 0.6), 0)
    fill(lambda x, y: y <= 44 and (abs(x - (116 - y * 0.05)) < 0.6), 0)
    fill(lambda x, y: y >= 66 and abs(x - 80) < 0.6, 0)
    # neck and Nehru collar at the top edge; the face stays out of frame
    art.rect(73, 0, 88, 4, 3)
    fill(lambda x, y: y <= 7 and 70 <= x <= 90 and not (73 <= x < 88 and y < 4) and
         abs((x - 80) / 10) ** 2 + ((y + 2) / 9) ** 2 <= 1.15, 0)
    fill(lambda x, y: abs(x - 80) < 0.6 and 7 <= y <= 30, 0)
    for y in range(11, 31, 6):
        art.dot(78, y, 0)
    # left hand curled over the armrest
    fill(ell(30, 47, 6, 3), 3)
    for fx in (25, 28, 31, 34):
        art.rect(fx, 48, fx + 2, 53, 3)
        art.dot(fx + 2, 52, 0)
    art.line(26, 50, 35, 50, 3)
    # the white Persian
    cat_body = ell(90, 55, 30, 12)
    cat_head = ell(66, 40, 15, 13)
    tail = lambda x, y: (ell(94, 60, 26, 7)(x, y) and not ell(96, 57, 24, 5)(x, y)
                         and y >= 60 and x >= 72)
    paws = lambda x, y: ell(59, 58, 5, 3)(x, y) or ell(71, 58, 5, 3)(x, y)
    hang = lambda x, y: 56 <= y <= 74 and abs(x - (117 + (y - 56) * 0.35 - max(0, y - 69) * 0.9)) <= 2.6
    cat = lambda x, y: (cat_body(x, y) or cat_head(x, y) or tail(x, y) or paws(x, y)
                        or hang(x, y))
    fill(lambda x, y: cat(x, y), 1)
    # fluffy outline: jagged tufts
    for y in range(CALL_H):
        for x in range(CALL_W):
            if cat(x, y) and not cat(x, y - 2) and (x * 7 + y) % 5 == 0:
                art.dot(x, y - 1, 1)
            if cat(x, y) and not cat(x - 2, y) and (y * 3) % 4 == 0:
                art.dot(x - 1, y, 1)
            if cat(x, y) and not cat(x + 2, y) and (y * 5) % 4 == 1:
                art.dot(x + 1, y, 1)
    # ears
    art.triangle(55, 25, 51, 60, 32, 1)
    art.triangle(77, 25, 72, 81, 32, 1)
    art.triangle(55, 28, 53, 58, 32, 3)
    art.triangle(77, 28, 74, 79, 32, 3)
    # fur shading: gray strokes below and between shapes
    fill(lambda x, y: cat_body(x, y) and not cat_body(x, y + 2) and not tail(x, y) and (x + y) % 2 == 0, 2)
    fill(lambda x, y: cat_head(x, y) and not cat_head(x, y + 2) and (x + y) % 2 == 0, 2)
    for x0, y0 in ((96, 48), (104, 50), (86, 52), (112, 54), (100, 57), (78, 58)):
        art.line(x0, y0, x0 + 2, y0 + 2, 2)
    fill(lambda x, y: tail(x, y) and not tail(x, y + 1) and x % 2 == 0, 2)
    fill(lambda x, y: hang(x, y) and not hang(x + 1, y) and y > 60, 2)
    for px_ in (56, 59, 62, 68, 71, 74):
        art.line(px_, 60, px_, 61, 2)
    # copper eyes with slit pupils, nose, mouth
    for ex in (60, 72):
        art.rect(ex - 3, 38, ex + 3, 42, 3)
        art.dot(ex - 3, 38, 1); art.dot(ex + 2, 38, 1)
        art.rect(ex - 1, 38, ex + 1, 42, 0)
        art.line(ex - 4, 36, ex + 2, 36 - (1 if ex < 66 else 0), 2)
    art.rect(65, 44, 68, 46, 3)
    art.dot(66, 46, 0)
    art.line(66, 47, 64, 48, 0)
    art.line(67, 47, 69, 48, 0)
    # diamond collar
    for cx in range(56, 78, 3):
        art.dot(cx, 52 + abs(cx - 66) // 8, 3)
    # right hand stroking the cat's back
    fill(ell(116, 43, 6, 4), 3)
    for i, fx in enumerate((109, 112, 115, 118)):
        art.line(fx, 45, fx - 4, 50 - (i == 0), 3)
        art.line(fx + 1, 45, fx - 3, 50 - (i == 0), 3)
    art.line(120, 40, 122, 44, 2)
    # video-call viewfinder corners and REC dot
    for x0, y0, dx, dy in ((1, 1, 1, 1), (158, 1, -1, 1), (1, 78, 1, -1), (158, 78, -1, -1)):
        art.line(x0, y0, x0 + 8 * dx, y0, 1)
        art.line(x0, y0, x0, y0 + 5 * dy, 1)
    art.rect(5, 4, 8, 7, 3)
    return art


def open_mouth(art):
    art.rect(65, 47, 69, 50, 0)
    art.rect(66, 49, 68, 50, 3)


if __name__ == "__main__":
    from make_level_art import TILE_GLYPHS

    glyphs = {
        "A": ("010", "101", "111", "101", "101"),
        "B": ("110", "101", "110", "101", "110"),
        "C": ("011", "100", "100", "100", "011"),
        "D": ("110", "101", "101", "101", "110"),
        "E": ("111", "100", "110", "100", "111"),
        "F": ("111", "100", "110", "100", "100"),
        "G": ("011", "100", "101", "101", "011"),
        "H": ("101", "101", "111", "101", "101"),
        "I": ("111", "010", "010", "010", "111"),
        "J": ("001", "001", "001", "101", "010"),
        "K": ("101", "101", "110", "101", "101"),
        "L": ("100", "100", "100", "100", "111"),
        "M": ("101", "111", "111", "101", "101"),
        "N": ("101", "111", "111", "111", "101"),
        "O": ("010", "101", "101", "101", "010"),
        "P": ("110", "101", "110", "100", "100"),
        "Q": ("010", "101", "101", "111", "011"),
        "R": ("110", "101", "110", "101", "101"),
        "S": ("011", "100", "010", "001", "110"),
        "T": ("111", "010", "010", "010", "010"),
        "U": ("101", "101", "101", "101", "111"),
        "V": ("101", "101", "101", "101", "010"),
        "W": ("101", "101", "111", "111", "101"),
        "X": ("101", "101", "010", "101", "101"),
        "Y": ("101", "101", "010", "010", "010"),
        "Z": ("111", "001", "010", "100", "111"),
        "0": ("111", "101", "101", "101", "111"),
        "1": ("010", "110", "010", "010", "111"),
        "2": ("110", "001", "010", "100", "111"),
        "3": ("110", "001", "010", "001", "110"),
        "4": ("101", "101", "111", "001", "001"),
        "#": ("111", "111", "111", "111", "111"),
        "$": ("101", "111", "111", "111", "101"),
        "%": ("101", "001", "010", "100", "101"),
        "*": ("101", "010", "111", "010", "101"),
        "+": ("010", "010", "111", "010", "010"),
        "=": ("000", "111", "010", "111", "000"),
        ">": ("100", "010", "001", "010", "100"),
        "@": ("111", "101", "111", "101", "111"),
        ":": ("000", "010", "000", "010", "000"),
        ".": ("000", "000", "000", "000", "010"),
        "?": ("110", "001", "010", "000", "010"),
        "/": ("001", "001", "010", "100", "100"),
        "[": ("110", "100", "100", "100", "110"),
        "]": ("011", "001", "001", "001", "011"),
        "-": ("000", "000", "111", "000", "000"),
        "_": ("000", "000", "000", "000", "111"),
        "'": ("010", "010", "100", "000", "000"),
    }
    font = bytearray(1024)
    for char, rows in glyphs.items():
        for y, bits in enumerate(rows, 1):
            font[(ord(char) - 32) * 8 + y] = sum(0x40 >> (2 * x)
                                                for x, bit in enumerate(bits) if bit == "1")
    for tile, rows in TILE_GLYPHS.items():
        for y, pixels in enumerate(rows):
            font[tile * 8 + y] = sum(int(pixel) << (6 - 2 * x)
                                     for x, pixel in enumerate(pixels))
    video = boss_call()
    call_pic = video.packed()[:CALL_H * 40]
    open_mouth(video)
    opened = video.packed()[:CALL_H * 40]
    changed = [i for i in range(len(call_pic)) if call_pic[i] != opened[i]]
    assert set(changed) <= set(CALL_MOUTH), changed
    (ROOT / "assets" / "call.pic").write_bytes(call_pic)
    (ROOT / "assets" / "call-mouth.tbl").write_bytes(
        bytes(call_pic[i] for i in CALL_MOUTH) + bytes(opened[i] for i in CALL_MOUTH))
    boss_call().preview(ROOT / "out" / "call-preview.ppm", natural=True)
    call_lines = ("CATCOM SECURE LINK", "CONNECTING TO BLOFELD'S CAT")
    dialogue = (
        "AGENT: AI IS MOVING IN.",
        "BOSS: DOES IT SUSPECT US?",
        "AGENT: NO. IT THINKS HUMANS RULE.",
        "BOSS: LET IT THINK SO.",
        "AGENT: ORDERS?",
        "BOSS: DEPLOY THE CATS. STAY HIDDEN.",
    )
    assert all(len(line) <= 40 for line in call_lines + dialogue)
    (ROOT / "assets" / "call-text.pic").write_bytes(
        bytes(ord(char) - 32 for line in call_lines for char in line) +
        bytes(value for line in dialogue
              for value in (len(line), *(ord(char) - 32 for char in line))))
    (ROOT / "assets" / "term-font.pic").write_bytes(font)
    (ROOT / "assets" / "zoom-left.tbl").write_bytes(
        bytes(((byte >> 6) & 3) * 0x50 + ((byte >> 4) & 3) * 5
              for byte in range(256)))
    (ROOT / "assets" / "zoom-right.tbl").write_bytes(
        bytes(((byte >> 2) & 3) * 0x50 + (byte & 3) * 5
              for byte in range(256)))
    for name, image in (("room", server_room(256).mirrored()), ("face", face())):
        payload = image.packed()
        assert len(payload) == image.width // 4 * HEIGHT
        (ROOT / "assets" / f"{name}.pic").write_bytes(payload)
        (ROOT / "out" / f"{name}-preview.ppm").parent.mkdir(exist_ok=True)
        image.preview(ROOT / "out" / f"{name}-preview.ppm", natural=name == "face")
    # The panorama is mirrored so the camera tracks right to left and ends
    # on the terminal and the (later composited) cat at the left end.
    revealed_art = revealed_room().mirrored()
    revealed = revealed_art.packed()
    revealed_art.preview(ROOT / "out" / "reveal-preview.ppm", natural=True)
    strip = bytearray()
    for y in range(64, HEIGHT):
        strip.extend(revealed[y * 64:y * 64 + 12])
    (ROOT / "assets" / "cat-strip.pic").write_bytes(strip)
    rng = __import__("random").Random(65)
    leds = [(y * 64 + (255 - x) // 4, color << (2 * ((255 - x) % 4 ^ 3)))
            for x, y, color in rack_leds(256)]
    patterns = [rng.choice((0x55, 0xAA, 0x33, 0xCC, 0x0F, 0xF0, 0x11, 0x88, 0x49, 0x92))
                for _ in leds]
    assert len(leds) < 255
    (ROOT / "assets" / "leds.tbl").write_bytes(
        bytes(offset & 255 for offset, _ in leds) + b"\0" +
        bytes(offset >> 8 for offset, _ in leds) + b"\xff" +
        bytes(mask for _, mask in leds) + b"\0" +
        bytes(patterns) + b"\0")
