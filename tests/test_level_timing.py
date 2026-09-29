"""Every floor crossing between safe spots (rack shadow, switch, relay) must
leave a human-sized timing window: the cat walks 1 px per 3 frames, and the
cameras sweep per tools/make_level_art.py CAMERAS (same stepping as tick_cam)."""

from pathlib import Path
import sys

root = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(root / "tools"))
from make_level_art import CAMERAS  # noqa: E402

art = (root / "assets/level-rooms.pic").read_bytes()
MIN_WINDOW = 60  # frames (~1.2 s) of valid start times per crossing
SWITCH = int(__import__("re").search(r"switch_timer=a=(\d+) call sfx_switch",
                                     (root / "main.k65").read_text()).group(1))


def col(px):
    return (px + 2) // 4 + 2  # set_col


def trace(cfg, frames):
    _, lo, hi, speed, pause, _, _ = cfg
    c, d, w, out = lo, 0, speed, []
    for _ in range(frames):
        if w == 0:
            w = speed
            c += 1 if d == 0 else -1
            if c == hi:
                d, w = 1, pause
            if c == lo:
                d, w = 0, pause
        else:
            w -= 1
        out.append(c)
    return out


for room in range(3):
    floor = art[room * 960 + 880:room * 960 + 920]
    cams = [(trace(k, 4000), k[5], k[6]) for k in CAMERAS[room] if k[0]]
    p = 0
    while p <= 140:
        if floor[col(p)] != 70:
            p += 1
            continue
        a = p - 1
        while p <= 140 and floor[col(p)] == 70:
            p += 1
        steps = (p - a) * 3
        switch = room == 1 and floor[col(a)] == 82  # leaving the switch: gated cam off
        run = best = 0
        for t0 in range(1000, 3000):
            ok = all(not any(abs(col(a + 1 + k // 3) - tr[t0 + k]) <= hw and not (g and switch and k < SWITCH)
                             for tr, hw, g in cams) for k in range(steps))
            run = run + 1 if ok else 0
            best = max(best, run)
        assert best >= MIN_WINDOW, (room, col(a), col(min(p, 140)), best)

print("test_level_timing.py: all checks passed")
