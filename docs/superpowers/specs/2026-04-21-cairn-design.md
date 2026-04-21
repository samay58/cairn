# Cairn design spec

**Date**: 2026-04-21
**Owner**: Samay
**Status**: Draft for approval

> Written via `/forge` from the Cairn project brief (Apr 21, 2026). All ten §3 open questions resolved; two sub-decisions (card-kind letter mapping, full-content cache opt-in mechanism) resolved alongside. Repo location confirmed. This spec is the single source of truth from here forward. When it conflicts with the brief, this spec wins.

---

## Purpose

A terminal-native bridge between MyMind and the AI and developer tools Samay already uses. The hero is "your AI can finally use what you saved, with citations and permission." Cairn makes a personal MyMind library queryable from the terminal and turns it into a first-class context source for Claude Code, Claude Desktop, and any other MCP client.

Three product surfaces, ordered by priority: search, context packs, MCP. Ingestion is infrastructure, not a feature.

## Non-goals

Cairn is not a MyMind clone, not a second home for cards, not a capture tool, not a dashboard, not a browser-replacement viewer. MyMind's own surfaces remain the capture layer. Cairn is about egress into work contexts.

## Binding constraints

MyMind has no public API today. They have stated a sanctioned API plus CLI is in development, no shipping timeline. The only sanctioned egress is the manual "Export my mind" button producing `cards.csv` and media files.

Cairn ships on the import path today and is designed so the future API landing is a config flag, not a rewrite. Session-cookie scraping against `access.mymind.com` is prohibited, including as an optional mode. The ethical posture, brittleness, and portfolio optics all argue against it.

## Locked decisions

| Decision | Value | Rationale |
|---|---|---|
| Name | `cairn` | Short, unclaimed, phonetically clean. |
| License | MIT, public GitHub | Portfolio-visible; simplifies the Homebrew tap. |
| Repo location | `~/cairn/` | Matches existing pattern for dev projects. |
| Phoenix vault write path | `~/phoenix/Clippings/MyMind/` plus `~/phoenix/Clippings/MyMind/_media/` | Uses the existing `Clippings/` convention; keeps external saved content out of active and reference layers. |
| Embedding default | Local MiniLM-class, ~60MB one-time download, lazy compute | Privacy-first. Configurable upgrade to Voyage or OpenAI later. |
| LLM for `ask` and `digest` | Anthropic default; prompt on first invocation if `ANTHROPIC_API_KEY` absent | Matches primary workflow. Configurable override via env or config. |
| MCP client autoconfig scope (v1) | Claude Code and Claude Desktop | Primary surfaces. Manual snippet emission for Cursor, Continue, Zed. |
| Sync cadence (v1) | On-demand import only; no daemon, no watcher | Matches the export-button reality. Daemon is a Phase 5 concern. |
| MyMind accounts | Single account; `--profile` flag reserved in parser but not wired | One account today. Future-proof without building the subsystem. |
| `cairn ask` scope | Phase 4, post-integrity-gate | Earns its slot by daily use. |
| Integrity gate commitment | Yes, hard stop at one week post-Phase 3 if unused in at least three distinct work contexts | Guardrail against the documented system-building pattern. |
| Card kind letter mapping | `a` article · `i` image · `q` quote · `n` note | Phase 1 parser may extend the enum; these four are the fixture set for Phase 0. |
| Full-content cache opt-in | Global config toggle: `storage.cache_full_content = true` | Simple, explicit, auditable. Privacy-first default. |

## Architecture

A single Go binary. SQLite via `modernc.org/sqlite` (pure Go, no CGo). FTS5 shadow table maintained by triggers. Content-addressed media in `~/.cairn/media/`. Chunks, not cards, are the retrieval unit. Embeddings arrive in Phase 2 as a separate lazy layer.

The source-of-truth abstraction is a `Source` interface with two implementations: `ImportSource` (reads a MyMind export folder) and, later, `APISource` (OAuth device flow against the future MyMind API). Every downstream component consumes `Source` and does not know which is providing data.

### Tech stack

| Layer | Choice |
|---|---|
| Language | Go |
| Storage | SQLite via `modernc.org/sqlite`, FTS5 |
| Embeddings | Local MiniLM (exact loader picked in Phase 2 after testing `hugot` and ONNX options) |
| MCP | `mark3labs/mcp-go` unless the official Anthropic Go SDK is current at build time; check at Phase 3 start |
| TUI | `charmbracelet/bubbletea` + `lipgloss` + `glamour` + `huh` + `fang` |
| Distribution | Homebrew tap plus direct binary plus `go-selfupdate` |

No runtime dependencies beyond this list without raising a question.

## Data model

```
cards          id, mymind_id, kind, title, url, body, captured_at, updated_at, source, deleted_at
card_meta      card_id, key, value
tags           card_id, tag
media          card_id, kind, path, sha256, mime
chunks         id, card_id, modality, text, start_offset, end_offset, checksum
embeddings     chunk_id, model, vector
fts(cards)     title, body, tags_flat           [FTS5 shadow]
sync_log       started_at, finished_at, delta_count, status
mcp_audit      ts, client, tool, params_hash, result_summary, decision
```

Chunking target: 200 to 600 tokens, semantic-unit boundaries (paragraph or heading). Deleted cards become tombstones with `deleted_at` set; hard-delete after 30 days.

## Command surface

```
cairn import <path>         Ingest a MyMind export folder
cairn status                Library size, last sync, MCP state, permissions
cairn search <query>        Hybrid retrieval, card-list output
cairn find                  Full-screen fuzzy TUI, @1..@N ephemeral handles
cairn get <card>            Render full card in terminal
cairn open <card>           Open in MyMind browser
cairn pack <query>          Cited context pack for an external AI
cairn ask <question>        RAG synthesis with citations       [Phase 4]
cairn mcp start             Local MCP server over stdio
cairn mcp install <client>  Write client config
cairn mcp audit             Human-readable tool-call log
cairn mcp permissions       Inspect and modify per-client scopes
cairn export                Mirror cards to Phoenix vault
cairn config                Edit config
```

Every data-returning command supports `--json`, `--jsonl`, `--plain`, `--limit N`, `--no-color`. After any search or list, cards receive ephemeral `@1..@N` handles valid until the next list. `cairn get @2` works without UUIDs. The CLI composes: `cairn search "x" --json | jq '.[].url'` must work on day one.

No destructive commands in v1. No `cairn delete`, no `cairn bulk-edit`.

## Output contract

Card list, never table. Format per result: `@N` handle, title, a single `type · source · date` line in tertiary color, a one-line "why shown" string derived from match signals, then an excerpt.

No emoji anywhere. Card-type single letters (`a`, `i`, `q`, `n`) in tertiary color only. One Unicode glyph permitted for chrome: `·` as separator, plus `›` as command-prompt marker inside the TUI. Sentence case only. Default width 80 columns; wider only opt-in via `--width`.

No-results states are designed, not apologetic. Errors name the failed operation and the last known good state. Progress is honest: sync shows the current card title; `ask` shows which cards it is reading. No opaque spinner lasting longer than 500ms. Spinner frames: `⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏` at 100ms. Progress bars are single-line braille blocks.

## Visual tokens

| Role | Light bg | Dark bg |
|---|---|---|
| Primary text | `#2C2A28` | `#FAF7F2` |
| Secondary | `#8B8680` | `#8B8680` |
| Tertiary | `#6B6965` | `#6B6965` |
| Accent | `#E55B3C` | `#E55B3C` |
| Success | `#6B8E4E` | `#6B8E4E` |
| Warn | `#C89B3C` | `#C89B3C` |
| Error | `#B44C38` | `#B44C38` |

Accent is reserved for sync-success, current-card indicator, and match-found. Never decorative.

## Retrieval

1. Parse query for filters: `type:`, `from:`, `since:`, `#tag`, `space:`.
2. FTS5 pass over title, body, tags.
3. Vector pass over chunk embeddings (Phase 2 onward).
4. Reciprocal-rank fusion, then rerank top 50 down to top 10.
5. Boost: exact title match, domain match, recent saves, user-added notes, highlights.
6. Attach a one-line "why shown" string.

Exact queries (quoted strings, URLs, code snippets) skip the vector stage to preserve precision.

## MCP surface

Five tools:

```
search_mind(query, limit?, filters?)         Snippet results with "why shown"
get_card(card_id, include_full_text?)        Card details; full text is gated
get_related(card_id, limit?)                 Cross-links
create_context_pack(query, max_tokens?)      Bundled evidence pack
save_to_mind(url_or_text, note?)             Optional; disabled by default
```

Every call writes to `mcp_audit`. `cairn mcp audit` renders the log. Default permissions for a newly installed client: `search_mind` and `get_related` allowed; `get_card` with `include_full_text=true` prompts once per session; `save_to_mind` disabled. Per-client overrides live in `~/.cairn/config.toml`.

Audit render format:

```
Today

  10:42  Claude searched "OAuth MCP authorization"
         Returned 5 snippets

  10:43  Claude requested full content for @2
         Allowed once by user

  10:49  Cursor searched "SQLite FTS5"
         Returned 8 snippets
```

`cairn mcp install claude-code` targets the local Claude Code MCP config. `cairn mcp install claude-desktop` targets `~/Library/Application Support/Claude/claude_desktop_config.json`. Both must merge without clobbering other servers. Any other client produces a copy-pasteable JSON snippet via `cairn mcp install manual`.

## Permission model

Defaults err toward privacy:

- No full-content cache unless `storage.cache_full_content = true` in config.
- No original asset cache unless explicitly opted in.
- MCP sees search snippets only by default; full content requires explicit grant (once, always, deny).
- All MCP invocations logged.

## Phoenix bridge

`cairn export` writes each card as a markdown file to `~/phoenix/Clippings/MyMind/{YYYY-MM-DD}-{slug}.md`. Frontmatter carries `mymind_id`, `url`, `tags[]`, `captured_at`, `kind`. Body holds article extraction or note text or OCR. Media lands at `~/phoenix/Clippings/MyMind/_media/{sha}.{ext}` with relative links, so Obsidian wiki-link resolution works without additional config.

Export is on-demand only in v1. No watcher, no daemon. A Phase 5 concern when the API lands.

## Storage layout

```
~/cairn/               source
~/.cairn/
  cairn.db             SQLite
  config.toml          config (max 15 keys in v1)
  media/               content-addressed opt-in assets
  logs/
    audit.log          mcp_audit mirror (text)
    sync.log
```

## Phased execution

### Phase 0. Design prototype

Fake CLI with static JSON fixtures. No real storage, no real retrieval.

Deliverables:

- Go program responding to every command in the command surface with hand-authored output.
- Golden-file snapshot tests for every output variant (terminal output is product UI).
- 25 fixture cards covering all four `kind` values in the real storage shape.
- `SCREENPLAY.md` walking a first-run session: `import` → `status` → `search` → `find` → `pack` → `mcp install claude-code` → `mcp audit`.

Acceptance: Samay reads the screenplay, runs each command against the fake CLI, signs off on every output. Zero snapshot failures.

### Phase 1. Import and local search

Real MyMind export parser. Real SQLite schema with migrations. FTS5 shadow table. `cairn import`, `cairn status`, `cairn search`, `cairn get`, `cairn open`.

Acceptance: Samay runs `cairn import` on his real export, then `cairn search "<thing he remembers>"`, and the card surfaces in the top 5.

### Phase 2. TUI, packs, Phoenix mirror

`cairn find` (bubbletea TUI with preview pane, filter tabs, keybinds), `cairn pack` with `--for {claude, chatgpt, cursor, markdown, json}`, `cairn export`, local embedding pipeline, reciprocal-rank fusion, "why shown" provenance.

Acceptance: Samay uses `cairn find` for a full real workday. Phoenix mirror appears in Obsidian and is visible to Claude Code. `cairn pack "<topic>"` produces a bundle he is willing to paste into a real conversation.

### Phase 3. MCP and audit. Integrity gate.

Five MCP tools over stdio, per-client permissions, audit log, prompt-injection test suite, `cairn mcp install claude-code` and `cairn mcp install claude-desktop` with safe JSON merging.

Acceptance gate: ship Phase 3. Use Cairn for one real week. If during that week Cairn is not reached for unprompted in at least three distinct work contexts (drafting, coding, research), stop here. Phases 4 and 5 do not ship. Hard stop.

### Phase 4. Ask, digest, related

Gated on Phase 3 passing.

`cairn ask` streams LLM synthesis with strict citations. `cairn digest --since 7d` clusters recent saves, names themes, writes a markdown brief to Phoenix, stays silent when there is nothing substantive. `cairn related @N` surfaces high-similarity cards.

Acceptance: `cairn ask` used on a real question with the output consumed downstream. Digest runs two weeks without becoming noise.

### Phase 5. Official API integration

Gated on MyMind shipping a public API. Until then, do not start.

OAuth device flow (RFC 8628) plus PKCE (RFC 7636). Delta sync against `/cards/delta`. Scope escalation UX for granular reads. Config flag `source = "api"` versus `source = "import"`; every downstream consumer is source-agnostic by design.

## Anti-patterns

Do not implement JWT-cookie session scraping against `access.mymind.com`, not even as an opt-in advanced mode. Do not build a TUI card browser that competes with MyMind's main surface; the TUI exists only for fast recall during work. Do not add features that serve system completeness rather than daily use. Do not cheerlead in output. Do not grow the config past 15 keys in v1. Do not gate v1 on the MyMind API. Do not over-engineer the embedding pipeline (no pgvector, no Qdrant, no LanceDB for v1). Do not ship `cairn here` or developer-context features until after Phase 3.

## Quality gates (apply every phase)

- First-install to first-successful-search under two minutes on an average library.
- Search works offline.
- Every human command has `--json`.
- No command reveals private content accidentally.
- Every MCP call is auditable.
- Errors are calm and actionable.
- No destructive command in v1.
- CLI composes under `jq`.
- No phase ships without snapshot tests for new output surfaces.

## Per-phase reporting

Each phase ends with a `PHASE-N-REPORT.md` covering what shipped, what was descoped, and what open questions surfaced. The integrity gate report is the most important one; Samay writes it himself based on actual use, not a self-graded checklist.

## Open items (deferred, not blockers)

- Homebrew tap GitHub org path. `samay/tap/cairn` in the brief. Resolve at Phase 3 when tap is set up (likely `samay58/tap/cairn` since the GitHub handle is `samay58`).
- MCP Go SDK final pick. `mark3labs/mcp-go` is the default; confirm against the official Anthropic Go SDK at Phase 3 start.
- Local embedding loader. Evaluate `hugot`, direct ONNX, and alternatives in Phase 2 when the work lands.
- MyMind export fixture diversity. Phase 0 fixtures cover all four kinds; Phase 1 parser may discover more and extend the enum.

## How to work on this

Raise ambiguity rather than guess. Prefer smaller, working increments over large plans. Show output early and often. Snapshot tests on every command are not optional; terminal output is UI. Do not add commands, flags, or config keys not listed here without raising them first. At each phase boundary produce a `PHASE-N-REPORT.md`. The integrity gate at Phase 3 is real; if it fails, stop.

## First action

Produce Phase 0 in a single session: fake CLI, 25 fixtures, golden-file snapshot tests, `SCREENPLAY.md`. Hand back for sign-off. No further decisions are blocking Phase 0.
