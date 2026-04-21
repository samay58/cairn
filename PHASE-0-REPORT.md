# Phase 0 report

**Shipped.** Every command in §2.6 of the spec is implemented and produces hand-authored output: `import`, `status`, `search`, `find`, `get`, `open`, `pack`, `ask`, `export`, `config`, and the `mcp` parent with `start`, `install` (claude-code and claude-desktop), `audit`, and `permissions` subcommands. The fixture corpus has 25 cards covering all four kinds (article `a`, note `n`, quote `q`, image `i`), with one fixture each for the not-found import error, empty search, and unknown handle error states. All output is pinned by golden-file snapshot tests. SCREENPLAY.md provides a first-run walkthrough with verbatim golden outputs.

**Descoped from Phase 0 (intentional).**
- No real storage. `~/.cairn/cairn.db` is never written; data is served entirely from `internal/fixtures/cards.json`.
- No interactive TUI. `cairn find` prints a static frame. Real keyboard navigation is Phase 2.
- No real MCP server. `cairn mcp start` exits immediately with a stub. JSON-RPC server is Phase 3.
- No real MCP install. `cairn mcp install` previews the config merge but does not write it.
- No ANSI color output. `--no-color` flag parses but is a no-op; styling is Phase 1.
- No real browser invocation. `cairn open` prints the URL but does not call `xdg-open` or `open`.
- No RAG synthesis. `cairn ask` is a stub redirecting to `pack`.
- No real phoenix export. `cairn export` is a dry-run listing filenames only.
- Handle resolution is positional (`@N` maps to fixture index N). Persisted last-result list via SQLite is Phase 1.

**Open items surfaced.**
- The note card in pack output (`c_0018`) renders `source=""` because notes have no source URL. This is correct behavior; Phase 1 fixture imports will carry real source values populated from the import pipeline.
- The `export` golden counts "9 media files" from the import fixture but only lists "5 files" in the export dry-run. The discrepancy is intentional: export only mirrors image and video attachments, not all media.
- **Status `permissions` column deferred.** Spec §2.6 lists status as "Library size, last sync, MCP state, permissions". Phase 0 status shows the first three; the per-client permissions column arrives in Phase 1 alongside real config persistence.
- **Phase 1 will introduce a `Source` interface.** Command handlers currently call `fixtures.All()` directly. When Phase 1 adds `ImportSource`/`APISource` per spec §"Architecture", every command file will need a small refactor to take a `Source` parameter. Expected, not a surprise.

**Known Phase 0 cosmetic notes.**
- Long excerpt lines in goldens exceed 80 columns. This is expected for content fields that wrap naturally in a terminal. No golden lines exceed 341 chars and none are unwieldy in context.
- SCREENPLAY.md has the same long-line pattern in its fenced output blocks (copied verbatim from goldens). Prose lines stay well under 80 chars.

**Gate status.** Ready for Samay's review. Sign-off unlocks Phase 1.
