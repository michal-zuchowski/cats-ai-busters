"""R4 joystick route on the user's XEX. --asset-overlay tests new room data on that engine.

No compilation, executable patching or file writes. The overlay changes only the in-memory
ATR room sector bytes and corresponding chunk checksum; new HUD/helper code needs a fresh build.
"""
import argparse
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
from sim import Sim, FRAME_CYCLES, SIOV
from solve_levels import boot


RESPAWNS = [0]


CAPTIONS = ("SWAT THE VACUUM THROUGH THE READER. ", "RIDE RIGHT. JUMP TO THE BALCONY.    ",
            "JUMP THE GAP TO THE SERVICE EXIT.   ", "SERVICE ACCESS OPEN. GO RIGHT.      ")


def invoke(s, name):
    """Call an existing compiled routine, preserving the suspended main-loop call stack."""
    m = s.mpu
    pc, sp = m.pc, m.sp
    ret = 0x07FF
    s.mem[0x100 + sp] = ret >> 8
    s.mem[0x100 + ((sp - 1) & 255)] = ret & 255
    m.sp, m.pc = (sp - 2) & 255, s.sym[name]
    for _ in range(2_000_000):
        if m.pc == s.sym["respawn"]:
            RESPAWNS[0] += 1
        if m.pc == 0x0800:
            m.pc, m.sp = pc, sp
            return
        if m.pc == SIOV:
            s._siov()
        else:
            m.step()
        if m.processorCycles >= s.next_tick:
            s.next_tick += FRAME_CYCLES
            for addr in (0x14, 0x13, 0x12):
                s.mem[addr] = (s.mem[addr] + 1) & 255
                if s.mem[addr]:
                    break
    raise RuntimeError("Compiled routine did not return: " + name)


def frame(s, joy=0, fire=False):
    s.mem[0xD300] = 0xFF ^ joy
    s.mem[0xD010] = 0 if fire else 1
    s.mem[0x14] = (s.mem[0x14] + 1) & 255
    s.next_tick = s.mpu.processorCycles + FRAME_CYCLES
    invoke(s, "plat_frame")


def entry(s):
    s["cur_room"], s["entry_col"], s["level_finished"] = 3, 0, 0
    s.mem[0xD300], s.mem[0xD010] = 0xFF, 1
    invoke(s, "enter_plat_room")


def reader(s):
    return tuple(set(s.screen(20, 12, 3)))


def door(s):
    return tuple(s.screen(r, 39, 1)[0] for r in (19, 20, 21))


def route(s, stand=17, board=(5, 8), rear=0, delay=0, edge=11, probe=None):
    """Joystick/FIRE only after room entry. Reads game state like a player reads the screen:
    approach the front robot, swing again after a miss, jump to its roof when it is near, ride,
    leave at the balcony and cross the second gap. Returns frames and counted respawns."""
    swings = hits = cooldown = 0
    boarded = False
    seen = set()
    if probe is not None:
        assert reader(s) == (83,) and door(s) == (81, 81, 81), (reader(s), door(s))
    for i in range(1500):
        joy, fire = 0, False
        if i >= delay:
            col, robot, x = s["player_col"], s["act_col"], s["player_x"]
            if not s["gate"]:
                if cooldown:
                    cooldown -= 1
                elif s["paw_hit"] and s["act_wob"]:
                    pass  # struck: wait while it is carried; a hit at col 7 falls one column short
                elif x >= stand and not s["paw_t"]:
                    swings += 1
                    fire, cooldown = True, 11 + 5 * (swings % 3)  # unsynchronised retries
                elif x < stand:
                    joy = 8
            elif col >= 33 and not s["air"]:
                joy = 8
            elif s["player_y"] == 152 and not s["air"]:
                joy = 8 if col < 27 else 9
            elif s["air"]:
                joy = 8
            elif s["ride"]:
                boarded = True
                if robot >= 19:
                    joy = 9
                elif col - robot < rear:
                    joy = 8
                elif col - robot > rear:
                    joy = 4
            elif s["player_y"] == 184:
                if s["act_dir"] == 1 and board[0] - RESPAWNS[0] <= robot - col <= board[1] - RESPAWNS[0]:
                    joy = 9
                elif col < edge:
                    joy = 8  # walk to the pit edge, then wait for the robot to pass by
        frame(s, joy, fire)
        if probe is not None and s["gate"] and "gate" not in probe:
            probe["gate"] = (reader(s), door(s))
        if not s["air"]:
            seen.add(s["player_y"])
        if s["level_finished"]:
            assert swings and boarded and s["gate"] and {171, 152, 184} <= seen, (swings, boarded, seen)
            assert s["player_y"] == 184 and s["player_col"] >= 33
            return i + 1, swings, RESPAWNS[0]
    raise AssertionError("Compiled R4 route stalled: " + str(
        {k: s[k] for k in ("player_x", "player_y", "ride", "carrier", "act_col", "gate", "air")}))


def main():
    args = argparse.ArgumentParser()
    args.add_argument("--asset-overlay", action="store_true")
    overlay = args.parse_args().asset_overlay
    s = Sim()
    if not overlay and "R4Goal" not in s.sym:
        raise RuntimeError("Build predates R4 changes. Build matching XEX/ATR, or use --asset-overlay for room-data QA only.")
    boot(s)
    s.mem[s.sym["CH"]] = 0x1A  # use the real debug entry to L03, not a fabricated physics state
    s.run(3)
    assert s["cur_level"] == 2
    if overlay:
        table = (ROOT / "assets/disk-chunks.bin").read_bytes()
        meta = table[60:65]  # chunk 12: L03 R4
        sector, count = int.from_bytes(meta[:2], "little"), int.from_bytes(meta[2:4], "little")
        assert count == 8
        disk = bytearray(s.atr)
        disk[(sector - 1) * 128:(sector - 1 + count) * 128] = (
            ROOT / "assets/plat-rooms.bin").read_bytes()[7168:8192]
        s.atr = bytes(disk)
        base = s.sym["ChunkTable"] + 60
        s.mem[base:base + 5] = list(meta)
    entry(s)
    assert s["m_exit"] == 184 and s["m_xmax"] == 19 and s["e_bspd"] == 0
    start = s.snapshot()
    frame(s, 1)
    for _ in range(60):
        frame(s)
    assert s["player_y"] == 184 and not s["ride"] and not s["gate"]
    # (stand x, boarding window, rear offset, input delay, pit-edge column): different approach spots, early/late
    # roof jumps, riding positions and start timings; failed swings/boardings simply retry.
    variants = ((17, (5, 7), 0, 0, 11), (19, (5, 6), 1, 7, 11), (18, (6, 7), 2, 19, 10),
                (21, (5, 5), 0, 11, 12), (17, (7, 7), 1, 31, 12), (20, (6, 6), 2, 3, 11), (19, (8, 8), 0, 5, 11))
    for stand, board, rear, delay, edge in variants:
        s.restore(start)
        RESPAWNS[0] = 0
        probe = None if overlay else {}
        n, swings, deaths = route(s, stand, board, rear, delay, edge, probe)
        if probe is not None:  # compiled reader update: RELAY 83 until the genuine gate, then DONE 84, door open
            assert probe["gate"][0] == (84,) and probe["gate"][1] != (81, 81, 81), probe
        assert deaths == (1 if board == (8, 8) else 0), deaths  # the late jump misses once, then retries
        print(f"R4 compiled-engine route: stand={stand} board={board} rear={rear} delay={delay} edge={edge}: "
              f"{n} frames, {swings} swings, {deaths} respawns, complete")
    if not overlay:
        s.restore(start)
        h = s.sym["Hints"]  # encoding check against old compiled text: ASCII - 32
        assert bytes(s.mem[h + 7:h + 11]) == bytes(c - 32 for c in b"JUMP")
        for i, text in enumerate(CAPTIONS):
            s["gate"], s["player_col"] = (0, 0) if i == 0 else (1, (0, 10, 23, 33)[i])
            s["ride"] = 0
            invoke(s, "plat_hud")
            assert s.screen(0, 0, 36) == bytes(ord(c) - 32 for c in text), (i, s.screen(0, 0, 36))
        s.restore(start)  # the caption loop left gate=1; reset to the start fixture first
        s["gate"] = 0
        s["e_bspd"], s["bcol"], s["player_x"], s["player_y"] = 2, 0, 0, 171
        invoke(s, "set_col")
        invoke(s, "on_vacuum")
        assert s.mpu.a == 0  # compiled root guard: dormant pursuer cannot be a carrier before the gate
        s["gate"] = 1
        invoke(s, "on_vacuum")
        assert s.mpu.a == 21 and s["carrier"] == 21  # gate open: B may carry
        s["gate"] = 1
        invoke(s, "act_init")
        assert s["gate"] == 0 and reader(s) == (83,) and door(s) == (81, 81, 81)  # reset: RELAY, closed door
    print("Actual 6502 logic passed; no VBI/display/audio emulation."
          + (" Asset overlay only: new HUD/reader/root-guard code remains uncompiled (not run)." if overlay else ""))


if __name__ == "__main__":
    main()
