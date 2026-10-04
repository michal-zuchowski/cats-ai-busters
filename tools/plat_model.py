"""Frame-faithful Python model of the platform-level runtime for Level 02.

Mirrors main.k65 plat_input / plat_phys / plat_fan / plat_lift / plat_crumble /
plat_switch / plat_exit (same frame order as plat_frame) so room routes can be
replayed and searched without a compiler. It is a source-level model: it does
not run the XEX.
"""

import make_level_art as L

JUMP, LEFT, RIGHT, FIRE = 1, 4, 8, 16   # key bits for step()
SUPPORT = (L.FLOOR, L.LIFT, L.CRUMB, L.CTRL)
CR_WARN, CR_GONE = 40, 120               # crumble: erase at 40, restore at 120
CR_SLOTS = 8
GUST_BIT = 64                            # fan_t & 64: gust half of a 128-frame cycle


def unpack(chunk):
    """chunk bytes -> (screen[24][40], meta dict)"""
    import make_plat_levels as P
    screen = [list(chunk[r * 40:(r + 1) * 40]) for r in range(24)]
    meta = dict(zip(P.META, chunk[960:960 + len(P.META)]))
    return screen, meta


class Sim:
    def __init__(self, chunk, entry_x=0):
        self.art, self.m = unpack(chunk)
        self.scr = [row[:] for row in self.art]
        m = self.m
        self.x, self.y = entry_x, m["sy"]
        self.air = self.vy = self.gt = self.jprep = self.lnd = 0
        self.t = 0
        self.fire_prev = 1
        self.lift_row, self.lift_dir, self.lift_t = m["lrow"], -1, 0
        self.sw_on = 0
        self.fan_t = 0
        self.cr = [[0, 0, 0] for _ in range(CR_SLOTS)]  # t, row, col
        self.done = False
        self.exited = False
        self.deaths = 0
        if m["lw"]:
            self.draw_lift(self.lift_tile())

    # -- helpers ------------------------------------------------------
    @property
    def col(self):
        return ((self.x + 2) >> 2) + 2

    def lift_tile(self):
        return L.LIFT | (0x80 if self.m["lpow"] and not self.sw_on else 0)

    def draw_lift(self, tile):
        for c in range(self.m["lcol"], self.m["lcol"] + self.m["lw"]):
            self.scr[self.lift_row][c] = tile if tile else self.art[self.lift_row][c]

    def tile_under(self):
        return self.scr[(self.y >> 3) - 1][self.col]

    def under(self):
        return self.y & 7 == 0 and (self.tile_under() & 0x7F) in SUPPORT

    def respawn(self):
        self.x, self.y = self.m["sx"], self.m["sy"]
        self.air = self.vy = self.jprep = self.lnd = 0
        self.deaths += 1
        self.crumble_reset()

    def crumble_reset(self):
        for s in self.cr:
            if s[0]:
                self.scr[s[1]][s[2]] = self.art[s[1]][s[2]]
            s[0] = 0

    # -- the frame ------------------------------------------------------
    def step(self, keys=0):
        self.input(keys)
        self.phys()
        self.fan()
        self.lift()
        self.crumble()
        if self.lnd:
            self.lnd -= 1
        self.t += 1

    def input(self, keys):
        if self.jprep:
            self.jprep -= 1
            if self.jprep:
                return
            self.vy, self.air, self.gt = -4, 1, 0
        elif keys & JUMP and not self.air and not self.lnd:
            self.jprep = 3
            return
        # plat_fire (edge triggered, grounded)
        if not keys & FIRE:
            self.fire_prev = 1
        elif self.fire_prev:
            self.fire_prev = 0
            if not self.air:
                self.switch()
        if self.lnd:
            return
        if not self.air and self.t & 1:
            return
        if keys & LEFT:
            if self.x:
                self.x -= 1
            return
        if keys & RIGHT:
            if self.x == 140:
                self.try_exit()
                return
            self.x += 1

    def switch(self):
        m = self.m
        if (m["swy"] and self.y == m["swy"] and not self.sw_on
                and abs(self.col - m["swc"]) <= 2):
            self.sw_on = 1
            self.scr[m["swy"] // 8 - 1][m["swc"]] = L.CTRL
            r = m["exit_y"] // 8 - 1
            for rr in range(r - 3, r):
                self.scr[rr][39] = L.HATCH
            if m["lpow"]:
                self.draw_lift(self.lift_tile())

    def exit_allowed(self):
        m = self.m
        if not m["exit_y"]:
            return True
        if m["swy"] and not self.sw_on:
            return False
        return not self.air and self.y == m["exit_y"]

    def try_exit(self):
        if self.exit_allowed():
            self.exited = True

    def phys(self):
        if not self.air:
            if self.under():
                return
            self.air, self.vy, self.jprep, self.lnd = 1, 0, 0, 0
        self.gt += 1
        if self.gt == 3:
            self.gt = 0
            if self.vy != 5:
                self.vy += 1
            if self.vy == 0:
                self.gt = 2
        if self.vy < 0:
            self.y += self.vy
        elif self.vy:
            for _ in range(self.vy):
                self.y += 1
                if self.under():
                    self.air = self.vy = 0
                    self.lnd = 10
                    break
        if self.y >= 196:
            self.respawn()

    def fan_draw(self):
        """Blades and airflow streaks, every 4 frames (mirrors fan_draw in main.k65)."""
        m = self.m
        gust = bool(self.fan_t & GUST_BIT)
        pose = ((self.fan_t >> 2) if gust else (self.fan_t >> 4)) & 1
        blades = L.FAN_A + L.FAN_B
        for dr in (0, 1):
            for dc in (0, 1):
                self.scr[m["fr0"] + 2 + dr][m["fan"] - 3 + dc] = blades[pose * 4 + dr * 2 + dc]
        for r in range(m["fr0"], m["fr1"]):
            for c in range(m["fan"], m["fc1"]):
                self.scr[r][c] = (L.AIR_A + (((self.fan_t >> 2) + r) & 1)) if gust else self.art[r][c]

    def fan(self):
        self.fan_t = (self.fan_t + 1) & 255
        m = self.m
        if m["fan"] and not self.fan_t & 3:
            self.fan_draw()
        if not m["fan"] or not self.air or not self.fan_t & GUST_BIT:
            return
        if not (m["fan"] <= self.col < m["fc1"]):
            return
        cb = (self.y >> 3) - 2
        if not (m["fr0"] <= cb < m["fr1"]) or self.x >= 140:
            return
        self.x += 1

    def lift(self):
        m = self.m
        if not m["lw"] or (m["lpow"] and not self.sw_on):
            return
        self.lift_t += 1
        if self.lift_t < m["lspd"]:
            return
        self.lift_t = 0
        ride = (not self.air and self.y == (self.lift_row + 1) * 8
                and self.col - m["lcol"] >= 0 and self.col - m["lcol"] < m["lw"])
        self.draw_lift(0)
        kk = self.lift_dir
        self.lift_row += kk
        if self.lift_row == m["lmin"]:
            self.lift_dir = 1
        if self.lift_row == m["lmax"]:
            self.lift_dir = -1
        self.draw_lift(self.lift_tile())
        if ride:
            self.y += -8 if kk == -1 else 8

    def crumble(self):
        for s in self.cr:
            if s[0]:
                s[0] += 1
                if s[0] == CR_WARN:
                    self.scr[s[1]][s[2]] = L.VOID
                elif s[0] == CR_GONE:
                    self.scr[s[1]][s[2]] = self.art[s[1]][s[2]]
                    s[0] = 0
        if self.air or self.y & 7 or self.tile_under() != L.CRUMB:
            return
        for s in self.cr:
            if not s[0]:
                s[:] = [1, (self.y >> 3) - 1, self.col]
                self.scr[s[1]][s[2]] = L.CRUMB | 0x80
                return

    # -- state key for searches ----------------------------------------
    def key(self):
        return (self.x, self.y, self.air, self.vy, self.gt, self.jprep, self.lnd,
                self.t & 1, self.lift_row, self.lift_dir, self.lift_t, self.sw_on,
                self.fan_t, self.fire_prev, tuple(map(tuple, self.cr)))


def run(sim, script):
    """script: list of (keys, frames). Returns sim."""
    for keys, n in script:
        for _ in range(n):
            sim.step(keys)
            if sim.exited:
                return sim
    return sim


def hold_until(sim, keys, cond, limit=600):
    for _ in range(limit):
        if cond(sim):
            return True
        sim.step(keys)
        if sim.exited:
            return cond(sim)
    return False


# ---------------------------------------------------------------------------
# Static route planner: stand/jump/fall graph over the art (no lift, crumble
# timing or switch), used to prove which ledges connect and to derive scripts
# that the dynamic model then replays.
# ---------------------------------------------------------------------------

DIRS = (0, LEFT, RIGHT)


class Static:
    def __init__(self, chunk, entry_x=0):
        self.art, self.m = unpack(chunk)
        self.sup = [[(t & 0x7F) in SUPPORT for t in row] for row in self.art]

    def stand(self, x, y):
        return y & 7 == 0 and (y >> 3) - 1 < 24 and self.sup[(y >> 3) - 1][((x + 2) >> 2) + 2]

    def flight(self, x, y, sched, takeoff, t0=None):
        """sched(f) -> key for frame f. takeoff: start with the jump impulse, else a fall
        that has just stepped off an edge. Returns ('land', x, y, frames) / ('dead', ...)."""
        m = self.m
        vy, gt = (-4, 0) if takeoff else (0, 0)
        f = 0
        if not takeoff:
            gt = 0
        while True:
            key = sched(f)
            if takeoff or f:
                if key & LEFT:
                    x = max(0, x - 1)
                elif key & RIGHT and x != 140:
                    x += 1
            gt += 1
            if gt == 3:
                gt = 0
                if vy != 5:
                    vy += 1
                if vy == 0:
                    gt = 2
            if vy < 0:
                y += vy
            else:
                for _ in range(vy):
                    y += 1
                    if self.stand(x, y):
                        return ("land", x, y, f + 1)
            if y >= 196:
                return ("dead", x, y, f + 1)
            if t0 is not None and m["fan"]:
                ft = (t0 + f + 1) & 255
                col = ((x + 2) >> 2) + 2
                cb = (y >> 3) - 2
                if (ft & GUST_BIT and m["fan"] <= col < m["fc1"] and m["fr0"] <= cb < m["fr1"]
                        and x < 140):
                    x += 1
            f += 1


def schedules(ks):
    for d1 in DIRS:
        for k in ks:
            for d2 in DIRS:
                yield (d1, k, d2)


def plan(chunk, start, ks=range(0, 28, 2), xstep=1, t0s=(None,), forbid=()):
    """Dijkstra (cost = frames) over standing positions of the static map.
    parents[node] = (prev, macro, cost). macro: ('walk', x) | ('jump', d1, k, d2, t0)
    | ('fall', dx, d1, k, d2, t0). Positions whose tile is in `forbid` are not used."""
    import heapq
    st = Static(chunk)
    parents = {start: None}
    dist = {start: 0}
    heap = [(0, start)]
    done = set()

    def relax(node, prev, macro, cost):
        if node in forbid:
            return
        if node not in dist or cost < dist[node]:
            dist[node] = cost
            parents[node] = (prev, macro, cost)
            heapq.heappush(heap, (cost, node))

    while heap:
        c, (x, y) = heapq.heappop(heap)
        if (x, y) in done:
            continue
        done.add((x, y))
        for dx in (-1, 1):
            nx = x + dx
            if 0 <= nx <= 140 and st.stand(nx, y):
                relax((nx, y), (x, y), ("walk", nx), c + 2)
        if x % xstep == 0 and st.stand(x, y):
            for t0 in t0s:
                for d1, k, d2 in schedules(ks):
                    sched = (lambda f, d1=d1, k=k, d2=d2: d1 if f < k else d2)
                    res = st.flight(x, y, sched, True, t0)
                    if res[0] == "land":
                        relax((res[1], res[2]), (x, y), ("jump", d1, k, d2, t0), c + 4 + res[3] + 10)
        for dx, key in ((-1, LEFT), (1, RIGHT)):
            nx = x + dx
            if 0 <= nx <= 140 and st.stand(x, y) and not st.stand(nx, y) and (nx + 2) >> 2 != (x + 2) >> 2:
                for t0 in t0s:
                    for d1, k, d2 in schedules(range(0, 20, 2)):
                        sched = (lambda f, d1=d1, k=k, d2=d2: d1 if f < k else d2)
                        res = st.flight(nx, y, sched, False, t0)
                        if res[0] == "land":
                            relax((res[1], res[2]), (x, y), ("fall", dx, d1, k, d2, t0), c + 2 + res[3] + 10)
    return parents


def cost_of(parents, node):
    return parents[node][2] if parents.get(node) else 0


def path_to(parents, node):
    out = []
    while parents[node]:
        prev, macro, _ = parents[node]
        out.append((prev, macro, node))
        node = prev
    return out[::-1]


def settle(sim, n=40):
    while sim.air or sim.lnd or sim.jprep:
        sim.step(0)
        n -= 1
        if n < 0:
            raise AssertionError("no settle")


def drive(sim, path, limit=2000):
    """Replay a planned path in the dynamic model. Returns False when the cat leaves the plan."""
    for prev, macro, node in path:
        settle(sim)
        if (sim.x, sim.y) != prev:
            return False
        kind = macro[0]
        if kind == "walk":
            key = RIGHT if macro[1] > sim.x else LEFT
            n = 0
            while sim.x != macro[1]:
                sim.step(key)
                n += 1
                if n > limit:
                    return False
        elif kind == "jump":
            _, d1, k, d2, t0 = macro
            if t0 is not None:
                while (sim.fan_t + 3) & 255 != t0:
                    sim.step(0)
            sim.step(JUMP)
            sim.step(JUMP)
            sim.step(JUMP)
            f = 0
            sim.step(JUMP | (d1 if f < k else d2))
            f += 1
            while sim.air:
                sim.step(d1 if f < k else d2)
                f += 1
        else:  # fall: step off the edge, then steer
            _, dx, d1, k, d2, t0 = macro
            key = RIGHT if dx > 0 else LEFT
            if t0 is not None:
                while (sim.fan_t + 1) & 255 != t0:
                    sim.step(0)
            while not sim.air:
                sim.step(key)
            f = 0
            while sim.air:
                sim.step(d1 if f < k else d2)
                f += 1
        settle(sim)
        if (sim.x, sim.y) != node:
            return False
    return True


def do_jump(sim, d1=0, k=0, d2=0):
    """press JUMP, then steer d1 for k air frames and d2 afterwards until landing."""
    for _ in range(3):
        sim.step(JUMP)
    f = 0
    sim.step(JUMP | (d1 if f < k else d2))
    f += 1
    while sim.air:
        sim.step(d1 if f < k else d2)
        f += 1
    settle(sim)


def search_jump(sim, goal, waits=range(0, 40), ks=range(0, 30, 2), key=0, dirs=DIRS):
    """Find (wait, d1, k, d2) whose jump from a copy of sim reaches goal(sim) without a death."""
    import copy
    for w in waits:
        base = copy.deepcopy(sim)
        for _ in range(w):
            base.step(key)
        for d1, k, d2 in schedules(ks):
            t = copy.deepcopy(base)
            deaths = t.deaths
            try:
                do_jump(t, d1, k, d2)
            except AssertionError:
                continue
            if t.deaths == deaths and goal(t):
                return w, d1, k, d2
    return None


def apply_jump(sim, w, d1, k, d2, key=0):
    for _ in range(w):
        sim.step(key)
    do_jump(sim, d1, k, d2)
