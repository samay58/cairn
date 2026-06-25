# Cairn first-run walkthrough

Cairn is currently Phase 2a. Import, status, search, get, open, and export
are real. Find, pack, ask, and MCP are designed surfaces that are not
implemented for imported libraries yet.

Every output block below was captured against `testdata/mymind_sample_export`
with a fresh `CAIRN_HOME`.

---

## Import

You have downloaded a MyMind export. Point Cairn at the export folder.

```console
$ cairn import testdata/mymind_sample_export
```

```
Reading export from testdata/mymind_sample_export
Rows: 4 read, 4 valid, 0 skipped.
Cards: 4 inserted, 0 updated, 0 unchanged, 0 tombstoned.
Media: 1 files. Chunks: 3.

Database at
  ~/.cairn/cairn.db
Run `cairn search "<query>"` or `cairn find`.
```

The row count is the CSV count. The card counts show what changed in the
database on this run. A no-op re-import should show valid rows and unchanged
cards, not pretend nothing was parsed.

---

## Status

`cairn status` is the agent-readable control panel: what was imported, what
was exported, and which commands are production surfaces.

```console
$ cairn status
```

```
cairn 0.1.0-phase2a

library   4 cards · last import 2026-06-25T18:08:15Z · 0 pending
storage   104.0 KB · media cache off
          ~/.cairn/cairn.db
import    4 read · 4 valid · 0 skipped · 0 warnings
changes   4 inserted · 0 updated · 0 unchanged · 0 tombstoned
mcp       not installed
clients   none
commands  real: import, status, search, get, open, export
          not implemented: find, pack, ask, mcp

Phase 2a. Import, search, get, open, and export are real.
```

Use `cairn status --json` when another tool needs to decide the next action.

---

## Search

Search by whatever you remember from the saved card.

```console
$ cairn search "craft"
```

```
@1  On craft
    q · Martha Beck · 2026-03-18
    Matched on title.
    The way you do anything is the way you do everything.
```

The `@N` handle refers to the most recent list. A later `get @1` or
`open @1` resolves through SQLite, not fixture position.

Filters work inline:

```console
$ cairn search "type:note"
```

```
@1  Cairn naming
    n · 2026-03-22
    Recent.
    Short. Unclaimed. Fits the category.
```

---

## Get

After the search above, `@1` refers to the craft quote.

```console
$ cairn get @1
```

```
@1  On craft
q · Martha Beck · 2026-03-18

The way you do anything is the way you do everything.
```

---

## Open

`cairn open @1` launches the URL in the default OS browser. Tests and dry
runs can set `CAIRN_DRY_OPEN=1`.

```console
$ CAIRN_DRY_OPEN=1 cairn open @1
```

```
Would open: https://access.mymind.com/cards/mm_2
```

---

## Export

`cairn export` mirrors the imported library into Phoenix markdown. The default
target is `~/phoenix/04-knowledge-base/research-archive/mymind-cards`.

```console
$ cairn export
```

```
Wrote 4 cards to ~/phoenix/04-knowledge-base/research-archive/mymind-cards
  media: 1 written, 0 skipped
  mirror: 4 cards, 1 media files
Next: cd ~/phoenix && qmd update
```

The raw MyMind export should stay in `~/phoenix/Clippings/mymind`. Cairn
refuses to export into the last import path, including case-only collisions
such as `MyMind` vs `mymind` on macOS.

---

## Designed But Not Implemented

These commands are intentionally honest until their phases earn implementation:

- `cairn find`: no real TUI for imported libraries yet.
- `cairn pack`: no real context pack for imported libraries yet.
- `cairn ask`: Phase 4, after the Phase 3 integrity gate.
- `cairn mcp *`: Phase 3.
- `cairn config`: defaults preview only.

For now, use `search`, `get`, and the Phoenix mirror as the production path.
