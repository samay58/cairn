# Cairn Phase 1 Implementation Plan

**Goal:** Replace Phase 0's hand-authored output with a real import-and-search pipeline. Parse a MyMind export into SQLite, maintain an FTS5 shadow index, make `cairn import`, `cairn status`, `cairn search`, `cairn get`, and `cairn open` work on real data. Leave `find`, `pack`, `ask`, `export`, `config`, `mcp *` on the Phase 0 fakes for now (they land in Phases 2 through 3).

**Architecture:** Introduce a `Source` interface that abstracts over the existing Phase 0 fixtures and a new SQLite-backed store. Every Phase 0 command handler keeps working by calling through `Source`; the five Phase 1 commands switch to the SQLite implementation when a database exists, fall back to fixtures otherwise. The import pipeline is a discrete package that produces rows to upsert; the SQLite package owns schema, migrations, FTS5 triggers, and handle persistence. Chunking runs at import time at paragraph granularity (200 to 600 tokens per chunk) so Phase 2 can layer embeddings without re-ingesting.

**Tech Stack:** Go 1.26, `modernc.org/sqlite` (pure-Go SQLite, no CGo), stdlib `encoding/csv`, stdlib `crypto/sha256`, existing `github.com/spf13/cobra` + internal render package. No new third-party deps beyond `modernc.org/sqlite`.

**Spec reference:** `docs/design/cairn-design.md` sections "Data model", "Retrieval", "Permission model", "Phase 1. Import and local search". Phase 0 plan is `docs/plans/cairn-phase-0.md`; Phase 0 report is `PHASE-0-REPORT.md`.

---

## Scope boundaries

**In Phase 1:**

- `cairn import <path>` — real MyMind export parser, real writes to `~/.cairn/cairn.db`, tombstone + hard-delete logic.
- `cairn status` — reports live counts from the database and the last import time.
- `cairn search <query>` — FTS5 retrieval over title, body, tags; filter parsing (`type:`, `from:`, `since:`, `#tag`); RRF-ready scoring (vector stage stubbed).
- `cairn get @N` — resolves the handle from the persisted last-list table, renders the full card.
- `cairn open @N` — resolves the handle, invokes the OS default browser.
- `Source` interface abstracts fixtures vs SQLite for every command.

**Deferred to later phases (keep the Phase 0 fakes):**

- `cairn find` — real bubbletea TUI in Phase 2.
- `cairn pack` — Phase 2 (real retrieval + profiles).
- `cairn export` — Phase 2 (write to `~/phoenix/Clippings/MyMind/`).
- `cairn ask` — Phase 4.
- `cairn mcp *` — Phase 3.
- `cairn config` — remains a defaults display until Phase 1 needs configuration (e.g., `storage.cache_full_content` toggle).
- Embeddings — Phase 2.
- Color output — not Phase 1's problem; plain text continues.

**Explicit non-goals:**

- No real MCP server or audit.
- No real Phoenix mirror.
- No daemon / watcher.
- No profile system (`--profile` stays reserved, unwired).

---

## Architecture notes

### The `Source` interface

Centralizes every way a command reads cards. Phase 0 commands currently import `internal/fixtures` directly; Phase 1 routes everything through `Source` so we can swap implementations.

```go
// internal/source/source.go
package source

type Source interface {
    Count() int
    All() []cards.Card
    ByHandle(n int) (cards.Card, error)
    Search(q string, filters Filters, limit int) []render.Match
    LastImport() (time.Time, bool)
    LastListSave(matches []render.Match) error   // persists @N handles after a list operation
}

type Filters struct {
    Kind    string    // "article" | "image" | "quote" | "note" | ""
    From    string    // domain match against card.Source
    Since   time.Time // zero value = unconstrained
    Tag     string    // exact-match tag filter
}
```

`FixtureSource` (new, wraps the existing fixtures package) implements this for Phase 0 compatibility. `SQLiteSource` (new) implements it using `~/.cairn/cairn.db`.

### Database selection logic

Commands ask `source.Open()` for a `Source`. The function:

1. Checks if `~/.cairn/cairn.db` exists and is readable.
2. If yes, returns `SQLiteSource` backed by that file.
3. If no, returns `FixtureSource`.

`cairn import` is the only command that creates `~/.cairn/cairn.db` — everything else is read-only from it. After a successful import, subsequent commands see the SQLite source.

### Schema

Per the spec, initial migration 0001 creates:

- `cards` (id, mymind_id, kind, title, url, body, captured_at, updated_at, source, deleted_at)
- `card_meta` (card_id, key, value)
- `tags` (card_id, tag)
- `media` (card_id, kind, path, sha256, mime)
- `chunks` (id, card_id, modality, text, start_offset, end_offset, checksum)
- `sync_log` (id, started_at, finished_at, delta_count, status)
- `handles` (position, card_id, created_at) — single-row-set for @N persistence, wiped and re-inserted per list.
- `cards_fts` (FTS5 virtual table over title, body, tags_flat).
- Triggers: `cards_ai`, `cards_au`, `cards_ad` keep `cards_fts` in sync.

Migration 0002 etc. reserved for future phases. We use a simple handmade migration runner — schema_version table + numbered up migrations, no down migrations.

### Chunking strategy

At import, for each card with body content, split on blank lines to paragraphs. Merge adjacent short paragraphs until a chunk hits ~200 tokens; split long paragraphs on sentence boundaries to stay under ~600 tokens. Token count approximated as `len(strings.Fields(text))`. Chunks store byte offsets into the original body so Phase 4's citation can point back.

### MyMind export format (assumed)

MyMind's export produces `cards.csv` plus a media folder. Exact schema is not publicly documented, so Phase 1 assumes a pragmatic shape and writes a defensive parser. Columns likely present (supply best-effort mapping; ignore unknowns):

- `id` (MyMind card id)
- `type` or `kind` (article, image, quote, note, link, pdf)
- `title`
- `url`
- `text` or `body` or `content`
- `excerpt` or `description`
- `source` (domain or publisher)
- `tags` (comma- or semicolon-separated)
- `created_at` / `captured_at` / `date`
- `media` or `attachment` (filename pointing into the media folder)

The parser treats column names case-insensitively, accepts synonyms (`body` ≡ `content` ≡ `text`), and never errors on unknown columns. Malformed rows produce a warning to stderr and are skipped. The first real import updates `IMPORT_FORMAT.md` with observed reality if it diverges from these assumptions.

### Handle persistence

Phase 0 used positional fixture indexing — `@2` always meant `fixtures.All()[1]`. Phase 1 replaces this with a `handles` table. Every command that prints a list (`search`, and eventually `find`) calls `Source.LastListSave(matches)` which wipes the table and inserts `(position, card_id)` rows. `ByHandle(n)` joins against the table and pulls the card. No TTL — always use the most recent save.

### FTS5 + retrieval

Phase 1 implements FTS5-only retrieval. The spec describes hybrid FTS5 + vector + RRF for Phase 2+. To keep Phase 2 cheap, the retrieval function signature already takes shape of a multi-source pipeline:

```go
// internal/retrieval/retrieval.go
type Stage interface {
    Score(query string, limit int) []Hit
}

type Hit struct { CardID string; Score float64; WhyShown string }

func Combine(stages []Stage, ...) []Hit { /* Reciprocal-Rank Fusion */ }
```

For Phase 1, only the FTS5 `Stage` exists. Phase 2 plugs in a vector `Stage` and `Combine` starts doing real RRF.

---

## File structure

```
internal/
  source/
    source.go              Source interface + Filters struct
    source_test.go         contract tests that any Source implementation must pass
    fixture.go             FixtureSource (wraps internal/fixtures)
    fixture_test.go
    open.go                source.Open() — picks SQLite if DB exists, falls back to fixtures

  storage/sqlite/
    sqlite.go              SQLiteSource (implements source.Source)
    sqlite_test.go
    schema.go              embed migrations
    schema/
      0001_init.sql        cards, tags, card_meta, media, chunks, sync_log, handles, FTS5
    migrate.go             bare-metal migration runner (schema_version table)
    migrate_test.go
    search.go              FTS5 query + filter parsing
    search_test.go
    handles.go             handles table read/write
    handles_test.go

  importer/
    importer.go            orchestrator: read export dir, parse rows, transform to cards
    importer_test.go
    csv.go                 csv parser: permissive column mapping
    csv_test.go
    media.go               media folder scan, sha256, mime detection
    media_test.go
    chunk.go               paragraph chunking
    chunk_test.go

  retrieval/
    retrieval.go           Stage interface + Combine (RRF, limit 50→10)
    retrieval_test.go

  commands/
    import.go              real import wiring (replaces Phase 0 fake)
    status.go              real status (reads sync_log)
    search.go              FTS5 search via Source
    get.go                 SQLite-backed @handle resolution
    open.go                OS-native browser invocation
    (find/pack/ask/export/config/mcp/* unchanged; still Phase 0 fakes)
    rootflags.go           (new) small helper to centralize output-format flag parsing

testdata/
  mymind_sample_export/    a tiny hand-crafted sample export used by importer tests
    cards.csv
    media/
      sample.png

docs/
  IMPORT_FORMAT.md         the assumed MyMind export schema, updated after first real import
```

Existing `internal/fixtures` stays; `FixtureSource` wraps it. Existing Phase 0 commands keep their goldens — Phase 1 commands get new goldens that exercise real SQLite paths via `testdata/mymind_sample_export/`.

---

## Conventions

- **TDD where the behavior is testable.** Schema tests, CSV parsing, FTS5 query construction, chunking, RRF — all test-first. Integration tests use an in-memory SQLite (`:memory:` or a temp file under `t.TempDir()`).
- **Golden files continue for command output.** Phase 1 command tests import a sample export into a temp DB, then snapshot the output against a golden. This makes Phase 1 commands' goldens reproducible in CI.
- **Commits per task.** No WIP commits. Use `git -c commit.gpgsign=false commit` everywhere.
- **No em-dashes. No emoji. Sentence case.** Phase 0 kill list stays in force.
- **Error messages: name the failed operation and the last successful state.** Spec §"Output contract".

---

## Task list

### Task 1: Add `modernc.org/sqlite` dependency

**Files:** modify `go.mod`, `go.sum`.

- [ ] **Step 1: Add the dep.**

Run:
```bash
cd ~/cairn && go get modernc.org/sqlite@latest
```

- [ ] **Step 2: Verify it compiles.**

Run:
```bash
cd ~/cairn && go build ./...
```
Expected: clean build.

- [ ] **Step 3: Write a one-shot sanity test `internal/storage/sqlite/sanity_test.go`.**

```go
package sqlite

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestOpenInMemory(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow("SELECT 1").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("got %d, want 1", n)
	}
}
```

- [ ] **Step 4: Run it.**

Run: `go test ./internal/storage/sqlite/...`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add go.mod go.sum internal/storage/sqlite/sanity_test.go && \
  git -c commit.gpgsign=false commit -m "chore(sqlite): add modernc.org/sqlite dependency and sanity test"
```

---

### Task 2: Migration runner + schema 0001

**Files:**
- Create: `internal/storage/sqlite/schema/0001_init.sql`
- Create: `internal/storage/sqlite/schema.go` (embeds the SQL files)
- Create: `internal/storage/sqlite/migrate.go`
- Create: `internal/storage/sqlite/migrate_test.go`

- [ ] **Step 1: Author `schema/0001_init.sql`.**

```sql
-- migration 0001: initial schema
CREATE TABLE cards (
    id           TEXT PRIMARY KEY,
    mymind_id    TEXT NOT NULL UNIQUE,
    kind         TEXT NOT NULL,
    title        TEXT NOT NULL,
    url          TEXT,
    body         TEXT,
    excerpt      TEXT,
    source       TEXT,
    captured_at  TIMESTAMP NOT NULL,
    updated_at   TIMESTAMP NOT NULL,
    deleted_at   TIMESTAMP
);
CREATE INDEX cards_captured_at_idx ON cards(captured_at);
CREATE INDEX cards_deleted_at_idx ON cards(deleted_at);

CREATE TABLE card_meta (
    card_id  TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    key      TEXT NOT NULL,
    value    TEXT,
    PRIMARY KEY (card_id, key)
);

CREATE TABLE tags (
    card_id  TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    tag      TEXT NOT NULL,
    PRIMARY KEY (card_id, tag)
);
CREATE INDEX tags_tag_idx ON tags(tag);

CREATE TABLE media (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id  TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    kind     TEXT NOT NULL,
    path     TEXT NOT NULL,
    sha256   TEXT NOT NULL,
    mime     TEXT
);
CREATE INDEX media_card_id_idx ON media(card_id);

CREATE TABLE chunks (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id       TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    modality      TEXT NOT NULL,
    text          TEXT NOT NULL,
    start_offset  INTEGER NOT NULL,
    end_offset    INTEGER NOT NULL,
    checksum      TEXT NOT NULL
);
CREATE INDEX chunks_card_id_idx ON chunks(card_id);

CREATE TABLE sync_log (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at    TIMESTAMP NOT NULL,
    finished_at   TIMESTAMP,
    delta_count   INTEGER NOT NULL DEFAULT 0,
    status        TEXT NOT NULL
);

CREATE TABLE handles (
    position    INTEGER PRIMARY KEY,
    card_id     TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    created_at  TIMESTAMP NOT NULL
);

CREATE VIRTUAL TABLE cards_fts USING fts5(title, body, tags_flat, content='');

CREATE TRIGGER cards_ai AFTER INSERT ON cards BEGIN
    INSERT INTO cards_fts(rowid, title, body, tags_flat)
    VALUES (new.rowid, new.title, coalesce(new.body, ''), '');
END;
CREATE TRIGGER cards_ad AFTER DELETE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, body, tags_flat)
    VALUES ('delete', old.rowid, old.title, coalesce(old.body, ''), '');
END;
CREATE TRIGGER cards_au AFTER UPDATE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, body, tags_flat)
    VALUES ('delete', old.rowid, old.title, coalesce(old.body, ''), '');
    INSERT INTO cards_fts(rowid, title, body, tags_flat)
    VALUES (new.rowid, new.title, coalesce(new.body, ''), '');
END;
```

Note: `tags_flat` starts empty; the importer fills it with a space-separated tag string after inserting all tag rows. Triggers do not auto-compute tags_flat from the `tags` table because FTS5 triggers on parent table changes only. The importer is responsible for `UPDATE cards_fts SET tags_flat = ? WHERE rowid = ?` after inserting tags.

- [ ] **Step 2: Write `schema.go` to embed migrations.**

```go
package sqlite

import "embed"

//go:embed schema/*.sql
var schemaFS embed.FS
```

- [ ] **Step 3: Write the failing migrate test `migrate_test.go`.**

```go
package sqlite

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateAppliesSchema0001(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	// schema_version tracks applied migrations.
	var version int
	if err := db.QueryRow("SELECT max(version) FROM schema_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Errorf("schema_version = %d, want 1", version)
	}

	// Every expected table is present.
	for _, name := range []string{"cards", "card_meta", "tags", "media", "chunks", "sync_log", "handles", "cards_fts"} {
		var got string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE name=?", name).Scan(&got)
		if err != nil {
			t.Errorf("table %q missing: %v", name, err)
		}
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate should succeed: %v", err)
	}
}
```

- [ ] **Step 4: Run, verify FAIL.**

Run: `go test ./internal/storage/sqlite/... -run TestMigrate`
Expected: FAIL (Migrate not defined).

- [ ] **Step 5: Implement `migrate.go`.**

```go
package sqlite

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

func Migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	applied := map[int]bool{}
	rows, err := db.Query("SELECT version FROM schema_version")
	if err != nil {
		return fmt.Errorf("read schema_version: %w", err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()

	entries, err := fs.ReadDir(schemaFS, "schema")
	if err != nil {
		return fmt.Errorf("list schema: %w", err)
	}
	type mig struct {
		version int
		name    string
	}
	var migs []mig
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		prefix, _, _ := strings.Cut(e.Name(), "_")
		v, err := strconv.Atoi(prefix)
		if err != nil {
			return fmt.Errorf("migration filename %q must start with integer: %w", e.Name(), err)
		}
		migs = append(migs, mig{version: v, name: e.Name()})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })

	for _, m := range migs {
		if applied[m.version] {
			continue
		}
		body, err := fs.ReadFile(schemaFS, "schema/"+m.name)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.name, err)
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply %s: %w", m.name, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_version(version) VALUES (?)", m.version); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 6: Run, verify PASS.**

Run: `go test ./internal/storage/sqlite/... -run TestMigrate`
Expected: PASS on both tests.

- [ ] **Step 7: Commit.**

```bash
git add internal/storage/sqlite/ && \
  git -c commit.gpgsign=false commit -m "feat(sqlite): migration runner and schema 0001 with FTS5"
```

---

### Task 3: `Source` interface and `FixtureSource`

**Files:**
- Create: `internal/source/source.go`
- Create: `internal/source/fixture.go`
- Create: `internal/source/fixture_test.go`

- [ ] **Step 1: Define `source.go` — the interface and `Filters` struct.**

```go
package source

import (
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/render"
)

type Filters struct {
	Kind  string
	From  string
	Since time.Time
	Tag   string
}

type Source interface {
	Count() int
	All() []cards.Card
	ByHandle(n int) (cards.Card, error)
	Search(query string, filters Filters, limit int) []render.Match
	LastImport() (time.Time, bool)
	LastListSave(matches []render.Match) error
}
```

- [ ] **Step 2: Write the failing test `fixture_test.go`.**

```go
package source

import (
	"testing"

	"github.com/samay58/cairn/internal/cards"
)

func TestFixtureSourceCount(t *testing.T) {
	s := NewFixtureSource()
	if got := s.Count(); got != 25 {
		t.Errorf("Count() = %d, want 25", got)
	}
}

func TestFixtureSourceByHandle(t *testing.T) {
	s := NewFixtureSource()
	c, err := s.ByHandle(2)
	if err != nil {
		t.Fatal(err)
	}
	if c.Kind != cards.KindQuote {
		t.Errorf("ByHandle(2).Kind = %q, want quote", c.Kind)
	}
	if _, err := s.ByHandle(99); err == nil {
		t.Error("ByHandle(99) should error")
	}
}

func TestFixtureSourceSearchExact(t *testing.T) {
	s := NewFixtureSource()
	matches := s.Search("oauth", Filters{}, 0)
	if len(matches) == 0 {
		t.Fatal("expected at least one match for 'oauth'")
	}
}

func TestFixtureSourceLastListSaveIsNoop(t *testing.T) {
	s := NewFixtureSource()
	if err := s.LastListSave(nil); err != nil {
		t.Errorf("LastListSave on fixture source should no-op, got %v", err)
	}
}
```

- [ ] **Step 3: Run, verify FAIL.**

Run: `go test ./internal/source/...`
Expected: FAIL.

- [ ] **Step 4: Implement `fixture.go`.**

```go
package source

import (
	"strings"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/fixtures"
	"github.com/samay58/cairn/internal/render"
)

type FixtureSource struct{}

func NewFixtureSource() *FixtureSource { return &FixtureSource{} }

func (f *FixtureSource) Count() int { return len(fixtures.All()) }

func (f *FixtureSource) All() []cards.Card { return fixtures.All() }

func (f *FixtureSource) ByHandle(n int) (cards.Card, error) {
	return fixtures.ByHandle(n)
}

func (f *FixtureSource) Search(query string, filters Filters, limit int) []render.Match {
	q := strings.ToLower(strings.TrimSpace(query))
	var matches []render.Match
	for _, c := range fixtures.All() {
		if !matchesFixture(c, q, filters) {
			continue
		}
		matches = append(matches, render.Match{
			Card:     c,
			WhyShown: whyShownFixture(c, q),
		})
	}
	if limit > 0 && limit < len(matches) {
		matches = matches[:limit]
	}
	return matches
}

func matchesFixture(c cards.Card, q string, f Filters) bool {
	if f.Kind != "" && string(c.Kind) != f.Kind {
		return false
	}
	if f.From != "" && !strings.Contains(strings.ToLower(c.Source), strings.ToLower(f.From)) {
		return false
	}
	if !f.Since.IsZero() && c.CapturedAt.Before(f.Since) {
		return false
	}
	if f.Tag != "" {
		found := false
		for _, t := range c.Tags {
			if strings.EqualFold(t, f.Tag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if q == "" {
		return true
	}
	hay := strings.ToLower(c.Title + " " + c.Body + " " + c.Excerpt + " " + strings.Join(c.Tags, " "))
	return strings.Contains(hay, q)
}

func whyShownFixture(c cards.Card, q string) string {
	if q == "" {
		return "recent"
	}
	if strings.Contains(strings.ToLower(c.Title), q) {
		return "matched on title"
	}
	for _, t := range c.Tags {
		if strings.EqualFold(t, q) {
			return "matched on tag " + t
		}
	}
	return "matched on body"
}

func (f *FixtureSource) LastImport() (time.Time, bool) { return time.Time{}, false }

func (f *FixtureSource) LastListSave(_ []render.Match) error { return nil }
```

- [ ] **Step 5: Run, verify PASS.**

Run: `go test ./internal/source/...`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/source/ && \
  git -c commit.gpgsign=false commit -m "feat(source): Source interface with FixtureSource implementation"
```

---

### Task 4: Wire existing commands through `Source`

**Files:** Modify every command handler that reads cards.

**Why now:** Before we build the SQLite backend, route Phase 0 commands through the interface so the Phase 1 SQLite source drops in without further command edits. No behavior change in this task — all goldens must still pass.

- [ ] **Step 1: Add a context-carrier for `Source`.**

Create `internal/commands/context.go`:

```go
package commands

import "github.com/samay58/cairn/internal/source"

// newSource returns the Source a command should use. Overridable in tests via
// the package-level variable below.
var newSource = func() source.Source {
	return source.NewFixtureSource()
}
```

- [ ] **Step 2: Update `search.go` to use `newSource()`.**

Replace the body of `fakeSearch` and the surrounding calls. New `search.go` (keep imports, change the RunE and remove `fakeSearch`):

```go
func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Hybrid retrieval, card-list output",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			limit, _ := cmd.Flags().GetInt("limit")

			src := newSource()
			matches := src.Search(query, source.Filters{}, limit)
			if err := src.LastListSave(matches); err != nil {
				return err
			}

			asJSON, _ := cmd.Flags().GetBool("json")
			asJSONL, _ := cmd.Flags().GetBool("jsonl")

			out := cmd.OutOrStdout()
			switch {
			case asJSON:
				fmt.Fprint(out, render.CardListJSON(matches))
			case asJSONL:
				fmt.Fprint(out, render.CardListJSONL(matches))
			default:
				if len(matches) == 0 {
					return writeNoResults(out, query)
				}
				fmt.Fprint(out, render.CardList(matches))
			}
			return nil
		},
	}
}
```

Import adjustment: add `"github.com/samay58/cairn/internal/source"`; remove `"github.com/samay58/cairn/internal/fixtures"` since `fakeSearch` is gone.

Remove the `fakeSearch` function from the file entirely. `writeNoResults` stays.

Note: `FixtureSource.Search` is a substring match, which is coarser than Phase 0's hand-picked `fakeSearch` map. The existing `search_oauth.txt`, `search_oauth.json`, `search_oauth.jsonl`, and `search_oauth_limit2.txt` goldens were authored from hand-picked results. Their content (OAuth article, PKCE article, the cairn device-flow note) all contain "oauth" as a substring and should still surface. Run the tests and inspect any diffs; if the match set or ordering differs, regenerate the goldens with `UPDATE_GOLDEN=1`, then manually confirm the output still represents the same design intent.

- [ ] **Step 3: Update `get.go` and `open.go` to use `newSource().ByHandle(n)` instead of `fixtures.ByHandle(n)`.**

For both files, replace:
```go
import "github.com/samay58/cairn/internal/fixtures"
...
c, err := fixtures.ByHandle(n)
```
with:
```go
import "github.com/samay58/cairn/internal/source"
...
src := newSource()
c, err := src.ByHandle(n)
```

Remove the `fixtures` import if no other references remain.

- [ ] **Step 4: Update `find.go` and `export.go` the same way (they call `fixtures.All()` for the top-3 / count).**

`find.go`: replace `all := fixtures.All()` with `src := newSource(); all := src.All()`.
`export.go`: same swap. The hard-coded count stays at `len(all)`.

- [ ] **Step 5: Run all tests.**

Run: `cd ~/cairn && go test ./...`
Expected: pass. If any golden fails because `FixtureSource.Search` orders results differently than Phase 0's `fakeSearch`, regenerate with `UPDATE_GOLDEN=1`, diff, confirm the output is reasonable, then re-run to verify.

- [ ] **Step 6: Commit.**

```bash
git add internal/commands/ && \
  git -c commit.gpgsign=false commit -m "refactor(commands): route all card reads through source.Source"
```

---

### Task 5: `source.Open()` selects fixture or SQLite

**Files:**
- Create: `internal/source/open.go`
- Create: `internal/source/open_test.go`

- [ ] **Step 1: Write the failing test.**

```go
package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenFallsBackToFixtureWhenNoDB(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "does-not-exist.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*FixtureSource); !ok {
		t.Errorf("expected *FixtureSource, got %T", s)
	}
}

func TestOpenReturnsSQLiteWhenDBExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cairn.db")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// SQLiteSource ships in Task 6. For now, any non-fixture implementation is acceptable.
	if _, ok := s.(*FixtureSource); ok {
		t.Error("expected SQLite-backed source, got fixture source")
	}
}
```

The second test will be skipped until Task 6 lands. Gate it:
```go
func TestOpenReturnsSQLiteWhenDBExists(t *testing.T) {
	t.Skip("enable when SQLiteSource lands in Task 6")
}
```

- [ ] **Step 2: Implement `open.go`.**

```go
package source

import (
	"fmt"
	"os"
)

// Open returns a Source reading from dbPath if the file exists, else a fixture
// source. In Task 6 the SQLite branch will return a real SQLiteSource; until
// then it returns an error to surface programming mistakes (the test gates the
// real call behind t.Skip).
func Open(dbPath string) (Source, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return NewFixtureSource(), nil
	} else if err != nil {
		return nil, fmt.Errorf("stat %s: %w", dbPath, err)
	}
	return nil, fmt.Errorf("SQLite source not yet implemented (Task 6)")
}
```

- [ ] **Step 3: Run, verify the skipped test skips and the other passes.**

Run: `go test ./internal/source/...`
Expected: PASS (skipped test counts as pass).

- [ ] **Step 4: Commit.**

```bash
git add internal/source/open.go internal/source/open_test.go && \
  git -c commit.gpgsign=false commit -m "feat(source): Open() dispatcher with fixture fallback"
```

---

### Task 6: `SQLiteSource` read path

**Files:**
- Create: `internal/storage/sqlite/sqlite.go`
- Create: `internal/storage/sqlite/sqlite_test.go`
- Create: `internal/storage/sqlite/handles.go`
- Create: `internal/storage/sqlite/handles_test.go`

- [ ] **Step 1: Write the failing test `sqlite_test.go`.**

```go
package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
	_ "modernc.org/sqlite"
)

func seed(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cairn.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(`INSERT INTO cards(id, mymind_id, kind, title, body, captured_at, updated_at) VALUES
	('c_1','mm_1','article','Deep Work','Rules for focused success.',?, ?),
	('c_2','mm_2','quote','On craft','The way you do anything is the way you do everything.',?, ?)`,
		now, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO sync_log(started_at, finished_at, delta_count, status) VALUES (?,?,2,'ok')`,
		now, now)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSQLiteSourceCount(t *testing.T) {
	db := seed(t)
	s := &SQLiteSource{DB: db}
	if got := s.Count(); got != 2 {
		t.Errorf("Count() = %d, want 2", got)
	}
}

func TestSQLiteSourceLastImport(t *testing.T) {
	db := seed(t)
	s := &SQLiteSource{DB: db}
	ts, ok := s.LastImport()
	if !ok {
		t.Fatal("LastImport should return ok=true after seed")
	}
	if time.Since(ts) > time.Minute {
		t.Errorf("LastImport too stale: %v", ts)
	}
}

func TestSQLiteSourceAllReturnsCards(t *testing.T) {
	db := seed(t)
	s := &SQLiteSource{DB: db}
	got := s.All()
	if len(got) != 2 {
		t.Fatalf("All() len = %d, want 2", len(got))
	}
	kinds := map[cards.Kind]bool{}
	for _, c := range got {
		kinds[c.Kind] = true
	}
	if !kinds[cards.KindArticle] || !kinds[cards.KindQuote] {
		t.Errorf("missing expected kinds, got %v", kinds)
	}
}
```

- [ ] **Step 2: Run, verify FAIL.**

- [ ] **Step 3: Implement `sqlite.go` (read path only — search in Task 8, handles in separate file).**

```go
package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
)

type SQLiteSource struct {
	DB *sql.DB
}

func Open(path string) (*SQLiteSource, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &SQLiteSource{DB: db}, nil
}

func (s *SQLiteSource) Close() error { return s.DB.Close() }

func (s *SQLiteSource) Count() int {
	var n int
	_ = s.DB.QueryRow(`SELECT count(*) FROM cards WHERE deleted_at IS NULL`).Scan(&n)
	return n
}

func (s *SQLiteSource) All() []cards.Card {
	rows, err := s.DB.Query(`SELECT id, mymind_id, kind, title, coalesce(url,''), coalesce(body,''), coalesce(excerpt,''), coalesce(source,''), captured_at FROM cards WHERE deleted_at IS NULL ORDER BY captured_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []cards.Card
	for rows.Next() {
		var c cards.Card
		var captured string
		if err := rows.Scan(&c.ID, &c.MyMindID, &c.Kind, &c.Title, &c.URL, &c.Body, &c.Excerpt, &c.Source, &captured); err != nil {
			continue
		}
		c.CapturedAt, _ = time.Parse(time.RFC3339, captured)
		// Tags populated in a follow-up query for cards that need them (Task 7).
		out = append(out, c)
	}
	return out
}

func (s *SQLiteSource) ByHandle(n int) (cards.Card, error) {
	// Implemented in Task 9 once the handles table is wired.
	return cards.Card{}, fmt.Errorf("ByHandle not yet implemented")
}

func (s *SQLiteSource) Search(query string, filters source.Filters, limit int) []render.Match {
	// Implemented in Task 8.
	return nil
}

func (s *SQLiteSource) LastImport() (time.Time, bool) {
	var ts sql.NullString
	err := s.DB.QueryRow(`SELECT finished_at FROM sync_log WHERE status='ok' ORDER BY finished_at DESC LIMIT 1`).Scan(&ts)
	if err != nil || !ts.Valid {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, ts.String)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func (s *SQLiteSource) LastListSave(_ []render.Match) error {
	// Implemented in Task 9.
	return nil
}
```

- [ ] **Step 4: Run, verify PASS.**

Run: `go test ./internal/storage/sqlite/...`
Expected: the three sqlite_test.go tests pass.

- [ ] **Step 5: Wire `SQLiteSource` into `source.Open()`.**

Edit `internal/source/open.go`:

```go
package source

import (
	"fmt"
	"os"

	"github.com/samay58/cairn/internal/storage/sqlite"
)

func Open(dbPath string) (Source, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return NewFixtureSource(), nil
	} else if err != nil {
		return nil, fmt.Errorf("stat %s: %w", dbPath, err)
	}
	return sqlite.Open(dbPath)
}
```

Unskip `TestOpenReturnsSQLiteWhenDBExists` in `open_test.go` (the seed file is empty; `sqlite.Open` will Migrate it and return a valid SQLiteSource — the test only checks it's not a FixtureSource).

Run: `go test ./internal/source/... ./internal/storage/sqlite/...`
Expected: all pass.

- [ ] **Step 6: Commit.**

```bash
git add internal/storage/sqlite/sqlite.go internal/storage/sqlite/sqlite_test.go internal/source/open.go internal/source/open_test.go && \
  git -c commit.gpgsign=false commit -m "feat(sqlite): SQLiteSource read path (Count, All, LastImport)"
```

---

### Task 7: Card tags round-trip

**Files:**
- Modify: `internal/storage/sqlite/sqlite.go` (fill `Card.Tags` on `All()`)
- Modify: `internal/storage/sqlite/sqlite_test.go` (add tag round-trip test)

- [ ] **Step 1: Add failing test.**

Append to `sqlite_test.go`:

```go
func TestSQLiteSourceAllIncludesTags(t *testing.T) {
	db := seed(t)
	_, err := db.Exec(`INSERT INTO tags(card_id, tag) VALUES ('c_1', 'productivity'), ('c_1','focus')`)
	if err != nil {
		t.Fatal(err)
	}
	s := &SQLiteSource{DB: db}
	got := s.All()
	var found cards.Card
	for _, c := range got {
		if c.ID == "c_1" {
			found = c
		}
	}
	if len(found.Tags) != 2 {
		t.Errorf("expected 2 tags on c_1, got %v", found.Tags)
	}
}
```

- [ ] **Step 2: Run, verify FAIL.**

- [ ] **Step 3: Modify `All()` to join tags.**

Replace `All()` body with:

```go
func (s *SQLiteSource) All() []cards.Card {
	rows, err := s.DB.Query(`SELECT id, mymind_id, kind, title, coalesce(url,''), coalesce(body,''), coalesce(excerpt,''), coalesce(source,''), captured_at FROM cards WHERE deleted_at IS NULL ORDER BY captured_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []cards.Card
	for rows.Next() {
		var c cards.Card
		var captured string
		if err := rows.Scan(&c.ID, &c.MyMindID, &c.Kind, &c.Title, &c.URL, &c.Body, &c.Excerpt, &c.Source, &captured); err != nil {
			continue
		}
		c.CapturedAt, _ = time.Parse(time.RFC3339, captured)
		out = append(out, c)
	}
	for i, c := range out {
		trows, err := s.DB.Query(`SELECT tag FROM tags WHERE card_id = ? ORDER BY tag`, c.ID)
		if err != nil {
			continue
		}
		for trows.Next() {
			var t string
			if err := trows.Scan(&t); err == nil {
				out[i].Tags = append(out[i].Tags, t)
			}
		}
		trows.Close()
	}
	return out
}
```

- [ ] **Step 4: Run, verify PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/storage/sqlite/ && \
  git -c commit.gpgsign=false commit -m "feat(sqlite): include tags when loading all cards"
```

---

### Task 8: FTS5 search

**Files:**
- Modify: `internal/storage/sqlite/sqlite.go` (`Search` method)
- Create: `internal/storage/sqlite/search.go` (query construction)
- Create: `internal/storage/sqlite/search_test.go`

- [ ] **Step 1: Write the failing test.**

```go
package sqlite

import (
	"testing"
	"time"

	"github.com/samay58/cairn/internal/source"
)

func TestSQLiteSourceSearchFTS(t *testing.T) {
	db := seed(t)
	s := &SQLiteSource{DB: db}

	matches := s.Search("craft", source.Filters{}, 0)
	if len(matches) == 0 {
		t.Fatal("expected a match for 'craft'")
	}
	if matches[0].Card.Title != "On craft" {
		t.Errorf("top hit = %q, want %q", matches[0].Card.Title, "On craft")
	}
	if matches[0].WhyShown == "" {
		t.Error("WhyShown should not be empty")
	}
}

func TestSQLiteSourceSearchKindFilter(t *testing.T) {
	db := seed(t)
	s := &SQLiteSource{DB: db}

	// Only quotes.
	matches := s.Search("", source.Filters{Kind: "quote"}, 0)
	for _, m := range matches {
		if m.Card.Kind != "quote" {
			t.Errorf("got kind %q", m.Card.Kind)
		}
	}

	// Only articles captured after year 3000 (none).
	far := time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)
	zero := s.Search("", source.Filters{Since: far}, 0)
	if len(zero) != 0 {
		t.Errorf("expected empty, got %d", len(zero))
	}
}

func TestSQLiteSourceSearchLimit(t *testing.T) {
	db := seed(t)
	s := &SQLiteSource{DB: db}
	matches := s.Search("", source.Filters{}, 1)
	if len(matches) != 1 {
		t.Errorf("limit=1 returned %d matches", len(matches))
	}
}
```

- [ ] **Step 2: Run, verify FAIL.**

- [ ] **Step 3: Implement `search.go` — query parser for filter tokens, then FTS5 MATCH.**

```go
package sqlite

import (
	"strings"
	"time"

	"github.com/samay58/cairn/internal/source"
)

// parseQuery strips filter tokens (type:, from:, since:, #tag) from the raw
// query string, merging them into filters. Remaining words are the FTS5 query.
// This is a permissive parser; unknown tokens are kept as plain terms.
func parseQuery(raw string, in source.Filters) (ftsQuery string, out source.Filters) {
	out = in
	var terms []string
	for _, tok := range strings.Fields(raw) {
		switch {
		case strings.HasPrefix(tok, "type:"):
			if out.Kind == "" {
				out.Kind = strings.TrimPrefix(tok, "type:")
			}
		case strings.HasPrefix(tok, "from:"):
			if out.From == "" {
				out.From = strings.TrimPrefix(tok, "from:")
			}
		case strings.HasPrefix(tok, "since:"):
			if out.Since.IsZero() {
				if t, err := time.Parse("2006-01-02", strings.TrimPrefix(tok, "since:")); err == nil {
					out.Since = t
				}
			}
		case strings.HasPrefix(tok, "#"):
			if out.Tag == "" {
				out.Tag = strings.TrimPrefix(tok, "#")
			}
		default:
			terms = append(terms, tok)
		}
	}
	return strings.Join(terms, " "), out
}

// escapeFTSTerm quotes a single term for FTS5 MATCH so special chars don't blow
// up the query. FTS5 accepts a double-quoted phrase.
func escapeFTSTerm(t string) string {
	t = strings.ReplaceAll(t, `"`, `""`)
	return `"` + t + `"`
}

func buildFTSExpression(q string) string {
	fields := strings.Fields(q)
	if len(fields) == 0 {
		return ""
	}
	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = escapeFTSTerm(f)
	}
	return strings.Join(quoted, " ")
}
```

- [ ] **Step 4: Replace `Search` in `sqlite.go`.**

```go
func (s *SQLiteSource) Search(rawQuery string, filters source.Filters, limit int) []render.Match {
	ftsQ, merged := parseQuery(rawQuery, filters)

	where := []string{`cards.deleted_at IS NULL`}
	args := []any{}
	join := ""
	if ftsQ != "" {
		join = "JOIN cards_fts ON cards_fts.rowid = cards.rowid"
		where = append(where, `cards_fts MATCH ?`)
		args = append(args, buildFTSExpression(ftsQ))
	}
	if merged.Kind != "" {
		where = append(where, `cards.kind = ?`)
		args = append(args, merged.Kind)
	}
	if merged.From != "" {
		where = append(where, `cards.source LIKE ?`)
		args = append(args, "%"+merged.From+"%")
	}
	if !merged.Since.IsZero() {
		where = append(where, `cards.captured_at >= ?`)
		args = append(args, merged.Since.Format(time.RFC3339))
	}
	if merged.Tag != "" {
		where = append(where, `EXISTS (SELECT 1 FROM tags WHERE tags.card_id = cards.id AND tags.tag = ?)`)
		args = append(args, merged.Tag)
	}

	q := `SELECT cards.id, cards.mymind_id, cards.kind, cards.title, coalesce(cards.url,''),
	coalesce(cards.body,''), coalesce(cards.excerpt,''), coalesce(cards.source,''), cards.captured_at FROM cards ` +
		join + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY cards.captured_at DESC`

	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var matches []render.Match
	for rows.Next() {
		var c cards.Card
		var captured string
		if err := rows.Scan(&c.ID, &c.MyMindID, &c.Kind, &c.Title, &c.URL, &c.Body, &c.Excerpt, &c.Source, &captured); err != nil {
			continue
		}
		c.CapturedAt, _ = time.Parse(time.RFC3339, captured)
		matches = append(matches, render.Match{Card: c, WhyShown: whyShownFTS(c, ftsQ)})
	}
	return matches
}

func whyShownFTS(c cards.Card, q string) string {
	if q == "" {
		return "recent"
	}
	lo := strings.ToLower(q)
	if strings.Contains(strings.ToLower(c.Title), lo) {
		return "matched on title"
	}
	return "matched on body"
}
```

Add `"fmt"` and `"strings"` to the imports if not already present.

- [ ] **Step 5: Run, verify PASS on all three search tests.**

Run: `go test ./internal/storage/sqlite/... -run TestSQLiteSourceSearch`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/storage/sqlite/ && \
  git -c commit.gpgsign=false commit -m "feat(sqlite): FTS5 search with filter parsing"
```

---

### Task 9: Handle persistence

**Files:**
- Create: `internal/storage/sqlite/handles.go`
- Create: `internal/storage/sqlite/handles_test.go`
- Modify: `internal/storage/sqlite/sqlite.go` (fill `ByHandle` and `LastListSave`)

- [ ] **Step 1: Write the failing test.**

```go
package sqlite

import (
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/render"
)

func TestHandlesRoundTrip(t *testing.T) {
	db := seed(t)
	s := &SQLiteSource{DB: db}

	matches := []render.Match{
		{Card: cards.Card{ID: "c_1", Title: "Deep Work", CapturedAt: time.Now()}},
		{Card: cards.Card{ID: "c_2", Title: "On craft", CapturedAt: time.Now()}},
	}
	if err := s.LastListSave(matches); err != nil {
		t.Fatal(err)
	}

	c2, err := s.ByHandle(2)
	if err != nil {
		t.Fatal(err)
	}
	if c2.ID != "c_2" {
		t.Errorf("@2 = %q, want c_2", c2.ID)
	}

	// Out of range.
	if _, err := s.ByHandle(99); err == nil {
		t.Error("@99 should error")
	}

	// New save clears old.
	single := []render.Match{{Card: cards.Card{ID: "c_1", Title: "Deep Work", CapturedAt: time.Now()}}}
	if err := s.LastListSave(single); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ByHandle(2); err == nil {
		t.Error("after single-item save, @2 should error")
	}
}
```

- [ ] **Step 2: Run, verify FAIL.**

- [ ] **Step 3: Implement handle methods on `SQLiteSource`.**

Append to `sqlite.go` (replacing the stubs):

```go
func (s *SQLiteSource) ByHandle(n int) (cards.Card, error) {
	row := s.DB.QueryRow(`SELECT cards.id, cards.mymind_id, cards.kind, cards.title, coalesce(cards.url,''),
		coalesce(cards.body,''), coalesce(cards.excerpt,''), coalesce(cards.source,''), cards.captured_at
		FROM handles JOIN cards ON cards.id = handles.card_id WHERE handles.position = ? AND cards.deleted_at IS NULL`, n)
	var c cards.Card
	var captured string
	if err := row.Scan(&c.ID, &c.MyMindID, &c.Kind, &c.Title, &c.URL, &c.Body, &c.Excerpt, &c.Source, &captured); err != nil {
		return cards.Card{}, fmt.Errorf("no card at handle @%d (run a list command to refresh)", n)
	}
	c.CapturedAt, _ = time.Parse(time.RFC3339, captured)
	return c, nil
}

func (s *SQLiteSource) LastListSave(matches []render.Match) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM handles`); err != nil {
		tx.Rollback()
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i, m := range matches {
		if _, err := tx.Exec(`INSERT INTO handles(position, card_id, created_at) VALUES (?, ?, ?)`, i+1, m.Card.ID, now); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
```

- [ ] **Step 4: Run, verify PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/storage/sqlite/ && \
  git -c commit.gpgsign=false commit -m "feat(sqlite): handles table for @N persistence across commands"
```

---

### Task 10: Importer CSV parser

**Files:**
- Create: `internal/importer/csv.go`
- Create: `internal/importer/csv_test.go`
- Create: `testdata/mymind_sample_export/cards.csv`

- [ ] **Step 1: Author a minimal sample CSV `testdata/mymind_sample_export/cards.csv` for tests.**

```csv
id,type,title,url,body,excerpt,source,tags,captured_at
mm_1,article,Deep Work,https://cal.newport.com/books/deep-work/,Rules for focused success in a distracted world.,,cal.newport.com,"focus;productivity",2026-03-01T09:00:00Z
mm_2,quote,On craft,,The way you do anything is the way you do everything.,,Martha Beck,"craft;philosophy",2026-03-18T14:05:00Z
mm_3,image,Braun T3 radio,https://www.vitsoe.com/,,,vitsoe.com,"design;rams",2026-03-20T11:00:00Z
mm_4,note,Cairn naming,,Short. Unclaimed. Fits the category.,,,"cairn;project",2026-03-22T18:40:00Z
```

This is a handcrafted sample. First real import updates `IMPORT_FORMAT.md` if columns differ.

- [ ] **Step 2: Write the failing test.**

```go
package importer

import (
	"path/filepath"
	"testing"
)

func TestParseCardsCSVHappyPath(t *testing.T) {
	got, warnings, err := ParseCardsCSV(filepath.Join("..", "..", "testdata", "mymind_sample_export", "cards.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Errorf("parsed %d cards, want 4", len(got))
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	// Check a specific known field.
	first := got[0]
	if first.MyMindID != "mm_1" || first.Title != "Deep Work" {
		t.Errorf("first card: %+v", first)
	}
	if len(first.Tags) != 2 {
		t.Errorf("first card tags = %v", first.Tags)
	}
}

func TestParseCardsCSVMalformedRowProducesWarning(t *testing.T) {
	// Write a bad CSV into a temp file.
	path := filepath.Join(t.TempDir(), "bad.csv")
	body := "id,type,title\nmm_1,article,Good\n,,\nmm_3,,Missing type\n"
	if err := writeFile(path, body); err != nil {
		t.Fatal(err)
	}
	got, warnings, err := ParseCardsCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("got %d valid cards, want 1 (Good)", len(got))
	}
	if len(warnings) == 0 {
		t.Error("expected warnings for malformed rows")
	}
}
```

Add a small helper `writeFile(path, body string) error` to the test file using `os.WriteFile`.

- [ ] **Step 3: Run, verify FAIL.**

- [ ] **Step 4: Implement `csv.go`.**

```go
package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

// ParseCardsCSV reads a MyMind-style cards.csv. Column names are matched
// case-insensitively; synonyms (body/text/content) are accepted. Rows missing
// required fields (id, kind, title) produce a warning and are skipped.
func ParseCardsCSV(path string) ([]cards.Card, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // permissive

	header, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("read header: %w", err)
	}
	cols := normalizeHeader(header)

	var out []cards.Card
	var warnings []string
	lineNo := 1
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		lineNo++
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("line %d: %v", lineNo, err))
			continue
		}
		c, ok, warn := rowToCard(cols, row)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("line %d: %s", lineNo, warn))
			continue
		}
		out = append(out, c)
	}
	return out, warnings, nil
}

func normalizeHeader(h []string) map[string]int {
	idx := map[string]int{}
	for i, name := range h {
		idx[strings.ToLower(strings.TrimSpace(name))] = i
	}
	return idx
}

func pick(cols map[string]int, row []string, names ...string) string {
	for _, n := range names {
		if i, ok := cols[n]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
	}
	return ""
}

func rowToCard(cols map[string]int, row []string) (cards.Card, bool, string) {
	id := pick(cols, row, "id", "mymind_id", "card_id")
	kindRaw := pick(cols, row, "type", "kind")
	title := pick(cols, row, "title")
	if id == "" || kindRaw == "" || title == "" {
		return cards.Card{}, false, "missing id/type/title"
	}
	kind, err := cards.KindFromString(strings.ToLower(kindRaw))
	if err != nil {
		return cards.Card{}, false, fmt.Sprintf("unknown kind %q", kindRaw)
	}
	captured := pick(cols, row, "captured_at", "created_at", "date")
	capturedAt, err := time.Parse(time.RFC3339, captured)
	if err != nil {
		capturedAt = time.Now().UTC()
	}
	tagsRaw := pick(cols, row, "tags")
	var tags []string
	if tagsRaw != "" {
		splitter := ";"
		if !strings.Contains(tagsRaw, ";") {
			splitter = ","
		}
		for _, t := range strings.Split(tagsRaw, splitter) {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
	}
	return cards.Card{
		ID:         id,
		MyMindID:   id,
		Kind:       kind,
		Title:      title,
		URL:        pick(cols, row, "url", "link"),
		Body:       pick(cols, row, "body", "text", "content"),
		Excerpt:    pick(cols, row, "excerpt", "description"),
		Source:     pick(cols, row, "source", "domain"),
		Tags:       tags,
		CapturedAt: capturedAt,
	}, true, ""
}
```

- [ ] **Step 5: Run, verify PASS.**

Run: `go test ./internal/importer/...`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/importer/csv.go internal/importer/csv_test.go testdata/ && \
  git -c commit.gpgsign=false commit -m "feat(importer): permissive MyMind cards.csv parser"
```

---

### Task 11: Media discovery + chunking

**Files:**
- Create: `internal/importer/media.go`
- Create: `internal/importer/media_test.go`
- Create: `internal/importer/chunk.go`
- Create: `internal/importer/chunk_test.go`
- Create: `testdata/mymind_sample_export/media/sample.png` (can be a tiny 1x1 PNG or any small binary)

- [ ] **Step 1: Create the sample media file.**

```bash
# 67-byte minimal PNG (1x1 transparent)
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82' > ~/cairn/testdata/mymind_sample_export/media/sample.png
```

- [ ] **Step 2: Write the failing test `media_test.go`.**

```go
package importer

import (
	"path/filepath"
	"testing"
)

func TestScanMediaHashesAndDetectsMime(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "mymind_sample_export", "media")
	items, err := ScanMedia(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d media items, want 1", len(items))
	}
	if items[0].Mime != "image/png" {
		t.Errorf("mime = %q, want image/png", items[0].Mime)
	}
	if len(items[0].SHA256) != 64 {
		t.Errorf("sha256 = %q (len %d), want 64 hex chars", items[0].SHA256, len(items[0].SHA256))
	}
}
```

- [ ] **Step 3: Implement `media.go`.**

```go
package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type MediaItem struct {
	Path   string
	SHA256 string
	Mime   string
}

func ScanMedia(dir string) ([]MediaItem, error) {
	var out []MediaItem
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		buf := make([]byte, 512)
		n, _ := f.Read(buf)
		mime := http.DetectContentType(buf[:n])
		// Re-read from start for hash.
		if _, err := f.Seek(0, 0); err != nil {
			return err
		}
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		out = append(out, MediaItem{
			Path:   path,
			SHA256: hex.EncodeToString(h.Sum(nil)),
			Mime:   mime,
		})
		return nil
	})
	return out, err
}
```

- [ ] **Step 4: Run, verify PASS.**

- [ ] **Step 5: Write the failing test `chunk_test.go`.**

```go
package importer

import (
	"strings"
	"testing"
)

func TestChunkShortBodyReturnsOne(t *testing.T) {
	chunks := Chunk("A short body.")
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0].Text != "A short body." {
		t.Errorf("text = %q", chunks[0].Text)
	}
	if chunks[0].StartOffset != 0 || chunks[0].EndOffset != len("A short body.") {
		t.Errorf("offsets = %d..%d", chunks[0].StartOffset, chunks[0].EndOffset)
	}
}

func TestChunkLongBodySplitsOnParagraphs(t *testing.T) {
	long := strings.Repeat("paragraph one word word word word word word word word word.\n\n", 40)
	chunks := Chunk(long)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for long body, got %d", len(chunks))
	}
	// Every chunk between ~200 and ~600 words per spec.
	for i, ch := range chunks {
		words := len(strings.Fields(ch.Text))
		if words > 700 {
			t.Errorf("chunk %d has %d words, want <=700", i, words)
		}
	}
}

func TestChunkChecksumsStable(t *testing.T) {
	body := "Once more with feeling.\n\nSecond paragraph."
	a := Chunk(body)
	b := Chunk(body)
	for i := range a {
		if a[i].Checksum != b[i].Checksum {
			t.Errorf("chunk %d checksum differs between runs", i)
		}
	}
}
```

- [ ] **Step 6: Implement `chunk.go`.**

```go
package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type ChunkItem struct {
	Text        string
	StartOffset int
	EndOffset   int
	Checksum    string
}

const (
	targetMinWords = 200
	targetMaxWords = 600
)

func Chunk(body string) []ChunkItem {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	paras := splitParagraphs(body)
	if len(paras) == 0 {
		return nil
	}
	var chunks []ChunkItem
	var buf strings.Builder
	bufWords := 0
	start := paras[0].start
	for _, p := range paras {
		pWords := len(strings.Fields(p.text))
		if bufWords+pWords > targetMaxWords && bufWords >= targetMinWords {
			chunks = append(chunks, makeChunk(buf.String(), start, p.start))
			buf.Reset()
			bufWords = 0
			start = p.start
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(p.text)
		bufWords += pWords
	}
	if buf.Len() > 0 {
		chunks = append(chunks, makeChunk(buf.String(), start, start+buf.Len()))
	}
	return chunks
}

type paraSpan struct {
	text  string
	start int
}

func splitParagraphs(body string) []paraSpan {
	var out []paraSpan
	pos := 0
	for _, chunk := range strings.Split(body, "\n\n") {
		if trimmed := strings.TrimSpace(chunk); trimmed != "" {
			out = append(out, paraSpan{text: trimmed, start: pos})
		}
		pos += len(chunk) + 2
	}
	return out
}

func makeChunk(text string, start, end int) ChunkItem {
	h := sha256.Sum256([]byte(text))
	return ChunkItem{
		Text:        text,
		StartOffset: start,
		EndOffset:   end,
		Checksum:    hex.EncodeToString(h[:]),
	}
}
```

- [ ] **Step 7: Run all importer tests, verify PASS.**

Run: `go test ./internal/importer/...`
Expected: PASS.

- [ ] **Step 8: Commit.**

```bash
git add testdata/mymind_sample_export/media/sample.png internal/importer/media.go internal/importer/media_test.go internal/importer/chunk.go internal/importer/chunk_test.go && \
  git -c commit.gpgsign=false commit -m "feat(importer): media discovery (sha256 + mime) and paragraph chunking"
```

---

### Task 12: Import orchestrator

**Files:**
- Create: `internal/importer/importer.go`
- Create: `internal/importer/importer_test.go`

- [ ] **Step 1: Write the failing test.**

```go
package importer

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/samay58/cairn/internal/storage/sqlite"
	_ "modernc.org/sqlite"
)

func TestImportEndToEndSampleExport(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cairn.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := sqlite.Migrate(db); err != nil {
		t.Fatal(err)
	}

	result, err := Import(db, filepath.Join("..", "..", "testdata", "mymind_sample_export"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 4 {
		t.Errorf("inserted %d cards, want 4", result.Inserted)
	}
	if result.MediaCount != 1 {
		t.Errorf("media %d, want 1", result.MediaCount)
	}

	// sync_log has an entry.
	var status string
	var finished sql.NullString
	if err := db.QueryRow(`SELECT status, finished_at FROM sync_log ORDER BY id DESC LIMIT 1`).Scan(&status, &finished); err != nil {
		t.Fatal(err)
	}
	if status != "ok" {
		t.Errorf("sync_log status %q, want 'ok'", status)
	}
	if !finished.Valid {
		t.Error("sync_log finished_at should be set")
	}

	// cards_fts populated.
	var ftsCount int
	if err := db.QueryRow(`SELECT count(*) FROM cards_fts`).Scan(&ftsCount); err != nil {
		t.Fatal(err)
	}
	if ftsCount != 4 {
		t.Errorf("cards_fts rows = %d, want 4", ftsCount)
	}

	// chunks populated for non-empty bodies.
	var chunkCount int
	if err := db.QueryRow(`SELECT count(*) FROM chunks`).Scan(&chunkCount); err != nil {
		t.Fatal(err)
	}
	if chunkCount < 2 {
		t.Errorf("chunks count = %d, want at least 2 (cards with bodies)", chunkCount)
	}
	_ = time.Now()
}

func TestImportMissingDirReturnsError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cairn.db")
	db, _ := sql.Open("sqlite", dbPath)
	defer db.Close()
	_ = sqlite.Migrate(db)

	_, err := Import(db, "/tmp/does-not-exist-x9z9z")
	if err == nil {
		t.Error("expected error for missing dir")
	}
}
```

- [ ] **Step 2: Run, verify FAIL.**

- [ ] **Step 3: Implement `importer.go`.**

```go
package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

type Result struct {
	Inserted     int
	Updated      int
	Tombstoned   int
	MediaCount   int
	ChunkCount   int
	Warnings     []string
}

func Import(db *sql.DB, exportDir string) (Result, error) {
	var r Result
	if _, err := os.Stat(exportDir); err != nil {
		return r, fmt.Errorf("read export dir: %w", err)
	}

	start := time.Now().UTC()
	var syncID int64
	res, err := db.Exec(`INSERT INTO sync_log(started_at, status) VALUES (?, 'running')`, start.Format(time.RFC3339))
	if err != nil {
		return r, err
	}
	syncID, _ = res.LastInsertId()

	parsed, parseWarns, err := ParseCardsCSV(filepath.Join(exportDir, "cards.csv"))
	if err != nil {
		markSyncFailed(db, syncID, err)
		return r, err
	}
	r.Warnings = append(r.Warnings, parseWarns...)

	// Load existing IDs for tombstoning.
	existing := map[string]bool{}
	rows, _ := db.Query(`SELECT mymind_id FROM cards WHERE deleted_at IS NULL`)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			existing[id] = true
		}
	}
	rows.Close()

	tx, err := db.Begin()
	if err != nil {
		markSyncFailed(db, syncID, err)
		return r, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range parsed {
		if err := upsertCard(tx, c, now); err != nil {
			tx.Rollback()
			markSyncFailed(db, syncID, err)
			return r, err
		}
		if existing[c.MyMindID] {
			r.Updated++
			delete(existing, c.MyMindID)
		} else {
			r.Inserted++
		}
		for _, ch := range Chunk(c.Body) {
			if _, err := tx.Exec(`INSERT INTO chunks(card_id, modality, text, start_offset, end_offset, checksum) VALUES (?, 'text', ?, ?, ?, ?)`,
				c.ID, ch.Text, ch.StartOffset, ch.EndOffset, ch.Checksum); err != nil {
				tx.Rollback()
				markSyncFailed(db, syncID, err)
				return r, err
			}
			r.ChunkCount++
		}
	}

	// Everything left in `existing` was absent from this export: tombstone.
	for id := range existing {
		if _, err := tx.Exec(`UPDATE cards SET deleted_at = ? WHERE mymind_id = ?`, now, id); err != nil {
			tx.Rollback()
			markSyncFailed(db, syncID, err)
			return r, err
		}
		r.Tombstoned++
	}

	// Media scan.
	mediaDir := filepath.Join(exportDir, "media")
	if _, err := os.Stat(mediaDir); err == nil {
		items, scanErr := ScanMedia(mediaDir)
		if scanErr != nil {
			r.Warnings = append(r.Warnings, fmt.Sprintf("media scan: %v", scanErr))
		}
		for _, it := range items {
			if _, err := tx.Exec(`INSERT INTO media(card_id, kind, path, sha256, mime) VALUES ('', 'image', ?, ?, ?)`,
				it.Path, it.SHA256, it.Mime); err != nil {
				r.Warnings = append(r.Warnings, fmt.Sprintf("media insert %s: %v", it.Path, err))
			}
			r.MediaCount++
		}
	}

	if err := tx.Commit(); err != nil {
		markSyncFailed(db, syncID, err)
		return r, err
	}

	finish := time.Now().UTC()
	_, _ = db.Exec(`UPDATE sync_log SET finished_at = ?, delta_count = ?, status = 'ok' WHERE id = ?`,
		finish.Format(time.RFC3339), r.Inserted+r.Updated+r.Tombstoned, syncID)

	return r, nil
}

func upsertCard(tx *sql.Tx, c cards.Card, updatedAt string) error {
	_, err := tx.Exec(`INSERT INTO cards(id, mymind_id, kind, title, url, body, excerpt, source, captured_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mymind_id) DO UPDATE SET
			title = excluded.title,
			url = excluded.url,
			body = excluded.body,
			excerpt = excluded.excerpt,
			source = excluded.source,
			captured_at = excluded.captured_at,
			updated_at = excluded.updated_at,
			deleted_at = NULL`,
		c.ID, c.MyMindID, string(c.Kind), c.Title, nullish(c.URL), nullish(c.Body), nullish(c.Excerpt), nullish(c.Source),
		c.CapturedAt.Format(time.RFC3339), updatedAt)
	if err != nil {
		return err
	}
	// Tags: replace-all semantics for simplicity.
	if _, err := tx.Exec(`DELETE FROM tags WHERE card_id = ?`, c.ID); err != nil {
		return err
	}
	for _, t := range c.Tags {
		if _, err := tx.Exec(`INSERT INTO tags(card_id, tag) VALUES (?, ?)`, c.ID, t); err != nil {
			return err
		}
	}
	// Chunks: replace-all.
	if _, err := tx.Exec(`DELETE FROM chunks WHERE card_id = ?`, c.ID); err != nil {
		return err
	}
	return nil
}

func nullish(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func markSyncFailed(db *sql.DB, syncID int64, err error) {
	if syncID == 0 {
		return
	}
	_, _ = db.Exec(`UPDATE sync_log SET finished_at = ?, status = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), "error: "+err.Error(), syncID)
}
```

- [ ] **Step 4: Run, verify PASS.**

Run: `go test ./internal/importer/... -run TestImport`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/importer/importer.go internal/importer/importer_test.go && \
  git -c commit.gpgsign=false commit -m "feat(importer): end-to-end Import orchestrator with tombstones"
```

---

### Task 13: Hard-delete after 30 days

**Files:**
- Modify: `internal/importer/importer.go`
- Modify: `internal/importer/importer_test.go`

- [ ] **Step 1: Add failing test.**

```go
func TestImportHardDeletesAfter30Days(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cairn.db")
	db, _ := sql.Open("sqlite", dbPath)
	defer db.Close()
	_ = sqlite.Migrate(db)

	old := time.Now().UTC().Add(-31 * 24 * time.Hour).Format(time.RFC3339)
	_, err := db.Exec(`INSERT INTO cards(id, mymind_id, kind, title, captured_at, updated_at, deleted_at)
		VALUES ('c_old','mm_old','article','Stale', ?, ?, ?)`, old, old, old)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Import(db, filepath.Join("..", "..", "testdata", "mymind_sample_export")); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM cards WHERE id = 'c_old'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("stale card should be hard-deleted, count = %d", n)
	}
}
```

- [ ] **Step 2: Run, verify FAIL.**

- [ ] **Step 3: Add the hard-delete pass to `Import` (after the Commit, before returning).**

```go
cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
if _, err := db.Exec(`DELETE FROM cards WHERE deleted_at IS NOT NULL AND deleted_at < ?`, cutoff); err != nil {
	r.Warnings = append(r.Warnings, fmt.Sprintf("hard-delete: %v", err))
}
```

- [ ] **Step 4: Run, verify PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/importer/ && \
  git -c commit.gpgsign=false commit -m "feat(importer): hard-delete tombstoned cards after 30 days"
```

---

### Task 14: Real `cairn import` command wiring

**Files:**
- Modify: `internal/commands/import.go`
- Modify: `internal/commands/import_test.go`

- [ ] **Step 1: Update `import_test.go`.**

Add a test that imports the sample export into a temp DB and snapshots the output. Remove or update `TestImportOK` and `TestImportNotFound` to point at the new behavior. Because the new implementation uses real file stat (not the magic path branch from Phase 0), the "not-found" error test can now check an actual non-existent path. The happy-path test imports the sample dir into `t.TempDir()`.

```go
func TestImportSampleExport(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("CAIRN_HOME", tmp)
	defer os.Unsetenv("CAIRN_HOME")

	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"import", filepath.Join("..", "..", "testdata", "mymind_sample_export")})
	if err := root.Execute(); err != nil {
		t.Fatalf("import failed: %v\n%s", err, out.String())
	}
	golden.Assert(t, "import_ok_phase1.txt", out.String())
}

func TestImportRealPathNotFound(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("CAIRN_HOME", tmp)
	defer os.Unsetenv("CAIRN_HOME")

	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"import", "/tmp/does-not-exist-x9z9z"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	golden.Assert(t, "import_err_notfound_phase1.txt", out.String())
}
```

Delete the Phase 0 `TestImportOK` and `TestImportNotFound` tests and their goldens (`import_ok.txt`, `import_err_notfound.txt`).

- [ ] **Step 2: Implement the new `import.go`.**

```go
package commands

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samay58/cairn/internal/importer"
	"github.com/samay58/cairn/internal/storage/sqlite"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Ingest a MyMind export folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			exportDir := args[0]
			if _, err := os.Stat(exportDir); err != nil {
				writeImportNotFound(out, exportDir)
				return nil
			}
			dbPath := cairnDBPath()
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return err
			}
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				return err
			}
			defer db.Close()
			if err := sqlite.Migrate(db); err != nil {
				return err
			}

			fmt.Fprintf(out, "Reading export from %s\n", exportDir)
			result, err := importer.Import(db, exportDir)
			if err != nil {
				fmt.Fprintf(out, "Error: import failed.\n\n%v\n", err)
				return nil
			}
			fmt.Fprintf(out, "Parsed %d cards; %d inserted, %d updated, %d tombstoned.\n",
				result.Inserted+result.Updated, result.Inserted, result.Updated, result.Tombstoned)
			fmt.Fprintf(out, "Media: %d files. Chunks: %d.\n\n", result.MediaCount, result.ChunkCount)
			fmt.Fprintf(out, "Database at %s.\nRun `cairn search \"<query>\"` or `cairn find`.\n", dbPath)
			for _, w := range result.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
			}
			return nil
		},
	}
}

func cairnDBPath() string {
	if v := os.Getenv("CAIRN_HOME"); v != "" {
		return filepath.Join(v, "cairn.db")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cairn", "cairn.db")
}

func writeImportNotFound(out *os.File, path string) {
	fmt.Fprintln(out, "Error: import failed.")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Could not read export directory: %s\n", path)
	fmt.Fprintln(out, "Check the path and try again.")
}
```

Fix the `writeImportNotFound` signature: it should accept `io.Writer`, not `*os.File`. Use:
```go
func writeImportNotFound(out io.Writer, path string) {
```
and add `"io"` to imports.

- [ ] **Step 3: Author the Phase 1 goldens.**

`internal/commands/testdata/golden/import_ok_phase1.txt`:
```
Reading export from ../../testdata/mymind_sample_export
Parsed 4 cards; 4 inserted, 0 updated, 0 tombstoned.
Media: 1 files. Chunks: 2.

Database at <TMP>/cairn.db.
Run `cairn search "<query>"` or `cairn find`.
```

Because the path is a temp dir, golden will need a small replacement step. Handle it in-test: before calling `golden.Assert`, substitute `tmp` in the captured output with the literal `<TMP>`:

```go
// Inside TestImportSampleExport before Assert:
outStr := strings.ReplaceAll(out.String(), tmp, "<TMP>")
golden.Assert(t, "import_ok_phase1.txt", outStr)
```

`internal/commands/testdata/golden/import_err_notfound_phase1.txt`:
```
Error: import failed.

Could not read export directory: /tmp/does-not-exist-x9z9z
Check the path and try again.
```

- [ ] **Step 4: Update the test helper and regenerate/inspect goldens.**

Add the `strings.ReplaceAll` substitution to both import tests. Run with UPDATE_GOLDEN=1 once to produce initial goldens, verify the content matches expected structure, then commit.

- [ ] **Step 5: Ensure Phase 0 goldens that reference the old import output are gone.**

Run: `ls internal/commands/testdata/golden/import_ok.txt internal/commands/testdata/golden/import_err_notfound.txt 2>/dev/null`
If any remain, `git rm` them.

- [ ] **Step 6: Run the full suite.**

Run: `cd ~/cairn && go test ./...`
Expected: PASS everywhere. If `SCREENPLAY.md` has a fenced block from the old import goldens, update it to reference the new import_ok_phase1 content.

- [ ] **Step 7: Commit.**

```bash
git add internal/commands/ SCREENPLAY.md && \
  git -c commit.gpgsign=false commit -m "feat(import): real CSV-to-SQLite ingestion via importer"
```

---

### Task 15: Real `cairn search` via SQLite

**Files:**
- Modify: `internal/commands/search.go` (read from `source.Open()`)
- Modify: `internal/commands/search_test.go`
- Create: `internal/commands/testdata/golden/search_phase1_craft.txt`

- [ ] **Step 1: Switch search.go to use `source.Open(cairnDBPath())` instead of `newSource()`.**

Replace:
```go
src := newSource()
```
with:
```go
src, err := source.Open(cairnDBPath())
if err != nil {
	return err
}
```

Imports: add `"github.com/samay58/cairn/internal/source"`.

Keep `newSource` as a compat shim used by `get.go`, `open.go`, `find.go`, and `export.go` — update those to also call `source.Open(cairnDBPath())`. Delete `newSource` once all callers are migrated.

Actually — simplify: delete `newSource` entirely. Every command that reads cards does `src, err := source.Open(cairnDBPath())`. Fewer layers.

- [ ] **Step 2: Write a Phase 1 search test that imports then searches.**

```go
func TestSearchPhase1CraftAfterImport(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("CAIRN_HOME", tmp)
	defer os.Unsetenv("CAIRN_HOME")

	// Import sample first.
	{
		root := NewRoot()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"import", filepath.Join("..", "..", "testdata", "mymind_sample_export")})
		if err := root.Execute(); err != nil {
			t.Fatalf("import: %v", err)
		}
	}

	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"search", "craft"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "search_phase1_craft.txt", out.String())
}
```

- [ ] **Step 3: Run, verify FAIL (golden doesn't exist).**

Regenerate with `UPDATE_GOLDEN=1 go test ./internal/commands/... -run TestSearchPhase1Craft`, inspect the output (should be a single card "On craft" by Martha Beck), verify clean.

- [ ] **Step 4: Audit fixture-based search tests.**

The Phase 0 `TestSearchOAuth*` tests used the fixture source. After Task 4, those still work because the DB doesn't exist in test (fallback to fixture). Confirm they still pass. If they fail because of env-var contamination from the new import test, ensure each test calls `os.Unsetenv("CAIRN_HOME")` on a deferred basis OR migrate them to also use a temp DB. Safer: set `CAIRN_HOME` to a path with no DB for the fixture-based tests:

```go
tmp := t.TempDir()
os.Setenv("CAIRN_HOME", tmp)
defer os.Unsetenv("CAIRN_HOME")
```

and update each Phase 0 search test with that setup block. This ensures isolation.

- [ ] **Step 5: Run full suite.**

Run: `cd ~/cairn && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/commands/ && \
  git -c commit.gpgsign=false commit -m "feat(search): route through source.Open(), SQLite when available"
```

---

### Task 16: Real `cairn get` via handle lookup

**Files:**
- Modify: `internal/commands/get.go`
- Modify: `internal/commands/get_test.go`
- Create: `internal/commands/testdata/golden/get_phase1_at1.txt`

- [ ] **Step 1: Update `get.go` to use `source.Open(cairnDBPath())`.**

Same swap as search.go. Delete the `newSource` call.

- [ ] **Step 2: Add a Phase 1 test that exercises the full flow: import, search (to populate handles), get @1.**

```go
func TestGetPhase1AfterSearch(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("CAIRN_HOME", tmp)
	defer os.Unsetenv("CAIRN_HOME")

	importSample(t) // helper: runs `cairn import <sample dir>` via the command
	searchCraft(t)  // helper: runs `cairn search craft` so handles are saved

	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"get", "@1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "get_phase1_at1.txt", out.String())
}
```

Implement `importSample` and `searchCraft` as package-local test helpers in a new `commands_testutil_test.go` file.

- [ ] **Step 3: Generate golden, inspect, confirm "On craft" card is rendered with Martha Beck / 2026-03-18 / craft tags.**

- [ ] **Step 4: Audit Phase 0 get tests.**

`TestGetByHandle` used fixtures. Migrate it to the isolated temp-dir pattern so it still uses fixtures (no DB). `TestGetUnknownHandle` error should come from the fixture source (`fixtures.ByHandle(99)`), same as before.

- [ ] **Step 5: Run full suite, commit.**

```bash
git add internal/commands/ && \
  git -c commit.gpgsign=false commit -m "feat(get): resolve @handle via SQLite handles table"
```

---

### Task 17: Real `cairn open`

**Files:**
- Modify: `internal/commands/open.go`
- Modify: `internal/commands/open_test.go`

- [ ] **Step 1: Make `cairn open` actually open the URL via the OS default browser, behind a flag.**

Default behavior: print "Would open: <url>" (no browser invocation) unless `--launch` is passed. Why: tests and Phase 0 screenplay expect the non-launching behavior. A real user runs `cairn open @1 --launch` (or we change the default once we're confident).

Actually — reread spec: `cairn open` "Open in MyMind browser". That implies launching. Compromise: default to launching, but if stdout is not a TTY or `CAIRN_DRY_OPEN=1` is set, fall back to the Phase 0 "Would open" message. Tests set the env var.

```go
package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <card>",
		Short: "Open a card in the default browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			n, err := strconv.Atoi(strings.TrimPrefix(args[0], "@"))
			if err != nil {
				fmt.Fprintf(out, "Invalid handle: %q.\n", args[0])
				return nil
			}
			src, err := source.Open(cairnDBPath())
			if err != nil {
				return err
			}
			c, err := src.ByHandle(n)
			if err != nil {
				fmt.Fprintln(out, err.Error())
				return nil
			}
			url := c.URL
			if url == "" {
				url = fmt.Sprintf("https://access.mymind.com/cards/%s", c.MyMindID)
			}
			if os.Getenv("CAIRN_DRY_OPEN") == "1" {
				fmt.Fprintf(out, "Would open: %s\n", url)
				return nil
			}
			return launchBrowser(out, url)
		},
	}
}

func launchBrowser(out io.Writer, url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		fmt.Fprintf(out, "Open this URL manually: %s\n", url)
		return nil
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(out, "Failed to open browser: %v\nURL: %s\n", err, url)
		return nil
	}
	fmt.Fprintf(out, "Opened: %s\n", url)
	return nil
}
```

- [ ] **Step 2: Update the test to set `CAIRN_DRY_OPEN=1`.**

Modify `TestOpenByHandle`:

```go
func TestOpenByHandle(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("CAIRN_HOME", tmp)
	os.Setenv("CAIRN_DRY_OPEN", "1")
	defer os.Unsetenv("CAIRN_HOME")
	defer os.Unsetenv("CAIRN_DRY_OPEN")

	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"open", "@1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "open_at1.txt", out.String())
}
```

The Phase 0 `open_at1.txt` golden mentions "Phase 0 fake". Update the golden to the new shape:

```
Would open: https://datatracker.ietf.org/doc/html/rfc8628
```

(single line, nothing else). This is Phase 1's quieter output.

- [ ] **Step 3: Run, verify PASS.**

- [ ] **Step 4: Commit.**

```bash
git add internal/commands/ && \
  git -c commit.gpgsign=false commit -m "feat(open): OS-native browser launch with CAIRN_DRY_OPEN escape hatch"
```

---

### Task 18: Real `cairn status` from SQLite

**Files:**
- Modify: `internal/commands/status.go`
- Modify: `internal/commands/status_test.go`
- Create: `internal/commands/testdata/golden/status_imported.txt`

- [ ] **Step 1: Write the failing test `TestStatusImported`.**

```go
func TestStatusImported(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("CAIRN_HOME", tmp)
	defer os.Unsetenv("CAIRN_HOME")
	importSample(t)

	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	outStr := strings.ReplaceAll(out.String(), tmp, "<TMP>")
	// Replace last-import timestamp with <TS> so golden stays stable.
	outStr = replaceTimestamp(outStr)
	golden.Assert(t, "status_imported.txt", outStr)
}
```

Add a `replaceTimestamp` helper that matches `2026-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z` and replaces with `<TS>`.

- [ ] **Step 2: Rewrite `status.go`.**

```go
package commands

import (
	"fmt"

	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Library size, last sync, MCP state, permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			dbPath := cairnDBPath()
			src, err := source.Open(dbPath)
			if err != nil {
				return err
			}

			count := src.Count()
			lastImport, hasImport := src.LastImport()

			fmt.Fprintln(out, "cairn 0.1.0-phase1")
			fmt.Fprintln(out)
			if hasImport {
				fmt.Fprintf(out, "library   %d cards · last import %s\n", count, lastImport.Format("2006-01-02T15:04:05Z"))
				fmt.Fprintf(out, "storage   %s\n", dbPath)
			} else {
				fmt.Fprintf(out, "library   %d cards (fixtures; no database yet)\n", count)
				fmt.Fprintln(out, "storage   run `cairn import <path>` to create a database")
			}
			fmt.Fprintln(out, "mcp       not installed")
			fmt.Fprintln(out, "clients   none")
			return nil
		},
	}
}
```

- [ ] **Step 3: Update the Phase 0 TestStatus fixture test.**

The Phase 0 `status.txt` golden must be regenerated — the version string, the "25 cards" count from fixtures, and the "no database" branch all change. Run the fixture-source path (no import) and update the golden:

```go
func TestStatusNoDatabase(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("CAIRN_HOME", tmp)
	defer os.Unsetenv("CAIRN_HOME")

	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "status_nodatabase.txt", out.String())
}
```

Delete the old `TestStatus` and its golden `status.txt`.

- [ ] **Step 4: Generate both new goldens, inspect.**

`status_nodatabase.txt`:
```
cairn 0.1.0-phase1

library   25 cards (fixtures; no database yet)
storage   run `cairn import <path>` to create a database
mcp       not installed
clients   none
```

`status_imported.txt`:
```
cairn 0.1.0-phase1

library   4 cards · last import <TS>
storage   <TMP>/cairn.db
mcp       not installed
clients   none
```

- [ ] **Step 5: Run tests, commit.**

```bash
git add internal/commands/ && \
  git -c commit.gpgsign=false commit -m "feat(status): report live counts from source, show fixture fallback"
```

---

### Task 19: SCREENPLAY.md update

**Files:**
- Modify: `SCREENPLAY.md`

- [ ] **Step 1: Rewrite the import, status, search, get, open sections to reflect Phase 1 behavior.**

Each section's fenced output blocks must match the new Phase 1 goldens byte-for-byte. Keep the find, pack, mcp sections unchanged (they still use Phase 0 fakes).

Add a short header note at the top: "This walkthrough uses the Phase 1 sample export at `testdata/mymind_sample_export/`. Real exports produce more cards and richer content, but the flow is identical."

- [ ] **Step 2: Commit.**

```bash
git add SCREENPLAY.md && \
  git -c commit.gpgsign=false commit -m "docs: refresh SCREENPLAY for Phase 1 real data"
```

---

### Task 20: IMPORT_FORMAT.md + PHASE-1-REPORT.md

**Files:**
- Create: `docs/IMPORT_FORMAT.md`
- Create: `PHASE-1-REPORT.md`

- [ ] **Step 1: Document the assumed import format.**

`docs/IMPORT_FORMAT.md` describes the columns the Phase 1 parser looks for, with synonyms, required-vs-optional, and how unknowns are handled. One screen of prose. Close with: "First real import updates this file if columns differ."

- [ ] **Step 2: Author the phase report.**

`PHASE-1-REPORT.md`:

```markdown
# Phase 1 report

**Shipped.** Real import pipeline, SQLite store, FTS5 shadow index, handle persistence, and the five target commands (import, status, search, get, open) running on real data. Phase 0 fixture source remains available as a fallback when no database exists.

**What's still Phase 0 fake.**
- cairn find (real TUI arrives Phase 2)
- cairn pack / export (Phase 2)
- cairn mcp (Phase 3)
- cairn ask (Phase 4)
- cairn config (defaults display only)
- embeddings (Phase 2)

**Open items surfaced.**
- [list anything found during implementation, e.g., MyMind CSV column names that differed from assumptions]

**Acceptance check.**
Run `cairn import <real-export>` then `cairn search "<remembered query>"`. The remembered card surfaces in the top 5. If not, file an issue against the FTS5 query construction.

**Gate status.** Phase 1 → Phase 2 sign-off is Samay running real queries against his real library for a full day. Sign-off unlocks Phase 2 (bubbletea TUI, pack, export, embeddings).
```

- [ ] **Step 3: Final test + smoke run.**

```bash
cd ~/cairn && go test ./... && go build -o cairn ./cmd/cairn
./cairn import testdata/mymind_sample_export
./cairn status
./cairn search craft
./cairn search type:article
./cairn get @1
./cairn open @1   # should actually launch a browser; skip if that's annoying
rm cairn
```

All must pass. Every line of output should read cleanly.

- [ ] **Step 4: Commit.**

```bash
git add docs/IMPORT_FORMAT.md PHASE-1-REPORT.md && \
  git -c commit.gpgsign=false commit -m "docs: IMPORT_FORMAT and Phase 1 report"
```

---

## Self-review

**Spec coverage.**
- Real export parser: Tasks 10, 11, 12.
- SQLite schema + migrations: Task 2.
- FTS5 shadow table + triggers: Task 2 (schema has triggers baked in).
- FTS5-only retrieval for Phase 1: Task 8.
- `--json`, `--jsonl`, `--plain`, `--limit`: already in place from Phase 0, routed through new Source in Tasks 4 and 15.
- Card-list output per §Output contract: unchanged from Phase 0 render package.
- Handle persistence: Task 9.
- `cairn import`: Task 14.
- `cairn status`: Task 18.
- `cairn search`: Task 15.
- `cairn get`: Task 16.
- `cairn open`: Task 17.
- Tombstoning + 30-day hard delete: Tasks 12, 13.

**Placeholder scan.** No TBD / TODO / "implement later" in the plan body. Every code block is complete.

**Type consistency.** `source.Source`, `source.Filters`, `render.Match`, `cards.Card`, `sqlite.SQLiteSource`, `sqlite.Migrate`, `importer.Import`, `importer.Result` are used identically across every task they appear in.

**Minor cosmetic gap.** `--plain` is not explicitly honored by the command-level code paths (they default to plain when `--json` and `--jsonl` are both false). That's fine for Phase 1; the flag exists for future symmetry.

**Golden refactor.** Several Phase 0 goldens (status.txt, import_ok.txt, import_err_notfound.txt, open_at1.txt) get replaced by Phase 1 variants. Each deletion is called out in its task.

**Known risk.** The assumed MyMind CSV schema may not match reality. Task 14 and Task 20 both account for this: the parser is permissive, produces warnings, and IMPORT_FORMAT.md is explicitly a living document.

