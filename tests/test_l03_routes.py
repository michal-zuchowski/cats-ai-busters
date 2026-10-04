"""Route / behaviour checks for the four level-03 rooms, run on the faithful frame model
(tests/l03_model.py) over the generated maps with the unchanged jump physics.  Bots are plain
input policies; nothing here pokes an objective flag - docks, gates, ride and shoves happen only
through the model's own contact/actor code."""
import sys
import re
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from l03_model import Game
from l03_model import SRC, FONT, ActT, ROOT


def body(name):
    text = re.search(r"func " + name + r" \{(.*?)\n\}", SRC, re.S).group(1)
    return re.sub(r"\s+", " ", re.sub(r"//[^\n]*", "", text)).strip()


# Bind the hand-written model's critical branches to the K65 implementation.
for name in ("paw_both", "plat_pursuer", "plat_beam"):
    assert body(name).startswith("a=cur_level a?2 != { return }"), name
assert "a?3 < { a=1 return }" in body("roof_on") and "a+1" not in body("roof_on")
assert "a=act_dir a?0 == { a=act_wob" in body("a_events")
assert "goto" not in body("a_step")
assert "a?m_dock >= { dk=a=240 act_wob=a=0 gate=a=1 tmp=a=0 draw_gate }" in body("a_events")
assert "a?48 >= {" in body("plat_pursuer")
assert "a?153 < { return } a?184 >= { return }" in body("plat_beam")
assert "a=(ap),y a&0x7F a?70 == { a=1 return }" in body("under")
assert "a=e_mode a&3" in body("plat_exit_allowed") and "a=dk" not in body("plat_exit_allowed")
assert "a=gate a?0 == { return }" in body("plat_exit_allowed")
assert all("a=e_mode a&3" in body(n) for n in ("act_init", "plat_reset"))
assert "a=e_mode a&1 a?0 != { a=gate a?0 != { x=9 } }" in body("plat_hud")
assert "a=e_mode a&2 a?0 != { a=gate a?0 == { return } }" in body("on_vacuum")
assert "a=gate a?0 != { a=e_mode a&2 a?0 != { wm=a=m_xmax } }" in body("a_step")
assert '"EXIT   "' in SRC and "HintOff { 0 7 14 21 28 35 42 49 56 63 }" in SRC
assert body("respawn").count("crumble_reset") == 1 and "== { plat_reset }" in body("respawn")
frame_order = body("plat_frame")
assert [frame_order.index(n) for n in ("plat_phys", "plat_actor", "plat_pursuer", "paw_both", "plat_hit", "plat_beam")] == sorted(
    frame_order.index(n) for n in ("plat_phys", "plat_actor", "plat_pursuer", "paw_both", "plat_hit", "plat_beam"))
assert "x?4 >= { x?8 < { a=52 } }" in body("draw_player")

# Compare the inexpensive seated-edge predicate with actual opaque art, not another rectangle model.
atlas = (ROOT / "assets/cat-sprites.bin").read_bytes()
mapping = (ROOT / "assets/cat-frame-map.bin").read_bytes()


def seated_overlap(px, logical, bcol):
    sprite = atlas[mapping[logical] * 128:(mapping[logical] + 1) * 128]
    for y in range(24):
        for x in range(16):
            opaque = any(sprite[p * 32 + y] & (128 >> (x % 8)) for p in (x // 8, 2 + x // 8))
            wx, wy = px + x - bcol * 4, 171 + y - 192
            if opaque and 0 <= wx < 12 and 0 <= wy < 16:
                tile = ActT[6 + (wy // 8) * 3 + wx // 4]
                if FONT[tile * 8 + wy % 8] & (192 >> (2 * (wx % 4))):
                    return True
    return False


for px in range(66, 78):  # all sub-pixel positions across the carrier's three support columns
    for bcol in range(17):  # from the pursuer's home to dome-to-dome contact
        for logical in range(50, 54):  # seated/shove, both facings
            g = Game(2)
            g.player_x, g.player_y = px, 171
            g.set_col()
            g.ride, g.carrier, g.facing = 1, 0, logical & 1
            g.act_col[0], g.act_col[21] = 19, bcol
            assert bool(g.b_touch()) == seated_overlap(px, logical, bcol), (px, bcol, logical)
g.lnd = 10
assert not g.b_touch()  # forepaw-first landing is not the seated silhouette yet
g.lnd, g.jprep = 0, 3
assert not g.b_touch()
g.jprep, g.player_x = 0, 62
g.act_col[21] = 0
g.set_col()
assert not g.on_vacuum()  # no unsupported one-column left overhang


def run(g, pol, n=3000, stop=("respawn",)):
    ev = []
    for i in range(n):
        g.frame(pol(g, i))
        ev += g.events
        g.events = []
        if any(e in ev for e in stop) or g.level_finished or any(e.startswith("room") for e in ev):
            break
    return ev, i


def exit_pol(g):
    if g.m["exit_y"] < 184 and g.player_col >= 30 and g.player_y == 184 and not g.air and not g.jprep and not g.lnd:
        return {"up": 1, "right": 1}
    return {"right": 1}


def swat_pol(dd=3, stop_col=28, hold=None):
    st = {"f": 0}

    def pol(g, i):
        d = g.act_col[0] - g.player_col
        if st["f"] and i < st["f"] + 2:
            return {}
        if not st["f"] and 0 <= d <= dd and g.act_dir[0] == 1 and not g.paw_t and g.player_col < stop_col:
            st["f"] = i
            return {"fire": 1}
        if hold and g.player_col >= hold and not st["f"]:
            return {}
        return exit_pol(g)
    return pol


def ride_pol(shoves=True, jump_col=28, board_col=4, rear=0):
    st = {"f": 0}

    def pol(g, i):
        d = g.act_col[0] - g.player_col
        if not g.ride and g.player_y == 168 and g.player_col >= 33 and not g.air:
            return {"right": 1}  # on the exit ledge
        if not g.ride:
            if g.air:
                return {"right": 1}
            if g.player_col < board_col:
                return {"right": 1}
            if 0 <= d <= 4 and g.player_y == 184:
                return {"up": 1, "right": 1}
            return {}
        if st["f"] and i < st["f"] + 2:
            return {}
        rel = g.player_col - g.act_col[0]
        if shoves and g.bk < 6 and not g.air and not g.jprep and g.act_col[0] < jump_col:
            if rel > rear:
                return {"left": 1}
            if rel < rear or g.facing:
                return {"right": 1}
        if shoves and g.bk >= 6 and g.facing and not g.paw_t and not g.air:
            return {"right": 1}
        if shoves and g.bk >= 6 and not g.paw_t and not g.air and not g.jprep:
            st["f"] = i
            return {"fire": 1}
        if g.act_col[0] >= jump_col:
            return {"up": 1, "right": 1}
        return {}
    return pol


def idle_until(g, n, pred):
    for _ in range(n):
        if pred(g):
            return True
        g.frame({})
    return pred(g)


def clean(g):  # screen buffer equals the art except where an actor is drawn
    cols = set()
    for s in (0, 21):
        if s == 21 and not g.e_bspd:
            continue
        cols |= {(g.m["arow"] + r, g.act_col[s] + c) for r in (0, 1) for c in range(3)}
    for r in range(24):
        for c in range(40):
            if (r, c) in cols:
                continue
            expected = g.art[r][c]
            if g.e_mode & 3 and not g.gate and c == 39 and (g.m["exit_y"] >> 3) - 4 <= r < (g.m["exit_y"] >> 3) - 1:
                expected = 81
            if g.e_mode & 1 and not g.dk and r == 19 and g.m["z0"] <= c <= g.m["z1"]:
                expected = 79 if g.rtclok & 16 else 0xCF
            if g.e_mode & 2 and r == 20 and g.m["z0"] <= c < g.m["z0"] + 3:
                expected = 84 if g.gate else 83
            if g.scr[r][c] != expected:
                return False
    return True


# ---------------------------------------------------------------- R1 cleaning corridor
ev, n = run(Game(0), lambda g, i: exit_pol(g))
assert "respawn" in ev and "hit0" not in ev, ("R1 floor bypass must fail", ev)
ev, n = run(Game(0), swat_pol())
assert ev[:1] == ["hit0"] and "room1" in ev and "respawn" not in ev, ev
g = Game(0)  # timed jump over the robot, no FIRE at all
st = {}


def jump_pol(g, i):
    d = g.act_col[0] - g.player_col
    if g.player_col < 14:
        return {"right": 1}
    if "j" not in st and g.act_dir[0] == 1 and 0 <= d <= 4 and g.player_y == 184 and not g.air:
        st["j"] = 1
        return {"up": 1, "right": 1}
    return exit_pol(g) if "j" in st else {}
ev, n = run(g, jump_pol)
assert "room1" in ev and "respawn" not in ev and "hit0" not in ev, ev
g = Game(0)  # safe niche: climb the two platforms and wait while the robot patrols
climb = {"s": 0}


def niche(g, i):
    if g.air or g.jprep:
        return {"right": 1}
    if g.player_y == 168 and g.player_col >= 18 and not g.air:
        return {}
    if g.player_y == 184 and not g.air and g.player_col >= 7 and g.player_col < 12:
        return {"up": 1, "right": 1}
    if g.player_y == 168 and g.player_col < 14 and not g.air:
        return {"right": 1}
    if g.player_y == 168 and not g.air and g.player_col >= 14:
        return {"up": 1, "right": 1}
    return {"right": 1} if g.player_y == 184 and g.player_col < 7 else {}
ev, n = run(g, niche, n=1000)
assert "respawn" not in ev and n == 999 and g.player_y == 168 and g.player_col >= 18, (ev, g.player_y, g.player_col)

# ---------------------------------------------------------------- R2 charger
g = Game(1)
assert not g.dk and g.scr[21][22] == g.art[21][22] == 12 and g.scr[20][22] == 88
assert g.m["exit_y"] == 184 and not g.gate and all(g.scr[r][39] == 81 for r in (19, 20, 21))
policy = swat_pol(1, 14, 13)
def floor_exit_pol(g, i):
    inp = policy(g, i)
    assert not inp.get("up") and g.player_y == 184 and not g.air
    return inp
ev, n = run(g, floor_exit_pol, n=800, stop=("room2",))  # real dock, then an uninterrupted floor route
assert ev[:2] == ["hit0", "dock"] and "room2" in ev and "respawn" not in ev, ev
g = Game(1)  # strike docks, then lasers (kill zone) are off; wait for the timeout; vacuum recovers
pol = swat_pol(1, 14, 13)
for i in range(400):
    g.frame(pol(g, i) if not g.dk else {})
    if g.dk:
        break
assert g.dk == 240 and g.act_col[0] >= 22 and "dock" in g.events
assert g.gate and all(g.scr[r][39] == g.art[r][39] == 85 for r in (19, 20, 21))
col0 = g.act_col[0]
assert idle_until(g, 400, lambda g: g.dk == 0)
assert "undock" in g.events and g.act_wob[0] == 24 and g.act_dir[0] == 1 and g.act_col[0] == col0
for _ in range(80):
    g.frame({})
assert g.act_col[0] < col0  # it patrols again (retreats toward the cat side)
g = Game(1)  # reproduce a slow player: the laser timer ends, but the opened door stays open
pol = swat_pol(1, 14, 13)
for i in range(800):
    g.frame(pol(g, i))
    if g.dk and g.player_col >= 31:
        break
else:
    raise AssertionError("no docked floor approach")
assert g.gate and not g.air and g.player_y == 184
g.events = []
for _ in range(350):
    g.frame({})
assert not g.dk and g.gate and "respawn" not in g.events
assert all(g.scr[r][39] == 85 for r in (19, 20, 21)) and g.plat_exit_allowed()
assert g.scr[19][31] & 127 == 79  # laser returns without relocking the door
ev, n = run(g, lambda g, i: {"right": 1}, n=200, stop=("room2",))
assert "room2" in ev and "respawn" not in ev
g = Game(1)  # misses retry: no-fire walking into the vacuum respawns, state resets
ev, n = run(g, lambda g, i: exit_pol(g))
assert "respawn" in ev and not g.dk and g.act_col[0] == g.m["acol"] and clean(g)
g = Game(1)  # lasers: a jump while the beam is live kills, a walk at floor height does not
for _ in range(30):
    g.frame({})
g.player_x, g.player_y, g.air = 96, 168, 1
g.set_col()
g.events = []
for _ in range(30):
    g.frame({})
    if g.events:
        break
assert "respawn" in g.events, "airborne in the laser band must die"
g = Game(1)
g.act_col[0], g.act_wob[0] = 22, 0
g.player_x, g.player_y = 96, 184
g.set_col()
g.dk = 100  # docked: the beam is off, so the same jump is safe
for _ in range(5):
    g.frame({})
assert not g.events or "respawn" not in g.events
g = Game(1)  # a cat standing at the robot's column (immune after a swat) keeps its floor
g.player_x, g.player_y = 4 * (g.act_col[0] - 1) - 2, 184
g.set_col()
assert g.scr[22][g.player_col] in (95, 124, 125) and g.art[22][g.player_col] == 70
g.player_y = 184
g.air = 0
g.act_wob[0] = 40
assert g.under()
g.art[22][g.player_col] = 79 | 0x80  # a pit beam under a base drawing is not support
assert not g.under()
g = Game(1)
g.player_x, g.player_y = 140, 184
g.set_col()
g.plat_exit()
assert g.cur_room == 1  # end jump / walking under the beam cannot skip the charger objective
g.dk = 1
assert not g.plat_exit_allowed()  # a timer alone is not the completed charger objective
g.gate = 1
assert g.plat_exit_allowed()
g.dk = 0
assert g.plat_exit_allowed()  # docking unlock persists after its short hazard window
g.player_y = 168
assert not g.plat_exit_allowed()  # former podium height is no longer the exit
g.respawn()
assert not g.gate and not g.dk and all(g.scr[r][39] == 81 for r in (19, 20, 21))
g.act_col[0], g.act_wob[0], g.act_dir[0] = 23, 40, 1
g.a_events()
assert not g.dk  # striking from the wrong side must not dock the vacuum
g.act_dir[0] = 0
g.a_events()
assert g.dk == 240
assert g.gate and all(g.scr[r][39] == 85 for r in (19, 20, 21))
g = Game(1)
g.player_x = 60
g.set_col()
for y, danger in ((152, False), (153, True), (183, True), (184, False)):
    g.player_y = y
    g.events = []
    g.plat_beam()
    assert ("respawn" in g.events) == danger, (y, g.events)
    g.player_x = 60
    g.set_col()

# ---------------------------------------------------------------- R3 ride and shove
# no ride at all: the floor stops at the pit
g = Game(2)
ev, n = run(g, lambda g, i: exit_pol(g))
assert "respawn" in ev
def rear_pol(rel_t, jump_col=99):  # board, then stand at carrier column act+rel_t facing the travel direction
    base = ride_pol(shoves=False, jump_col=jump_col)

    def pol(g, i):
        if g.ride and not g.air:
            rel = g.player_col - g.act_col[0]
            if rel > rel_t:
                return {"left": 1}
            if rel < rel_t or g.facing:
                return {"right": 1}
            return {}
        return base(g, i)
    return pol
g = Game(2)  # rear of the roof, no FIRE: the pursuer's dome really touches the rider and catches it
ev, n = run(g, rear_pol(0, jump_col=28), n=2500)
assert ev == ["respawn"] and g.ride == 0 and g.bk == 0 and clean(g) and g.act_col[0] == g.m["acol"], ev
g = Game(2)  # documented dodge: standing at the very front edge never overlaps the gap-3 pursuer
ev, n = run(g, rear_pol(2), n=2500)
assert ev == [] and g.bk == 0
g = Game(2)
ev, n = run(g, ride_pol(), n=1500, stop=("room3",))
assert "hit21" in ev and "respawn" not in ev and "room3" in ev, ev
# a true seated shove leaves the carrier alone: clone the same moment with and without FIRE
a = Game(2)
pa = rear_pol(0)
for i in range(1500):
    a.frame(pa(a, i))
    if a.ride and a.bk >= 6 and not a.facing and not a.paw_t:
        break
assert a.ride == 1 and a.carrier == 0 and a.bk >= 6 and a.paw_kind in (0, 2)
b = a.clone()
a.frame({"fire": 1})
b.frame({})
assert a.paw_kind == 2  # seated swing
trail_a, trail_b = [], []
for _ in range(40):
    a.frame({})
    b.frame({})
    trail_a.append((a.act_col[0], a.act_dir[0], a.act_t))
    trail_b.append((b.act_col[0], b.act_dir[0], b.act_t))
assert "hit21" in a.events and "hit21" not in b.events and "respawn" not in a.events
assert trail_a == trail_b, "carrier column/direction/step cadence unchanged by the shove"
assert a.act_wob[21] > 0 or a.act_col[21] < b.act_col[21] or a.bk == 0 < b.bk
assert a.bk == 0 and a.act_col[21] < b.act_col[21]
# early dismount: jumps from the roof at cols 20..24 fall into the pit; then retry/reboard works
for jc in (21, 23):
    g = Game(2)
    ev, n = run(g, ride_pol(shoves=True, jump_col=jc), n=1500, stop=("room3",))
    assert "respawn" in ev and "room3" not in ev, (jc, ev)
    ev2, _ = run(g, ride_pol(shoves=True), n=1500, stop=("room3",))
    assert "room3" in ev2, ("reboard after retry", ev2)
# a jump that lands on the post-pit floor (not the podium) still needs the exit jump
g = Game(2)
ev, n = run(g, ride_pol(jump_col=26), n=1500, stop=("room3",))
assert "respawn" not in ev and g.player_y == 184 and g.player_col >= 33, (ev, g.player_y, g.player_col)
ev, n = run(g, lambda g, i: exit_pol(g), n=300, stop=("room3",))
assert "room3" in ev

# ---------------------------------------------------------------- R4 reader, gate, ride, shove, exit
g = Game(3)
assert g.gate == 0
assert g.e_bspd == 0 and not g.on_vacuum()
g.frame({"up": 1})
for _ in range(60):
    g.frame({})
assert g.player_y == 184 and not g.ride and not g.gate  # the formerly stuck rear carrier is absent
# The shared roof helper also rejects any inactive parked pursuer if one is configured later.
g.e_bspd, g.player_x, g.player_y, g.act_col[21] = 2, 0, 171, 0
g.set_col()
assert not g.on_vacuum()
g.gate = 1
assert g.on_vacuum() == 21
g = Game(3)
ev, n = run(g, lambda g, i: exit_pol(g))
assert "respawn" in ev and "gate" not in ev  # walking the floor never opens the gate
g = Game(3)
ev, n = run(g, swat_pol(2, 28, 7), n=600, stop=("respawn", "gate"))
assert ev[-2:] == ["hit0", "gate"] or ev[-1] == "gate", ev
assert g.gate == 1 and g.m["z0"] <= g.act_col[0] <= g.m["z1"] and g.scr[19][39] == g.art[19][39]
assert all(g.scr[20][c] == 84 for c in (12, 13, 14))
g = Game(3)  # no gate: even a jump onto the ledge cannot leave the room
g.player_x, g.player_y = 140, 184
g.set_col()
g.frame({"right": 1})
assert not g.level_finished and "level_end" not in g.events
def service_pol():
    sw, rd = swat_pol(2, 28, 7), ride_pol(shoves=False, jump_col=19)
    def pol(g, i):
        if not g.gate:
            return sw(g, i)
        if g.player_col >= 33 and not g.air:
            return {"right": 1}
        if g.player_y == 152 and not g.air:
            return {"right": 1} if g.player_col < 27 else {"up": 1, "right": 1}
        return rd(g, i)
    return pol

g = Game(3)
pol = service_pol()
heights = set()
for i in range(1500):
    g.frame(pol(g, i))
    if not g.air:
        heights.add(g.player_y)
    if g.level_finished or "respawn" in g.events:
        break
assert g.events == ["hit0", "gate", "level_end"], (g.events, g.player_x, g.player_y)
assert {171, 152, 184} <= heights and g.player_y == 184 and g.player_col >= 34
# retry reset clears gate, ride, dk, and redraws the gate wall
g.respawn()
assert g.gate == 0 and g.ride == 0 and g.dk == 0 and g.bk == 0 and g.act_col[0] == g.m["acol"]
assert any(g.scr[r][39] != g.art[r][39] for r in range(17, 21))  # gate wall redrawn
assert all(g.scr[20][c] == 83 for c in (12, 13, 14))
# every genuine rightward strike at a normal patrol position that connects reaches the reader within the stun
for stand in range(17, 22):
    for wait in range(120):
        g = Game(3)
        g.player_x = stand
        g.set_col()
        g.run({}, wait)
        g.frame({"fire": 1})
        hit_t = None
        for t in range(140):
            g.frame({})
            if "hit0" in g.events:
                hit_t = t
            assert "respawn" not in g.events
            if "gate" in g.events:
                assert hit_t is not None and t - hit_t < 40, (stand, wait)  # gate while still stunned
            g.events = []
        assert (hit_t is None) == (not g.gate), (stand, wait)  # no hit-without-gate
# ---------------------------------------------------------------- R4 captions and reader hooks (source/data)
goal = re.search(r"data R4Goal \{(.*?)\n\}", SRC, re.S).group(1)
lines = re.findall(r'"([^"\n]*)"', re.sub(r"charset [^\n]*", "", goal))
assert "nocross" in goal and len(lines) == 4 and all(len(x) == 36 for x in lines), lines
assert lines[0].startswith("SWAT THE VACUUM") and "READER" in lines[0]
assert "BALCONY" in lines[1] and lines[1].startswith("RIDE") and "GAP" in lines[2] and "EXIT" in lines[2]
assert lines[3].startswith("SERVICE ACCESS OPEN") and "RIGHT" in lines[3]
assert set("".join(lines)) <= set(" !~#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ")
assert "data R4Off { 0 36 72 108 }" in SRC and sum(map(len, lines)) == 144  # 144 bytes < one page
hints = re.sub(r"charset [^\n]*", "", re.search(r"data Hints \{.*?\n\}", SRC, re.S).group(0))
names = [x.strip() for x in re.findall(r'"([^"\n]*)"', hints)]
r4h = list(map(int, re.search(r"R4Hint \{ ([\d ]+) \}", SRC).group(1).split()))
assert [names[i] for i in r4h] == ["SWAT", "RIDE", "JUMP", "EXIT"], names
rg = body("r4_goal")
assert rg.startswith("x=0 a=gate a?0 != { x=1 a=player_col a?23 >= { x=2 } a?33 >= { x=3 } }"), rg
assert "msg_len=a=36 draw_row0_msg" in rg and "msg_src+1=a=&>R4Goal" in rg and "a=R4Off,x" in rg
assert "a=R4Hint,x x=a a=ride a?0 != { a=act_col a?17 >= { x=1 } }" in rg
assert "a=cur_room a?3 == { r4_goal }" in body("plat_hud")
# draw_reader: reader room only; RELAY (83) until the crossing, then DONE (84); not the R2 dock.
assert body("draw_reader").startswith("a=cur_level a?2 != { return } a=e_mode a&2 a?0 == { return } tmp=a=83 a=gate a?0 != { tmp=a=84 }")
assert "x=20 y=m_z0 cnt=a=3 put_run" in body("draw_reader")
ae = body("a_events")
dock, reader = ae.split("a=e_mode a&2 a?0 == { return }")
assert "draw_reader" not in dock and "draw_gate }" in dock  # R2 dock: no reader redraw
assert reader.endswith("gate=a=1 tmp=a=0 draw_gate draw_reader") and reader.count("draw_reader") == 1
assert "a=m_arow a?0 == { return } draw_reader draw_act" in body("act_init")  # retry/reset restores RELAY
assert body("plat_reset").endswith("act_init") and body("act_init").count("draw_reader") == 1
assert sum(body(n).count("draw_reader") for n in re.findall(r"func (\w+) \{", SRC)) == 2  # act_init + a_events only

# R2 dock latch: after the wobble recovery the normal patrol limit applies again.
g = Game(1)
pol = swat_pol(1, 14, 13)
for i in range(400):
    g.frame(pol(g, i) if not g.dk else {})
    if g.dk:
        break
assert g.gate and g.e_mode & 1 and not g.e_mode & 2
assert idle_until(g, 400, lambda g: g.dk == 0 and g.act_wob[0] == 0)
peak = 0
for _ in range(500):
    g.frame({})
    if g.act_col[0] <= g.m["amax"]:
        peak = max(peak, g.act_col[0])
    assert g.act_col[0] <= g.m["amax"] or g.act_dir[0] == 1  # may only be above amax while heading back
assert peak <= g.m["amax"]

# ---------------------------------------------------------------- R4 robustness: timing, dismount, recovery
def r4_bot(stand=17, board=(5, 7), edge=11, rear=0, dismount=None, delay=0, retry_shift=True, mid_x=99):
    """Input policy only: swing again after a miss, wait at the first pit for the patrolling
    vacuum, jump to its roof, ride, leave at the balcony and cross the gaps by position."""
    st = {"cool": 0, "swings": 0, "left": 0, "mid": 0, "deaths": 0}

    def pol(g, i):
        col, robot, x = g.player_col, g.act_col[0], g.player_x
        win = [b - (st["deaths"] if retry_shift else 0) for b in board]
        if i < delay:
            return {}
        if not g.gate:
            if st["cool"]:
                st["cool"] -= 1
            elif g.paw_hit and g.act_wob[0]:
                return {}  # carried toward the reader; hits land at the patrol columns 8..10 and reach the reader in time
            elif x >= stand and not g.paw_t and not g.air:
                st["swings"] += 1
                st["cool"] = (11, 16, 21)[st["swings"] % 3]
                return {"fire": 1}
            elif x < stand:
                return {"right": 1}
            return {}
        if g.player_y == 184 and not g.air and not g.ride and col >= 33:
            return {"right": 1}
        if g.player_y == 184 and not g.air and not g.ride and 23 <= col < 33:
            st["mid"] = 1  # landed on the floor between the pits: stop at its edge, then leap
            return {"right": 1} if x < mid_x else {"up": 1, "right": 1}
        if g.player_y == 152 and not g.air:
            return {"right": 1} if col < 27 else {"up": 1, "right": 1}
        if g.air:
            return {"right": 1}
        if g.ride:
            if dismount is not None and dismount <= robot < 19 and not st["left"]:
                st["left"] = 1
                return {"up": 1, "right": 1}
            if robot >= 19:
                return {"up": 1, "right": 1}
            return {"right": 1} if col - robot < rear else {"left": 1} if col - robot > rear else {}
        if g.player_y == 184:
            if g.act_dir[0] == 1 and win[0] <= robot - col <= win[1]:
                return {"up": 1, "right": 1}
            if col < edge:
                return {"right": 1}
        return {}
    pol.st = st
    return pol


def r4_run(n=6000, **kw):
    g, pol, deaths, seen = Game(3), r4_bot(**kw), 0, set()
    for i in range(n):
        g.frame(pol(g, i))
        deaths += g.events.count("respawn")
        pol.st["deaths"] = deaths
        g.events = []
        if not g.air:
            seen.add(g.player_y)
        if g.level_finished:
            assert g.player_y == 184 and g.player_col >= 34 and g.gate and 184 in seen, seen
            return i + 1, deaths, pol.st
    raise AssertionError(("R4 bot stalled", kw, deaths, g.player_x, g.player_y, g.gate, g.ride))


runs = 0
for stand in (17, 19, 21):  # forepaw contact needs x 17..21 for the 8..10 patrol
    for board in ((5, 5), (6, 6), (7, 7), (5, 7)):  # roof-jump distance: a 3-column, ~24 frame window
        for rear in (0, 1, 2):
            n, deaths, _ = r4_run(stand=stand, board=board, rear=rear, delay=3 * stand % 17)
            assert deaths == 0, (stand, board, rear, deaths)
            runs += 1
mids = []
for dismount in (16, 17, 18):  # leaving the carrier early: balcony or mid-floor landing, no retry needed
    n, deaths, st = r4_run(dismount=dismount)
    assert deaths == 0 and st["left"] == 1, (dismount, deaths)
    mids.append(st["mid"])
assert mids == [0, 1, 1]  # the mid-floor edge leap recovers without a respawn
for mx in (99, 103, 106, 109):  # the lower mid floor (cols 23..29) now gives an 11 px jump window
    for dismount in (17, 18):
        n, deaths, st = r4_run(dismount=dismount, mid_x=mx)
        assert deaths == 0 and st["mid"] == 1, (mx, dismount, deaths)
for x in range(90, 112):  # walking-off positions: only the first pit and the 110.. void are fatal
    g = Game(3)
    g.gate, g.player_x, g.player_y = 1, x, 184
    g.set_col()
    dead = False
    g.frame({"up": 1, "right": 1})
    for _ in range(200):
        g.frame({"right": 1})
        dead |= "respawn" in g.events
        if g.player_col >= 34 and not g.air:
            break
    assert dead == (x < 99) or x >= 110, (x, dead)  # 99..109 clear the second pit onto floor 33..39
late = [(b, r4_run(board=(b, b))[1]) for b in (8, 9)]  # a late jump falls in the pit, resets the room, retries
assert late == [(8, 1), (9, 2)], late
print(f"R4 model timing/boarding variants: {runs} complete with 0 respawns; dismount {mids}; late jumps {late}")

# A rider boarded before the reader opens cannot carry the cat past the first pit: no softlock, no bypass.
g, pol = Game(3), None
for i in range(3000):
    robot, col = g.act_col[0], g.player_col
    if g.ride:
        inp = {"up": 1, "right": 1} if robot >= 9 and not g.air else {}
    elif g.air or (g.player_y == 184 and g.player_x < 18):
        inp = {"right": 1}
    elif g.player_y == 184 and g.act_dir[0] == 1 and 0 <= robot - col <= 4:
        inp = {"up": 1, "right": 1}
    else:
        inp = {}
    g.frame(inp)
    g.events = []
    assert not g.gate and not g.level_finished, i
assert g.m["xmax"] == 19 and g.m["amax"] == 10

print("test_l03_routes.py: all checks passed")
