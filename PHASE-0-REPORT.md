# Phase 0 report

**Shipped.** Every command in §2.6 of the spec is implemented and produces
hand-authored output: `import`, `status`, `search`, `find`, `get`, `open`,
`pack`, `ask`, `export`, `config`, and the `mcp` parent with `start`,
`install` (claude-code, claude-desktop, manual), `audit`, and `permissions`
subcommands. The fixture corpus has 25 cards covering all four kinds. All
human-readable output is pinned by golden-file snapshot tests, with an
additional contract test enforcing no em dashes and 80-column plain-text
goldens. `SCREENPLAY.md` matches the current goldens verbatim.

**Craft fixes landed after the first Phase 0 pass.**
- Command handlers now read through `internal/source.Source` via
  `NewRootWithSource`, so Phase 1 can swap the fixture-backed source without
  rewriting command scaffolding.
- Data-returning commands now honor their declared format flags instead of
  silently ignoring them.
- Search and render JSON now use typed DTOs with `handle`, `card`, and
  `why_shown` fields.
- `cairn pack` now escapes XML correctly, omits empty `source` attributes, and
  honors `--limit`.
- Plain-text outputs wrap to the default width, including image-card excerpt
  fallbacks and status permissions summary.
- Error paths such as `import /tmp/does-not-exist` and `get @99` now return
  non-zero exits instead of printing error copy and succeeding.

**Descoped from Phase 0 (intentional).**
- No real storage. `~/.cairn/cairn.db` is never written; data comes from
  `internal/fixtures/cards.json`.
- No interactive TUI. `cairn find` prints a static frame. Real keyboard
  navigation is Phase 2.
- No real MCP server. `cairn mcp start` exits immediately with a stub.
  JSON-RPC server is Phase 3.
- No real MCP install. `cairn mcp install` previews the config merge but does
  not write it.
- No ANSI color output. `--no-color` parses but is a no-op. Styling is Phase 1.
- No real browser invocation. `cairn open` prints the URL but does not call
  `open` or `xdg-open`.
- No RAG synthesis. `cairn ask` is a stub redirecting the user to `pack`.
- No real phoenix export. `cairn export` is a dry-run listing filenames only.
- Handle resolution is still positional. `@N` maps to fixture index `N`. Phase 1
  replaces this with persisted last-list handles.

**Open items surfaced.**
- `cairn pack --json` and `--jsonl` now emit typed payloads keyed by citation.
  If Phase 1 wants a different machine format, change it deliberately rather
  than by accident.
- The XML pack output is now valid and readable, but it is still a hand-authored
  Phase 0 format. Phase 2 can revisit the final context-pack shape if daily use
  suggests a better default.

**Gate status.** Ready for Phase 1 start.
