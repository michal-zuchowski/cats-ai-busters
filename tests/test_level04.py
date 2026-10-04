"""L04 source/data checks and frame-level route replay; does not execute a K65 build."""
import hashlib
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
import make_ai_scene as A
import make_atr as ATR
import make_level_art as L
import make_plat_levels as P
import plat_model as M

main = (ROOT / "main.k65").read_text()
rooms = (ROOT / "assets/plat-rooms.bin").read_bytes()
assert rooms == b"".join(P.ROOMS)
assert hashlib.sha256(rooms[:5120] + rooms[6144:7168]).hexdigest() == "72914d2e88474c393a75bc7febdd697330bc0f7734ee5add073f84745344a568"


def body(name):
    i = main.index("{", main.index(f"func {name} {{"))
    depth = 0
    for j in range(i, len(main)):
        depth += main[j] == "{"
        depth -= main[j] == "}"
        if depth == 0:
            return re.sub(r"\s+", " ", re.sub(r"//[^\n]*", "", main[i + 1:j])).strip()


class Finale(M.Sim):
    """Plain jump/lift physics plus L04 floor contact and final-action gates."""
    def __init__(self, chunk):
        super().__init__(chunk)
        self.dlg_done = self.scenes = 0
        self.facing = 0

    def glass_ready(self):
        delta = self.m["glass"] * 4 - self.x
        return (self.dlg_done and self.m["glass"] and self.y == self.m["desk"]
                and not self.air and not self.jprep and not self.lnd
                and not getattr(self, "paw_t", 0) and self.facing == 0
                and 12 <= delta < 16)

    def under(self):
        if not self.y & 7:
            r = self.y // 8 - 1
            if self.scr[r][self.col] & 127 == L.BEAM and self.art[r][self.col] & 127 == L.FLOOR:
                return True
        return super().under()

    def switch(self):
        # Model the shared pixel-edge gate, not the old six-column shortcut.
        if self.glass_ready():
            self.done = True
            self.stages = (54, 55, 56, 57)

    def step(self, keys=0):
        self.input(keys)
        if self.done:
            return
        self.phys()
        self.lift()
        on = not self.t & 32
        for c in range(self.m["h0"], self.m["h1"] + 1):
            self.scr[22][c] = L.BEAM | 128 if on else self.art[22][c]
        if on and self.y == 184 and self.m["h0"] <= self.col <= self.m["h1"]:
            self.respawn()
        if self.m["dlg"] and not self.dlg_done and not self.air and self.y < 137 and self.col >= 20:
            self.dlg_done = 1
            self.scenes += 1
            self.fire_prev = 0
        if self.lnd:
            self.lnd -= 1
        self.t += 1


def walk(s, col):
    direction = M.RIGHT if s.col < col else M.LEFT
    assert M.hold_until(s, direction, lambda g: g.col == col), (s.x, s.y, col)
    M.settle(s)


# Room 1: three rising steps, transfer to boarding dock, mandatory lift, roof hatch.
r1 = Finale(P.ROOMS[8])
for col, height in ((2, 168), (8, 152), (15, 136), (22, 152)):
    walk(r1, col)
    M.do_jump(r1, M.RIGHT, 100, M.RIGHT)
    assert r1.y == height and r1.deaths == 0, (r1.x, r1.y, height, r1.deaths)
walk(r1, 31)
assert M.hold_until(r1, 0, lambda g: g.lift_row == 18 and g.lift_t == 0)
walk(r1, 33)
assert r1.y == (r1.lift_row + 1) * 8 and r1.deaths == 0
assert M.hold_until(r1, 0, lambda g: g.lift_row == 10 and g.lift_t == 0)
M.do_jump(r1, M.RIGHT, 100, M.RIGHT)
assert r1.y == 80 and r1.deaths == 0
assert M.hold_until(r1, M.RIGHT, lambda g: g.exited)
assert r1.scenes == 0

art1, m1 = M.unpack(P.ROOMS[8])
supports = {(r, c) for r in range(24) for c in range(40) if art1[r][c] & 127 in M.SUPPORT}
expected = {(r, c) for r, c0, c1 in ((22, 0, 39), (20, 4, 8), (18, 11, 15),
                                  (16, 18, 22), (18, 27, 31), (9, 35, 39))
            for c in range(c0, c1 + 1)}
assert supports == expected
assert 136 - m1["exit_y"] > 26  # highest other fixed support exceeds the measured jump apex
for height in (184, 168, 152, 136, 88):
    r1.y, r1.air = height, 0
    assert not r1.exit_allowed()
r1.y, r1.air = 80, 1
assert not r1.exit_allowed()

# Electric state and contact agree at every phase and boundary, with no phantom support in L03.
assert "a?79 == { a=cur_level a?3 == { goto under_base } a=0 return }" in body("under")
haz = body("plat_haz")
for needle in ("a&0x20", "a?184", "a?m_h0", "a?player_col", "respawn"):
    assert needle in haz
for t in range(64):
    for col in (6, 7, 20, 34, 35):
        s = Finale(P.ROOMS[8])
        s.t, s.x = t, (col - 2) * 4 - 2
        s.step()
        assert s.deaths == int(t < 32 and 7 <= col <= 34), (t, col)
        for c in range(7, 35):
            assert s.scr[22][c] == (L.BEAM | 128 if t < 32 else L.FLOOR)
    s = Finale(P.ROOMS[8])
    s.t, s.x, s.y, s.air, s.vy = t, 70, 180, 1, -1
    s.step()
    assert s.deaths == 0  # airborne cat is above the electrified surface
    s = Finale(P.ROOMS[8])
    s.t, s.x, s.y = t, 22, 168
    s.step()
    assert s.deaths == 0  # same columns are safe on an elevated ledge

# Room 2: jump across descending ledges, confront AI, then jump to the glass yourself.
r2 = Finale(P.ROOMS[9])
assert r2.y == 80 == m1["exit_y"]
assert r2.m["desk"] == 120 and r2.m["glass"] == 33
assert not r2.m["lw"] and not r2.m["arow"]  # cinematic cleanup has no live lift/actor to repaint
walk(r2, 6)
M.do_jump(r2, M.RIGHT, 100, M.RIGHT)
assert r2.y == 96 and r2.deaths == 0
walk(r2, 16)
M.do_jump(r2, M.RIGHT, 100, M.RIGHT)
assert r2.y == 112 and r2.deaths == 0 and r2.scenes == 1 and not r2.done
walk(r2, 24)
M.do_jump(r2, M.RIGHT, 100, M.RIGHT)
assert r2.y == 120 and r2.deaths == 0 and r2.scenes == 1 and not r2.done
walk(r2, 32)
r2.step(M.FIRE)
assert r2.done
# Preserve the original dialogue/action edge coverage: the first FIRE starts
# dialogue, held FIRE is ignored, and only a fresh release/new press pushes.
s = Finale(P.ROOMS[9])
s.x, s.y = 118, 120  # direct/debug approach cannot bypass the scene with FIRE
s.step(M.FIRE)
assert not s.done and s.dlg_done
s.step(M.FIRE)
assert not s.done  # held skip/action cannot leak into the final push
s.step(0)
s.step(M.FIRE)
assert s.done
s.respawn()
assert s.dlg_done and s.scenes == 1  # falling does not force another monologue
s.x, s.y = 90, 136
s.step()
assert s.scenes == 1
for x in range(117, 121):
    s = Finale(P.ROOMS[9])
    s.dlg_done, s.x, s.y, s.fire_prev = 1, x, 120, 1
    s.step(M.FIRE)  # each case starts with a fresh released FIRE edge
    assert s.done and s.stages == (54, 55, 56, 57), x
for attr in ("air", "jprep", "lnd", "paw_t"):
    s = Finale(P.ROOMS[9])
    s.dlg_done, s.x, s.y, s.fire_prev = 1, 118, 120, 1
    setattr(s, attr, 1)
    s.step(M.FIRE)
    assert not s.done, attr
s = Finale(P.ROOMS[9])
s.dlg_done, s.x, s.y, s.facing, s.fire_prev = 1, 118, 120, 1, 1
s.step(M.FIRE)
assert not s.done  # wrong-facing FIRE cannot trigger the push
s = Finale(P.ROOMS[9])
s.x, s.y, s.t = 74, 184, 32
s.step()
assert not s.dlg_done  # floor is not a safe cinematic perch
assert "glass_ready" in body("plat_push")
ready = body("glass_ready")
for needle in ("a=dlg_done", "a=m_glass", "a=player_y", "a=air", "a=jprep",
               "a=lnd", "a=paw_t", "a=facing", "a<< a<<",
               "c+ a-player_x", "a?12", "a?16"):
    assert needle in ready
assert "a=54" in body("final_scene") and "a=55" in body("final_scene")
assert "a=56" in body("final_scene") and "a=57" in body("final_scene")
assert body("final_scene").index("a=56") < body("final_scene").index("gcol++")
assert body("final_scene").index("put_run") < body("final_scene").index("AUDC1=a=0")
scene = body("final_scene")
events = (
    "wm=a=13 gcol=a=m_glass clear_glass_scene kk=a=0 glass_glyph gcol++ draw_glass_scene",
    "wm=a=13 clear_glass_scene kk=a=0 glass_glyph gcol++ draw_glass_scene",
    "wm=a=13 clear_glass_scene wm=a=14 kk=a=1 glass_glyph draw_glass_scene",
    "wm=a=14 clear_glass_scene wm=a=15 gcol=a=36 kk=a=2 glass_glyph draw_glass_scene",
    "wm=a=15 clear_glass_scene wm=a=17 gcol=a=36 kk=a=3 glass_glyph draw_glass_scene",
)
last = -1
for event in events:
    pos = scene.index(event)
    assert pos > last
    last = pos
assert scene.index("tmp=a=12 x=18 y=35 cnt=a=4 put_run") > last
ending = body("final_scene")
pmg_off = ending.rindex("GRACTL=a=0")
pcm_end = ending.rindex("AUDC1=a=0")
text_switch = ending.index("clear_term_screen")
assert pcm_end < pmg_off < text_switch
for register in ("SDMCTL=a=0x22", "SDLSTL=a=&<dl_term",
                 "SDLSTH=a=&>dl_term", "CHBAS=a=&>TermFont"):
    assert ending.index(register, pmg_off) < text_switch
assert "a?3 != { return }" in body("plat_dlg") and "a?137 >= { return }" in body("plat_dlg")
assert "a?20 < { return }" in body("plat_dlg") and "ai_cutscene" in body("plat_dlg")
assert "dlg_done=a=0" in body("init_plat") and "dlg_done" not in body("respawn")

# Streamed cinematic, mouth packing and all three overwritten memory regions.
image, closed, opened = A.portrait()
portrait = (ROOT / "assets/ai-portrait.pic").read_bytes()
assert portrait == A.pack(image) + A.pack(closed) + A.pack(opened) and len(portrait) == 3260
assert len(A.pack(closed)) == len(A.pack(opened)) == 30 and closed.tobytes() != opened.tobytes()
assert A.pack(image.crop(A.MOUTH)) == A.pack(closed)
assert ATR.EXTRA_CHUNKS[25][0] == "ai-portrait" and len(ATR.EXTRA_CHUNKS) == 26
disk, _, table = ATR.build_atr(bytes(128), ATR.EXTRA_CHUNKS)
assert table == (ROOT / "assets/disk-chunks.bin").read_bytes() and len(table) == 130
entry = table[125:130]
assert entry[2:4] == bytes((26, 0))
assert entry[4] == sum(portrait.ljust(3328, b"\0")) & 255
assert 0xA000 + 3328 <= 0xB000 < 0xB400 < 0xBC00
memory = bytearray([0xAA] * 65536)


def load(chunk, dest):
    entry = table[chunk * 5:chunk * 5 + 5]
    start, count = int.from_bytes(entry[:2], "little"), int.from_bytes(entry[2:4], "little")
    memory[dest:dest + count * 128] = disk[16 + (start - 1) * 128:16 + (start - 1 + count) * 128]


load(14, 0xA000)
load(15, 0xA600)
load(21, 0xB400)
saved_cry, saved_music = memory[0xA600:0xB000], memory[0xB400:0xB600]
load(25, 0xA000)
assert memory[0xA000:0xA000 + 3260] == portrait and memory[0xA600:0xB000] != saved_cry
assert memory[0xB400:0xB600] == saved_music
load(14, 0xA000)
memory[0xB000:0xB000 + 960] = memory[0xA000:0xA000 + 960]
load(15, 0xA600)
assert memory[0xA000:0xA400] == P.ROOMS[9]
assert memory[0xB000:0xB000 + 960] == P.ROOMS[9][:960]
assert memory[0xA600:0xB000] == saved_cry and memory[0xB400:0xB600] == saved_music
assert memory[0x1000:0xA000] == bytes([0xAA]) * 0x9000  # resident atlas/code untouched
assert memory[0xBC00:0xC000] == bytes([0xAA]) * 1024  # only init_pmg, not SIO, owns the cat pages
mouth = body("ai_mouth")
for src, dest in ((3200, 2256), (3206, 2296), (3212, 2336), (3218, 2376), (3224, 2416)):
    assert f"ZoomBuf+{src},x" in mouth and f"ZoomBuf+{dest},y" in mouth
assert "y?6" in mouth and "x=ai_pose" in mouth
scene = body("ai_cutscene")
assert "a=25 load_checked_chunk" in scene and "a?180" in scene and "a?5" in scene
assert scene.count("load_checked_chunk") == 3 and scene.count("call music_failure") == 0
checked = body("load_checked_chunk")
assert "a?0" in checked and "!= { music_failure }" in checked
assert checked.index("disk_failed") < checked.index("disk_checksum")
restore = scene.split("ai_restore:", 1)[1]
for required in ("paint_room_art", "a=15 load_checked_chunk", "draw_glass", "draw_hud_static",
                 "clear_row0", "clear_row1", "&<dl_term", "LvC1,x", "LvC2,x", "init_pmg",
                 "fire_prev=a=0", "NMIEN=a=0xC0"):
    assert required in restore, required
assert all(name not in scene for name in ("enter_plat_room", "init_plat", "RevealImg", "stop_music", "level_music"))
tick = body("ai_tick")
assert "a^30" in tick and "a?2" in tick and "a?4" in tick and "sfx_switch" in tick
assert "input_pressed x=a CH=a=0xFF a=x" in tick and "fire_prev=a=1" in tick
assert "goto" not in tick  # skip returns through the call stack before parent cleanup
assert "0x4D &<ZoomBuf &>ZoomBuf" in main and "for x=0..78 eval [0x0D]" in main
assert "level_finished=a=1" in body("final_scene") and "CryBuf" in body("final_scene")

print("test_level04.py: source/data checks passed; both no-death routes, floor phases, AI/finale gates")
