# Phase 1 report

**Shipped.** Real MyMind export pipeline; SQLite store with FTS5 shadow
index; `Source` interface with `FixtureSource` and `SQLiteSource`
implementations; five commands running on real data: `cairn import`,
`cairn status`, `cairn search`, `cairn get`, `cairn open`. Handle
persistence via the `handles` table so `@N` survives across
commands. Tombstone plus 30-day hard-delete on import.

**Acceptance check (live).** Ran `cairn import` against a real 43-card
MyMind export at `~/phoenix/Clippings/mymind/`. 43 inserted, 0
warnings, DB at 156 KB. `cairn search "ai"` returns real titles.
`cairn search "type:note"` correctly filters to note-kind cards.
`cairn get @N` resolves handles saved by the most recent list.

**Schema deltas uncovered by the live import.** The hand-crafted
sample export guessed wrong on four counts, all fixed in
`feat(importer): handle real MyMind export schema`:

- `cards.csv` ships with a UTF-8 BOM.
- Column names are `content`, `note`, `created` (not `body`,
  `captured_at`). `note` is the user's annotation and merges into the
  card body when surfacing search or get results.
- Type values are capitalized and include `Article`, `Document`,
  `Embed`, `Note`, `WebPage`, `YouTubeVideo`. All map to the four
  Phase 1 kinds (`a/i/q/n`) for display consistency; Phase 2 may
  introduce finer-grained kinds.
- Attachments live at the export root, not in a `media/` subdir. The
  scanner now walks the export root when no subdir exists.

Documented fully in `docs/IMPORT_FORMAT.md`.

**What's still Phase 0 fake.**
- `cairn find` (real TUI in Phase 2)
- `cairn pack` and `cairn export` (Phase 2)
- `cairn mcp *` (Phase 3)
- `cairn ask` (Phase 4)
- `cairn config` remains a defaults display
- Embeddings and Phoenix mirror (Phase 2)

**Open items.**
- Tag hydration on `SQLiteSource.All()` is O(N) queries (one per
  card). For a 43-card library this is imperceptible. Before the
  corpus grows past ~5k cards, move to a single `WHERE card_id IN (...)` bulk fetch.
- Media rows currently land with an empty `card_id` because the export
  does not expose per-card attachment linkage. Phase 2 will establish
  the mapping when real card-to-media joining arrives.
- `source.Open` lives under `internal/commands` rather than
  `internal/source` to break an import cycle caused by
  `sqlite.SQLiteSource.Search` referencing `source.Filters`. The
  cleanest resolution is extracting `Filters` to a neutral package in
  Phase 2 (for example `internal/query`).
- Card kinds currently collapse `WebPage`, `Document`, `Embed`, and
  `YouTubeVideo` all to `article`. Finer display is a Phase 2 UI
  concern.

**Gate status.** Phase 1 to Phase 2 sign-off: use cairn for one real
week against the live library. If on day seven `cairn search` is
reached for in at least three distinct work contexts (drafting,
coding, research), Phase 2 ships. If not, stop here per the spec's
integrity gate, even though `find`, `pack`, `ask`, and `mcp` are
visibly incomplete. The integrity gate is real.
