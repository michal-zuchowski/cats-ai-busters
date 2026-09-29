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


root = Path(__file__).resolve().parents[1]
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
    b = body(name)
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
assert len(sprites) == 28 * 128
frames = [sprites[i * 128:(i + 1) * 128] for i in range(28)]
assert len(set(frames[:12])) == 6 and frames[12] != frames[13]
assert all(f[23] | f[32 + 23] | f[22] | f[32 + 22] == 0 for f in frames[:14])  # no dark legs
for f in frames:
    assert all(b == 0 for chunk in range(4) for b in f[chunk * 32 + 24:chunk * 32 + 32])
assert sum(bool(f[64 + 23] | f[96 + 23]) for f in frames[:12]) >= 8
draw = body("draw_player")
assert all(p in draw for p in ("PmP0,y", "PmP1,y", "PmP2,y", "PmP3,y", "HPOSP0", "HPOSP3"))
assert "a=player_x" in draw and "a?12" in draw and "c- a+14" in draw
for base, off in (("PmP0", 0), ("PmP1", 32), ("PmP2", 64), ("PmP3", 96)):
    addr = int(re.search(r"var " + base + r" = 0x([0-9A-F]+)", main).group(1), 16)
    player = (addr + off - 0xBC00) // 256
    assert (player, (addr + off) % 256) == (int(base[-1]), 184), base
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
    "MsgEnding": "PHASE 02 DELAYED. AGENCY UNDETECTED.",
}
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
assert "&<RoomArt+960" in loader and "&<RoomArt+1920" in loader
assert "cur_room--" in body("read_and_move")
assert "cur_room++" in body("read_and_move")

# The gameplay hand-off must happen after cut_to_black, and level_finished
# must gate the main loop so the ending freezes the game once it plays.
scene = re.search(r"main \{(.*)\}\s*$", main, re.S).group(1)
assert scene.index("init_system") < scene.index("goto intro_skip") < scene.index("build_storm_list")
tail = scene[scene.index("cut_to_black"):]
assert tail.index("cut_to_black") < tail.index("init_gameplay") < tail.index("game_frame")
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
assert len(music) == 432 and 'binary "assets/music.bin"' in main
assert "AUDCTL=a=0x50" in body("init_gameplay")  # 16-bit 1.79 MHz lead on ch1+2
for k in set(music[:256]) - {0, 1}:  # every lead note in tune within 2 cents
    f = 1789790 / (2 * (music[336 + k - 2] + 256 * music[352 + k - 2] + 7))
    assert abs(1200 * math.log2(f / 440) % 100 - 50) > 48, f
assert set(music[256:272]) <= {0, 1, 2} and set(music[272:304]) <= {0, 1, 2, 3}
assert "a&127" not in body("music_step") and "Music+400,x" in body("tick_music")  # 256-step song
assert "sfx_timer" in body("tick_music") and "AUDC4" in body("sfx_alert")  # drums yield to SFX
assert "tick_audio" in body("music_vbi") and "tick_audio" not in body("game_frame")  # steady tempo
assert "VVBLKD" in body("init_gameplay")
assert "tick_music" in body("tick_audio") and "AUDC3=a=0" in body("level_ending")
print("test_gameplay_logic.py: all checks passed")

font = (root / "assets/term-font.pic").read_bytes()
assert font[73 * 8 + 2] == 0xAE and font[75 * 8 + 4] == 0xBA  # LED bytes blink_leds toggles
assert "TermFont+586" in body("blink_leds") and "TermFont+604" in body("blink_leds")
assert "blink_leds" in body("game_frame")
