# Tools and tests (Go)

[← back to doc/README.md](../README.md)

All tooling is Go (module root = repo root); run from the repo root. Generators
write into `assets/` (see [assets.md](assets.md)), previews into `out/`.
Generators are deterministic and were ported bit-for-bit from earlier Python
tools (`tools/pyrand` reproduces CPython's `random.Random`). Nothing here
compiles K65 — compilation is `tools/run.sh` / `ci/k65-build.sh`.

## CLIs (`go run ./tools/cmd/<name>`)

| CLI | Purpose | Inputs | Outputs / flags |
| --- | --- | --- | --- |
| `make-intro-art` | Intro pictures and tables | – (procedural, `introart` + `levelart`) | `assets/`: `call.pic`, `call-mouth.tbl`, `call-text.pic`, `term-font.pic`, `zoom-left.tbl`, `zoom-right.tbl`, `room.pic`, `face.pic`, `cat-strip.pic`, `leds.tbl`; previews `out/*-preview.ppm` |
| `make-level-art [preview-dir]` | Level 01 rooms, cameras, brushes | – | `level-rooms.pic`, `level-cams.bin`, `cone-step.bin`, `vacuum-brushes.bin`; with a dir: `room0..2.png` with cat overlay |
| `make-plat-levels` | Platform rooms L02–L04 + death cry | – (`platlevels`) | `plat-rooms.bin`, `plat-cry.pcm` |
| `make-cat-sprite [preview.png]` | Cat PMG atlas | – | `cat-sprites.bin`, `cat-frame-map.bin`; preview PNG (name `glass-knock-preview.png` gives the glass preview) |
| `make-meow` | Meow PCM | `assets/meow-source.wav` | `meow.pcm` |
| `make-music` | Title and per-level scores | – (`music`) | `music.bin`, `music-level-01..04.bin` |
| `make-title` | Title bitmap | `assets/title-screen.png` | `title-screen.pic` |
| `make-ai-scene` | AI face cutscene picture | – | `ai-portrait.pic`, `out/ai-cutscene-preview.png` |
| `make-atr` | Disk image and chunk table | `out/cats-ai-busters.xex` (full mode) and chunk assets | `--table-only` (also `-table-only`): writes only `assets/disk-chunks.bin`, works before the first build. Full: `out/cats-ai-busters.atr` + `disk-chunks.bin`. `-h` help; other args error |
| `show-room <0-9>` | ASCII preview of a platform room | `plat-rooms` data via `platlevels` | stdout |
| `solve-levels [1\|2\|3]` | BFS bot plays levels on the 6502 emulator | built XEX, ATR, `.sym` | stdout; slow, needs a fresh build |

**Ordering:** regenerate chunk-source assets → `make-atr --table-only` →
compile K65 → `make-atr`. A stale `disk-chunks.bin` makes the compiler fail
without a clear message (details in [gameplay.md](gameplay.md)).

## Packages (`tools/`)

| Package | Role |
| --- | --- |
| `k65src` | Locates the repo root and reads `src/*.k65` in link order (`Read`, `Files`, `Path`); tests grep the source through it |
| `atr` | 90K ATR builder, boot loader, chunk list `ExtraChunks()`, chunk table; layout: boot sector 1, resident segment from sector 2, chunks from sector 290 |
| `sim` | Headless 6502 emulator harness: loads XEX + ATR from `out/` and `.sym` (`LoadSymbols`), fakes joystick, RTCLOK and SIOV; no VBI/DLI/audio. `cpu.go` = CPU |
| `solver` | BFS over input macros against a `Machine` interface (`Solve(level, limit)`) |
| `platmodel` | Frame-faithful Go model of the `plat_*` logic (L02), route planner/search |
| `levelart` | Tiles/glyphs, level 01 room screens, cameras, brushes |
| `introart` | Intro pictures (panorama, face, call, scenes) |
| `platlevels` | Platform room layouts, metadata, `show.go` previews |
| `catsprite` | Ginger cat as four PMG players, frame map |
| `music` | POKEY scores (435-byte tracks) |
| `meow` | WAV → 4-bit PCM |
| `title` | PNG → ANTIC E title picture |
| `aiscene` | AI portrait picture and mouth patches |
| `pyrand` | CPython MT19937 `random.Random` clone |
| `tests/l03model` | Go port of level 03 logic used by L03 route tests |

## Tests (`go test ./...`)

CI also runs gofmt on `tools`/`tests` and `go vet`. Tests marked **XEX/ATR**
require a built `out/cats-ai-busters.xex` (and ATR/`.sym`) and **skip**
otherwise; the rest only read sources/assets.

| File | Verifies | Needs build |
| --- | --- | --- |
| `cat_streaming_test.go` | cat atlas chunk/streaming contract | no |
| `gameplay_logic_test.go` | level 01 logic and source contracts | no |
| `intro_scene_test.go` | intro scene sources/assets | no |
| `l03_assets_test.go` | level 03 room assets | no |
| `l03_routes_test.go` | L03 routes with `tests/l03model` | no |
| `level_timing_test.go` | level timing constants | no |
| `level02_test.go` | L02 via `platmodel` | no |
| `level04_test.go` | L04 rooms/dialogue/finale sources and assets | no |
| `meow_test.go` | meow PCM generation/format | no |
| `music_test.go` | music tracks format/determinism | no |
| `solver_macros_test.go` | solver macros with a fake machine | no |
| `title_screen_test.go` | title picture and chunks (synthetic `atr.BuildATR`) | no |
| `l04_loader_runtime_test.go` | `TestL04LoaderSourceContract`: no; `TestL04LoaderRuntime`: runs loader in `sim` | runtime test: **XEX/ATR/.sym** |
| `l03_runtime_test.go` | `TestL03RuntimeR4Route`: runs the R4 route in `sim` | **XEX** |
| `atr_test.go` | `TestATRLayout` against the built XEX/ATR | **XEX/ATR** |
| `disk_chunks_test.go` | `TestDiskChunksMatchATR`: `disk-chunks.bin` matches the ATR | **ATR** |
| `helpers_test.go` | shared helpers (`requireBuilt`, `readAsset`, …) | – |
| `tools/sim/cpu_test.go`, `tools/k65src/k65src_test.go` | CPU and source-reader unit tests | no |

**`L03_ASSET_OVERLAY=1`** (L03 runtime test): replaces only the R4 room chunk
and its checksum in the in-memory ATR so the previously built code runs on
freshly generated data, without recompiling. Without it, the test needs a
fresh authorised build.

```
go test ./...                               # everything runnable
go test ./tests -run 'Level02|Level04'      # subset
L03_ASSET_OVERLAY=1 go test ./tests -run L03Runtime
```

(Test name patterns above are illustrative; use `go test ./tests -list .` for exact names.)
