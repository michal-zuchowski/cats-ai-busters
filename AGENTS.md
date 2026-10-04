# K65 project

Target platform: Atari XE/XL. This is a K65 language project, not C or a generic 6502 assembler.

- Search `reference/graph.jsonl` first for task keywords (for example `rg -i 'raster|irq' reference/graph.jsonl`, or use IDE text search). `node` records identify a source `path` and `line_start`/`line_end`; `edge` records connect sections, linked pages, grammar rules and platform examples by ID. Read only the matching source sections when needed; if no node matches, search the original docs. The graph is an index, not a replacement for the source text or persistent LLM memory.
- The main documentation starts at `reference/doc/docs/index.md`; `reference/compiler/compiler.inc` is the compiler's authoritative grammar. The graph and source documents are snapshots bundled with this plugin, not automatically updated when reference files change.
- Working code examples: `reference/examples/atari-xl/` and `reference/examples/a2600-tutorial-03/`. C64 guidance and code snippets are in `reference/doc/docs/platform-c64.md` and `reference/doc/docs/examples.md`.
- Online: [K65 documentation by zbyti](https://zbyti.github.io/k65-mkdocs/) and [margorski's Atari 2600 AI example](https://github.com/margorski/k65-vcs-ai-example). See `reference/UPSTREAM.md` for attribution. The external example is linked, not copied into this project.
- Edit `main.k65`, `defs.k65`, and the project's `.k65proj` file. Keep generated `.xex`, `.bin`, `.prg`, `.gmap`, `.lst`, and `.sym` files in `out/`. Treat `reference/` as read-only background material, not project source.
- Build using the K65 run configuration or manually invoke **Build K65 Project** on the project's `.k65proj`. The plugin supplies its own compiler; no local K65 checkout is needed for standalone projects.
- The bundled compiler currently runs on macOS/Linux amd64/arm64; creating or editing a project on Windows works, but compiling there is not yet supported by this plugin.
- K65 project lists may execute Squirrel (`$`) or shell commands (`!`). Do not execute example lists, untrusted project files, or build commands automatically during editing, indexing, or on save. Compile only after an explicit user request.

## Model Usage Policy

- Expensive, high-capability models (e.g. Claude Opus, GPT-*-sol class) are reserved for planning, task breakdown, delegation, supervision, code review and verification.
- Implementation is delegated to cheaper models (e.g. Claude Sonnet, Haiku, GPT-*-mini/luna class) running as sub-agents with complete context, a bounded scope, acceptance criteria and explicit verification steps.
- The expensive coordinating model owns the plan, assigns implementation tasks, monitors progress and reviews every delegated diff. It independently verifies the requested behavior rather than accepting the implementing model's completion claim.
- Substantive corrections found during review go back to the cheaper implementing model; the expensive model must not take over bulk implementation.
- Use existing tests, linters and runtime checks for acceptance. Compile K65 only after an explicit user request; otherwise clearly distinguish source/data checks from fresh compiled-game verification.
- Small, trivial edits (a few lines, docs, config) may be done directly when delegation would cost more than the change.
