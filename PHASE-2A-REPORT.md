# Phase 2a report

**Shipped.** Real `cairn export` that mirrors cards into a Phoenix vault as
`{YYYY-MM-DD}-{slug}.md` markdown files with YAML frontmatter
(`mymind_id`, `url`, `tags`, `captured_at`, `kind`) and content-addressed
attachments under `_media/{aa}/{bb}/{sha}.{ext}`. Media rows now carry a
real `card_id` resolved from the MyMind filename convention
(filename-stem equals `mymind_id`); orphan attachments surface as
warnings instead of FK-violating inserts. Migration 0002 cleans up
Phase 1 leftovers and `PRAGMA foreign_keys = ON` enforces the constraint
going forward. Fresh installs without an imported library are refused
cleanly so fixture cards never land in a user vault.

**Acceptance check (local).** End-to-end test coverage: `TestExportDryRunWritesNothing`, `TestExportRealWritesMarkdown`,
`TestExportSecondRunIsUnchanged`, `TestExportFreshInstallRefusesWithoutImport`
under `internal/commands/`. Live smoke:

```
cairn import ~/phoenix/Clippings/mymind/
cairn export --dry-run --to /tmp/cairn-vault
cairn export --to /tmp/cairn-vault
```

produces ~43 markdown files at the vault root, the single
`MDE0O3xIKzJh4Y.pdf` attachment under `_media/<fan>/<out>/...pdf`,
and the card markdown links resolve in Obsidian via the relative path.

**What's still Phase 0 fake.**

- `cairn find`: real TUI in Phase 2d.
- `cairn pack` and `cairn ask`: Phases 2b and 4.
- `cairn mcp *`: Phase 3.
- `cairn config` remains a defaults display.
- `render.Match` still card-level (chunk-level provenance is Phase 2c).
- `source.Filters` still lives in `internal/source`; no cycle has forced relocation yet.

**Open items.**

- Slug collisions across different cards on the same capture date are
  handled with `-2`, `-3`, ... suffixes, but two different cards with
  the same MyMind id (shouldn't happen) would overwrite in place. The
  importer's duplicate-id guard prevents that upstream.
- The writer reads existing vault files to recover their `mymind_id`
  frontmatter when deciding whether a filename is "ours". This couples
  `writer.go` to `markdown.go`'s frontmatter format. If the format
  changes, add a migration that rewrites existing vault files.
- `--limit` is not wired on `cairn export`; export always writes every
  card in the source. No user has asked for a partial export.

**Gate status.** Phase 2a to Phase 2b sign-off: if the Phoenix-mirrored
markdown is opened in Obsidian in at least three different contexts
within a week (drafting, Claude Code lookup, knowledge-graph linking),
continue to Phase 2b (`cairn pack`). Otherwise stop and reassess which
surface earns the next slot.
