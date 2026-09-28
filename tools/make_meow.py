from array import array
from pathlib import Path
import sys
import wave


ROOT = Path(__file__).resolve().parents[1]
RATE = 3900
START = 0.13
SAMPLES = 12 * 256


def convert():
    with wave.open(str(ROOT / "assets/meow-source.wav")) as wav:
        assert (wav.getnchannels(), wav.getsampwidth()) == (1, 2)
        source = array("h", wav.readframes(wav.getnframes()))
        source_rate = wav.getframerate()
    if sys.byteorder == "big":
        source.byteswap()

    filtered = []
    for i in range(SAMPLES):
        center = (START + i / RATE) * source_rate
        left = int(center) - 12
        weighted = total = 0.0
        for j in range(left, left + 25):
            weight = 12 - abs(j - center)
            if weight > 0 and 0 <= j < len(source):
                weighted += source[j] * weight
                total += weight
        filtered.append(weighted / total)

    peak = max(abs(value) for value in filtered)
    result = bytes(max(0, min(15, round(7.5 + 7.5 * value / peak))) for value in filtered)
    assert len(result) == SAMPLES and max(result) - min(result) >= 12
    return result


if __name__ == "__main__":
    (ROOT / "assets/meow.pcm").write_bytes(convert())
