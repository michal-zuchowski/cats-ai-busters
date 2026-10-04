"""ASCII preview of a platform room: python3 tools/show_room.py <room 0-9>"""
import sys
import make_level_art as L
import make_plat_levels as P

CH = {L.VOID: " ", L.FLOOR: "=", L.LIFT: "L", L.CRUMB: "c", L.CTRL: "C", L.WALL: "W", L.HATCH: "H",
      L.CEIL: "#", L.CEIL_EDGE: "#", L.CEIL_JOINT: "#", L.EDGE: "#", L.CONDUIT: "|", L.PANEL: "p"}
CH.update({t: "f" for t in (*L.FAN_A, *L.FAN_B)})


def show(n):
    chunk = P.ROOMS[n]
    for r in range(24):
        line = "".join(CH.get(chunk[r * 40 + c] & 0x7F, "r" if chunk[r * 40 + c] else " ")
                       for c in range(40))
        print(f"{r:2} {line}")
    print("   " + "".join(str(c % 10) for c in range(40)))


if __name__ == "__main__":
    show(int(sys.argv[1]))
