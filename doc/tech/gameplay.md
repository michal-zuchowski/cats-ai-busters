# Level 01 — technical plan (not implemented)

This is the implementation target for [The Blind Spot](../scenario/level-01.md).
The current `main.k65` still stops after the CATCOM intro; none of the
controls or gameplay systems below exist yet.

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
