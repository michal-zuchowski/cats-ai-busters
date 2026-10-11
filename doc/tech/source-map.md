# Source map (`src/*.k65`)

[← back to doc/README.md](../README.md)

Files in **link order** from `main.k65proj` (order affects memory layout and
disk chunk addresses; do not reorder). Each file starts with a header comment
that is the authoritative function list; this is the overview. Tools and tests
read all files as one text via `k65src.Read()`. Asset details:
[assets.md](assets.md). "Chunk n" = disk chunk index in
`assets/disk-chunks.bin` (table in `bitmaps.k65`).

Exact link order (sections below group files by topic, keeping this order
within each group):

```
hardware_defs meow_sample memory_map intro_text hud_strings bitmaps display_lists
system intro_camera intro_scenes l01_drawing l01_cameras l01_player audio l01_rooms
l01_hud disk_io l01_init level_text plat_tables plat_surfaces plat_drawing plat_rooms
plat_paw plat_switch_crumble plat_physics plat_fan_lift plat_vacuums plat_hazards
ai_cutscene final_scene plat_loop title_screen main_scene
```

## Definitions and data

| File | What it does | Key items | Assets |
| --- | --- | --- | --- |
| `hardware_defs` | Atari XL/XE register definitions | GTIA, ANTIC, POKEY, PIA/MMU, SIO/OS vectors, OS shadows | – |
| `meow_sample` | Meow PCM data (CC0, 256-aligned + 64 B silence) | `meow_ptr`, `Meow` | `meow.pcm` |
| `memory_map` | Zero-page and page-6 variables shared by all scenes; screen/PMG addresses | intro, L01, disk (`disk_*`), platform (`m_*`, actors) state, `lives` `0x06A9`; `ZoomBuf` `0xA000`, `CryBuf` `0xA600`, `StageRoom` `0xA800`, `StageMusic` `0xAC00` (overlap CryBuf before platform gameplay; `preload_ready` `0x06AA`), `TermScreen` `0xB000`, `Music` `0xB400`, `TitleBottom` `0xB600`, `PmMem` `0xBC00` | – |
| `intro_text` | Terminal font (also level tile graphics), zoom tables, video-call patches/text, terminal lines | `TermFont`, `ZoomLeft/Right`, `CallMouth`, `CallText`, `Line0..Line6` | `term-font.pic`, `zoom-left.tbl`, `zoom-right.tbl`, `call-mouth.tbl`, `call-text.pic` |
| `hud_strings` | Level 01 HUD strings (ascii-32 encoding) | `MsgHud`, `MsgLevel`, `MsgCamOff`, `MsgStop`, `MsgAlert`, `MsgGameOver`, `MsgRoute` … | – |
| `bitmaps` | Picture data and RAM buffers | `RoomArtBuf` `0xA000`, `CamCfg` `0xA400`, `ConeStep` `0xA480`, `ChunkTable`, `FrameMap`, `RevealImg`, `Leds`, `CatStrip`, `FaceImg`, `CallImg`, disk self-test messages | `disk-chunks.bin`, `cat-frame-map.bin`, `room.pic`, `leds.tbl`, `cat-strip.pic`, `face.pic`, `call.pic` |
| `display_lists` | ANTIC display lists | `dl_term` (terminal / L01 play), `dl_call`, `dl_ai`, `dl_title`, `dl_storm`, `dl_reveal`, `dl_face` | – |

## System

| File | What it does | Key items |
| --- | --- | --- |
| `system` | Input, timing, one-time init | `input_pressed`, `check_intro_skip`, `wait_frames`, `init_system` |

## Intro cutscene

| File | What it does | Key items | Assets used (via data) |
| --- | --- | --- | --- |
| `intro_camera` | Storm pan over the panorama, LED blink, zoom bitmap, cat reveal | `build_storm_list`, `storm_pan`, `blink_rack_leds`, `show_zoom_bitmap`, `show_cat`, `build_reveal_list` | `room.pic`, `leds.tbl`, `cat-strip.pic` |
| `intro_scenes` | Terminal text, cuts, face close-up, meow, video call | `reveal_line0..6`, `blink_prompt`, `switch_to_*`, `zoom_to_terminal`, `pupils_stage1..3`, `play_meow`, `video_call`, `cut_to_black` | `face.pic`, `call.pic`, `meow.pcm`, zoom/call tables |

## Level 01 (stealth, `l01_`)

| File | What it does | Key items |
| --- | --- | --- |
| `l01_drawing` | Message rows, room header, cat PMG sprite drawing, walk animation | `draw_row0_msg`, `draw_player`, `tick_cat`, `repaint_current`, `init_pmg` |
| `l01_cameras` | Camera sweep, visibility test, light cones | `cam_sees`, `load_cams`, `tick_cams`, `draw_cone`, `redraw_cones`, `tick_switch` |
| `l01_player` | Detection, alert/respawn, record-console action, joystick | `check_detection`, `trigger_alert`, `start_playback`, `read_and_move` |
| `l01_rooms` | Room art streaming, room entry, ending, LED blink | `load_room_chunk`, `load_room_art` (chunks 0–2), `enter_room`, `level_ending`, `blink_leds` |
| `l01_hud` | Bottom HUD and the L01 frame loop | `draw_hud`, `draw_lives`, `lose_life`, `game_over`, `game_frame` |
| `l01_init` | Level 01 start | `init_gameplay` (self-test, assets, music, first room) |
| `level_text` | Per-level story/dialogue/hint text and per-level tables | `Txt`, `Dlg`, `Fin`, `Hints`, `LvBase`, `LvRooms` (3 4 4 2), `LvC1/LvC2`, `RowLo/RowHi` |

Linked between these: `audio` and `disk_io` (see below).

## Audio and disk (resident services)

| File | What it does | Key items | Chunks |
| --- | --- | --- | --- |
| `audio` | POKEY effects and music sequencer (VBI-driven) | `sfx_switch/alert/success`, `tick_audio`, `music_step`, `tick_music` | reads `Music` buffer |
| `disk_io` | SIO sector reads (no DOS), checksums, asset loads, music start/stop, early level 2 staging | `disk_read`, `disk_checksum`, `disk_selftest`, `load_level_assets`, `load_checked_chunk`, `start_music`, `level_music`, `init_audio`, `l01_preload`, `preload_level2`, `take_staged_room`, `take_staged_music` | 0, 3, 4, 5, 16, 17–21 |

## Platform engine (levels 02–04, `plat_`)

Data-driven: rooms are disk chunks (`plat-rooms.bin`), see `make-plat-levels`.

| File | What it does | Key items |
| --- | --- | --- |
| `plat_tables` | Data tables | `ActT`, `FanT`, `HintOff/DlgOff/FinOff`, `R4Goal/R4Hint`, `GlassPoses` |
| `plat_surfaces` | Surface helpers, vacuum roofs as platforms | `row_ptr`, `put_run`, `vacuum_roof`, `on_vacuum`, `under`, `BrushSpin` (`vacuum-brushes.bin`) |
| `plat_drawing` | Actor/object drawing | `spin_brushes`, `draw_act`, `draw_lift`, `draw_gate`, `draw_reader`, glass-on-desk scene |
| `plat_rooms` | Room setup/transitions; loads room chunks 5–14 and death cry (chunk 15) | `act_init`, `plat_reset`, `respawn`, `enter_plat_room`, `plat_exit` |
| `plat_paw` | Paw swat, contact tests, glass push | `swat`, `paw_contact`, `paw_both`, `plat_push` (→ `final_scene`) |
| `plat_switch_crumble` | Controller switch, crumbling floor | `plat_switch`, `plat_crumble`, `crumble_reset` |
| `plat_physics` | Input, gravity, jump, collision | `plat_input`, `plat_phys`, `plat_fire`, `l01_hop` |
| `plat_fan_lift` | Fan airflow, lift, carrying | `plat_fan`, `plat_lift`, `carry` |
| `plat_vacuums` | Vacuum robots (docked, patrol, pursuer) and hit tests | `dock_tick`, `plat_actor`, `plat_pursuer`, `plat_hit` |
| `plat_hazards` | Lasers, electrified floor | `plat_beam`, `plat_haz` |
| `plat_loop` | Goals, HUD, frame loop, level progression | `r4_goal`, `plat_hud`, `plat_frame`, `frame_any`, `init_plat`, `next_level`, `dbg_level` |

`ai_cutscene` and `final_scene` are linked between `plat_hazards` and `plat_loop`:

| File | What it does | Key items | Chunks |
| --- | --- | --- | --- |
| `ai_cutscene` | Full-screen talking AI face and in-level dialogue trigger (L04) | `ai_cutscene`, `plat_dlg`, `ai_mouth`, `ai_tick` | 25 (portrait → `0xA000`), 15 (reloads cry) |
| `final_scene` | Finale: glass pushed off the desk, computer dies (PCM cry) | `final_scene` (four poses), sets `level_finished` | uses `CryBuf` |

## Entry

| File | What it does | Key items | Chunks |
| --- | --- | --- | --- |
| `title_screen` | Bitmap title, title music, waits for start | `title_screen` | 22, 23, 24 (title), 17 (music) |
| `main_scene` | `main`: intro sequence → title → `init_gameplay` → per-frame loop (`frame_any` / `next_level`) | `main` | – |

## Rules

Put code in the file of its topic; keep a file to one topic and roughly under
500 lines; update the file's header comment and the table in
[AGENTS.md](../../AGENTS.md) when adding/renaming/removing files, and update
`main.k65proj` in the right position.
