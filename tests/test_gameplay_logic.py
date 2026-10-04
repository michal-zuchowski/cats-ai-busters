"""Structural + behavioral checks for Level 01 ("The Blind Spot") gameplay
in main.k65.

Mirrors the style of test_intro_scene.py: parse the K65 source as text and
assert invariants the compiler itself cannot check for us, since compiling
or running the level is out of scope for this session (see AGENTS.md).

Beyond pure threshold/string matching, several checks below pin down the
*meaning* of specific branches (which literal a= value each condition
selects) so that reintroducing the switch-polarity bug, the redraw/erase-cat
bug, or the frozen switch_timer bug would fail a test, not just a search.
"""

from pathlib import Path
import re
import sys


root = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(root / "tools"))
import make_level_art as L
main = (root / "main.k65").read_text()
room_art = (root / "assets/level-rooms.pic").read_bytes()


def body(name):
    match = re.search(r"(?:data|func|inline|naked) " + name + r" \{(.*?)\n\}", main, re.S)
    assert match, name
    return match.group(1)


def thresholds(name):
    """Literal column thresholds: every `x?N` (a CPX comparing the current
    column against N), as opposed to other state compared via `a?0` (e.g.
    cam_state, switch_timer)."""
    return [int(n) for n in re.findall(r"x\?(\d+)", body(name))]


def data_bytes(name):
    match = re.search(r"data " + name + r" \{([^}]*)\}", main, re.S)
    assert match, name
    b = match.group(1)
    b = re.sub(r"//[^\n]*", "", b)  # strip line comments (they may contain digits)
    return [int(n) for n in re.findall(r"\d+", b)]


def ascii32(text):
    return [ord(c) - 32 for c in text]


def branch_values_inline(text):
    """`a?0 == { a=X } else { a=Y }` -> (X, Y): the values chosen when the
    tested byte is zero, and when it is nonzero, respectively."""
    m = re.search(r"a\?0\s*\n\s*==\s*\{\s*a=(\d+)\s*\}\s*else\s*\{\s*a=(\d+)\s*\}", text)
    assert m, text
    return int(m.group(1)), int(m.group(2))


# ---------------------------------------------------------------------------
# Cameras: one rule (cam_sees) decides both the lit floor that is drawn and
# whether the cat is caught, so visuals and detection cannot drift apart.
assert "cam_sees" in body("repaint_current")
assert "cam_sees" in body("check_detection")
cams = (root / "assets/level-cams.bin").read_bytes()
assert len(cams) == 3 * 2 * 8
assert len((root / "assets/cone-step.bin").read_bytes()) == 160
for r in range(3):
    for s in range(2):
        col, lo, hi, speed, pause, hw, gate, _ = cams[(r * 2 + s) * 8:(r * 2 + s + 1) * 8]
        if col:
            assert lo <= col <= hi <= 39 and speed and hw
            assert lo - hw > 2  # room entry column is never watched
# Only room 1's first camera is gated by the switch.
assert [cams[(r * 2 + s) * 8 + 6] for r in range(3) for s in range(2)] == [0, 0, 1, 0, 0, 0]
# Switch polarity: switch_timer counts frames the gated camera stays dark.
assert "dark" in re.search(r"var switch_timer.*", main).group(0)
assert re.search(r"a=switch_timer\s*\n\s*a\?0\s*\n\s*==\s*\{\s*a=1\s*\}\s*else\s*\{\s*a=0\s*\}",
                 body("tick_cam"))

# switch_timer must count down every gameplay frame; cameras tick too.
tick_switch = body("tick_switch")
assert re.search(r"switch_timer\s*--", tick_switch)
game_frame = body("game_frame")
assert "tick_switch" in game_frame and "tick_cams" in game_frame

# ---------------------------------------------------------------------------
# redraw_zone repaints monitored floor and sprite rows; after every pass the
# four-column cat must be restored, including during alert recovery.
redraw_call_sites = [
    m.start() for m in re.finditer(r"\bredraw_zone\b", main)
    if main[max(0, m.start() - 5):m.start()] != "func "
]
assert len(redraw_call_sites) >= 2, "expected redraw_zone called from game_frame's two branches"
for pos in redraw_call_sites:
    tail = main[pos + len("redraw_zone"):pos + len("redraw_zone") + 200]
    tail_no_comments = re.sub(r"//[^\n]*", "", tail)
    next_call = re.search(r"\S+", tail_no_comments)
    assert next_call and next_call.group(0) == "draw_player", (
        "redraw_zone call not immediately followed by draw_player", tail
    )

# The cat is a 16x24 PMG sprite (P0/P1 dark, P2/P3 fur): a
# 12-drawing stride picked from the pixel x (planted paws do not slide),
# 2 idle, mirrored set.  All four legs are fur (P2/P3), so the near and far
# pairs look alike and the stride repeats after 6 drawings.
sprites = (root / "assets/cat-sprites.bin").read_bytes()
# Packed atlas: identical drawings share a physical frame; FrameMap maps logical -> physical.
frame_map = (root / "assets/cat-frame-map.bin").read_bytes()
assert len(frame_map) == 58 and len(sprites) == (max(frame_map) + 1) * 128
assert len(sprites) == 43 * 128 and 'binary "assets/cat-frame-map.bin"' in main
assert sorted(set(frame_map)) == list(range(43))
frames = [sprites[p * 128:(p + 1) * 128] for p in frame_map]
assert len(frames) == 58
assert len({f for f in frames}) == 43 and len(sprites) < 50 * 128  # packed atlas
import hashlib
assert hashlib.sha256(b"".join(frames[:50])).hexdigest() == "015127fe60b2317ff0b6e85f8503c8a8d122624b32cfdaa93b57af2003c159b6"
assert re.search(r"func draw_frame \{[^}]*?x=a\s*a=FrameMap,x\s*x=a", main, re.S)
assert len(set(frames[:12])) == 6 and frames[12] != frames[13]
assert all(f[23] | f[32 + 23] | f[22] | f[32 + 22] == 0 for f in frames[:14])  # no dark legs
for f in frames:
    assert all(b == 0 for chunk in range(4) for b in f[chunk * 32 + 24:chunk * 32 + 32])
assert sum(bool(f[64 + 23] | f[96 + 23]) for f in frames[:12]) >= 8
def fur_pixels(frame, row):
    return [x for x in range(16) if frame[64 + (x // 8) * 32 + row] & (0x80 >> (x % 8))]

for frame in frames[29:32]:
    paws = [x for row in range(18, 24) for x in fur_pixels(frame, row)]
    assert paws and min(paws) >= 1 and max(paws) <= 14
for frame in frames[29:31]:
    muzzle = max(x for row in range(4, 12) for x in fur_pixels(frame, row))
    assert max(fur_pixels(frame, 12)) > muzzle
    assert max(fur_pixels(frame, 14)) > muzzle
    assert fur_pixels(frame, 12)[-3:] == [13, 14, 15]
    assert fur_pixels(frame, 14)[-3:] == [13, 14, 15]
assert any(fur_pixels(frames[30], row) != fur_pixels(frames[12], row) for row in range(10, 17))
assert len(set(frames[28:36])) == 8
# Fore paws touch the floor before the hind paws (frame 32).
assert fur_pixels(frames[32], 23) and min(fur_pixels(frames[32], 23)) >= 11
for frame in frames[31:33]:
    hind_lowest = max(row for row in range(17, 24) if any(x < 8 for x in fur_pixels(frame, row)))
    fore_lowest = max(row for row in range(17, 24) if any(x >= 10 for x in fur_pixels(frame, row)))
    assert fore_lowest == 23 and hind_lowest <= 20
assert any(x < 8 for x in fur_pixels(frames[33], 23))
assert fur_pixels(frames[33], 23) == fur_pixels(frames[34], 23)  # planted paws do not slide
assert frames[35] == frames[12]  # recovery joins the idle pose without a snap
head_rows = [min(row for row in range(24) if any(x >= 10 for x in fur_pixels(frames[k], row)))
             for k in (33, 34, 35)]
assert head_rows[0] > head_rows[1] > head_rows[2]
for k in range(8):
    for row in range(24):
        assert fur_pixels(frames[36 + k], row) == sorted(15 - x for x in fur_pixels(frames[28 + k], row))
assert 14 in fur_pixels(frames[44], 3) and 0 in fur_pixels(frames[45], 19)
assert 15 in fur_pixels(frames[48], 18) and {13, 14} <= set(fur_pixels(frames[48], 19))
assert frames[44] != frames[45] and frames[44] != frames[12]
for k in range(2):
    for row in range(24):
        assert fur_pixels(frames[46 + k], row) == sorted(15 - x for x in fur_pixels(frames[44 + k], row))
for row in range(24):
    assert fur_pixels(frames[49], row) == sorted(15 - x for x in fur_pixels(frames[48], row))
assert len(frames) == 58
assert 15 in fur_pixels(frames[56], 12) and 11 >= max(fur_pixels(frames[55], 12), default=0)
assert max(fur_pixels(frames[54], 12), default=0) <= 11
glass_pixels = {
    (33 * 4 + px, 128 + py)
    for py, row in enumerate(L.TILE_GLYPHS[L.GLASS])
    for px, value in enumerate(row)
    if value != "0"
}
for logical in (54, 55):
    for origin in range(117, 121):
        image_pixels = {
            (origin + x, 120 + y)
            for y in range(24)
            for x in range(16)
            if any(frames[logical][(colour * 2 + (x // 8)) * 32 + y] & (0x80 >> (x % 8))
                   for colour in range(2))
        }
        assert not image_pixels & glass_pixels, (logical, origin, image_pixels & glass_pixels)
for origin in range(117, 121):
    assert (origin + 15, 132) in glass_pixels
    assert any(frames[56][(colour * 2 + 1) * 32 + 12] & 1 for colour in range(2))
draw = body("draw_frame")
assert all(p in draw for p in ("PmMem,x", "PmMem+256,x", "PmMem+512,x", "PmMem+768,x", "HPOSP0", "HPOSP3"))
dp = body("draw_player")
assert "a=player_x" in dp and "a?12" in dp and "c- a+14" in dp
assert "jprep--" in body("plat_input") and "jprep=a=3" in body("plat_input")
assert "a=jprep" in dp and "a=28 jump_facing return" in dp
assert re.search(r"a\?1\s+>= \{ a=31 \} else \{ a=30 \}", dp)
assert "lnd=a=10" in body("plat_phys")
assert re.search(r"a\?9\s+>= \{ a=32 \}", dp)
assert re.search(r"a\?6\s+>= \{ a=33 \}", dp)
assert re.search(r"a\?3\s+>= \{ a=34 \} else \{ a=35 \}", dp)
assert re.search(r"a=lnd\s+a\?0\s+!= \{ return \}", body("plat_input"))
assert "c- a+8" in body("jump_facing")
assert re.search(r"a=vy\s+a\?0\s+== \{ gt=a=2 \}", body("plat_phys"))
assert "paw_t--" in dp and "c- a+44" in dp and "c- a+2" in dp
assert re.search(r"paw_t--\s+a=paw_t\s+a\?3\s+< \{ a=28 jump_facing return \}", dp)
assert "a=48" in dp and "a=49" in dp and "a?7" in dp
swat = body("swat")
assert swat.index("paw_t=a=10") < swat.index("on_vacuum")
assert "paw_kind=a=2" in swat  # seated: hind shove; grounded stays the forepaw swat
assert "act_wob=" not in swat and "act_dir=" not in swat  # FIRE alone cannot frighten the vacuum
contact = body("paw_contact")
assert re.search(r"a=paw_t\s+a\?4\s+< \{ return \}\s+a\?8\s+>= \{ return \}", contact)
assert "a=paw_hit" in contact and "paw_hit=a=1" in contact and "paw_hit=a" in swat
assert re.search(r"a\?kk\s+== \{ return \}\s+\} always", contact)  # loop needs JMP, not a relative branch
assert "a=(sp),y" in contact and "a&PixelMask,x" in contact and "a^1" in contact
assert body("plat_frame").index("plat_actor") < body("plat_frame").index("paw_both") < body("plat_frame").index("plat_hit")
assert "paw_contact" in body("paw_both") and body("paw_both").count("paw_contact") == 2
tips = list(zip(data_bytes("PawX"), data_bytes("PawY")))
assert len(tips) == 7 and data_bytes("PawK") == [0, 3, 5] and data_bytes("PawE") == [3, 5, 7]
for frame, mirror, group in ((48, False, tips[:3]), (49, True, tips[:3]), (45, False, tips[3:5]),
                             (47, True, tips[3:5]), (52, False, tips[5:]), (53, True, tips[5:])):
    for x, y in group:
        assert (15 - x if mirror else x) in fur_pixels(frames[frame], y)
for frame in (52, 53):  # seated shove: toes on the floor row, haunch folded above them
    assert fur_pixels(frames[frame], 23) and fur_pixels(frames[frame - 2], 23)
    assert max(row for row in range(24) if fur_pixels(frames[frame], row)) == 23
assert frames[50] != frames[52] and frames[51] != frames[53] and frames[50] != frames[12]
assert fur_pixels(frames[50], 0) == [9, 12]  # seated head/ears, not the standing back and raised tail
assert all(not any(x < 5 for x in fur_pixels(frames[50], y)) for y in range(9))
assert all(len(fur_pixels(frames[50], y)) <= 7 for y in range(8))
assert all(x in fur_pixels(frames[50], y) for y in range(18, 24) for x in (12,))
assert fur_pixels(frames[50], 21) == list(range(1, 9)) + [12]
assert fur_pixels(frames[50], 22) == list(range(9)) + [12]
assert fur_pixels(frames[50], 23) == list(range(9)) + [11, 12, 13]  # original roof/catch footprint
for row in range(24):
    assert fur_pixels(frames[51], row) == sorted(15 - x for x in fur_pixels(frames[50], row))
    assert fur_pixels(frames[53], row) == sorted(15 - x for x in fur_pixels(frames[52], row))
font = (root / "assets/term-font.pic").read_bytes()
actor_tiles = data_bytes("ActT")

def paw_touches(px, py, facing, col=10, row=21, timer=7):
    if not 4 <= timer < 8:
        return False
    for x, y in tips[:3]:
        x = px + (15 - x if facing else x) - col * 4
        y = py + y - (24 + row * 8)
        if 0 <= x < 12 and 0 <= y < 16:
            tile = actor_tiles[(y // 8) * 3 + x // 4]
            if font[tile * 8 + y % 8] & (192 >> (2 * (x % 4))):
                return True
    return False

assert not paw_touches(24, 184, 0) and paw_touches(25, 184, 0)  # one-pixel gap vs contact
assert not paw_touches(52, 184, 1) and paw_touches(51, 184, 1)
assert not paw_touches(25, 185, 0) and paw_touches(26, 185, 0)  # transparent silhouette corner
assert not paw_touches(25, 184, 0, timer=10)  # raised paw does not hit
assert not paw_touches(25, 184, 0, timer=3)  # recoil cannot hit
assert not paw_touches(25, 160, 0)  # wrong height cannot hit
def seated_touch(px, facing, bcol, timer=7):
    # seated rider origin 171 on a row-21 carrier; pursuer dome tiles drawn with the chase lamp glyphs
    if not 4 <= timer < 8:
        return False
    for x, y in tips[5:]:
        wx = px + (15 - x if facing else x) - bcol * 4
        wy = 171 + y - (24 + 21 * 8)
        if 0 <= wx < 12 and 0 <= wy < 16:
            tile = actor_tiles[6 + (wy // 8) * 3 + wx // 4]
            if font[tile * 8 + wy % 8] & (192 >> (2 * (wx % 4))):
                return True
    return False

assert max(y for x, y in tips[5:]) == 23  # toes reach world Y 194, the dome's first visible row
cat_x = lambda col: col * 4 - 8  # player_col = ((x+2)//4)+2 -> col -2 gives the sprite origin
# right-facing rider (hind paw trailing left): pursuer dome ends just behind the toes
assert seated_touch(cat_x(20), 0, 16) and not seated_touch(cat_x(20), 0, 14)
assert not seated_touch(cat_x(20), 0, 16, timer=3) and not seated_touch(cat_x(20), 0, 16, timer=10)
# mirrored left-facing rider reaches a pursuer on the right only, never the one behind
assert [b for b in range(10, 40) if seated_touch(cat_x(20), 1, b)] == [20, 21]
assert not any(seated_touch(cat_x(20), 1, b) for b in range(10, 20))
assert [b for b in range(10, 40) if seated_touch(cat_x(20), 0, b)] == [16, 17]
# pursuer spacing 4 behind a carrier at col 19: reachable from cat cols 18..19, not from the far end of the roof
assert seated_touch(cat_x(18), 0, 15) and seated_touch(cat_x(19), 0, 15) and not seated_touch(cat_x(21), 0, 15)
brushes = (root / "assets/vacuum-brushes.bin").read_bytes()
assert len(brushes) == 24 and len({brushes[i:i + 3] for i in range(0, 12, 3)}) == 4
assert "TermFont+765,y=a" in body("spin_brushes") and "TermFont+1005,y=a" in body("spin_brushes")
assert "spin_brushes" in body("plat_actor")
assert "c- a+3" in body("vacuum_roof")
for name in ("under", "swat", "plat_actor"):
    assert "on_vacuum" in body(name)
assert "vacuum_roof" in body("hit_check")
assert "a?91" not in body("under") and "a?94" in body("under")
assert re.search(r"player_y\+\+\s+under\s+a\?0", body("plat_phys"))
assert body("plat_actor").index("ride=a=0") < body("plat_actor").index("a?m_aspd")
# The paws rest on the first visible roof row, not five pixels into its dome.
for row in (18, 21):
    roof = row * 8 + 3
    assert roof + 23 == 24 + row * 8 + 2
    for speed in range(1, 6):
        y = roof - 1
        for _ in range(speed):
            y += 1
            if y == roof:
                break
        assert y == roof  # a non-tile-aligned roof cannot be skipped while falling
assert re.search(r"a=act_wob,x\s+a\?0\s+!= \{ return \}", body("hit_check"))

# Exit gates precede both room advancement and the level-ending hand-off.
exit_body = body("plat_exit")
assert re.match(r"\s*plat_exit_allowed\s+a\?0\s+== \{ return \}", re.sub(r"//[^\n]*", "", exit_body))
assert exit_body.index("plat_exit_allowed") < exit_body.index("cur_room++")
assert exit_body.index("plat_exit_allowed") < exit_body.index("call level_ending")
gate = re.sub(r"//[^\n]*", "", body("plat_exit_allowed"))
# unrestricted -> controller gate (sw_on) -> grounded -> exact exit height
assert re.search(r"a=m_exit\s+a\?0\s+== \{ a=1 return \}\s+a=m_swy\s+a\?0\s+!= \{\s+a=sw_on\s+"
                 r"a\?0\s+== \{ return \}\s+\}\s+a=air\s+a\?0\s+!= \{ a=0 return \}\s+"
                 r"a=player_y\s+a\?m_exit\s+== \{ a=1 return \}\s+a=0$", gate.strip())

# Metadata layout: 32 bytes copied, new fields directly after m_exit and below lift_row.
plat_rooms = (root / "assets/plat-rooms.bin").read_bytes()
assert len(plat_rooms) == 10 * 1024
addr = {n: int(a, 16) for n, a in re.findall(r"var (m_\w+) = (0x[0-9A-Fa-f]+)", main)}
assert addr["m_exit"] - addr["m_sx"] == 18
assert [addr[n] - addr["m_sx"] for n in ("m_swy", "m_swc", "m_lpow", "m_fc1", "m_fr0", "m_fr1", "m_lspd")] \
    == [19, 20, 21, 22, 23, 24, 25]
assert "y?32" in body("enter_plat_room")
assert 0x06B0 + 32 <= int(re.search(r"var lift_row = (0x[0-9A-Fa-f]+)", main).group(1), 16)
rooms = [plat_rooms[i * 1024:(i + 1) * 1024] for i in range(10)]
# Level 04 redesign has its own route, hazard and cinematic checks.
import runpy
runpy.run_path(str(root / "tests/test_level04.py"), run_name="level04")

# Level 02 mechanics/route proofs on the frame model (tests/test_level02.py).
runpy.run_path(str(root / "tests/test_level02.py"), run_name="level02")

init_pmg = body("init_pmg")
assert "PMBASE=a=0xB8" in init_pmg and "SDMCTL=a=0x3A" in init_pmg and "GRACTL=a=2" in init_pmg
assert "init_pmg" in body("init_gameplay")
assert "tick_cat" in body("game_frame")
assert body("read_and_move").count("cat_stepped") == 3
assert "set_col" in body("cat_stepped") and "set_col" in body("enter_room")
assert "a?140" in body("read_and_move") and "player_x=a=144" in body("read_and_move")
assert "player_x=a=0" in body("trigger_alert")
assert len(room_art) == 3 * 40 * 24
floor_rows = [room_art[r * 960 + 880:r * 960 + 920] for r in range(3)]
assert all(set(row) <= {70, 78, 81, 82, 83} and 70 in row and row[2] != 70 for row in floor_rows)

# ---------------------------------------------------------------------------
# relay_done is permanent progress and must never be reset by trigger_alert
# (only written to, e.g. `relay_done=a=1`, in handle_relay/init_gameplay).
assert not re.search(r"relay_done\s*(=|\+\+|--)", body("trigger_alert"))

# handle_relay must not touch relay_done unless relay_progress has reached
# the completion threshold used elsewhere in the same function body.
handle_relay = body("handle_relay")
assert re.search(r"a=relay_progress\s*\n\s*a\?90", handle_relay)
assert handle_relay.index("a?90") < handle_relay.index("relay_done=a=1")

# ---------------------------------------------------------------------------
# The objective stays visible and is restored after an alert rather than
# disappearing before the player has learned the controls.
assert "hint_timer" not in game_frame
alert_clear_branch = game_frame[game_frame.index("alert_timer--"):]
assert "clear_row0" in alert_clear_branch and "redraw_zone" in alert_clear_branch
assert "&<MsgStop" in alert_clear_branch and "draw_row0_msg" in alert_clear_branch

# ---------------------------------------------------------------------------
# HUD strings: the ascii-32 encoded data tables must match the literal
# strings documented alongside them, and the msg_len used at each call site
# must match the table's actual length.
hud_strings = {
    "MsgStop": "STOP THE RELAY. STAY UNSEEN.",
    "MsgAlert": "UNRECOGNIZED DEVICE",
    "MsgRoute": "ROUTE: TEST / ISOLATED",
}
assert '"PHASE 02 DELAYED. AGENCY UNDETECTED."' in body("Txt") and "TxtOff { 0 36" in main
for name, text in hud_strings.items():
    assert data_bytes(name) == ascii32(text), name
    length = len(text)
    for call_site in re.finditer(r"&<" + name + r"\b", main):
        # the next `msg_len=a=<N>` after selecting this string must equal
        # its real length, wherever init_gameplay/trigger_alert/etc uses it
        tail = main[call_site.end():call_site.end() + 200]
        m = re.search(r"msg_len=a=(\d+)", tail)
        assert m, name
        assert int(m.group(1)) == length, (name, m.group(1), length)

# "CONTROL INFRASTRUCTURE" is reused verbatim from the intro's Line4 data
# (bytes 10..31, 22 ascii-32 codes) rather than duplicated -- draw_room_header
# must reference Line4+10 with msg_len=22, matching the substring's length.
header = body("draw_room_header")
assert "&<Line4+10" in header and "msg_len=a=22" in header
line4 = data_bytes("Line4")
assert line4[10:32] == ascii32("CONTROL INFRASTRUCTURE")

# The map loader must initialize both background row pointers after its
# three-page copy, and room entry must repaint the dynamic play row.
enter_room = body("enter_room")
assert "redraw_zone" in enter_room and "repaint_current" in body("redraw_zone")
assert enter_room.index("load_room_art") < enter_room.index("load_cams")
loader = body("load_room_art")
assert "call load_room_chunk" in loader and "&<RoomArtBuf" in loader  # a3e.3: per-room disk chunk
assert "call disk_read" in body("load_room_chunk") and "a=ChunkTable,x" in body("load_room_chunk")
assert "cur_room--" in body("read_and_move")
assert "cur_room++" in body("read_and_move")

# The gameplay hand-off must happen after cut_to_black, and level_finished
# must gate the main loop so the ending freezes the game once it plays.
scene = re.search(r"main \{(.*)\}\s*$", main, re.S).group(1)
assert scene.index("init_system") < scene.index("build_storm_list") < scene.index("video_call")
assert "goto intro_skip" not in scene[:scene.index("build_storm_list")]  # intro enabled
tail = scene[scene.index("cut_to_black"):]
assert tail.index("cut_to_black") < tail.index("init_gameplay") < tail.index("frame_any")
assert "level_finished" in tail

# HUD row 24 sits below the floor, recoloured by its own DLI, refreshed every frame.
dl = body("dl_term")
assert "0x90" in dl and dl.index("0x90") < dl.rindex("0x04")
assert "hud_dli" in body("init_gameplay") and "NMIEN=a=0xC0" in body("init_gameplay")
assert "draw_hud" in body("game_frame") and "draw_hud_static" in enter_room
assert "y?232" in body("clear_term_screen")  # 25 rows = 1000 bytes

assert "MsgFireHud" in body("fire_prompt")
assert body("draw_hud").count("fire_prompt") == 2 and "a?6" in body("draw_hud") and "a?33" in body("draw_hud")  # switch + relay prompts

import math
music = (root / "assets/music.bin").read_bytes()
assert len(music) == 435 and 'binary "assets/music.bin"' not in main
assert "init_audio" in body("init_gameplay") and "AUDCTL=a=0x50" in body("init_audio")
for k in set(music[:256]) - {0, 1}:  # every lead note in tune within 2 cents
    f = 1789790 / (2 * (music[336 + k - 2] + 256 * music[352 + k - 2] + 7))
    assert abs(1200 * math.log2(f / 440) % 100 - 50) > 48, f
assert set(music[256:272]) <= {0, 1, 2} and set(music[272:304]) <= {0, 1, 2, 3}
assert "a&127" not in body("music_step") and "Music+400,x" in body("tick_music")  # 256-step song
assert "sfx_timer" in body("tick_music") and "AUDC4" in body("sfx_alert")  # drums yield to SFX
assert "tick_audio" in body("music_vbi_imm") and "tick_audio" not in body("game_frame")  # steady tempo
assert "VVBLKI" in body("init_audio")
audio = re.sub(r"//[^\n]*", "", body("tick_audio"))
assert "call tick_music" in audio and "music_ready" in audio and "sfx_timer--" in audio
assert "AUDC3=a=0" in body("level_ending")
print("test_gameplay_logic.py: all checks passed")

font = (root / "assets/term-font.pic").read_bytes()
assert font[73 * 8 + 2] == 0xAE and font[75 * 8 + 4] == 0xBA  # LED bytes blink_leds toggles
assert "TermFont+586" in body("blink_leds") and "TermFont+604" in body("blink_leds")
assert "blink_leds" in body("game_frame")

# level_ending shows the per-level message from Txt/TxtOff (a3e.5 prefetch placeholder is gone)
ending = body("level_ending")
assert "TxtOff,x" in ending and "prefetch_next_level" not in main
