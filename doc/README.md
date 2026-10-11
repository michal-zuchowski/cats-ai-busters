# Cats AI Busters — documentation

Atari XE/XL game written in **K65** (not C, not generic 6502 assembler); all
tooling and tests are Go (standard library only). This file is the single
entrypoint to the docs; project rules for contributors are in
[AGENTS.md](../AGENTS.md).

## Game overview

A ginger agent cat sabotages a rogue AI in a data center. Actual flow, as
scripted in `main` (`src/main_scene.k65`):

1. **Intro** (cutscene, skippable with FIRE/key): pan past server racks →
   zoom into the terminal → terminal text → cat reveal → face close-up →
   meow → boss-cat video call.
2. **Title screen** (`title_screen`): bitmap title + music; FIRE/Return/Space starts.
3. **Level 01 — stealth** (`init_gameplay`, 3 rooms): dodge camera light cones,
   use switch and record console.
4. **Levels 02–04 — platformers** (`next_level` → `init_plat`; 4 / 4 / 2 rooms):
   L02 vertical route (lift, crumbling floor, fan), L03 AI vacuum robots
   (swat, dock, ride), L04 defenses.
5. **AI cutscene + finale** (L04 last room): `ai_cutscene` (talking AI face)
   then `final_scene` (the cat knocks the glass off the desk, the computer dies).
   After the last level the game idles.

Story/design: [intro](scenario/intro.md), [level 01](scenario/level-01.md),
[level 02](scenario/level-02.md), [level 03](scenario/level-03.md),
[level 04](scenario/level-04.md).

## Build pipeline

```
 tools/cmd/*  (Go generators, deterministic)          tests/*_test.go  (go test ./...)
 make-intro-art  make-level-art  make-cat-sprite            |  check sources (k65src), assets,
 make-plat-levels make-music make-meow make-title           |  chunk table, ATR layout and,
 make-ai-scene                                              |  when built, run out/*.xex in
        |                                                   |  the tools/sim 6502 emulator
        v                                                   |
 assets/*.pic *.tbl *.bin *.pcm            assets/disk-chunks.bin  <-- make-atr --table-only
        |  (binary "assets/..." in src)             |  (chunk table, included as ChunkTable)
        v                                           v
 src/*.k65 (34 files, link order = main.k65proj)
   system/hardware : hardware_defs memory_map system display_lists bitmaps
                     hud_strings level_text
   intro           : intro_text intro_camera intro_scenes meow_sample
   level 01        : l01_drawing l01_cameras l01_player l01_rooms l01_hud l01_init
   platform engine : plat_tables plat_surfaces plat_drawing plat_rooms plat_paw
                     plat_switch_crumble plat_physics plat_fan_lift plat_vacuums
                     plat_hazards plat_loop
   cutscene/finale : ai_cutscene final_scene title_screen main_scene
   audio / disk    : audio disk_io
        |
        v   k65 compiler  `k65.exe @main.k65proj`  (-lowAddr 0x1000 -hiaddr 0x9FFF)
 out/cats-ai-busters.xex  (+ .gmap .lst .sym)
        |
        v   make-atr (boot sector + resident segment + disk chunks)
 out/cats-ai-busters.atr  (90K, boots without DOS)
        |                                   ^ chunks (rooms, music, title, cat atlas,
        v                                   |  AI portrait, ...) streamed at runtime
 emulator (Atari800MacX)  <----- disk_io (SIO reads, checksum-verified)
```

## Runtime flow

```
 boot sector -> loads resident program (0x1000..) -> main
   init_system
   INTRO: storm_pan -> zoom_to_terminal -> terminal lines -> show_cat/reveal
          -> face (pupils, meow) -> video_call          (FIRE/key => intro_skip)
   cut_to_black
   title_screen : loads title chunks 22-24 + music 17, waits for FIRE/Return/Space
   lives = 9 ; init_gameplay (cur_level 0, music, level 01 room 0)
   loop each frame:
     wait; dbg_level (debug keys: 1 = restart level 01, 2-4 = jump to level 02-04)
     level_finished == 0 ?
        yes -> frame_any: cur_level==0 ? game_frame (L01 stealth) : plat_frame
        no  -> next_level: cur_level++ ; init_plat   (no-op after level 04)
   lose_life (L01 alert start / platform respawn): lives-- ; lives == 0 -> game_over:
     "GAME OVER" ~3 s -> goto intro_skip (stack reset) -> title_screen -> new game, lives 9
   L01 --level_ending--> L02 --level_ending--> L03 --level_ending--> L04
   L04 room 2: plat_dlg -> ai_cutscene -> plat_push -> final_scene -> idle

 Memory (resident program 0x1000..0x9FFF, 0xA000+ is streamed scratch):
   0xA000 RoomArtBuf / ZoomBuf   0xA400 CamCfg   0xA480 ConeStep   0xA600 CryBuf
   0xB000 TermScreen   0xB400 Music (512 B)   0xB600 TitleBottom
   0xBC00 PMG (cat sprite)       RevealImg (resident, 4K aligned) is reused
   for the cat atlas after the intro (chunk 16)

 Disk (ATR 720 x 128 B): sector 1 boot, sectors 2.. resident segment,
   chunks from sector 290: 0-2 room art, 3 CamCfg, 4 ConeStep, 5-14 plat
   rooms, 15 death cry, 16 cat atlas, 17 title music, 18-21 level music,
   22-24 title bitmap, 25 AI portrait (26 chunks, table in disk-chunks.bin)
```

## Where is what

| Doc | Content |
| --- | --- |
| [scenario/intro.md](scenario/intro.md) | Intro story and beats |
| [scenario/level-01.md](scenario/level-01.md) … [level-04.md](scenario/level-04.md) | Level story and design |
| [tech/gameplay.md](tech/gameplay.md) | Level 01 rules, runtime/memory, disk boot and chunks, platform levels 02–04 notes, music, title (long, partly historical/Polish) |
| [tech/graphics.md](tech/graphics.md) | Graphics decisions (ANTIC modes, intro art) |
| [tech/source-map.md](tech/source-map.md) | Every `src/*.k65` file: purpose, key functions, assets used |
| [tech/tools.md](tech/tools.md) | Go packages, CLIs, flags and every test |
| [tech/assets.md](tech/assets.md) | Every `assets/` file: format, generator, consumer |
| [../AGENTS.md](../AGENTS.md) | Contributor rules (source layout, Go tooling) |

## Quick commands

Build/run need the K65 compiler (see `tools/run.sh`, `ci/k65-build.sh`);
compile only on explicit request.

```sh
go run ./tools/cmd/make-atr --table-only   # refresh assets/disk-chunks.bin (before compiling)
k65.exe @main.k65proj                      # compile -> out/cats-ai-busters.xex
go run ./tools/cmd/make-atr                # pack XEX + chunks -> out/cats-ai-busters.atr
tools/run.sh                               # compile + make-atr + launch Atari800MacX (macOS)
go test ./...                              # tests (XEX/ATR-dependent ones skip if missing)
gofmt -l . && go vet ./...                 # must be clean
```

Regenerate assets (deterministic; see [assets.md](tech/assets.md) for which
tool writes what): `go run ./tools/cmd/make-<name>` for `intro-art`,
`level-art`, `cat-sprite`, `plat-levels`, `music`, `meow`, `title`,
`ai-scene`. Ordering rule: `assets/disk-chunks.bin` is compiled into the XEX
(`ChunkTable`) and must match the chunk sources, so after changing any
streamed asset run `make-atr --table-only` → compile → `make-atr`. A
missing or stale table makes the compiler abort silently
(see "Gotcha" in [tech/gameplay.md](tech/gameplay.md)). Since chunks start at
a fixed sector, changing the program size alone does not move them.

Inspect: `go run ./tools/cmd/show-room <0-9>` (ASCII platform room),
`go run ./tools/cmd/solve-levels [1|2|3]` (bot proof, needs build + ATR).
