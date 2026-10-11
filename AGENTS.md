# K65 project

Documentation entrypoint: [doc/README.md](doc/README.md) (overview, pipeline, source map, tools, assets).

Target platform: Atari XE/XL. This is a K65 language project, not C or a generic 6502 assembler.

- Search `reference/graph.jsonl` first for task keywords (for example `rg -i 'raster|irq' reference/graph.jsonl`, or use IDE text search). `node` records identify a source `path` and `line_start`/`line_end`; `edge` records connect sections, linked pages, grammar rules and platform examples by ID. Read only the matching source sections when needed; if no node matches, search the original docs. The graph is an index, not a replacement for the source text or persistent LLM memory.
- The main documentation starts at `reference/doc/docs/index.md`; `reference/compiler/compiler.inc` is the compiler's authoritative grammar. The graph and source documents are snapshots bundled with this plugin, not automatically updated when reference files change.
- Working code examples: `reference/examples/atari-xl/` and `reference/examples/a2600-tutorial-03/`. C64 guidance and code snippets are in `reference/doc/docs/platform-c64.md` and `reference/doc/docs/examples.md`.
- Online: [K65 documentation by zbyti](https://zbyti.github.io/k65-mkdocs/) and [margorski's Atari 2600 AI example](https://github.com/margorski/k65-vcs-ai-example). See `reference/UPSTREAM.md` for attribution. The external example is linked, not copied into this project.
- Edit the `src/*.k65` files and `main.k65proj`; keep generated `.xex`, `.bin`, `.prg`, `.gmap`, `.lst`, and `.sym` files in `out/`. Treat `reference/` as read-only background material, not project source.

- Build using the K65 run configuration or manually invoke **Build K65 Project** on the project's `.k65proj`. The plugin supplies its own compiler; no local K65 checkout is needed for standalone projects.
- The bundled compiler currently runs on macOS/Linux amd64/arm64; creating or editing a project on Windows works, but compiling there is not yet supported by this plugin.
- K65 project lists may execute Squirrel (`$`) or shell commands (`!`). Do not execute example lists, untrusted project files, or build commands automatically during editing, indexing, or on save. Compile only after an explicit user request.

## Source layout rules

- `main.k65proj` lists the `src/*.k65` files in link order. Preserve that order: data/var/func order affects memory layout and disk chunk addresses.
- Put new code in the file whose topic it belongs to. Add a new file only for a new topic; name it after what it does (e.g. `bitmaps`, `l01_cameras`, `plat_vacuums`; `l01_` = level 01 stealth, `plat_` = platform levels 02-04), never by number or as a catch-all.
- Every file starts with a header comment listing what is inside it and what each part is for; update it whenever you add, move or remove data or functions.
- Keep a file to one topic and roughly under 500 lines; split by topic when it grows past that.
- When adding, renaming or removing a file, update `main.k65proj` (in the right position) and the table below.
- Tools and tests read the whole source via `k65src.Read()` (`tools/k65src`), never a single file path.

Source layout (`src/`, in project order):

| Area | Files |
| --- | --- |
| Definitions/data | `hardware_defs` (Atari registers), `meow_sample` (PCM), `memory_map` (variables), `intro_text`, `hud_strings`, `bitmaps`, `display_lists` |
| System | `system` (input, wait, init) |
| Intro cutscene | `intro_camera`, `intro_scenes` |
| Level 01 (stealth) | `l01_drawing`, `l01_cameras`, `l01_player`, `audio`, `l01_rooms`, `l01_hud`, `disk_io`, `l01_init`, `level_text` |
| Platform levels 02-04 | `plat_tables`, `plat_surfaces`, `plat_drawing`, `plat_rooms`, `plat_paw`, `plat_switch_crumble`, `plat_physics`, `plat_fan_lift`, `plat_vacuums`, `plat_hazards`, `ai_cutscene`, `final_scene`, `plat_loop` |
| Entry | `title_screen`, `main_scene` (`main` and the scene script) |

## Tools and tests (Go only)

- All tools and tests are written in Go (module in `go.mod`, standard library only; no Python, no third-party packages without a strong reason).
- Libraries live in `tools/<pkg>` (e.g. `tools/levelart`, `tools/atr`, `tools/sim`); runnable tools are thin `tools/cmd/<name>/main.go` wrappers, run as `go run ./tools/cmd/<name>`.
- Tests live in `tests/*_test.go` (plus unit tests next to packages) and run with `go test ./...`. Tests that need a compiled `out/*.xex` skip when it is missing.
- Generators must be deterministic: rerunning a tool on unchanged inputs must reproduce the committed `assets/` byte for byte.
- Run `gofmt -l .` and `go vet ./...` clean before finishing.

## Model Usage Policy

- Expensive, high-capability models (e.g. Claude Opus, GPT-*-sol class) are reserved for planning, task breakdown, delegation, supervision, code review and verification.
- Implementation is delegated to cheaper models (e.g. Claude Sonnet, Haiku, GPT-*-mini/luna class) running as sub-agents with complete context, a bounded scope, acceptance criteria and explicit verification steps.
- The expensive coordinating model owns the plan, assigns implementation tasks, monitors progress and reviews every delegated diff. It independently verifies the requested behavior rather than accepting the implementing model's completion claim.
- Substantive corrections found during review go back to the cheaper implementing model; the expensive model must not take over bulk implementation.
- Use existing tests, linters and runtime checks for acceptance. Compile K65 only after an explicit user request; otherwise clearly distinguish source/data checks from fresh compiled-game verification.
- Small, trivial edits (a few lines, docs, config) may be done directly when delegation would cost more than the change.
