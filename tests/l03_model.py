"""Frame-by-frame Python port of the level-03 (cur_level == 2) platformer logic of main.k65.

The model mirrors plat_frame: plat_input, plat_phys, plat_actor, plat_pursuer, paw_both,
plat_hit, plat_beam, draw_player's counters.  Data tables (ActT, PawX/PawY/PawK/PawE,
PixelMask), the room art + metadata (assets/plat-rooms.bin) and the font (assets/term-font.pic)
are read from the real files; the control flow is ported function by function and each port is
named after its K65 function so a reviewer can diff it.  Nothing is compiled or executed from K65.
"""

from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
SRC = (ROOT / "main.k65").read_text()
ROOMS = (ROOT / "assets/plat-rooms.bin").read_bytes()
FONT = (ROOT / "assets/term-font.pic").read_bytes()
META = ("sx", "sy", "fan", "lrow", "lcol", "lw", "lmin", "lmax", "arow", "amin", "amax",
        "aspd", "acol", "h0", "h1", "glass", "dlg", "desk", "exit_y")
L03_META = {"xmax": 3, "dock": 4, "z0": 6, "z1": 7, "bcol": 26, "bmin": 27, "bmax": 28,
            "bspd": 29, "mode": 30}
BASE, ROOM_COUNT = 4, 4  # plat-rooms.bin chunk of L03 room 0 (LvBase[2] - 5), LvRooms[2]


def data(name):
    m = re.search(r"data " + name + r" \{([^}]*)\}", SRC, re.S)
    return [int(n) for n in re.findall(r"\d+", re.sub(r"//[^\n]*", "", m.group(1)))]


ActT, PawX, PawY = data("ActT"), data("PawX"), data("PawY")
PawK, PawE, PixelMask = data("PawK"), data("PawE"), data("PixelMask")
assert data("LvBase")[2] - data("LvBase")[1] == BASE + 0 and data("LvBase")[1] == 5 and data("LvRooms")[2] == ROOM_COUNT
FLOOR, LIFT_G, BASE_GLYPHS, BEAM_G, WALL_G = 70, 94, (95, 124, 125), 79, 81


def s8(v):
    return v - 256 if v & 0x80 else v


class Game:
    def __init__(self, room=0):
        self.facing = 0
        self.fire_prev = 1
        self.level_finished = 0
        self.rtclok = 0
        self.events = []
        self.enter_room(room, 0)

    # -------------------------------------------------------------- rooms / meta
    def enter_room(self, room, entry_col):
        self.cur_room = room
        chunk = ROOMS[(BASE + room) * 1024:(BASE + room + 1) * 1024]
        self.art = [list(chunk[r * 40:(r + 1) * 40]) for r in range(24)]
        self.scr = [row[:] for row in self.art]
        raw = chunk[960:]
        self.m = {k: raw[i] for i, k in enumerate(META)}
        self.m.update({k: raw[v] for k, v in L03_META.items()})
        assert self.m["fan"] == 0 and self.m["lw"] == 0 and self.m["h0"] == 0
        self.player_x, self.player_y = entry_col, self.m["sy"]
        self.air = self.vy = self.jprep = self.lnd = self.paw_t = self.paw_kind = self.paw_hit = 0
        self.gt = 0
        self.set_col()
        # plat_dyn_init
        self.e_mode, self.e_bspd = self.m["mode"], self.m["bspd"]
        self.act_init()

    def set_col(self):
        self.player_col = (((self.player_x + 2) & 0xFF) >> 2) + 2

    # -------------------------------------------------------------- drawing helpers
    def put_run(self, row, col, cnt, tile):
        for c in range(col, col + cnt):
            self.scr[row][c] = self.art[row][c] if tile == 0 else tile

    def draw_act(self, slot):
        wm = 0
        if self.act_wob[slot] != 0:
            wm = 0x80
        if self.dk != 0 and slot == 0 and not (self.rtclok & 8):
            wm = 0x80
        base = 6 if slot == 21 and self.bw != 0 else 0
        col = self.act_col[slot]
        for i in range(3):
            self.scr[self.m["arow"]][col + i] = ActT[base + i] | wm
        for i in range(3):
            self.scr[self.m["arow"] + 1][col + i] = ActT[base + 3 + i] | wm

    def erase_act(self, slot):
        col = self.act_col[slot]
        self.put_run(self.m["arow"], col, 3, 0)
        self.put_run(self.m["arow"] + 1, col, 3, 0)

    def draw_gate(self, tile):
        row = (self.m["exit_y"] >> 3) - 4
        for i in range(3):
            self.put_run(row + i, 39, 1, tile)

    def draw_reader(self):
        if self.e_mode & 2:
            self.put_run(20, self.m["z0"], 3, 84 if self.gate else 83)

    # -------------------------------------------------------------- init / reset
    def act_init(self):
        self.act_col = {0: self.m["acol"], 21: self.m["bcol"]}
        self.act_dir = {0: 0, 21: 0}
        self.act_t = 0
        self.act_wob = {0: 0, 21: 0}
        self.ride = self.carrier = self.slot = self.dk = self.gate = self.bt = self.bw = self.bk = 0
        self.kk = 0
        if self.m["arow"] == 0:
            return
        self.draw_reader()
        self.draw_act(0)
        if self.e_bspd:
            self.draw_act(21)
        if self.e_mode & 3:
            self.draw_gate(WALL_G)

    def plat_reset(self):
        if self.m["arow"] == 0:
            return
        self.erase_act(0)
        if self.e_bspd:
            self.erase_act(21)
        if self.e_mode & 3:
            self.draw_gate(0)
        self.act_init()

    def respawn(self):
        self.events.append("respawn")
        self.player_x, self.player_y = self.m["sx"], self.m["sy"]
        self.air = self.vy = self.jprep = self.lnd = self.paw_t = self.paw_kind = self.paw_hit = 0
        self.set_col()
        self.plat_reset()

    # -------------------------------------------------------------- support
    def vacuum_roof(self):
        return (self.m["arow"] << 3) + 3

    def roof_on(self, slot):
        return ((self.player_col - self.act_col[slot]) & 0xFF) < 3

    def on_vacuum(self):
        if self.m["arow"] == 0 or self.vacuum_roof() != self.player_y:
            return 0
        if self.roof_on(0):
            self.carrier = 0
            return 1
        if self.e_bspd == 0 or not self.roof_on(21):
            return 0
        if self.e_mode & 2 and not self.gate:
            return 0
        self.carrier = 21
        return 21

    def under(self):
        if self.on_vacuum():
            return 1
        if self.player_y & 7:
            return 0
        row, col = (self.player_y >> 3) - 1, self.player_col
        if row >= 24:
            return 0  # the real code reads meta bytes here; none is a support glyph
        g = self.scr[row][col] & 0x7F
        if g in (FLOOR, LIFT_G):
            return 1
        if g in BASE_GLYPHS:
            return int(self.art[row][col] & 0x7F == FLOOR)
        return 0

    # -------------------------------------------------------------- input / physics
    def plat_input(self, inp):
        if self.jprep:
            self.jprep -= 1
            if self.jprep:
                return
            self.vy, self.air, self.gt, self.paw_t = 0xFC, 1, 0, 0
        if inp.get("up") and not self.air and self.lnd == 0:
            self.jprep = 3
            return
        self.plat_fire(inp)
        if self.level_finished or self.lnd or self.paw_t:
            return
        if not self.air and (self.rtclok & 1):
            return
        if inp.get("left"):
            self.facing = 1
            if self.player_x == 0:
                return
            self.player_x -= 1
            self.set_col()
            return
        if inp.get("right"):
            self.facing = 0
            if self.player_x == 140:
                self.plat_exit()
                return
            self.player_x += 1
            self.set_col()

    def plat_fire(self, inp):
        if not inp.get("fire"):
            self.fire_prev = 1
            return
        if not self.fire_prev:
            return
        self.fire_prev = 0
        if self.air or self.paw_t:
            return
        self.swat()

    def swat(self):
        self.paw_t, self.paw_kind, self.paw_hit = 10, 0, 0
        if self.on_vacuum():
            self.paw_kind = 2

    def plat_exit_allowed(self):
        if self.e_mode & 3 and self.gate == 0:
            return 0
        if self.m["exit_y"] == 0:
            return 1
        return int(not self.air and self.player_y == self.m["exit_y"])

    def plat_exit(self):
        if not self.plat_exit_allowed():
            return
        if self.cur_room == ROOM_COUNT - 1:
            self.level_finished = 1
            self.events.append("level_end")
            return
        self.events.append("room%d" % (self.cur_room + 1))
        self.enter_room(self.cur_room + 1, 0)

    def plat_phys(self):
        if not self.air:
            if self.under():
                return
            self.jprep = self.lnd = self.paw_t = 0
            self.air, self.vy = 1, 0
        self.gt += 1
        if self.gt == 3:
            self.gt = 0
            if self.vy != 5:
                self.vy = (self.vy + 1) & 0xFF
            if self.vy == 0:
                self.gt = 2
        if self.vy & 0x80:
            self.player_y -= (-s8(self.vy))
        elif self.vy:
            cnt = self.vy
            while cnt:
                self.player_y += 1
                if self.under():
                    self.air = self.vy = 0
                    cnt = 1
                    self.lnd = 10
                cnt -= 1
        if self.player_y >= 196:
            self.respawn()

    # -------------------------------------------------------------- actors
    def carry(self, slot):
        if not self.ride or self.carrier != slot:
            return
        if self.kk == 1 and self.player_x < 136:
            self.player_x += 4
        if self.kk == 2 and self.player_x >= 4:
            self.player_x -= 4
        self.set_col()

    def dock_tick(self):
        self.dk -= 1
        if self.dk == 0:
            self.act_wob[0], self.act_dir[0], self.act_t = 24, 1, 0
            self.events.append("undock")
        self.draw_act(0)

    def a_step(self):
        m = self.m
        wm = m["amax"]
        if self.e_mode:
            if self.act_wob[0] or (self.e_mode & 2 and self.gate):
                wm = m["xmax"]
        if self.act_dir[0] == 0:
            if self.act_col[0] >= wm:
                self.act_dir[0] = 1
            else:
                self.act_col[0] += 1
                self.kk = 1
        elif m["amin"] >= self.act_col[0]:
            self.act_dir[0] = 0
        else:
            ok = True
            if self.e_bspd and self.act_col[0] - self.act_col[21] < 4:
                ok = False
            if ok:
                self.act_col[0] -= 1
                self.kk = 2

    def a_events(self):
        m = self.m
        if self.e_mode & 1 and self.act_dir[0] == 0 and self.act_wob[0] and self.act_col[0] >= m["dock"]:
            self.dk, self.act_wob[0] = 240, 0
            self.gate = 1
            self.draw_gate(0)
            self.events.append("dock")
        if not self.e_mode & 2 or self.gate:
            return
        if m["z0"] <= self.act_col[0] <= m["z1"]:
            self.gate = 1
            self.draw_gate(0)
            self.draw_reader()
            self.events.append("gate")

    def plat_actor(self):
        if self.m["arow"] == 0:
            return
        if self.act_wob[0]:
            self.act_wob[0] -= 1
        if self.act_wob[21]:
            self.act_wob[21] -= 1
        self.ride = 0
        if not self.air and self.on_vacuum():
            self.ride = 1
        self.slot = 0
        if self.dk:
            self.dock_tick()
            return
        self.act_t += 1
        if self.act_t < self.m["aspd"]:
            return
        self.act_t = 0
        self.erase_act(0)
        self.kk = 0
        self.a_step()
        self.draw_act(0)
        self.a_events()
        self.carry(0)

    def b_right(self):
        if self.act_col[21] >= self.m["bmax"]:
            return
        if self.act_col[0] - self.act_col[21] < 4:
            return
        self.act_col[21] += 1
        self.kk = 1

    def b_left(self):
        if self.m["bmin"] >= self.act_col[21]:
            return
        self.act_col[21] -= 1
        self.kk = 2

    def b_move(self):
        if self.act_wob[21]:
            (self.b_right if self.act_dir[21] == 0 else self.b_left)()
            return
        if self.ride:
            self.b_right()
            return
        if self.m["bcol"] < self.act_col[21]:
            self.b_left()
        if self.act_col[21] < self.m["bcol"]:
            self.b_right()

    def b_touch(self):
        if self.lnd or self.jprep or not self.ride or self.carrier != 0 or self.act_wob[21]:
            return 0
        x = self.player_x + (2 if self.facing else 0)
        return int(x < self.act_col[21] * 4 + 10)

    def plat_pursuer(self):
        if self.e_bspd == 0:
            return
        if self.e_mode & 2 and self.gate == 0:
            return
        self.slot = 21
        self.bw = 0
        if self.ride:
            self.bw = 1
        if self.act_col[21] != self.m["bcol"]:
            self.bw = 1
        if self.bw:
            if not self.b_touch():
                self.bk = max(self.bk - 1, 0)
            else:
                self.bk += 1
                if self.bk >= 48:
                    self.slot = 0
                    self.respawn()
                    return
        self.bt += 1
        if self.bt < self.m["bspd"]:
            self.slot = 0
            return
        self.bt = 0
        self.erase_act(21)
        self.kk = 0
        self.b_move()
        self.draw_act(21)
        self.carry(21)
        self.slot = 0

    # -------------------------------------------------------------- contact
    def paw_contact(self, slot):
        if self.paw_t < 4 or self.paw_t >= 8 or self.paw_hit or self.m["arow"] == 0:
            return
        if self.air or self.jprep:
            return
        if slot != 0 and self.e_bspd == 0:
            return
        if self.ride and self.carrier == slot:
            return
        kk = PawK[self.paw_kind]
        while True:
            px = PawX[kk]
            if self.facing:
                px = (~px & 0xFF) + 16 & 0xFF
            px = (px + self.player_x - self.act_col[slot] * 4) & 0xFF
            if px < 12:
                py = (PawY[kk] + self.player_y - (self.m["arow"] * 8 + 24)) & 0xFF
                if py < 16:
                    tmp = 3 if py & 8 else 0
                    if slot != 0 and self.bw:
                        tmp += 6
                    glyph = ActT[(px >> 2) + tmp]
                    if FONT[glyph * 8 + (py & 7)] & PixelMask[px & 3]:
                        self.paw_hit = 1
                        d = self.facing if self.paw_kind == 0 else self.facing ^ 1
                        self.act_dir[slot] = d
                        self.act_wob[slot] = 40
                        self.bk = 0
                        self.draw_act(slot)
                        self.events.append("hit%d" % slot)
                        return
            kk += 1
            if PawE[self.paw_kind] == kk:
                return

    def paw_both(self):
        self.paw_contact(0)
        self.paw_contact(21)

    def hit_check(self, slot):
        if self.act_wob[slot]:
            return
        if slot == 0:
            if self.dk:
                return
        elif not self.bw:
            return
        if self.player_y < self.vacuum_roof() + 1:
            return
        if ((self.player_col - self.act_col[slot]) & 0xFF) >= 3:
            return
        self.respawn()

    def plat_hit(self):
        if self.m["arow"] == 0:
            return
        self.hit_check(0)
        if self.e_bspd:
            self.hit_check(21)

    def plat_beam(self):
        if not self.e_mode & 1:
            return
        tile = 0 if self.dk else (79 if self.rtclok & 16 else 0xCF)
        self.put_run(19, self.m["z0"], self.m["z1"] - self.m["z0"] + 1, tile)
        if self.dk:
            return
        if self.player_col < self.m["z0"] or self.m["z1"] < self.player_col:
            return
        if self.player_y < 153 or self.player_y >= 184:
            return
        self.respawn()

    # -------------------------------------------------------------- one frame
    def draw_player(self):
        if self.jprep or self.air:
            return
        if self.paw_t:
            self.paw_t -= 1
            return
        if self.lnd:
            self.lnd -= 1

    def frame(self, inp=None):
        inp = inp or {}
        self.rtclok = (self.rtclok + 1) & 0xFF
        self.plat_input(inp)
        if self.level_finished:
            return
        self.plat_phys()
        self.plat_actor()
        self.plat_pursuer()
        self.paw_both()
        self.plat_hit()
        self.plat_beam()
        self.draw_player()

    def run(self, inp, n):
        for _ in range(n):
            self.frame(inp)

    # -------------------------------------------------------------- search helpers
    def clone(self):
        g = object.__new__(Game)
        g.__dict__ = self.__dict__.copy()
        for k in ("act_col", "act_dir", "act_wob"):
            setattr(g, k, dict(getattr(self, k)))
        g.scr = [row[:] for row in self.scr]
        g.events = list(self.events)
        return g

    def key(self):
        return (self.cur_room, self.player_x, self.player_y, self.air, self.vy, self.gt, self.jprep,
                self.lnd, self.paw_t, self.paw_kind, self.paw_hit, self.facing, self.fire_prev,
                self.ride, self.carrier, self.dk, self.gate, self.bt, self.bw, self.bk, self.act_t,
                self.rtclok & 15, self.level_finished, tuple(sorted(self.act_col.items())),
                tuple(sorted(self.act_dir.items())), tuple(sorted(self.act_wob.items())))
