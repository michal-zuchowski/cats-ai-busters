"""Level 02 redesign: source wiring plus route/mechanic proofs on tools/plat_model.py.

plat_model.py is a frame-level Python model of main.k65's plat_* functions. These checks
prove the designed routes, crumble/fan/switch/lift behaviour and softlock freedom at the
source+asset level; they do not execute a compiled XEX.
"""

import copy
import re
import sys
from pathlib import Path

root = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(root / "tools"))
import make_level_art as L  # noqa: E402
import plat_model as M  # noqa: E402
from make_plat_levels import META, ROOMS  # noqa: E402

main = (root / "main.k65").read_text()
plat_rooms = (root / "assets/plat-rooms.bin").read_bytes()
assert [plat_rooms[i * 1024:(i + 1) * 1024] for i in range(4)] == ROOMS[:4]  # assets are current


def body(name):
    start = main.index(f"func {name} {{")
    depth, i = 0, main.index("{", start)
    for j in range(i, len(main)):
        depth += main[j] == "{"
        depth -= main[j] == "}"
        if depth == 0:
            return re.sub(r"\s+", " ", re.sub(r"//[^\n]*", "", main[i + 1:j])).strip()


# ---------------------------------------------------------------- source wiring
fire = body("plat_fire")
assert "a=air a?0 != { return }" in fire  # grounded only
assert "a=TRIG0 a&1 a?0 != { fire_prev=a=1 return }" in fire and "fire_prev=a=0" in fire  # press edge
assert "a=cur_level a?1 == { plat_switch return } a?2 == { swat return }" in fire
ready = body("sw_ready")
for needle in ("a=m_swy", "a?player_y", "a=sw_on", "a=air", "c+ a-m_swc", "c- a+2", "a?5 >= { a=0 return }"):
    assert needle in ready, needle
sw = body("plat_switch")
assert sw.startswith("sw_ready a?0 == { return }") and "sw_on=a" in sw and "tmp=a=62" in sw and "tmp=a=85" in sw
assert "a=m_lpow a?0 != { draw_lift }" in sw
under = body("under")
assert all(g in under for g in ("a?70", "a?94", "a?127", "a?62"))
assert body("respawn").count("crumble_reset") == 1
assert body("crumble_reset").startswith("a=cur_level a?1 != { return } x=0")  # L03/L04 alias these bytes
assert body("plat_dyn_init").startswith("sw_on=a=0 fan_t=a x=0 { cr_t,x=a x++ x?8 } !=")
lift = body("plat_lift")
assert "a=m_lpow a?0 != { a=sw_on a?0 == { return } }" in lift and "a?tmp < { return }" in lift
assert "a=m_lspd a?0 == { a=24 }" in lift  # Levels 03/04 keep the 24 frame lift period
assert "plat_crumble" in body("plat_frame").split("plat_lift")[1].split("plat_actor")[0]
fan = body("plat_fan")
assert "a=fan_t a&64" in fan and "a?m_fan < { return }" in fan and "a?m_fc1 >= { return }" in fan
assert "a=air a?0 == { return }" in fan and "RTCLOK" not in fan and "COLOR2" not in fan
assert "fan_draw" in fan and "FanT,x" in body("fan_draw") and "put_run" in body("fan_draw")
hud = body("plat_hud")
assert "a=fan_t a&64" in hud and "sw_ready a?0 != { x=7 }" in hud
assert re.search(r'"PUSH   "\s+"FIRE   "\s+"DOCK   "', main) and "HintOff { 0 7 14 21 28 35 42 49 56 63 }" in main
for name in ("paw_both", "plat_pursuer", "plat_beam"):
    assert body(name).startswith("a=cur_level a?2 != { return }"), name
assert body("plat_exit_allowed").startswith("a=cur_level a?2 == {")
inp, phys = body("plat_input"), body("plat_phys")  # physics / pose-driving constants are untouched
assert "vy=a=0xFC" in inp and "jprep=a=3" in inp and "walk_timer=a=4" in inp
assert "lnd=a=10" in phys and "a?3 == {" in phys and "gt=a=2" in phys
addr = {n: int(a, 16) for n, a in re.findall(r"var (\w+) = (0x[0-9A-Fa-f]+)", main)}
used = [addr[n] for n in ("sw_on", "fan_t", "cr_i")] + [addr[n] + i for n in ("cr_t", "cr_r", "cr_c") for i in range(8)]
assert len(set(used)) == len(used) and min(used) > addr["paw_hit"] and max(used) <= 0x06FF
fant = [int(v) for v in re.search(r"data FanT \{([^}]*)\}", main).group(1).split()]
assert fant == list(L.FAN_A) + list(L.FAN_B)

# ---------------------------------------------------------------- map invariants
SUPPORT = {L.FLOOR, L.LIFT, L.CRUMB, L.CTRL}
EXPECT = {
    0: [(22, 0, 39), (20, 22, 25), (18, 16, 19), (16, 23, 26), (14, 16, 19), (12, 6, 13), (13, 35, 39)],
    1: [(22, 0, 39), (13, 0, 6), (13, 10, 13), (12, 17, 20), (11, 25, 28), (20, 26, 29), (18, 32, 35),
        (16, 26, 29), (14, 32, 35), (11, 33, 39)],
    2: [(22, 0, 39), (11, 0, 9), (11, 18, 25), (9, 29, 39), (19, 12, 15), (16, 17, 20), (13, 22, 25)],
    3: [(22, 0, 39), (9, 0, 7), (10, 17, 31), (6, 18, 21), (6, 25, 28), (6, 33, 39)],
}
metas = [dict(zip(META, ROOMS[n][960:960 + len(META)])) for n in range(4)]
for n, spans in EXPECT.items():
    want = {(r, c) for r, a, b in spans for c in range(a, b + 1)}
    got = {(r, c) for r in range(24) for c in range(40) if ROOMS[n][r * 40 + c] & 0x7F in SUPPORT}
    assert got == want, n  # decorative tiles never support: exactly the designed platforms
    row = metas[n]["sy"] // 8 - 1
    assert (row, 2) in want and (row, 4) in want and metas[n]["sx"] == 8  # entry (x=0) and respawn (x=8)
assert [m["sy"] for m in metas] == [184, 112, 96, 80]  # coherent climb: entry = previous exit height
assert [m["exit_y"] for m in metas] == [112, 96, 80, 56]
assert all(m["sy"] == (metas[i - 1]["exit_y"] if i else 184) for i, m in enumerate(metas))
assert all(not (m["arow"] or m["h0"] or m["glass"] or m["dlg"]) for m in metas)  # no vacuums/hazards
for n in range(4):
    row = metas[n]["exit_y"] // 8 - 1
    assert all(ROOMS[n][row * 40 + c] & 0x7F in SUPPORT for c in (37, 38, 39))
    closed = L.WALL if metas[n]["swy"] else L.HATCH  # controller rooms start with the exit sealed
    assert all(ROOMS[n][r * 40 + 39] == closed for r in range(row - 3, row))
assert [(m["swy"], m["swc"], m["lpow"]) for m in metas] == [(104, 9, 1), (0, 0, 0), (0, 0, 0), (88, 26, 0)]
assert ROOMS[0][12 * 40 + 9] == L.CTRL | L.INV and ROOMS[3][10 * 40 + 26] == L.CTRL | L.INV
assert [m["fan"] for m in metas] == [0, 0, 10, 0]
fm = metas[2]
assert (fm["fan"], fm["fc1"], fm["fr0"], fm["fr1"]) == (10, 18, 5, 11)
assert all(ROOMS[2][r * 40 + c] == L.VOID for r in range(5, 11) for c in range(10, 18))  # zone is empty air
blade_tiles = [ROOMS[2][(fm["fr0"] + 2 + dr) * 40 + fm["fan"] - 3 + dc] for dr in (0, 1) for dc in (0, 1)]
assert blade_tiles == list(L.FAN_A) and set(L.FAN_A) != set(L.FAN_B)
font = (root / "assets/term-font.pic").read_bytes()
assert all(font[g * 8:(g + 1) * 8] != bytes(8) for g in fant + [L.AIR_A, L.AIR_B, L.CRUMB, L.CTRL])
assert font[L.AIR_A * 8:L.AIR_A * 8 + 8] != font[L.AIR_B * 8:L.AIR_B * 8 + 8]


# ---------------------------------------------------------------- helpers
def blank(chunk, *spans):
    b = bytearray(chunk)
    for r, c0, c1 in spans:
        b[r * 40 + c0:r * 40 + c1 + 1] = bytes([L.VOID]) * (c1 - c0 + 1)
    return bytes(b)


def replay(chunk, start, goal, **kw):
    par = M.plan(chunk, start, **kw)
    assert goal in par, ("unreachable", start, goal)
    sim = M.Sim(chunk)
    sim.x, sim.y = start
    assert M.drive(sim, M.path_to(par, goal)), "replay left the plan"
    assert sim.deaths == 0 and (sim.x, sim.y) == goal
    return M.cost_of(par, goal), sim, par


def jumps(par, goal):
    return sum(1 for _, mac, _ in M.path_to(par, goal) if mac[0] == "jump")


def settle_press(sim, key):
    sim.step(key)
    sim.step(0)


def fire_at(chunk, x, y, air=0):
    s = M.Sim(chunk)
    s.x, s.y, s.air = x, y, air
    s.step(M.FIRE)
    return s


# ---------------------------------------------------------------- Room 1: service shaft
c1 = ROOMS[0]
centres = [(a + b) / 2 for _, a, b in EXPECT[0][1:6]]
dirs = [centres[i + 1] > centres[i] for i in range(3)]
assert dirs == [False, True, False]  # climb zigzags left, right, left ...
assert [EXPECT[0][i][0] for i in range(1, 6)] == [20, 18, 16, 14, 12]  # ... rising two rows each time
par1 = M.plan(c1, (0, 184))
cost, sim, _ = replay(c1, (0, 184), (28, 104))  # controller ledge with the unchanged jump
assert sim.col == 9
assert not any(y == 112 and x >= 130 for x, y in par1)  # no walking/jumping bypass to the exit ledge
assert not any(y < 104 for _, y in par1)
assert not any(y == 112 and x >= 130 for x, y in M.plan(c1, (28, 104)))  # not from the console ledge either
idle = M.Sim(c1)
for _ in range(500):
    idle.step(0)
assert idle.lift_row == 22 and idle.sw_on == 0 and idle.scr[22][33] == L.LIFT | 0x80  # stationary, red
for x in range(141):  # FIRE works within 2 columns of the console only
    assert bool(fire_at(c1, x, 104).sw_on) == (abs(((x + 2) >> 2) + 2 - 9) <= 2), x
assert not fire_at(c1, 28, 96).sw_on  # wrong height
assert not fire_at(c1, 28, 104, air=1).sw_on  # airborne
held = M.Sim(c1); held.x, held.y, held.fire_prev = 28, 104, 0  # FIRE already held: no edge
held.step(M.FIRE); held.step(M.FIRE)
assert not held.sw_on
held.step(0); held.step(M.FIRE)
assert held.sw_on
s = fire_at(c1, 28, 104)
assert s.sw_on and s.scr[12][9] == L.CTRL and all(s.scr[r][39] == L.HATCH for r in (10, 11, 12))
for _ in range(40):
    s.step(0)
assert s.lift_row != 22 and s.scr[s.lift_row][33] == L.LIFT  # powered lift runs, no longer red
def exit_try(sw, y, air):
    g = M.Sim(c1)
    g.x, g.y, g.sw_on, g.air = 140, y, sw, air
    g.step(M.RIGHT)
    return g.exited


assert not exit_try(0, 112, 0) and not exit_try(1, 104, 0) and not exit_try(1, 112, 1)  # needs switch, height, ground
assert exit_try(1, 112, 0)
sim = M.Sim(c1)  # complete: climb, FIRE, drop, ride the lift, exit
M.drive(sim, M.path_to(par1, (28, 104)))
settle_press(sim, M.FIRE)
assert sim.sw_on
assert M.drive(sim, M.path_to(M.plan(c1, (28, 104)), (124, 184)))
assert M.hold_until(sim, 0, lambda t: t.lift_row == 22 and t.y == 184)
assert M.hold_until(sim, 0, lambda t: t.lift_row == 13) and sim.y == 112
assert M.hold_until(sim, M.RIGHT, lambda t: t.x == 140) and sim.y == 112
sim.step(M.RIGHT); sim.step(M.RIGHT)
assert sim.exited and sim.deaths == 0
cyc = M.Sim(c1); cyc.sw_on = 1
seen = set()
for _ in range(2 * 9 * 18 + 20):  # the lift returns to the bottom every cycle: reboard is always possible
    cyc.step(0)
    seen.add(cyc.lift_row)
assert seen == set(range(13, 23))

# ---------------------------------------------------------------- Room 2: catwalks
c2 = ROOMS[1]
start, goal = (0, 112), (140, 96)
crumb_only = blank(c2, (22, 0, 39), (20, 26, 29), (18, 32, 35), (16, 26, 29), (14, 32, 35))
stable_only = blank(c2, (13, 10, 13), (12, 17, 20), (11, 25, 28))
cost_s, sim_s, par_s = replay(crumb_only, start, goal)  # each route works alone
cost_t, sim_t, par_t = replay(stable_only, start, goal)
cost_f, _, _ = replay(c2, start, goal)
assert cost_f <= cost_s <= cost_t - 30, (cost_f, cost_s, cost_t)  # the shortcut is shorter in frames
assert jumps(par_s, goal) < jumps(par_t, goal) and sim_s.t < sim_t.t
assert any(M.Sim(c2).art[r][c] == L.CRUMB for r, c in ((13, 10), (12, 17), (11, 25)))
row = 13
cm = M.Sim(c2); cm.x, cm.y = 44, 112  # on the first crumbling run
col = cm.col
assert cm.scr[row][col] == L.CRUMB
log = []
for _ in range(135):
    cm.step(0)
    log.append(cm.scr[row][col])
assert log[0] == L.CRUMB | 0x80 and cm.deaths == 0  # warning from the first contact frame ...
assert all(t == L.CRUMB | 0x80 for t in log[:39])  # ... solid and visibly red for 40 frames
assert all(t == L.VOID for t in log[39:119])  # then gone
assert log[119] == L.CRUMB and log[-1] == L.CRUMB  # and restored at frame 120 without any retry
assert cm.y == 184 and not cm.air  # the missed/crumbled cat is caught by the floor, no softlock
rr = M.Sim(c2); rr.x, rr.y = 44, 112
for _ in range(45):
    rr.step(0)
assert rr.scr[row][col] == L.VOID
rr.y, rr.air = 196, 1
rr.step(0)  # retry (respawn) restores every active crumble at once
assert rr.deaths == 1 and rr.scr[row][col] == L.CRUMB and not any(sl[0] for sl in rr.cr)
full = M.Sim(c2); full.x, full.y = 44, 112
for sl in full.cr:
    sl[:] = [1, 0, 0]  # every slot busy: the tile stays a plain solid tile, never silently corrupt
full.step(0)
assert full.scr[row][full.col] == L.CRUMB and not full.air
for first in (10, 17, 25):  # lower catches under all crumbles
    below = [r for r in range(14, 23) if (r, first) in {(r, c) for r, a, b in EXPECT[1] for c in range(a, b + 1)}]
    assert below
for x in range(0, 141, 8):  # from anywhere on the floor the stable route still reaches the exit
    assert (140, 96) in M.plan(stable_only, (x, 184), xstep=2), x

# ---------------------------------------------------------------- Room 3: cooling channel
c3 = ROOMS[2]
start, goal = (0, 96), (140, 80)
windy = blank(c3, (22, 0, 39), (19, 12, 15), (16, 17, 20), (13, 22, 25))
gusts = tuple(range(64, 128, 2))
cost_w, sim_w, _ = replay(windy, start, goal, t0s=gusts)  # the wind-assisted gap alone gets through
assert goal not in M.plan(windy, start)  # no wind: the 8-column gap is impossible
no_gap = blank(c3, (11, 0, 9))
cost_st, sim_st, _ = replay(no_gap, (8, 184), goal)  # longer stable alternative with no wind
assert cost_st > cost_w


def pushed(x, y, t_before, air=1):
    f = M.Sim(c3)
    f.x, f.y, f.air, f.vy, f.fan_t = x, y, air, 0, t_before
    f.fan()
    return f.x - x


assert pushed(34, 80, 64) == 1  # gust phase, airborne, inside the visible zone
assert pushed(34, 80, 0) == 0  # calm phase
assert pushed(34, 80, 64, air=0) == 0  # grounded
assert pushed(20, 80, 64) == 0 and pushed(24, 80, 64) == 0  # waiting nook (cols <= 9) outside the zone
assert pushed(28, 80, 64) == 0 and pushed(32, 80, 64) == 1 and pushed(60, 80, 64) == 1 and pushed(64, 80, 64) == 0  # zone starts at col 10 and ends before col 18
assert [pushed(34, y, 64) for y in (48, 56, 96, 104)] == [0, 1, 1, 0]  # body rows 5..10 only
assert pushed(140, 80, 64) == 0
nook = M.Sim(c3)
for _ in range(300):
    nook.step(0)  # waits through gust and calm phases on ledge A, never moved
assert (nook.x, nook.y, nook.deaths) == (0, 96, 0)
vis = M.Sim(c3)
poses = {}
for _ in range(256):
    vis.step(0)
    gust = bool(vis.fan_t & 64)
    zone = {vis.scr[r][c] for r in range(5, 11) for c in range(10, 18)}
    blades = tuple(vis.scr[7 + dr][7 + dc] for dr in (0, 1) for dc in (0, 1))
    assert zone <= ({L.AIR_A, L.AIR_B} if gust else {L.VOID})  # airflow is drawn exactly in the zone
    if vis.fan_t % 4 == 0:
        poses.setdefault(gust, set()).add(blades)
assert poses[True] == poses[False] == {tuple(L.FAN_A), tuple(L.FAN_B)}  # blades visibly turn in both phases
sp = M.Sim(c3)
turns = {True: 0, False: 0}
last = None
for _ in range(256):
    sp.step(0)
    b = sp.scr[7][7]
    if sp.fan_t % 4 == 0:
        if last is not None and b != last:
            turns[bool(sp.fan_t & 64)] += 1
        last = b
assert turns[True] > 3 * turns[False] > 0  # the fan spins up during the gust

# ---------------------------------------------------------------- Room 4: roof lift
c4 = ROOMS[3]
start, goal = (0, 80), (140, 56)
par4 = M.plan(c4, start)
assert not any(y == 88 and x >= 58 for x, y in par4) and not any(y <= 64 for _, y in par4)  # lift required
assert not any(y == 56 for _, y in M.plan(c4, (60, 88)))  # catwalk is 4 rows above the console ledge
for col in list(range(18, 22)) + list(range(25, 29)):  # every catwalk miss is caught by the console ledge
    assert next(r for r in range(7, 23) if (r, col) in {(10, c) for c in range(17, 32)} | {(22, c) for c in range(40)}) == 10
s = M.Sim(c4)
M.drive(s, M.path_to(par4, (46, 184)))
assert M.hold_until(s, 0, lambda t: t.y == 88)  # riding up past the console ledge
on_cl = lambda t: t.y == 88 and t.col >= 17 and not t.air
res = [M.search_jump(s, on_cl, waits=[w]) for w in range(20)]
assert sum(1 for r in res if r) >= 10  # readable transfer: many consecutive frames work
M.apply_jump(s, *next(r for r in res if r))
g4 = M.Sim(c4); g4.x, g4.y = 140, 56  # no remote/early unlock: gate shut, no L03 hand-off
g4.step(M.RIGHT); g4.step(M.RIGHT); assert not g4.exited
M.hold_until(s, M.RIGHT, lambda t: t.col == 26)
far = copy.deepcopy(s); far.x = 60; far.step(M.FIRE); assert not far.sw_on
settle_press(s, M.FIRE)
assert s.sw_on and all(s.scr[r][39] == L.HATCH for r in (3, 4, 5)) and s.scr[10][26] == L.CTRL
M.hold_until(s, M.LEFT, lambda t: t.x <= 58)
on_lift = lambda t: not t.air and t.y == (t.lift_row + 1) * 8 and 0 <= t.col - 13 < 3
res = M.search_jump(s, on_lift, waits=range(0, 330, 2), ks=range(0, 16, 2))
assert res, "reboard"
M.apply_jump(s, *res)
assert M.hold_until(s, 0, lambda t: t.y == 56)
on_top = lambda t: t.y == 56 and not t.air and t.col >= 18
res = M.search_jump(s, on_top, waits=range(12))
assert res, "lift top to catwalk"
M.apply_jump(s, *res)
final = M.plan(blank(c4, (22, 0, 39)), (s.x, 56))
assert (140, 56) in final and M.drive(s, M.path_to(final, (140, 56)))  # crumbling catwalk, real timers
s.step(M.RIGHT); s.step(M.RIGHT)
assert s.exited and s.sw_on and s.deaths == 0

# ---------------------------------------------------------------- softlock closure
for n, start in ((0, (0, 184)), (1, (0, 112)), (2, (0, 96)), (3, (0, 80))):
    reach = M.plan(ROOMS[n], start, xstep=4)
    assert (140, metas[n]["exit_y"]) in reach or n in (0, 3)  # lift rooms need the dynamic lift
    for node in sorted(reach):
        if node[1] != 184:  # every ledge position can leave by walking off an edge, to the floor or lower
            assert any(y > node[1] for _, y in M.plan(ROOMS[n], node, xstep=4)), (n, node)
assert all((x, 184) in M.plan(c1, (x, 184)) for x in (0, 70, 140))

print("test_level02.py: all checks passed")
