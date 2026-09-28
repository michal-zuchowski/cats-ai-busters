"""Checks that the compact intro art and its mutable regions match ANTIC."""

from pathlib import Path
import re


root = Path(__file__).resolve().parents[1]
main = (root / "main.k65").read_text()
room = (root / "assets/room.pic").read_bytes()
face = (root / "assets/face.pic").read_bytes()
strip = (root / "assets/cat-strip.pic").read_bytes()
font = (root / "assets/term-font.pic").read_bytes()
zoom_left = (root / "assets/zoom-left.tbl").read_bytes()
zoom_right = (root / "assets/zoom-right.tbl").read_bytes()
call = (root / "assets/call.pic").read_bytes()
call_mouth = (root / "assets/call-mouth.tbl").read_bytes()
call_text = (root / "assets/call-text.pic").read_bytes()


def body(name):
    match = re.search(r"(?:data|func|inline) " + name + r" \{(.*?)\n\}", main, re.S)
    assert match, name
    return match.group(1)


def pokes(name):
    return {int(offset): int(value) for offset, value in
            re.findall(r"FaceImg\+(\d+)=a=(\d+)", body(name))}


assert len(room) == 64 * 96
assert all(pixel == 0 for pixel in room[95 * 64:96 * 64])  # no vanishing-point floor lines
assert len(face) == 40 * 96
assert len(strip) == 12 * 32
assert len(font) == 1024
def pixel(image, x, y, stride=40):
    return (image[y * stride + x // 4] >> (6 - 2 * (x % 4))) & 3


assert len(call) == 40 * 80
assert len(call_mouth) == 12 and call_mouth[:6] != call_mouth[6:]
assert list(call_mouth[:6]) == [call[i] for i in (1896, 1897, 1936, 1937, 1976, 1977)]
assert pixel(call, 66, 33) == pixel(call, 100, 55) == 1  # cat's head and body
assert pixel(call, 58, 40) == pixel(call, 70, 40) == 3 and pixel(call, 60, 40) == 0  # copper eyes, slit pupil
assert pixel(call, 116, 43) == pixel(call, 30, 47) == 3  # Blofeld's hands
assert pixel(call, 50, 75) == pixel(call, 110, 75) == 2  # suit on his lap
assert bytes(value + 32 for value in call_text[:45]).decode() == (
    "CATCOM SECURE LINKCONNECTING TO BLOFELD'S CAT")
dialogue = []
offset = 45
while offset < len(call_text):
    length = call_text[offset]
    assert 1 <= length <= 40
    dialogue.append(bytes(value + 32 for value in
                          call_text[offset + 1:offset + 1 + length]).decode())
    offset += length + 1
assert offset == len(call_text)
assert dialogue == [
    "AGENT: AI IS MOVING IN.",
    "BOSS: DOES IT SUSPECT US?",
    "AGENT: NO. IT THINKS HUMANS RULE.",
    "BOSS: LET IT THINK SO.",
    "AGENT: ORDERS?",
    "BOSS: DEPLOY THE CATS. STAY HIDDEN.",
]
assert len(zoom_left) == len(zoom_right) == 256
assert all(zoom_left[value] == ((value >> 6) & 3) * 0x50
           + ((value >> 4) & 3) * 5
           and zoom_right[value] == ((value >> 2) & 3) * 0x50
           + (value & 3) * 5 for value in range(256))
assert all(font[(ord(char) - 32) * 8 + y] & 0x03 == 0
           for char in "ABCDEFGHIJKLMNOPQRSTUVWXYZ" for y in range(8))
assert any(font[(ord(char) - 32) * 8 + y]
           for char in "GLOBAL POWER STRUCTURE: ANALYZING..." for y in range(8)
           if char != " ")


assert pixel(face, 46, 30) == 2  # terminal frame left of the cat
assert pixel(face, 56, 60) == 2 and pixel(face, 65, 62) == 2  # Y_
assert 'binary "assets/room.pic"' in body("RevealImg")
assert 'binary "assets/cat-strip.pic"' in body("CatStrip")
assert 'binary "assets/face.pic"' in body("FaceImg")
assert 'binary "assets/term-font.pic"' in body("TermFont")
assert 'binary "assets/call.pic"' in body("CallImg") and "align 4096" in body("CallImg")
assert 'binary "assets/call-mouth.tbl"' in body("CallMouth")
assert all(token in body("dl_call") for token in
           ("0x4D &<CallImg &>CallImg", "for x=0..78 eval [0x0D]",
            "0x44 &<TermScreen+840 &>TermScreen+840", "0x41 &<dl_call &>dl_call"))
assert 'binary "assets/call-text.pic"' in body("CallText")
assert "0x44 &<TermScreen &>TermScreen" in body("dl_term")
assert "for x=0..22 eval [0x04]" in body("dl_term")
assert "align 4096" in body("RevealImg") and "align 4096" in body("FaceImg")
assert pixel(face, 112, 27) == 0 and pixel(face, 111, 27) == 2
assert any(pixel(face, x, 67) == 3 for x in range(97, 127))  # light muzzle
assert all(room[y * 64 + 52:y * 64 + 64] == strip[(y-64)*12:(y-63)*12]
           for y in range(64, 70))  # no cat before the floor palette changes
assert all(room[y * 64 + 52:y * 64 + 64] != strip[(y-64)*12:(y-63)*12]
           for y in range(75, 85))  # cat is absent from opening panorama

assert len(re.findall(r"repeat 2 \{ 0x4E ", body("dl_face"))) == 96
assert "align 1024" in body("dl_storm") and "align 1024" in body("dl_face")
assert all(token in body("build_storm_list") for token in ("x=96", "a=0x4E",
           "a+64", "a+6", "dl_storm+3"))
assert all(token in body("pan_step") for token in ("x=96", "a+1", "a+6",
           "c+?"))
assert "x=1 wait_frames" in body("storm_pan")
assert 24 * 12 / 50 == 5.76
assert "a?24" in body("storm_pan") and "x=12 wait_frames" in body("storm_tick")
assert 96 * 4 == 24 * 16  # 96px of horizontal travel at 4px per step
assert all((y * 64 + offset) % 4096 <= 4096 - 40
           for y in range(96) for offset in range(25))
assert all(token in body("show_cat") for token in ("x=32", "y?12",
           "RevealImg+4148", "a+64", "a+12"))
assert all(token in body("build_reveal_list") for token in
           ("dl_storm+256,y", "dl_reveal+256,y", "y?70", "dl_reveal+420=a=0xCE"))
assert all(token in main for token in
           ("naked reveal_dli", "COLPF0=a=0x72", "COLPF1=a=0x08",
            "COLPF2=a=0x2C", "VDSLST=a=&<reveal_dli"))
assert all(token in body("build_zoom_bitmap") for token in
           ("RevealImg+1003", "ZoomBuf+40", "zoom_rows=a=48",
            "zoom_col", "a?20", "y?40", "a+64", "a+80"))
assert all(token in body("build_zoom_list") for token in
           ("ZoomBuf", "dl_reveal+3", "x=96", "a+40", "a+6"))
assert "x=16 wait_frames" in body("show_zoom_bitmap")
assert all(token in body("blink_rack_leds") for token in ("pan_phase",
           "RevealImg+1479", "RevealImg+2034", "RevealImg+3561"))

for name in ("pupils_stage1", "pupils_stage2", "pupils_stage3"):
    changes = pokes(name)
    assert 1 <= len(changes) <= 16
    assert all(1900 <= offset < 2300 and face[offset] != 0 and value == 0
               for offset, value in changes.items())

opened, closed = pokes("mouth_open"), pokes("mouth_close")
assert opened.keys() == closed.keys()
assert len(opened) == 8
assert all(opened[offset] == 0 and closed[offset] == face[offset]
           for offset in opened)

page_branch = re.search(r"meow_ptr\+1\+\+(.*?)x--\s*\n\s*x\?0 == break", main, re.S)
assert page_branch and "mouth_tick" in page_branch.group(1)
assert body("play_meow").count("WSYNC=a") == 3
assert "mouth_open" in body("mouth_tick") and "mouth_close" in body("mouth_tick")
assert "COLOR0=a=0x0E" in body("switch_to_term")
assert "COLOR4=a=0xB4" in body("switch_to_term")
assert all(step in body("zoom_to_terminal") for step in
           ("zoom_left=a=27", "zoom_left=a=13", "zoom_left=a=1",
            "zoom_rows=a=14", "zoom_rows=a=19", "zoom_rows=a=24"))
assert body("zoom_to_terminal").count("zoom_frame") == 3
assert "clear_term_screen" in body("zoom_to_terminal")
assert all(color in body("reveal_palette")
           for color in ("COLOR0=a=0x72", "COLOR1=a=0xB6", "COLOR2=a=0x1A"))
assert all(color in body("cat_palette")
           for color in ("COLOR0=a=0x72", "COLOR1=a=0x08", "COLOR2=a=0x2C"))
assert "reveal_palette" in body("switch_to_reveal")
assert "NMIEN=a=0xC0" in body("switch_to_reveal")
assert "NMIEN=a=0x40" in body("switch_to_face")
assert "cat_palette" in body("switch_to_face")
assert body("storm_tick").count("thunder_left=a=3") == 3
assert all("AUDC" + channel in body("storm_pan") for channel in ("2", "3", "4"))
assert "AUDC1" not in body("storm_tick")
assert all("AUDC" + channel + "=a=0" in body("cut_to_black")
           for channel in ("2", "3", "4"))
assert all("CallImg+%d=a" % offset in body("call_mouth") and
           "CallMouth+%d,x" % index in body("call_mouth").replace("CallMouth,x", "CallMouth+0,x")
           for index, offset in enumerate((1896, 1897, 1936, 1937, 1976, 1977)))
assert all(token in body("video_call") for token in
           ("switch_to_term", "TermScreen+107=a=14", "SDLSTL=a=&<dl_call",
            "COLOR2=a=0x2A", "TermScreen+840,y=a", "call_lines_left=a=6",
            "x=6 call_mouth", "x=0 call_mouth", "AUDC4=a=0",
            "x=152 wait_frames", "x=200 wait_frames"))  # ~4 s per subtitle

scene = re.search(r"main \{(.*)\}\s*$", main, re.S).group(1)
beats = ("build_storm_list", "switch_to_storm", "storm_pan", "show_zoom_bitmap",
         "zoom_to_terminal",
         "reveal_line0", "show_y_prompt", "show_cat", "build_reveal_list",
         "switch_to_reveal", "switch_to_face",
         "pupils_stage1", "pupils_stage2", "pupils_stage3",
         "play_meow", "video_call", "cut_to_black")
assert [scene.index(beat) for beat in beats] == sorted(scene.index(beat) for beat in beats)
assert "{} always" in scene[scene.index("cut_to_black"):]

print("test_intro_scene.py: all checks passed")
