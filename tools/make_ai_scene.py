"""Streamed ANTIC D AI monitor: 160x80 bitmap and two 24x5 mouth patches."""
from pathlib import Path

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parents[1]
PALETTE = ((10, 16, 25), (36, 100, 142), (236, 229, 203), (236, 163, 69))
MOUTH = (64, 56, 88, 61)


def pack(image):
    pixels = [image.getpixel((x, y)) for y in range(image.height) for x in range(image.width)]
    assert image.width % 4 == 0 and all(0 <= p <= 3 for p in pixels)
    return bytes((pixels[i] << 6) | (pixels[i + 1] << 4) | (pixels[i + 2] << 2) | pixels[i + 3]
                 for i in range(0, len(pixels), 4))


def portrait():
    image = Image.new("P", (160, 80))
    image.putpalette([v for rgb in PALETTE for v in rgb] + [0] * (768 - 12))
    d = ImageDraw.Draw(image)
    d.rectangle((8, 3, 151, 77), outline=1, width=2)
    d.rectangle((12, 7, 147, 70), outline=2)
    for y in range(13, 66, 8):
        d.line((16, y, 43, y), fill=1)
        d.line((112, y, 143, y), fill=1)
        d.rectangle((19, y - 1, 21, y + 1), fill=3)
        d.rectangle((137, y - 1, 139, y + 1), fill=3)
    d.polygon(((49, 17), (104, 17), (110, 23), (110, 58), (98, 66),
               (55, 66), (43, 58), (43, 23)), fill=1, outline=2)
    d.rectangle((49, 23, 104, 47), fill=0)
    d.line((51, 29, 68, 34), fill=3, width=2)
    d.line((84, 34, 102, 29), fill=3, width=2)
    d.rectangle((56, 36, 69, 39), fill=2)
    d.rectangle((84, 36, 97, 39), fill=2)
    d.line((76, 39, 76, 49), fill=0, width=2)
    d.rectangle((62, 54, 89, 62), fill=0)
    for x in range(24, 133, 12):
        d.rectangle((x, 73, x + 5, 74), fill=2 if x % 24 else 3)
    closed = Image.new("P", (24, 5))
    c = ImageDraw.Draw(closed)
    c.line((0, 2, 23, 2), fill=3)
    for x in range(2, 24, 5):
        c.line((x, 1, x, 3), fill=2)
    opened = Image.new("P", (24, 5))
    o = ImageDraw.Draw(opened)
    o.rectangle((0, 0, 23, 4), outline=3)
    for x in range(2, 24, 4):
        o.line((x, 1, x, 2 if x % 8 else 3), fill=2)
    image.paste(closed, MOUTH[:2])
    return image, closed, opened


if __name__ == "__main__":
    image, closed, opened = portrait()
    data = pack(image) + pack(closed) + pack(opened)
    assert len(data) == 3260
    (ROOT / "assets/ai-portrait.pic").write_bytes(data)
    out = ROOT / "out"
    out.mkdir(exist_ok=True)
    sheet = Image.new("RGB", (640, 160))
    for n, pose in enumerate((closed, opened)):
        frame = image.copy()
        frame.paste(pose, MOUTH[:2])
        sheet.paste(frame.convert("RGB").resize((320, 160), Image.Resampling.NEAREST), (n * 320, 0))
    sheet.save(out / "ai-cutscene-preview.png")
