# Level 01 — technical plan and current preview

This is the implementation target for [The Blind Spot](../scenario/level-01.md).
The first room is now a playable visual preview. `main.k65` temporarily jumps
from hardware initialization to `intro_skip`, so the XEX starts directly in
gameplay; remove that one `goto intro_skip` at the start of `main` to restore
the existing intro. All three rooms use a dense Druidarium-style ANTIC 4 tile set (steel-blue
racks with lit server bays, ceiling trays, a rust under-floor grille).
Pixel colour 3 in inverse tiles is red (PF3): rack LEDs and the watched
camera floor. `tools/make_level_art.py` produces the maps and tile glyphs;
`tools/make_intro_art.py` merges the glyphs into `assets/term-font.pic`.

The cat is a single-line PMG sprite, 16 × 24 pixels: P0/P1 carry the dark
rim, stripes and far legs, P2/P3 the ginger fur. `tools/make_cat_sprite.py`
writes `assets/cat-sprites.bin` (20 frames × 128 bytes): 8 walk frames, 2
idle frames with a tail flick, and the mirrored set. The walk is a
four-beat gait (near hind, near fore, far hind, far fore, a quarter cycle
apart). The frame comes from `player_x & 7`, and the cat moves 1 pixel per
gait phase, so planted paws do not slide. `player_col` is derived from the
cat's middle pixel for the tile-based camera, switch and relay rules. PMG
memory is `$B800` (players at `$BC00–$BFFF`); the linker uses `$2000–$9FFF`
and the intro buffers remain at `$A000` (zoom) and `$B000` (screen).

## Controls and rules

- One player, one joystick in port 1: four directions move the cat; diagonals
  do not increase speed. The fire button acts only when the cat overlaps the
  maintenance switch or relay cabinet. The hatch exits by walking into it
  after the relay has been diverted. No keyboard or second joystick needed
  for the first level.
- Move on a tile map; walls and racks block movement. The cat may walk
  through marked shadow floor tiles behind racks, where rack walls block
  camera sight. Read the
  joystick and fire button once per frame, move at a fixed cadence, and
  accept a fire press once per release rather than retriggering every frame.
- Each camera has a short, repeating sequence of facing directions and a
  matching lit floor area. Detection checks the cat's tile against the
  camera's current visible tiles, stopping at opaque rack tiles; the visual
  cone and the collision rule must be generated from the same data. The
  service switch overrides one camera's sequence for a limited number of
  frames; it never disables the other camera. An alert resets only the
  current screen's temporary state and cat position.
- Relay interaction needs the action button and a brief uninterrupted
  progress period while the cat stays adjacent; leaving or being seen
  cancels progress. The switch can be used again after every failed
  crossing. Store permanent progress as a small relay-completed flag and
  temporary state as the current room, positions, camera phase and timers.

## Picture and sound

- Gameplay uses a 40 × 24 ANTIC 4 character screen, as proposed in
  [graphics.md](graphics.md). Draw a small reusable tile set for floors,
  racks, wall, shadow, camera cones, switch, relay and hatch. Three maps fit
  in fixed screens, not a scrolling world. Keep colors readable in dark
  rooms: shadows distinct from walls, lit camera area visibly distinct from
  safe floor. Reserve a row for the CATCOM/status message and interaction
  progress; do not rely on color alone to show alert or completion.
- Use player/missile graphics for the cat if it stays clear at this scale;
  otherwise use a character sprite. Camera collision remains tile-based
  regardless of rendering. Update map cells that change (sweep, switch,
  status), not the entire screen on every frame. No gameplay scanline
  palette kernel or full-screen bitmap animation.
- POKEY provides a short looping, low-key pulse while moving through the
  data center, a cue for a switch press, a distinct camera-alert sting and
  a short success phrase. Sound effects take priority over music on a
  channel; restore the music voice afterwards. The PCM meow belongs to the
  intro and is not a gameplay control. Compose/arrange the actual tune only
  after the basic loop works; no new playback engine is specified yet.

## Runtime and memory

- The target is a standalone Atari XL/XE `.xex`, initially PAL/50 Hz, using
  K65 and the existing project build. Gameplay starts only after
  `video_call` and `cut_to_black` in `main.k65`; replace the final idle loop
  with an explicit handoff once gameplay is implemented.
- The intro currently uses its own display lists, palettes, face DLI, PCM,
  OS shadows and buffers (`$8000` zoom bitmap, `$9000` terminal screen).
  Before displaying the first gameplay frame, silence intro audio, disable
  its DLI, set the gameplay display list, colors and font through the
  relevant OS shadows, and initialize PMG only if used. Keep the OS VBI
  enabled as in the intro so frame timing and shadows remain valid. Do not
  reuse an intro buffer before the last intro routine has finished with it.
- The linker's existing `$2000–$7FFF` bank is nearly full with intro code
  and art. **First implementation gate:** measure the current link map and
  reserve concrete ranges for gameplay code/data, screen, charset and any
  PMG memory; verify on a 64 KB XL/XE with OS/BASIC mapping respected. Do
  not assume gameplay code fits in the remaining bank, or that `$A000+`
  is free. If it does not fit, plan a deliberate post-intro reuse/overlay
  of intro-only memory and a safe load/copy strategy within the single XEX
  before writing gameplay. Keep the three level maps compact (for example,
  tile IDs plus camera paths) rather than three screen-sized bitmaps.
- A small loop per frame is enough: sample input, advance camera phases and
  switch timer, apply movement/interactions, test visibility, update changed
  screen cells, then advance the POKEY music/effects. Keep camera state and
  level state separate from display bytes so retry is deterministic. Start
  with one screen and one camera, then add the aisle and relay only after
  the transition from the intro runs in the emulator.

**Acceptance for level 01:** a fresh boot plays the existing intro, hands
off without a blue/blank screen or stray DLI/audio, accepts joystick movement
and fire, visibly matches each camera's detection area, retries after an
alert, allows the switch to open a crossing, completes the relay and hatch,
and reaches the ending on the same 64 KB configuration.

## Earlier prototype notes (historical; superseded by the preview above)

`main.k65` contains a first pass at three one-row rooms. After resolving
K65's crash on bare forward calls (use `call name` when a routine is
declared later), a linker overflow was removed by sharing room-entry and
redraw code. With intro skipping added, the project **compiles with 28 bytes
free** in
`$2000–$7FFF`; emulator playability and audiovisual quality still need
verification. See `cats-ai-busters-ieb.4` for the remaining validation.

The intro can be skipped at the next frame wait with joystick fire (port 1),
Space or Return. `wait_frames` polls `TRIG0` and the OS `CH` key buffer only
while `cur_room` is `$FF`; gameplay uses room IDs 0–2. All skip paths jump
to the same handoff as a completed intro, where the stack is reset, the DLI
is disabled and audio is stopped before displaying gameplay. The terminal
font also includes the gameplay glyphs (`#$%*+= >@`), which were previously
blank and left an apparently empty screen after the call. The objective
message remains visible after the initial display and after an alert.

- **Memory reuse (the "first implementation gate" above):** gameplay adds no
  new display list, font, or palette. It reuses `dl_term`, `TermFont` and
  `switch_to_term`'s palette verbatim (proven working since the intro's
  terminal scene), and reuses `TermScreen` ($9000, dead the instant
  `cut_to_black` runs) as the only gameplay screen buffer, cleared via the
  existing `clear_term_screen`. All new gameplay code/data lives in the
  `$2000–$7FFF` bank alongside the intro, with little free space after
  linking. Avoid adding game data or code to this bank without first making
  room or redesigning the memory layout.
  New zero-page state is 22 fresh bytes at `$A3–$B8`, past the intro's own
  vars (`$80–$A2`), so it cannot collide with anything the intro still uses.
- **Single fixed play row:** all three rooms use one row (row 12) with
  permanent wall rows above/below; up/down are accepted but always blocked.
  This avoids needing a runtime row-address multiply (no hardware multiply
  on 6502) while still exposing all four joystick directions.
- **Tile rendering is shape-only, not color-coded:** floor/shelter/watched/
  wall/switch/relay/hatch/cat are distinct `TermFont` glyphs (ascii-32
  encoded, see the comment above `fill_walls` in `main.k65`), not ANTIC 4
  color-quadrant tricks. ANTIC 4's exact color-select semantics were not
  confirmed in `reference/` this session, so this is a deliberate,
  documented deviation from "keep colors readable... visibly distinct" to
  avoid shipping unverified color behavior; the HUD/status row already
  satisfies "do not rely on color alone." A follow-up could add color once
  verified on real hardware/emulator.
- **Sound**: a looping original spy-funk theme (`tools/make_music.py` →
  `assets/music.bin`, D minor, 4/4, 16 bars with a bridge and a drum fill
  every 4th bar, 16th = 5 frames), all short plucks, driven from the
  deferred VBI (`music_vbi`) so the tempo never drags on slow frames.
  8-bit 64 kHz pure tones are out of tune above C4, so the lead uses
  channels 1+2 joined as a 16-bit 1.79 MHz voice (AUDCTL `$50`); the bass
  stays 8-bit on channel 3 (C3–A3 are within 3 cents); channel 4 plays the
  drums and yields to the switch/alert/success cues while `sfx_timer`
  runs. The program loads from `$1000` (`-lowAddr`) for room; this assumes
  a DOS-less XEX loader, as in the emulator.
- **The cat is a character glyph, not player/missile graphics** — simpler
  and consistent with the tile-based rendering above; camera detection was
  always tile-based regardless of rendering, per this doc.
- The visual cone and the collision rule are generated from the same
  per-room column thresholds by construction: `paint_col_roomN` (what is
  drawn) and `roomN_lit` (what can be seen) are written side-by-side in
  `main.k65` and checked for matching thresholds by
  `tests/test_gameplay_logic.py`.
- Not yet verified in the emulator: camera visibility, joystick input,
  timing of the intro handoff and POKEY volume/distortion of the cues.

## Follow-up pass: mechanical bugs found by tracing the loop

The source checks caught logic problems in the first pass before compilation.
These corrections are in the successfully linked version, but have not yet
been exercised in the emulator.

- **`switch_timer` was set but never decremented.** `read_fire` sets it to
  150 when the switch is pressed, and room entry/alert handling resets it
  to 0, but no code ever counted it down, so once
  pressed it never expired on its own. Added `tick_switch` (mirrors
  `advance_camera`'s shape) and call it every non-alert frame from
  `game_frame`.
- **The switch's safe/danger polarity was inverted**, independent of the
  missing decrement. `switch_timer`'s own doc comment says it counts "frames
  the switch keeps segment A dark" (safe), but `paint_col_room1` and
  `room1_lit` both treated `switch_timer == 0` as safe and `switch_timer > 0`
  as dangerous — backwards, and it made the switch puzzle pointless (the
  segment was safe by default and only became dangerous right after using
  the switch). Fixed both to match the documented intent.
- **`redraw_zone` erased the cat glyph.** It repaints every tile in the
  current room's monitored zones every single frame (needed so camera/switch
  state animates), but it does not special-case the player's own column, and
  `draw_player` was only called on movement. Any frame where the player
  stood still inside a monitored zone — including the entire time they hold
  the relay terminal in room 2, which is itself inside `redraw_zone`'s
  range — silently erased the cat until the next keypress. Fixed by calling
  `draw_player` immediately after every `redraw_zone` call (both call sites,
  in `game_frame`'s normal and alert-recovery branches).
- **Room 2's detection didn't match what was rendered at the relay terminal
  and hatch.** `paint_col_room2` always draws columns 33 (relay terminal)
  and 38 (hatch) as fixed glyphs, never camera-colored, but `room2_lit` fell
  through to the camera-gated bucket for those same columns, so a lit
  camera could trigger an alert while the player stood on a tile that never
  visually showed any danger. Fixed `room2_lit` to explicitly treat columns
  33 and 38 as always-safe, matching the rendering.
- **The HUD objective vanished after the first alert.** The initial
  `hint_timer` never cleared the text at zero, and alert recovery blanked
  the row permanently. Removed the countdown: the objective stays visible
  during play and is restored after an alert.

`tests/test_gameplay_logic.py` was extended to check the *behavior* implied
by these fixes, not just the presence of code: it asserts the literal `a=`
value selected by each branch of the switch logic (not just that the
branches exist), asserts every `redraw_zone` call site is immediately
followed by `draw_player`, asserts `room2_lit` special-cases columns 33/38
the same way `paint_col_room2` does, and asserts post-alert objective
restoration.

## Disk boot (no DOS)

`tools/make_atr.py` builds `out/cats-ai-busters.atr`, a 90K single-density
image (720 × 128-byte sectors) that boots straight into the game without
any DOS. It parses `out/cats-ai-busters.xex`, pulls out the one segment
that starts at `-lowAddr` (0x1000 — the resident program), and writes:

- **Sector 1**: the standard Atari cold-boot header (`BFLAG=0`, `BRCNT=1`,
  `BLDADR=$0700`, `BINITAD=$0706`) followed by a ~110-byte hand-assembled
  6502 loader. The loader only needs one boot sector, so the OS's own
  cold-start code loads it in one shot. It repeatedly pokes the DCB
  (`$0300-$030B`, `DDEVIC=$31`, `DCOMND='R'`) and calls `SIOV` ($E459,
  always resident in OS ROM, no DOS required) to read the payload sectors
  straight to `$1000`, then `JMP $1000`.
- **Sectors 2..N**: the resident program's bytes verbatim, zero-padded to a
  sector boundary. The loader never parses or honors the XEX's `RUNAD`
  segment (always `$1000`, a `JMP main`) — since the entry address is
  static, the loader just jumps there directly once every sector is in.

`tools/run.sh` builds the XEX, then the ATR, then opens the `.atr` in
Atari800MacX (no more raw `.xex` launches). `tests/test_atr.py` re-parses
the ATR and asserts the header, boot sector fields, and that the sector
payload matches the XEX segment byte-for-byte.
