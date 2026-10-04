"""Original title spy-funk and four level-specific POKEY scores.

8-bit POKEY pure tones on the 64 kHz clock drift up to ~20 cents above C4,
so the lead plays on channels 1+2 joined into one 16-bit voice clocked at
1.79 MHz (AUDCTL $50): AUDF = 1789790 / (2 * f) - 7, in tune to a cent.
The bass stays 8-bit on ch3 (C3..A3 are within 3 cents there);
ch4 plays table drums (kick = falling pure tone, snare/hat = noise) and
yields to sound effects.

Layout of assets/music.bin (offsets used by main.k65):
  0 lead[256] (0 nothing, 1 cut, k >= 2: note k)  256 bass rhythm[16]
  272 drums[2*16]  304 bass low[16]  320 bass high[16]
  336 note AUDF1[16]  352 note AUDF2[16]  368 drum AUDF[4*8]  400 drum AUDC[4*8]
  432 frames/step   433 lead attack volume   434 bass attack volume
Each 435-byte track is streamed into a shared 512-byte RAM buffer.
"""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
N = {"C3": 243, "D3": 217, "G3": 162, "Bb3": 136, "F3": 182, "A3": 144, "C4": 121, "C#4": 114, "D4": 108,
     "E4": 96, "F4": 91, "G4": 81, "A4": 72, "Bb4": 68, "C5": 60, "C#5": 57, "D5": 53,
     "E5": 47, "F5": 45, "G5": 40, "A5": 35}
HOLD, REST = 0, 1


def audf16(name):
    step = {"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}[name[0]]
    step += name.count("#") - name.count("b")
    midi = 12 * (int(name[-1]) + 1) + step
    return round(1789790 / (2 * 440 * 2 ** ((midi - 69) / 12)) - 7)

# 16 bars: Dm F C A (two motifs), Gm Bb F A (bridge), Dm F C A (third motif). POKEY's 64 kHz pure tones drift out of tune above
# C5 (D5 +13, E5 +17 cents), so the lead stays in D4..C5 and every note is
# a short pluck: busy 16th figures instead of held notes. "-" = nothing new.
LEAD = """
D4 - D4 F4 - A4 - D4 - - C5 - A4 - F4 -
F4 - F4 A4 - C5 - F4 - - A4 - C5 A4 - -
E4 - E4 G4 - C5 - E4 - - G4 - E4 - C4 -
C#4 - E4 - A4 - E4 - C#4 E4 A4 - G4 - E4 -
A4 - A4 - G4 A4 - F4 - G4 - E4 - F4 D4 -
C5 - C5 - A4 - F4 - A4 C5 - A4 - F4 - -
G4 - G4 E4 - C4 - E4 G4 - C5 - Bb4 - G4 -
A4 - - C#4 - E4 - A4 - G4 - E4 - C#4 - -
G4 - Bb4 - D5 - - . D5 C5 Bb4 - A4 - G4 -
F4 - Bb4 - D5 - F5 - - D5 - Bb4 - C5 - -
C5 - A4 - F4 - A4 C5 - F5 - E5 - C5 - -
E5 - C#5 - A4 - E5 - C#5 - A4 - G4 - E4 -
D5 - - D5 - C5 - A4 - - F4 - A4 - C5 -
A4 - - A4 - F4 - C5 - - A4 - F4 - E4 -
G4 - - G4 - E4 - C5 - - E5 - D5 - C5 -
C#5 - E5 - A4 - - E4 - C#4 - E4 - A4 . .
"""
BASS_RHYTHM = (1, 0, 0, 2, 0, 0, 1, 0, 1, 0, 0, 2, 0, 0, 1, 2)  # 1 root, 2 octave up
DRUMS = (1, 0, 3, 0, 2, 0, 3, 0, 1, 0, 1, 3, 2, 0, 3, 3,        # kick, snare, hat
         1, 0, 1, 3, 2, 0, 2, 3, 1, 0, 2, 2, 2, 3, 2, 2)        # fill: every 4th bar
A = (("D3", "D4"), ("F3", "F4"), ("C3", "C4"), ("A3", "A4"))
ROOTS = A + A + (("G3", "G4"), ("Bb3", "Bb4"), ("F3", "F4"), ("A3", "A4")) + A
DRUM_F = ((0,) * 8,
          (40, 70, 110, 160, 220, 250, 255, 255),     # kick: pitch drops fast
          (6, 6, 8, 8, 10, 10, 12, 12),
          (1,) * 8)
DRUM_C = ((0,) * 8,
          (0xAE, 0xAC, 0xAA, 0xA7, 0xA4, 0xA2, 0xA1, 0),
          (0x8C, 0x8A, 0x88, 0x86, 0x84, 0x82, 0x81, 0),
          (0x85, 0x83, 0x81, 0, 0, 0, 0, 0))


def encode(lead, bass, drums, roots, settings, drum_gain=15):
    tokens = lead.split()
    assert len(tokens) == 256 and len(bass) == len(roots) == 16 and len(drums) == 32
    notes = sorted({t for t in tokens if t not in "-."}, key=audf16, reverse=True)
    assert len(notes) <= 16
    lead = [HOLD if t == "-" else REST if t == "." else notes.index(t) + 2 for t in tokens]
    freq = [audf16(n) for n in notes] + [0] * (16 - len(notes))
    out = bytes(lead) + bytes(bass) + bytes(drums)
    out += bytes(N[lo] for lo, _ in roots) + bytes(N[hi] for _, hi in roots)
    out += bytes(f & 255 for f in freq) + bytes(f >> 8 for f in freq)
    out += bytes(v for row in DRUM_F for v in row)
    out += bytes((v & 0xF0) | ((v & 15) * drum_gain // 15) for row in DRUM_C for v in row)
    return out + bytes(settings)


def music():
    return encode(LEAD, BASS_RHYTHM, DRUMS, ROOTS, (5, 12, 12))


# Eight-bar phrases repeat twice; rests and cut notes leave space for gameplay cues.
LEVEL_LEADS = (
    """
    D4 - - . - - - - A4 - - . - - - -
    F4 - - . - - - - D4 - - - . - - -
    E4 - - . - - - - C4 - - . - - - -
    G4 - - - . - - - E4 - - . - - - -
    F4 - - . - - - - Bb4 - - . - - - -
    D4 - - . - - - - F4 - - - . - - -
    C#4 - - . - - - - E4 - - . - - - -
    A4 - - - . - - - C#4 - - . - - - -
    """,
    """
    D4 - A4 - D5 - A4 - F4 - A4 - D5 - A4 -
    D4 - F4 - A4 - F4 - D5 - A4 - F4 - A4 -
    C4 - G4 - C5 - G4 - E4 - G4 - C5 - G4 -
    C4 - E4 - G4 - E4 - C5 - G4 - E4 - G4 -
    Bb4 - F4 - D4 - F4 - Bb4 - F4 - D5 - F4 -
    F4 - C5 - A4 - C5 - F4 - A4 - C5 - A4 -
    A4 - E4 - C#5 - E4 - A4 - E4 - C#5 - E4 -
    A4 - C#5 - E5 - C#5 - A4 - E4 - C#4 . . -
    """,
    """
    D5 - A4 D5 F5 - E5 D5 C5 - A4 C5 D5 F5 E5 -
    E5 - C5 E5 G5 - E5 D5 C5 - G4 C5 E5 D5 C5 -
    D5 - Bb4 D5 F5 - D5 C5 Bb4 - F4 Bb4 D5 C5 Bb4 -
    C#5 - A4 C#5 E5 - C#5 A4 G4 - E4 G4 A4 C#5 E5 -
    D5 F5 - A4 D5 - F5 E5 D5 C5 - A4 F4 A4 D5 -
    F5 - C5 A4 F5 - E5 C5 A4 F4 - A4 C5 E5 F5 -
    G5 - D5 Bb4 G5 - F5 D5 Bb4 G4 - Bb4 D5 F5 G5 -
    E5 C#5 - A4 E5 - C#5 A4 G4 E4 - C#4 E4 A4 . .
    """,
    """
    D4 . - - Eb4 . - - D4 . - - C#4 - . -
    D4 - . - A4 - . - Eb4 - . - D4 . - -
    E4 . - - F4 . - - E4 . - - Eb4 - . -
    C4 - . - G4 - . - Eb4 - . - C4 . - -
    Bb4 . - - A4 . - - F4 . - - E4 - . -
    F4 - . - Eb4 - . - D4 - . - C#4 . - -
    C#4 . - - E4 . - - A4 . - - Bb4 - . -
    A4 - . - G4 - . - E4 - C#4 . D4 - . -
    """,
)
LEVEL_ROOTS = (
    (("D3", "D4"),) * 2 + (("C3", "C4"),) * 2 + (("Bb3", "Bb4"),) * 2 + (("A3", "A4"),) * 2,
    (("D3", "D4"),) * 2 + (("C3", "C4"),) * 2 + (("Bb3", "Bb4"), ("F3", "F4")) + (("A3", "A4"),) * 2,
    A[:1] + (("C3", "C4"), ("Bb3", "Bb4"), ("A3", "A4")) + A[:2] + (("G3", "G4"), ("A3", "A4")),
    (("D3", "D4"),) * 2 + (("C3", "C4"),) * 2 + (("Bb3", "Bb4"), ("F3", "F4")) + (("A3", "A4"),) * 2,
)
LEVEL_BASS = (
    (1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0),
    (1, 0, 2, 0, 1, 0, 2, 0, 1, 0, 2, 0, 1, 0, 2, 0),
    (1, 0, 2, 1, 0, 2, 1, 0, 1, 2, 0, 1, 2, 0, 1, 2),
    (1, 0, 0, 0, 0, 0, 2, 0, 1, 0, 0, 0, 0, 0, 2, 0),
)
LEVEL_DRUMS = (
    (3, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0,
     3, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 3, 0),
    (1, 0, 3, 0, 2, 0, 3, 0, 1, 0, 3, 0, 2, 0, 3, 0,
     1, 0, 3, 0, 2, 0, 3, 0, 1, 3, 1, 3, 2, 0, 2, 3),
    DRUMS,
    (1, 0, 0, 0, 0, 0, 3, 0, 1, 0, 0, 0, 0, 0, 3, 0,
     1, 0, 0, 0, 0, 3, 0, 0, 1, 0, 3, 0, 2, 0, 3, 0),
)
LEVEL_SETTINGS = ((8, 6, 6), (5, 9, 10), (4, 12, 12), (6, 8, 8))


def level_music(level):
    assert 0 <= level < 4
    return encode(LEVEL_LEADS[level] * 2, LEVEL_BASS[level], LEVEL_DRUMS[level],
                  LEVEL_ROOTS[level] * 2, LEVEL_SETTINGS[level], (6, 10, 15, 8)[level])


if __name__ == "__main__":
    data = music()
    assert len(data) == 435 and data[:256].count(HOLD) < 256
    assert all(v > REST for v in data[304:336])
    assert abs(1789790 / (2 * (audf16("A4") + 7)) - 440) < 0.5
    (ROOT / "assets" / "music.bin").write_bytes(data)
    for level in range(4):
        (ROOT / "assets" / ("music-level-%02d.bin" % (level + 1))).write_bytes(level_music(level))
