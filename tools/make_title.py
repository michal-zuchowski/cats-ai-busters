"""Pack the approved four-color title into 188 ANTIC E rows; the last four are blank."""
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
PALETTE = ((10, 16, 25), (36, 57, 76), (236, 229, 203), (236, 163, 69))


def title_bitmap():
    with Image.open(ROOT / "assets/title-screen.png") as image:
        assert image.size == (160, 192), "Title must be native 160x192"
        rgb = image.convert("RGB")
        pixels = [PALETTE.index(rgb.getpixel((x, y))) for y in range(192) for x in range(160)]
    assert not any(pixels[188 * 160:]), "Last four rows must be blank for the display-list gap"
    return bytes((pixels[i] << 6) | (pixels[i + 1] << 4) | (pixels[i + 2] << 2) | pixels[i + 3]
                 for i in range(0, 188 * 160, 4))


if __name__ == "__main__":
    data = title_bitmap()
    assert len(data) == 7520
    (ROOT / "assets/title-screen.pic").write_bytes(data)
