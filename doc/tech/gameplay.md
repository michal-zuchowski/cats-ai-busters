# Level 01 — technical plan and current preview

This is the implementation target for [The Blind Spot](../scenario/level-01.md).
The game plays the intro after hardware initialization, then enters Level 01;
fire, Space or Return can skip the intro through the same gameplay handoff.
Returning from the face close-up disables its DLI before the CATCOM
terminal display list is shown; gameplay installs its own HUD DLI later.
All three rooms use a dense Druidarium-style ANTIC 4 tile set (steel-blue
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
- **Sound**: music is enabled again in `tick_audio`, alongside sound effects
  and PCM. The looping original spy-funk theme (`tools/make_music.py` →
  `assets/music.bin`, D minor, 4/4, 16 bars with a bridge and a drum fill
  every 4th bar, 16th = 5 frames) belongs only to the title screen.
  Four separate level scores are described below; all are driven from the
  immediate VBI (`music_vbi_imm`, hooked into `VVBLKI` and chained to the
  saved OS vector) so the tempo never drags on slow frames *and* keeps
  running while `disk_read` (a3e.2) has SIO set `CRITIC` and skip the
  deferred stage -- see "Disk loads and music" below.
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

### Resident `disk_read` and the chunk table

`tools/make_atr.py` also appends an `EXTRA_CHUNKS` list of files after the
resident payload's sectors, and writes `assets/disk-chunks.bin`: one 5-byte
entry per chunk (`<HHB>` = start sector, sector count, checksum — the sum
of the chunk's padded on-disk bytes mod 256). `main.k65` includes this as
`data ChunkTable { binary "assets/disk-chunks.bin" }`, so, like the other
generated `assets/*.bin` files (`cat-sprites.bin`, `level-cams.bin`, ...),
it's committed to git rather than built on the fly — a missing or stale
`assets/disk-chunks.bin` makes the K65 compiler abort silently on that
`data` line (see the gotcha below). Re-run `python3 tools/make_atr.py`
and commit the result whenever `EXTRA_CHUNKS` changes. `tools/run.sh`
builds the XEX, then re-runs `make_atr.py` (refreshing the chunk table
and building the ATR from the new XEX in one pass) before launching the
emulator.

`func disk_read` (params in the zero-page `disk_*` vars, `0xC7-0xCF`, right
after the audio vars) reads `disk_count` sectors starting at `disk_sector`
into `disk_dest` via `SIOV`, incrementing the DCB's buffer pointer and
sector number after each sector. Since SIO drives POKEY channels 3/4 as
its baud-rate clock, `disk_read` silences `AUDC3`/`AUDC4` and zeroes
`AUDCTL` for the transfer, then restores `AUDCTL=0x50` (ch1 at 1.79 MHz,
ch1+2 joined for the 16-bit lead) afterward — the bass/drum channels go
quiet for the read's duration; the lead itself stays audible throughout,
and `disk_busy` (see "Disk loads and music" below) keeps `tick_music`
from fighting SIO for `AUDF3`/`AUDC3`/`AUDF4`/`AUDC4`.

`func disk_selftest` reads chunk 0 of `ChunkTable` (room 0's art, since
a3e.3) into the intro's unused `ZoomBuf` scratch (`$A000`, dead once
gameplay starts, and the same address `load_room_art` itself streams into
as `RoomArtBuf`), recomputes the same checksum `make_atr.py` stored for
that chunk, and flashes `MsgDiskOk`/`MsgDiskErr` on the term screen for 90
frames via `clear_term_screen`/`hud_label`/`wait_frames`. `init_gameplay`
calls it right after `switch_to_term`, before any HUD/room state is set
up, so it never touches live gameplay memory; `load_room_art` reloads the
same chunk for real once room 0 is entered. Verified in the emulator with
the flash duration temporarily lengthened: `DISK OK` renders correctly
before level 1 starts.

### Disk loads and music (a3e.4)

The OS's vertical blank interrupt has two stages: an *immediate* stage
(always runs, updates `RTCLOK` etc., entered via `VVBLKI`) and a
*deferred* stage (entered via `VVBLKD`, but skipped whenever `CRITIC` is
set — which SIO does for the duration of a transfer). The music tick used
to live in the deferred stage (`music_vbi`, exiting via `XITVBV`), so a
multi-sector `disk_read` would silently pause the whole band for the
transfer's length.

`music_vbi_imm` now hooks `VVBLKI` instead: `init_audio` saves the OS's
existing immediate-stage vector into `old_vviblki`, installs
`music_vbi_imm`, and `music_vbi_imm` ends with `goto (old_vviblki)` — an
indirect jump, not `XITVBV` — chaining into the original handler so
`RTCLOK` and the OS's own stage-2 dispatch still happen exactly as before.
This is safe without extra register saves because the OS's NMI entry
already pushes A/X/Y before reaching `VVBLKI`, the same convention
`music_vbi`'s `XITVBV` exit already relied on.

Since `disk_read` drives POKEY channels 3/4 as SIO's baud-rate clock, the
now-always-running `tick_audio`/`tick_music` must not fight it for those
registers mid-transfer. `disk_read` sets `disk_busy=1` before muting
`AUDC3`/`AUDC4`/`AUDCTL` and clears it back to 0 only after `AUDCTL` is
restored to `$50`. `tick_music` checks `disk_busy` right after updating
the lead (`AUDC2`, ch1+2 — untouched by SIO, so it keeps sounding through
the whole load) and returns before touching `AUDF3`/`AUDC3`/`AUDF4`/
`AUDC4` while busy; `music_step`'s bass-note writes go through a
`bass_freq` scratch var instead of `AUDF3` directly, so a note picked
mid-load is queued and applied on the first `tick_music` after the load
clears, rather than lost or written unsafely during it. Net effect: only
the 16-bit lead is audible during a load (bass/drums stay silent, as
`disk_read` already left them), and the full band resumes on the next
tick once `disk_busy` clears — matching a3e.4's acceptance criteria.
Verified: build succeeds, full test suite passes, and the emulator boots
and plays level 1 normally (camera sweep, HUD, movement) with the new
`VVBLKI` hook installed and no hang/crash over 30+ seconds of run time.

**Gotcha**: an absent/stale `assets/disk-chunks.bin` at K65 compile time
(e.g. running the compiler directly without first running
`tools/make_atr.py`) causes a *silent* compiler abort (`Abort trap: 6`,
exit 134, no diagnostic) on the `data ChunkTable { binary ... }` line —
this looks identical to the classic "forward bare-call" abort, so if a
build aborts silently after adding a new `binary`-included data block,
check that the referenced file actually exists first.

### Streaming room art and level assets from disk (a3e.3)

Level 01's room art (three 960-byte rooms, `assets/level-rooms.pic`) and
per-level camera config (`CamCfg`, `assets/level-cams.bin`, 48 bytes) and
sweep timing (`ConeStep`, `assets/cone-step.bin`, 160 bytes) used to be
resident `data { binary ... }` blocks baked into the XEX. They are now
five disk chunks (`ChunkTable+0`/`+5`/`+10` = room 0/1/2 art, `+15` =
CamCfg, `+20` = ConeStep — 5 bytes/entry, so these are compile-time
constant offsets, not a runtime multiply) built by `tools/make_atr.py`,
which now supports an optional `(offset, length)` per `EXTRA_CHUNKS`
entry to slice a byte range out of a source file (used for the three
960-byte room slices; CamCfg/ConeStep still take a whole file).

They stream into three fixed RAM buffers inside the intro's unused
`ZoomBuf` region (`$A000-$AFFF`, dead once gameplay starts, same region
`disk_selftest` already borrows): `RoomArtBuf=$A000` (960 bytes, the
current room's clean tiles — reloaded by `load_room_art` every room
entry, since only one room's art needs to be resident at a time),
`CamCfg=$A400` and `ConeStep=$A480` (loaded once per level by the new
`func load_level_assets`, called from `init_gameplay` right after
`disk_selftest`). None of the consumers (`load_cams`, `tick_cam`,
`cone_rows_start`, `erase_slot`, `art_base`/`floor_ptr`) needed to change:
they already reference `RoomArt`/`CamCfg`/`ConeStep` purely through
indexed addressing or pointer vars, so repointing the labels at RAM
addresses populated at runtime is transparent to that code. `load_room_art`
itself changed from a resident-to-resident copy into a `disk_read` call
that picks `disk_sector`/`disk_count` from the room-indexed `ChunkTable`
offset and reads into `RoomArtBuf`; the `art_base`/`floor_ptr` 880-byte
pointer arithmetic and the TermScreen copy loop are unchanged, just
re-sourced from `RoomArtBuf`.

Net effect: resident free space grew from 3409 to 6356 bytes (a build
right before this change, with a3e.2/a3e.4 already applied, had 3409
free; after moving RoomArt/CamCfg/ConeStep off the binary it's 6356),
freeing roughly the ~2880 (room art) + 48 (CamCfg) + 160 (ConeStep) bytes
that moved to disk, minus the small amount of new `load_room_art`/
`load_level_assets` code and the 20 extra `ChunkTable` bytes now resident
for the two new chunk entries. Verified: build succeeds (6356 free
bytes), `tests/test_disk_chunks.py` confirms all five chunks' on-disk
bytes and checksums match the corresponding slices of
`assets/level-rooms.pic`/`level-cams.bin`/`cone-step.bin` byte-for-byte,
full test suite passes, and the emulator boots straight into level 1
with room 0's art, HUD and camera cone rendering identically to before
the change; moving the cat right showed the camera cone tracking
correctly against the disk-streamed `ConeStep` data (confirming
`load_level_assets` populated it correctly, not just `load_room_art`'s
room-0 chunk).

### Prefetching during the level ending (a3e.5)

`level_ending` (fires when the cat exits room 2 at `player_x=140` with
`relay_done` set) shows `MsgEnding`, plays `sfx_success`, then pauses for
150 frames before fading colors/PMG/audio to black. To prove the
continuous-loading trick works, that pause is now split in half (`x=75`
`wait_frames` twice) around a `call prefetch_next_level`: level 2 doesn't
exist yet, so a sixth `ChunkTable` entry (`+25`, `assets/next-level-
placeholder.bin`, a 512-byte deterministic filler) stands in for its data.
`prefetch_next_level` `disk_read`s it into `RoomArtBuf` (dead scratch by
the time the level ends) and checksums it exactly like `disk_selftest`
checksums chunk 0, storing 1/0 in `next_chunk_ok` — not shown on screen,
since a placeholder load shouldn't change the ending's look.

No new stutter risk: `disk_read` already sets `disk_busy` and mutes only
ch3/ch4 (a3e.4), and `music_vbi_imm` already keeps ticking via `VVBLKI`
regardless of `CRITIC`/SIO activity, so the lead keeps sounding through
the prefetch exactly as it does through any other `disk_read` call —
`level_ending`'s own `AUDC1..4=a=0` fade-out afterward is unaffected by
whether the load happened first.

Verified: build succeeds (6250 free bytes, down slightly from 6356 for
`prefetch_next_level`'s code and the 6th `ChunkTable` entry's 5 resident
bytes); `tests/test_disk_chunks.py`'s generic per-entry loop picked up
the new chunk automatically and confirms its on-disk bytes/checksum
match `assets/next-level-placeholder.bin`; `tests/test_gameplay_logic.py`
gained assertions that `level_ending` calls `prefetch_next_level` between
`sfx_success` and the color fade-out, and that `prefetch_next_level`
reads `ChunkTable+25` and checks against `ChunkTable+29`; full test suite
passes. The emulator boots and runs stably with the new build (process
alive, clean audio-init log, no crash/signal), but the interactive
session was screen-locked for this task (`CGWindowListCopyWindowInfo`
reports `onscreen=false` for all windows) — the same environment
limitation noted for a3e.4 — so reaching the actual level-1 ending
in-emulator (finish room 2's relay, then exit right) to screenshot the
pause could not be completed live; the mechanism was instead verified by
code parity with `disk_selftest`/`load_room_art`/`load_level_assets`
(already screenshot-verified in a3e.2/a3e.3) and by the checksum test.

## Poziomy platformowe 02–04

- Silnik: `init_plat`, `plat_frame`, `frame_any` (main.k65). Pokoje: 10 chunków 1 KB (`assets/plat-rooms.bin`, 960 B grafiki + 64 B meta), generowane przez `tools/make_plat_levels.py`; chunk 15 = krzyk PCM (`CryBuf` $A600, ładowany przy wejściu do L04 R1).
- Mapa pamięci: `RoomArtBuf` $A000, `CamCfg` $A400, `ConeStep` $A480, `CryBuf` $A600, `TermScreen` $B000.
- Platformy jednokierunkowe (FLOOR), skok vy=-4, grawitacja co 3 klatki; na szczycie vy=0 przez 1 klatkę zamiast 3 (bez zmiany wysokości skoku). Upadek poniżej y=196 = `respawn`.
- L02 (4 pokoje, bez odkurzaczy), wejście→wyjście: 184→112→96→80→56 (`exit_y`); pokój n+1 startuje na wysokości wyjścia pokoju n, więc wysokość nie jest gubiona. Wyjście wymaga lądowania na oznaczonej platformie po prawej (`exit_y`), a w pokojach ze sterownikiem (`swy`≠0) także `sw_on`; brama (kolumna 39) jest ścianą, dopóki sterownik nie zostanie włączony. 0 zachowuje dotychczasowe wyjścia L03/L04.
- Pokój 1 (szyb): zygzak lewo–prawo–lewo do sterownika (rząd 12, kolumna 9). FIRE (zbocze naciśnięcia, na ziemi, w ±2 kolumnach i na wysokości `swy`; `sw_ready` pokazuje podpowiedź FIRE) uruchamia stojącą dotąd (czerwoną) windę `lpow=1`; wjazd na platformę wyjścia (rząd 13). Brak obejścia po podłodze.
- Pokój 2 (pomosty): krótszy środek po kruszących się kafelkach (`CRUMB`=127, trzy platformy) lub dłuższe stałe schody. Kontakt → ostrzeżenie (kafel czerwony) od razu, po 40 klatkach kafel znika, po 120 wraca (`cr_t/cr_r/cr_c`, 8 slotów; `respawn` → `crumble_reset`). Podłoga jest siatką bezpieczeństwa.
- Pokój 3 (chłodnia): wentylator (łopatki `FanT`, 2×2 kafle, 2 pozy obracane co 4 klatki; szybciej w podmuchu) i smugi powietrza (glify 8/9) rysowane dokładnie w strefie (`m_fan`..`m_fc1-1`, rzędy `m_fr0`..`m_fr1-1`). Podmuch (`fan_t&64`, okres 128: 64 spokój/64 podmuch) przesuwa o +1 px/klatkę wyłącznie kota w powietrzu w strefie; kolumny 0–9 to osłonięta wnęka. Luka A→B jest do pokonania tylko z wiatrem; dłuższe stałe schody to alternatywa. HUD: WAIT/GUST.
- Pokój 4 (winda dachowa): winda z podłogi do rzędu 6 (`lspd`=12), przesiadka na półkę sterownika (rząd 10, kol. 26), FIRE otwiera bramę, ponowne wsiadanie, skok na kruchy pomost (rząd 6) i wyjście na dach. Chybione skoki lądują na półce lub podłodze.
- Metadane rozszerzone do 26 pól (`swy,swc,lpow,fc1,fr0,fr1,lspd` na offsetach 19–25); `enter_plat_room` kopiuje 32 B do `m_sx`. Nowe zmienne: `sw_on` 06E4, `fan_t` 06E5, `cr_*` 06E7–06FF.
- Weryfikacja źródłowa: `python3 tests/test_level02.py` (model `tools/plat_model.py`, klatka po klatce; nie uruchamia XEX). Podgląd map: `cd tools && python3 show_room.py <0-9>`.
- Od pierwszej dodatniej prędkości pionowej (`vy>=1`) kot sięga przednimi łapami w dół. W pozach opadania i pierwszego kontaktu przód jest opuszczony, zad uniesiony, a tylne łapy pozostają co najmniej 3 piksele ponad przednimi.
- Mechaniki: wentylator (podmuch), winda, odkurzacz (patrol, drap = ucieczka), hazard na rzędzie 22, dialog, scena końcowa L04.
- Weryfikacja: `cd tools && python3 solve_levels.py [1|2|3]` (BFS bot na `sim.py`, wolny).
- Skok: osiem póz dla obu kierunków. Przysiad przed odbiciem trwa 3 klatki, potem kot unosi przód tułowia i prostuje przednie łapy przed pyskiem (bez zawinięcia nadgarstków z animacji chodu). Lądowanie: 2 klatki kontaktu przednich łap i opuszczenia przodu tułowia, 3 klatki dosiadu tylnych i amortyzacji, 3 klatki prostowania, 2 klatki stania. Łapy pozostają oparte w tym samym miejscu podczas amortyzacji i prostowania. Wysokość i grawitacja bez zmian; podczas przysiadu i lądowania kot zatrzymuje ruch poziomy.
- FIRE na poziomie 03: na podłożu pacnięcie przednią łapą z góry (3 klatki uniesienia, 4 klatki uderzenia z podgiętymi palcami, 3 klatki cofnięcia), podczas jazdy kopnięcie tylną. `paw_contact` sprawdza wyłącznie cztery klatki uderzenia, najwyżej jedno trafienie na zamach: widoczne piksele końcówki łapy muszą pokryć niezerowy piksel aktualnego glifu odkurzacza. Sam FIRE, odległość w kolumnach ani puste narożniki sylwetki nie wywołują ucieczki. Trafienie odwraca odkurzacz od użytej łapy i rozbraja go na 40 klatek; na czas rozbrojenia czerwienieje. Ruch poziomy jest zatrzymany podczas zamachu.
- Odkurzacz ma dwie animowane szczotki pod obudową: cztery fazy obrotu co 4 klatki, generowane w `assets/vacuum-brushes.bin`. Obrót trwa także między krokami patrolu; kolizja używa aktualnej grafiki.
- Dach odkurzacza podpiera łapy na pierwszym widocznym rzędzie kopuły, nie na dolnej krawędzi jej kafelka: `player_y = m_arow*8 + 3` (dla rzędu 21: 171 zamiast 176). Wspólne `on_vacuum` obsługuje podparcie, przewożenie i wybór tylnej łapy; opadanie sprawdza powierzchnię po każdym pikselu, także poza granicą kafelka.
- Aktualny L03 używa dwóch odkurzaczy: kopnięcie podczas jazdy trafia osobnego prześladowcę, nigdy maszynę pod kotem.

## L03 actors (cur_level==2 only)
Two slots: A (act_*, slot 0) and pursuer B (bcol/bdir/bt/bwob at 06E8-06EB, slot 21 offset). State: carrier 06EC, slot 06ED, dk 06EE (dock/laser timer), gate 06EF (R2 dock achieved / R4 reader crossed), bw 06F0, bk 06F1 (catch), e_mode 06F2, e_bspd 06F3.
These bytes alias L02 crumble slots, including `e_mode`/`e_bspd`. L03 frame helpers and objective gates check `cur_level==2` before accessing them; zeroing mode at entry alone is insufficient once L02 fills crumble slots.
Metadata aliases: m_xmax 06B3, m_dock 06B4, m_z0 06B6, m_z1 06B7, m_bcol 06CA, m_bmin 06CB, m_bmax 06CC, m_bspd 06CD, m_mode 06CE (offsets 26-30; 32-byte copy) (mode bit0 dock, bit1 reader/gate).
Wsparcie: `roof_on` = oryginalne 3 kolumny `(player_col-act_col)<3` (bez nawisu); `under` traktuje podstawę robota (95/124/125) jako podłoże tylko gdy kafel w art (RoomArtBuf) to FLOOR, belka dołu (BEAM) nie podpiera.
Pościg B: `b_touch` sprawdza brzeg siedzącej sylwetki w rzędzie 23 z kopułą B (+2 px gdy kot patrzy w lewo), tylko dla jeźdźca (carrier==0), po zakończeniu lądowania i poza przygotowaniem skoku. `bk` rośnie o 1 na klatkę kontaktu, maleje o 1 bez kontaktu; 48 = złapany (respawn). Kot na samym przednim brzegu dachu (A+2) nie styka się z B (odstęp 3) - alternatywny unik. Kopnięcie siedzące (hit21) odpycha B przez 40 klatek bez zmiany ruchu nośnika; sylwetka rozróżnia pozycję złożoną i wyprost nogi w aktywnej części ataku.
Laser R2 (`plat_beam`): pas Y 176..183 w z0..z1, wyłączony podczas dokowania. Kolizja wymaga przecięcia pasa przez 24-pikselowe ciało (origin Y 153..183); podłoga Y 184 i pozycja całkowicie ponad pasem są bezpieczne. Dokowanie wymaga ruchu w prawo po trafieniu. `dk` daje 240 klatek wyłączenia lasera, ale pierwsze poprawne dokowanie trwale ustawia `gate` do następnego respawnu. Brama jest widocznie zamknięta przed wykonaniem celu i otwarta po nim; HUD pokazuje EXIT. Wyjście R2 jest na podłodze Y184, bez półki na wysokości głowy i końcowego skoku. Samo podejście do drzwi nie pomija celu; upływ czasu lasera nie zatrzaskuje otwartych drzwi.
Weryfikacja bez kompilacji: `python3 tests/test_l03_routes.py` używa modelu `tests/l03_model.py` (R1 pacnięcie/skok/nisza, R2 dok/timeout/retry/laser, R3 jazda/pościg/zeskok/retry, R4 czytnik/brama/jazda/dwie przerwy/wyjście na podłodze, bez pościgu). Zasięg modelu: logika L03; nie jest to dowód uruchomienia K65.
### L03 R4 (reader room, one carrier)
- Metadata: `mode`=2 (reader), `bspd`=0 (no pursuer B), `amax`=10 / `xmax`=19, reader z0=z1=12, `exit_y`=184 (floor-height hatch). A's patrol is 8..10 (`amin`=8, start 8; the old minimum 7 left a connected hit one column short of the reader); the gate (`a_events` after the reader crossing) switches the limit to `m_xmax` in reader mode only (`a_step`), so the R2 permanent dock latch no longer widens patrol after wobble recovery.
- Bug fixed: with B at column 0 the initial UP landed on B (ride, carrier 21, y 171) and, while `gate`=0, `plat_pursuer` never moved it - a stationary-rear-robot trap in the user's Oct 4 17:09 build. B was removed and `on_vacuum` returns 0 for an inactive reader-mode B (`e_mode&2`, gate 0) even if metadata adds one later.
- Steps: (1) walk right to the forepaw reach (cat x 17..21), swing, and swing again after a miss; every connected hit at patrol columns 8..10 carries it into the reader (column 12) about 17..32 frames later, inside the 40-frame stun, so a hit always opens the gate; standing further right (x >= 22) is within the robot's contact distance; (2) gate: reader tiles 83 `RELAY` -> 84 `DONE` (`draw_reader`, called after the actual crossing and from `act_init`/reset; R2 dock does not call it); (3) wait at the first pit edge and jump to the roof when A is 5..7 columns away and heading left (~24-frame window; ride y 171); (4) ride, jump to the balcony (row 18, cols 23..27, y 152) at A column >=19; (5) jump down/right from the balcony to the floor at cols 33..39 (y 184) and walk to the service exit. The physical floor is only row 22 cols 0..22 and 33..39; rows 22..23 cols 23..32 are clear. Walking the floor, or riding before the gate, never reaches the exit. An early dismount falls into the pit but permits a retry/reboard from the grounded lane; respawn resets gate/reader.
- Captions: `r4_goal` writes four 36-character strings from `R4Goal` (`nocross`, charset ASCII-32) to TermScreen row 0 col 0 through `draw_row0_msg`, chosen by gate and `player_col` (0: SWAT, 1: RIDE/BALCONY, col>=23: GAP/EXIT, col>=33: ACCESS OPEN); the HUD action comes from `R4Hint` {SWAT, RIDE, JUMP, EXIT}.
- Checks: `python3 tests/test_l03_assets.py`; `python3 tests/test_l03_routes.py` (model bots: 36 stand/boarding/rear variants with 0 respawns, every swing phase at x 17..21 gating inside the stun, per-frame three-column robot support, early dismount/reboard, late jump retry, pre-gate rider, captions/hook source checks); `python3 tests/test_l03_runtime.py --asset-overlay` runs the real 6502 `enter_plat_room`/`plat_frame` on the existing XEX via `tools/sim.py` with only the R4 ATR chunk 12 (and its checksum entry) replaced in memory; bot uses joystick/FIRE only, counts real `respawn` calls. Limits: no VBI/display/audio; overlay mode runs the old compiled code on the new room data only; the new captions, reader redraw/reset, door state and the gate-0/gate-1 root guard are checked only in the default branch, which needs a fresh authorised build and has not been run. `out/level03-room4-preview.png` is rendered from source data (initial `Game(3)` screen, font/palette, cat from the sprite atlas), not from a compiled emulator frame.

Sprites: 58 logical frames (50-53 seated/shove, 54-57 right-facing glass reach/contact/push/retract) map through assets/cat-frame-map.bin to 43 physical frames.
Siedzenie (50/51) ma osobny, pionowy tułów, podniesiony pysk, złożone tylne
łapy i niski zawinięty ogon; nie używa poziomego grzbietu animacji stania.
Rzędy styku 21–23 oraz palce aktywnego kopnięcia pozostają bez zmian, więc
poprawa sylwetki nie poszerza kolizji z prześladowcą. Podgląd 2:1:
`out/cat-seated-preview.png`.

## L04: defenses and animated AI confrontation

Two rebuilt rooms preserve L01–L03 data. The first has three rising ledges
(standing Y168/152/136), a Y152 boarding dock and a lift (rows 9–18,
12 frames per step) reaching the Y80 exit. The next room enters at Y80,
crosses Y96/Y112 ledges, has a Y136 recovery ledge and keeps the final
Y120 desk, glass column 33 and computer in their existing animation positions. The
L04 R2 shelf ends at column 34; the main computer unit occupies columns 35..38,
rows 17..21, below it. The front-paw contact window is the four-pixel range
`m_glass*4-player_x` 12..15, and the glass follows row/column positions
33/13 → 34/13 → 35/13 → 35/14 → 36/15 → 36/17 before water reaches the unit.
The exit height is enforced; no floor-walking shortcut reaches the core.

Both electric bands span columns 7–34 on tile row 22. `RTCLOK&32` controls
32 frames lit / 32 unlit. A lit tile over original FLOOR remains supporting
in L04, but `plat_haz` immediately respawns a cat touching Y184 inside the
band; the unlit state restores original art. Airborne cats and elevated
ledges are safe. L03 pit/laser BEAM tiles remain non-supporting.

`plat_dlg` triggers once on a grounded core perch (column >=20, Y<=136).
The existing `dlg_done` ($06D8) persists across falls and resets in `init_plat`.
The blocking `ai_cutscene` freezes movement, keeps the L04 VBI soundtrack
playing, hides PMG and disables the HUD DLI. ANTIC D shows a 160x80 AI
monitor with two mouth states and pulsing circuitry, plus five 36-character
subtitles held for 180 PAL frames each. Only AI lines animate the mouth
and use channel-4 speech bleeps; the cat replies leave it closed.
FIRE or any key skips after a release; cleanup disarms the FIRE edge so
the skip cannot also shove the glass. `plat_push` and its HUD prompt require
`dlg_done`; watching/skipping the exchange never performs the final action.

`tools/make_ai_scene.py` generates streamed chunk 25:
`assets/ai-portrait.pic`, 3200 bitmap bytes + two 30-byte mouth patches,
padded to 26 sectors (3328 B), loaded at $A000. It fits within one ANTIC
4K block and does not touch Music, PMG or the resident cat atlas.
It does overwrite RoomArtBuf and part of CryBuf. Every completion/skip
path checks and reloads the current room and chunk 15 (entire 2560-byte
cry), repaints the glass/HUD, and restores the gameplay display list,
palette and PMG without resetting player/objective state.

`python3 tests/test_level04.py` replays both no-death routes in a source-level
jump/lift/floor model and checks mandatory lift geometry, all 64 hazard
phases and contact boundaries, scene/finale gates and cinematic data/cleanup.
It does not execute K65 or establish emulator display/audio acceptance.
The preview is `out/ai-cutscene-preview.png`; rebuild XEX and pack its
matching ATR after preparing `assets/disk-chunks.bin`.

### Cat atlas streamed after the intro

The 5504-byte packed atlas is no longer a resident `CatSprites` section.
Disk chunk 16 loads it into the existing 6144-byte, 4K-aligned `RevealImg`
panorama region after `cut_to_black`/`switch_to_term`, before the first PMG
frame. The intro image stays in the XEX and works on a fresh boot; its RAM
is reused only after the intro ends. Room, camera, death-cry and PMG buffers
remain separate. Debug level changes never restart the intro.

Chunks now start at sector 290, after the reserved `$1000..$9FFF` resident
bank; the boot loader still reads only the actual resident payload.
Consequently changing the linked program size does not move disk chunks
or require a table/link/table/relink cycle. Chunk indices 0..15 are
unchanged. `python3 tests/test_cat_streaming.py` checks source wiring,
overlay bounds, logical frames and disk layout using synthetic payloads,
not a compiled game. A fresh XEX and ATR are still required together.
Prepare the stable table with `python3 tools/make_atr.py --table-only`,
build `main.k65proj`, then run `python3 tools/make_atr.py` to pack that XEX.
Table-only mode does not read or modify XEX/ATR and works before the first build.

### Title and level soundtracks

The intro is enabled again. FIRE or any keyboard key skips to the approved
bitmap title screen, which also appears after a complete intro. Skip polling
continues during the PCM meow, once per sample page. Shift and console buttons
are accepted too. Held controls must first be released on the title, so one
press cannot skip both scenes. The title plays the existing spy-funk;
FIRE, Return or Space starts level 01.
The original 432 score bytes are unchanged; three bytes were appended for
frames per sixteenth note and lead/bass attack volume.

`python3 tools/make_music.py` generates five 435-byte tracks:

| Scene | Asset | Character | PAL tempo |
|---|---|---|---|
| Title | `assets/music.bin` | Original D-minor spy-funk | 150 BPM |
| L01 | `assets/music-level-01.bin` | Quiet, sparse stealth plucks, restrained bass and hats | 93.75 BPM |
| L02 | `assets/music-level-02.bin` | Regular machinery-like arpeggios and alternating bass octaves | 150 BPM |
| L03 | `assets/music-level-03.bin` | Fast chase melody, busy bass and full drum fills | 187.5 BPM |
| L04 | `assets/music-level-04.bin` | Tense chromatic semitone figures, abrupt cuts and a low pulse | 125 BPM |

All scores loop over 256 steps (16 bars), with their own note tables,
melodies, rhythms and mix. New disk chunks 17 (title) and 18..21 (levels)
follow the unchanged cat-atlas chunk 16. Each occupies four 128-byte sectors;
the resident chunk table now has 26 entries (130 bytes). No soundtrack is
resident in the `$1000..$9FFF` bank. The shared `Music` buffer at
`$B400..$B5FF` sits after the 1000-byte terminal screen and before PMG memory;
it does not overlap the `$A600..$AFFF` finale PCM.

`start_music` silences all voices and clears `music_ready` before SIO.
The immediate VBI still services SFX, but cannot read the replacing score.
After checking SIO status, the padded chunk checksum and tempo/volume bounds, the
player resets its step, timer, note frequencies and envelopes before enabling
music. A failed track load stays silent and halts with visible `DISKERR`
instead of playing corrupt data. Normal room entry and respawn do not restart
the soundtrack; only title/level entry, including debug keys 1..4, selects a
new track. During normal room loads the lead keeps ticking, bass/drums yield
to SIO, and SFX retain channel-4 priority. The final PCM scene still freezes
the VBI player through `level_finished`.

The solver's bounded `boot` helper skips the intro, then releases/presses FIRE at the title before
searching a level. `tests/test_music.py` checks preservation of the title,
generated melodies/moods, tuning, memory bounds, disk data and source wiring.
These checks do not establish compiled K65 playback or audible POKEY quality;
a fresh authorized build, matching ATR and emulator listening are still needed.

### Approved bitmap title

`assets/title-screen.png` is the lossless native 160x192 version of the
approved Desktop mockup. `python3 tools/make_title.py` packs it as
`assets/title-screen.pic`, a four-color ANTIC E bitmap. The Atari palette uses
black, dark blue, light fur/text and amber; RGB colors depend on the emulator
or display. No title bitmap is linked into the resident bank.

Disk chunks 22..24 contain 100, 24 and 64 scanlines, loaded at `$A000`,
`$B000` and `$B600`. Their padded sector spans end at `$B000`, `$B400` and
`$C000`; none touches the live music buffer `$B400..$B5FF`. Three LMS entries
avoid ANTIC's 4K address wrapping. The final four rows are blank, matching the
mockup exactly. The bottom image temporarily uses PMG memory, with PMG and
display DMA disabled during loading. No DLI is needed for the static image.
On leaving the title, DMA is disabled before buffers are reused; gameplay
clears the terminal, restores its display list, reloads room data and
initializes PMG. Intro art remains resident and is not modified by title loads.

Title/music chunks share `load_checked_chunk` for SIO status and padded
checksum validation; errors restore a readable terminal and halt at
`DISKERR`. After regenerating artwork, prepare `make_atr.py --table-only`
before compiling and pack a matching ATR afterwards. `tests/test_title_screen.py`
reconstructs the bitmap from synthetic ATR sectors and its three DMA regions,
checks exact artwork preservation, 4K boundaries, live-music separation and
input/transition source contracts. It is not an emulator screenshot or
compiled runtime acceptance.
