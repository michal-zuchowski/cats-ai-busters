"""tools/solve_levels.py macro/key logic against a fake simulator (no compiler, no XEX)."""

import sys
from pathlib import Path

root = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(root / "tools"))
import solve_levels as S  # noqa: E402

NAMES = list(S.KEY_VARS) + list(S.L03_KEY_VARS) + ["cr_t", "cr_r", "cr_c", "RTCLOK", "m_h0"]


class BootFake:
    def __init__(self, intro=False):
        self.state = dict(player_y=0, level_finished=0, title_phase=1, music_ready=1, music_chunk=17, cur_room=0)
        if intro:
            self.state.update(cur_room=0xFF, title_phase=0, music_ready=0)
        self.frames = 0

    def __getitem__(self, n):
        return self.state[n]

    def run(self, frames, joy=0, fire=False):
        assert frames == 1 and joy == 0
        self.frames += 1
        if self.state["cur_room"] == 0xFF:
            assert fire
            self.state.update(cur_room=0, title_phase=1, music_ready=1)
        elif self.state["title_phase"] == 1:
            assert not fire  # don't hold FIRE forever in the release loop
            self.state["title_phase"] = 2
        elif self.state["title_phase"] == 2:
            assert fire
            self.state.update(title_phase=0, player_y=184, music_ready=0)
        else:
            assert not fire  # release immediately, don't trigger a level-01 action
            self.state.update(music_ready=1, music_chunk=18)


s = BootFake()
S.boot(s)
assert s.frames == 3 and s["music_ready"] and s["music_chunk"] == 18
s = BootFake(intro=True)
S.boot(s)
assert s.frames == 4 and s["music_ready"] and s["music_chunk"] == 18
try:
    S.boot(BootFake(), limit=2)
except RuntimeError as exc:
    assert "XEX/ATR" in str(exc)
else:
    raise AssertionError("boot silently accepted unfinished music loading")


class Fake:
    """Jump: 3 prep frames, 6 air frames, 4 landing frames. x >= 10 changes room (or level)."""

    def __init__(self, **kw):
        self.sym = {n: 0x100 + i * 8 for i, n in enumerate(NAMES)}
        self.mem = [0] * 0x400
        for n in NAMES:
            self[n] = 0
        for k, v in kw.items():
            self[k] = v
        self.frames = 0

    def __getitem__(self, n):
        return self.mem[self.sym[n]]

    def __setitem__(self, n, v):
        self.mem[self.sym[n]] = v

    def run(self, frames, joy=0, fire=False):
        for _ in range(frames):
            self.frames += 1
            if self["jprep"]:
                self["jprep"] -= 1
                if not self["jprep"]:
                    self["air"], self["vy"] = 6, 1
            elif self["air"]:
                self["air"] -= 1
                if not self["air"]:
                    self["lnd"] = 4
            elif self["lnd"]:
                self["lnd"] -= 1
            elif joy & S.U:
                self["jprep"] = 3
            if joy & S.R and not self["air"]:
                self["player_x"] += 1
            if self["player_x"] >= 10 and not self["cur_room"]:
                self["cur_room"] = 1
            if self["player_x"] >= 20 and self["cur_room"] == 1:
                self["level_finished"] = 1
            self["RTCLOK"] = (self["RTCLOK"] + 1) & 255


# anticipation: the jump macro must not stop at air==0 during jprep; it ends fully landed
s = Fake()
assert not S.run_macro(s, "jumpUp", S.U, 1)
assert s.frames == 1 + 3 + 6 + 4  # press + prep + flight + landing, no early stop on air == 0
assert s["lnd"] == 0 and s["air"] == 0 and s["jprep"] == 0

# stop at the first transition, not after the whole walk in the next room
s = Fake(player_x=7)
assert S.run_macro(s, "R40", S.R, 40)
assert s["cur_room"] == 1 and s["player_x"] == 10 and s.frames == 3
s = Fake(player_x=19, cur_room=1)
assert S.run_macro(s, "R16", S.R, 16) and s["level_finished"] and s.frames == 1
s = Fake(player_x=0)
assert not S.run_macro(s, "wait", 0, 40) and s.frames == 40

# key: exact, distinguishes every future-relevant byte
base = S.key(Fake())
for n in S.KEY_VARS:
    f = Fake()
    f[n] = 1
    assert S.key(f) != base, n
f = Fake(cur_level=1)
level2 = S.key(f)
for n in ("cr_t", "cr_r", "cr_c"):
    for i in range(8):
        g = Fake(cur_level=1)
        g.mem[g.sym[n] + i] = 1
        assert S.key(g) != level2, (n, i)
g = Fake(cur_level=2)
h = Fake(cur_level=2)
h.mem[h.sym["cr_t"]] = 5  # aliased actor bytes outside L02 are not crumble state
assert S.key(g) == S.key(h)
level3 = S.key(Fake(cur_level=2))
for n in S.L03_KEY_VARS:
    g = Fake(cur_level=2)
    g[n] = 1
    assert S.key(g) != level3, n
    g = Fake(cur_level=1)
    h = Fake(cur_level=1)
    g[n] = 1
    assert S.key(g) == S.key(h), ("not L03 state in L02", n)
assert S.key(Fake(player_x=3)) != S.key(Fake(player_x=2))  # no x//2 bucketing
assert S.key(Fake(fan_t=64)) != S.key(Fake(fan_t=65)) and S.key(Fake(act_col=3)) != S.key(Fake(act_col=4))
assert S.key(Fake(RTCLOK=1)) != S.key(Fake(RTCLOK=0))
assert S.key(Fake(RTCLOK=2)) == S.key(Fake(RTCLOK=0))  # parity only unless the hazard row exists
assert S.key(Fake(RTCLOK=2, m_h0=5)) != S.key(Fake(RTCLOK=0, m_h0=5))
assert S.key(Fake(cur_level=3, dlg_done=0)) != S.key(Fake(cur_level=3, dlg_done=1))

print("test_solver_macros.py: all checks passed")
