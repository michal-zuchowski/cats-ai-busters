from pathlib import Path
import re
import wave


root = Path(__file__).resolve().parents[1]
sample = (root / "sample.k65").read_text()
main = (root / "main.k65").read_text()
project = (root / "main.k65proj").read_text()
pcm = (root / "assets/meow.pcm").read_bytes()

pages = int(re.search(r"a\?&>Meow\+(\d+) == break", main).group(1)) // 256
assert len(pcm) == pages * 256
assert all(value <= 15 for value in pcm)
assert max(pcm) - min(pcm) >= 12
with wave.open(str(root / "assets/meow-source.wav")) as wav:
    assert wav.getnframes() / wav.getframerate() > 1
assert "align 256" in sample
assert 'binary "assets/meow.pcm"' in sample
scanlines_per_sample = re.search(r"func play_meow \{(.*?)\n\}", main, re.S).group(1).count("WSYNC=a")
assert scanlines_per_sample == 3  # face_kernel also plays one sample per 3 lines
assert 0.55 < len(pcm) * scanlines_per_sample / 15625 < 0.65
assert "a|0x10" in main and "AUDC1=a=0" in main
assert "meow_ptr+1++" in main and "for x=0..63 eval [7]" in sample  # kernel overrun pad
assert "play_meow" in main and "sample.k65 main" in project
assert "func play_meow" not in sample
assert project.index("sample.k65 main") < project.index("main.k65 main")
