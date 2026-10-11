# Assets (`assets/`)

[← back to doc/README.md](../README.md)

Generated binary data is committed to git and must be reproducible byte for
byte by its generator (`go run ./tools/cmd/<tool>`; see [tools.md](tools.md)).
Sizes are from the current files. "Resident" = linked into the XEX with
`binary "assets/..."`; "chunk n" = streamed from disk at runtime (index in
`disk-chunks.bin`; checksummed by `disk_io`). Pictures are packed ANTIC bitmaps
(`.pic`), `.tbl` are small lookup tables.

## Intro art — `make-intro-art` (previews go to `out/`)

| File | Bytes | What | Consumer |
| --- | ---: | --- | --- |
| `room.pic` | 6144 | 256×96 facility panorama (mirrored), 64 B × 96 rows | resident `RevealImg` (`bitmaps`); storm pan / reveal; RAM reused for cat atlas after the intro |
| `cat-strip.pic` | 384 | panorama patch with the cat (bottom rows) | resident `CatStrip`; `show_cat` patches it into the panorama |
| `leds.tbl` | 484 | rack LED positions, EOR masks, blink patterns | resident `Leds`; `blink_rack_leds` |
| `face.pic` | 3840 | 40×96 cat face close-up | resident `FaceImg`; pupil dilation |
| `call.pic` | 3200 | 40×80 video-call picture | resident `CallImg`; `video_call` |
| `call-mouth.tbl` | 12 | 6 closed + 6 open mouth bytes for the call | resident `CallMouth`; `call_mouth` |
| `call-text.pic` | 203 | on-screen text of the call | resident `CallText` |
| `term-font.pic` | 1024 | 1K charset: intro terminal font + level tile graphics (levelart glyphs) | resident `TermFont` (`intro_text`) |
| `zoom-left.tbl`, `zoom-right.tbl` | 256 each | per-row tables of the zoom-to-terminal push-in | resident `ZoomLeft/ZoomRight` |

## Level art — `make-level-art`

| File | Bytes | What | Consumer |
| --- | ---: | --- | --- |
| `level-rooms.pic` | 2880 | 3 × 960-byte level 01 rooms | chunks 0–2 → `RoomArtBuf` (`l01_rooms`) |
| `level-cams.bin` | 48 | camera config: per-room camera slots | chunk 3 → `CamCfg` (`load_level_assets`) |
| `cone-step.bin` | 160 | light-cone step table | chunk 4 → `ConeStep` |
| `vacuum-brushes.bin` | 24 | packed rotating vacuum-brush glyph rows (4 phases) | resident `BrushSpin` (`plat_surfaces`) |

## Platform levels — `make-plat-levels`

| File | Bytes | What | Consumer |
| --- | ---: | --- | --- |
| `plat-rooms.bin` | 10240 | 10 platform rooms × 1024 B (960 B art + 64 B metadata) for L02 (4), L03 (4), L04 (2) | chunks 5–14 (`plat_rooms`) |
| `plat-cry.pcm` | 2560 | computer death-cry PCM | chunk 15 → `CryBuf` (`plat_rooms` on L04 room 1, `ai_cutscene` reload, `final_scene`) |

## Sprites — `make-cat-sprite`

| File | Bytes | What | Consumer |
| --- | ---: | --- | --- |
| `cat-sprites.bin` | 5504 | atlas of 43 physical PMG frames (four 16x24 players per frame) | chunk 16 → `RevealImg` region after the intro (`load_level_assets`) |
| `cat-frame-map.bin` | 58 | 58 logical frames → physical frame | resident `FrameMap` |

## Audio

| File | Bytes | Generator | What | Consumer |
| --- | ---: | --- | --- | --- |
| `meow-source.wav` | 101968 | – (hand-supplied input) | CC0 meow recording, mono 16-bit 44.1 kHz | input of `make-meow` |
| `meow.pcm` | 3072 | `make-meow` | 4-bit samples at 3.9 kHz (12 × 256) | resident `Meow` (`meow_sample`) |
| `music.bin` | 435 | `make-music` | title score | chunk 17 → `Music` |
| `music-level-01.bin` … `-04.bin` | 435 each | `make-music` | per-level POKEY scores | chunks 18–21 → `Music` |

Track format (lead/bass/drums tables + 3 control bytes) is documented in
`tools/music/music.go` and, with tempos per level, in
[gameplay.md](gameplay.md).

## Title and AI scene

| File | Bytes | Generator | What | Consumer |
| --- | ---: | --- | --- | --- |
| `title-screen.png` | 2276 | – (hand-made input) | 160×192 4-colour mockup | input of `make-title` |
| `title-screen.pic` | 7520 | `make-title` | ANTIC E bitmap, 188 rows × 40 B (+4 blank) | chunks 22 (4000 B), 23 (960 B), 24 (2560 B) → `0xA000`, `0xB000`, `0xB600` |
| `ai-portrait.pic` | 3260 | `make-ai-scene` | 160×80 AI face (3200 B) + two 30-byte mouth patches | chunk 25 → `0xA000` (`ai_cutscene`) |

## Disk chunk table

| File | Bytes | Generator | What | Consumer |
| --- | ---: | --- | --- | --- |
| `disk-chunks.bin` | 130 | `make-atr` (`--table-only` or full) | 26 entries × 5 B: start sector (word), sector count (word), checksum (byte) | resident `ChunkTable` (`bitmaps`) |

Chunk index map (from `atr.ExtraChunks()`, disk order from sector 290):

| Chunk | Name | Source |
| --- | --- | --- |
| 0–2 | room0/1/2-art | `level-rooms.pic` slices of 960 B |
| 3 / 4 | cam-cfg / cone-step | `level-cams.bin` / `cone-step.bin` |
| 5–14 | plat-room0..9 | `plat-rooms.bin` slices of 1024 B |
| 15 | death-cry | `plat-cry.pcm` |
| 16 | cat-atlas | `cat-sprites.bin` |
| 17 | title-music | `music.bin` |
| 18–21 | level1..4-music | `music-level-01..04.bin` |
| 22–24 | title-top / middle / bottom | `title-screen.pic` slices 4000 / 960 / 2560 B |
| 25 | ai-portrait | `ai-portrait.pic` |

Chunks are zero-padded to 128-byte sectors; the checksum is the sum of the
padded bytes mod 256. If you change a chunk source, rerun
`make-atr --table-only`, recompile, then `make-atr` (see the README).
