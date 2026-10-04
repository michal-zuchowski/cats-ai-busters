"""Breadth-first 'bot' proving levels 02-04 are completable in the headless sim.

Usage: python3 tools/solve_levels.py [level 1..3]   (needs make_atr.py + a build)
Explores joystick macros from each room start; prints the winning macro list.
"""
import sys
from collections import deque

from sim import Sim

R, L, U, F = 8, 4, 1, 0x100
MACROS = [("R4", R, 4), ("L4", L, 4), ("R16", R, 16), ("L16", L, 16), ("R40", R, 40), ("L40", L, 40),
          ("jumpR", R | U, 1), ("jumpL", L | U, 1), ("jumpUp", U, 1),
          ("wait", 0, 40), ("fire", F, 3)]


def where(s):
    return s["cur_level"], s["cur_room"], s["level_finished"]


def run_macro(s, name, joy, n):
    """Run one macro; return True at the first room/level/level_finished transition (stops there)."""
    start = where(s)[:2]

    def frame(j, fire=False):
        s.run(1, j, fire)
        w = where(s)
        return w[:2] != start or w[2]

    if name.startswith("jump"):  # anticipation, flight and landing must all finish
        h = joy & ~U
        if frame(joy):
            return True
        for _ in range(200):
            if not (s["air"] or s["jprep"] or s["lnd"]):
                break
            if frame(joy if s["jprep"] else (h if s["air"] else 0)):
                return True
    elif joy == F:
        for _ in range(n):
            if frame(0, True):
                return True
        for _ in range(2):
            if frame(0):
                return True
    else:
        for _ in range(n):
            if frame(joy):
                return True
    return False


# Every byte that can change future behaviour; no bucketing.  Page-6 crumble slots alias
# other levels' actors, so they are only meaningful in level 02 (cur_level == 1).
KEY_VARS = ("cur_level", "cur_room", "player_x", "player_y", "air", "vy", "gt", "jprep", "lnd",
            "fire_prev", "facing", "walk_timer", "paw_t", "paw_kind", "paw_x", "paw_y", "paw_hit",
            "act_col", "act_dir", "act_t", "act_wob", "lift_row", "lift_dir", "lift_t", "dlg_done",
            "sw_on", "fan_t", "ride", "level_finished")
L03_KEY_VARS = ("bcol", "bdir", "bt", "bwob", "carrier", "dk", "gate", "bw", "bk",
                "e_mode", "e_bspd")


def key(s):
    k = tuple(s[v] for v in KEY_VARS)
    if s["cur_level"] == 1:
        k += tuple(s.mem[s.sym[v] + i] for v in ("cr_t", "cr_r", "cr_c") for i in range(8))
    elif s["cur_level"] == 2:
        k += tuple(s[v] for v in L03_KEY_VARS)
    # ground steps use RTCLOK&1; the L04 floor hazard uses RTCLOK&0x20 (64-frame period)
    return k + (s["RTCLOK"] & (0x3F if s["m_h0"] else 1),)


def boot(s, limit=600):
    """Skip the intro, release FIRE on the title, then start and wait for level 01 music."""
    for _ in range(limit):
        if (s["player_y"] == 184 and not s["level_finished"] and not s["title_phase"]
                and s["music_ready"] and s["music_chunk"] == 18):
            return
        s.run(1, fire=s["cur_room"] == 0xFF or s["title_phase"] == 2)
    raise RuntimeError("Game did not reach level 01 after the title; check matching XEX/ATR and DISKERR.")


def solve(level, limit=15000):
    s = Sim()
    boot(s)
    for _ in range(level):
        s["level_finished"] = 1
        s.run(20)
    s.run(5)
    target = s["cur_level"]
    total, paths = 0, []
    while True:  # one search per room: goal = next room / end of the selected level
        room = s["cur_room"]
        seen, q, found = set(), deque([(s.snapshot(), [])]), None
        n = 0
        while q and n < limit and not found:
            snap, path = q.popleft()
            for name, joy, cnt in MACROS:
                s.restore(snap)
                n += 1
                if run_macro(s, name, joy, cnt) or s["cur_level"] != target:
                    found = (s.snapshot(), path + [name])
                    break
                k = key(s)
                if k not in seen:
                    seen.add(k)
                    q.append((s.snapshot(), path + [name]))
        total += n
        if not found:
            return None, total, paths, room
        s.restore(found[0])
        paths.append(found[1])
        if s["level_finished"] or s["cur_level"] != target:
            return True, total, paths, room


if __name__ == "__main__":
    for lv in ([int(sys.argv[1])] if len(sys.argv) > 1 else (1, 2, 3)):
        ok, n, paths, room = solve(lv)
        print("level %02d:" % (lv + 1), "solved" if ok else "STUCK in room %d" % room, "(%d tried)" % n)
        for p in paths:
            print("  ", " ".join(p))
