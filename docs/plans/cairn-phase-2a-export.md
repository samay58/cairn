# Cairn Phase 2a Implementation Plan: Phoenix Mirror

**Goal:** Replace the Phase 0 fake `cairn export` with a real Phoenix-vault mirror. Every card with a body becomes a markdown file at `~/phoenix/Clippings/MyMind/{YYYY-MM-DD}-{slug}.md`, every media asset lands content-addressed at `~/phoenix/Clippings/MyMind/_media/{sha}.{ext}` with a relative link from the card's markdown. Along the way, fix the Phase 1 open item where every `media` row was written with an empty `card_id`: the real MyMind export keys attachments by filename-stem equal to `mymind_id`, so cards can own their media now.

**Architecture:** Add a new `internal/phoenix` package that knows nothing about SQLite or commands; it takes cards and their media, writes files to a configurable vault root, and reports what it wrote. The command layer (`internal/commands/export.go`) orchestrates: reads cards and their media from the `Source`, asks the phoenix writer to emit, reports. Source gains one method (`MediaFor(cardID)`) so `SQLiteSource` can serve per-card attachments. Importer gains filename-stem ↔ mymind_id lookup so media rows land with real `card_id`. A one-time schema migration (0002) turns on `PRAGMA foreign_keys = ON` and hard-deletes any legacy rows with empty `card_id` so the FK isn't immediately violated.

**Tech Stack:** Go 1.26, stdlib `os`, `io`, `path/filepath`, `crypto/sha256`, existing `modernc.org/sqlite`, existing `spf13/cobra` + render package. New internal package `internal/phoenix`; no new third-party deps. Markdown frontmatter is hand-rolled YAML (the spec forbids dep-creep and this is 6 keys, flat).

**Spec reference:** `docs/design/cairn-design.md` §"Phoenix bridge" (line 176-180) and §"Phase 2. TUI, packs, Phoenix mirror". `PHASE-1-REPORT.md` open items (media linkage; `source.Open` cycle). Phase 1 plan: `docs/plans/cairn-phase-1.md`.

---

## Scope boundaries

**In Phase 2a:**

- Real `cairn export` command:
  - Writes `{YYYY-MM-DD}-{slug}.md` per card under a configurable vault root (default `~/phoenix/Clippings/MyMind/`).
  - Writes media under `_media/{sha}.{ext}` with relative `![](../_media/...)` or `[filename](../_media/...)` links from the card.
  - `--dry-run` prints what would happen without touching disk (preserves Phase 0's dry-run output shape for discoverability).
  - `--to <path>` overrides the vault root. Used by tests, and by people whose Phoenix lives elsewhere.
  - Idempotent: re-running with no changes writes no new bytes (hash-compare before overwrite).
  - Collision handling: if two cards slug to the same string on the same day, suffix `-2`, `-3`, ….
- Media-to-card linkage in importer:
  - `media` rows now carry the real `card_id`.
  - Orphan media (no matching card stem) skipped with a warning, not inserted with empty `card_id`.
- Schema migration 0002:
  - Deletes any legacy `media` rows with `card_id = ''` (Phase 1 leftovers).
  - Turns `PRAGMA foreign_keys = ON` in the `Open` pragma block so the constraint actually enforces.
- `Source` interface gains `MediaFor(cardID string) []cards.Media`. `FixtureSource` returns empty slice; `SQLiteSource` queries the `media` table.
- `cairn export` wires through the real `src`, not the fixture.

**Deferred to later phases (keep current state):**

- `cairn find` — real bubbletea TUI in Phase 2d.
- `cairn pack` — Phase 2b (real retrieval + profiles).
- `cairn ask` — Phase 4.
- `cairn mcp *` — Phase 3.
- Embeddings, vector search, RRF, "why shown" upgrades — Phase 2c.
- Chunk-level `render.Match` — Phase 2c (card-level Match stays for now).
- Moving `source.Filters` out of `internal/source` — only if a new import cycle forces it; this plan does not.
- Phoenix-side index file (`_index.md` or similar) — not in spec. If it comes up, file it as Phase 5.
- Watcher or daemon that auto-exports on import — spec explicitly scopes to on-demand.

**Explicit non-goals:**

- No OCR, no reprocessing of card bodies during export — whatever the importer stored is what gets written.
- No delete-on-export: if a card is tombstoned, export leaves the existing markdown file alone. Phase 5+ concern.
- No URL-driven media resolution (downloading remote `cards.url` assets). Only files physically present in the MyMind export mirror through.

---

## Architecture notes

### What the real MyMind export actually looks like

Confirmed against the live export at `~/phoenix/Clippings/mymind/` (51-line `cards.csv`, one `MDE0O3xIKzJh4Y.pdf`):

- `cards.csv` header: `id,type,title,url,content,note,tags,created`.
- No `media` or `attachment` column.
- Attachments live at export root (no `media/` subdir).
- **Per-card linkage is derivable from filename:** the attachment `MDE0O3xIKzJh4Y.pdf` corresponds to card `id=MDE0O3xIKzJh4Y` in `cards.csv`. This is a MyMind convention, not an assumption.

Phase 1's importer scanned media but couldn't link it; the comment at `internal/importer/importer.go:262` explicitly defers linkage to Phase 2. This plan lands it.

### The `internal/phoenix` package

A narrow, dependency-free writer. Takes simple value types (cards + media), returns a report. Knows nothing about SQLite, commands, or flags.

```go
// internal/phoenix/phoenix.go
package phoenix

type Writer struct {
    Root   string   // e.g. "~/phoenix/Clippings/MyMind" (expanded)
    DryRun bool
}

type CardBundle struct {
    Card  cards.Card
    Media []cards.Media
}

type WriteReport struct {
    CardsWritten   int      // new files
    CardsUnchanged int      // identical content, no rewrite
    MediaWritten   int
    MediaSkipped   int      // already present with matching sha
    Warnings       []string
}

func (w *Writer) Write(bundles []CardBundle) (WriteReport, error)
```

Internals:

- `markdown.go` — frontmatter + body composition. No tables, no emoji, no em-dashes (kill-list compliance extends here).
- `paths.go` — `dailySlug(capturedAt, title)`, `collisionSuffix`, `mediaPath(sha, ext)`, `relMediaLink(fromCardPath, mediaPath)`.
- `writer.go` — the `Write` method. Creates dirs on demand, hashes existing files to skip unchanged writes, copies media with `io.Copy` + sha verification.

### The markdown shape

```markdown
---
mymind_id: MDE0O3xIKzJh4Y
url: https://example.com/post
tags: [ai, search, llm]
captured_at: 2026-04-20T20:55:33Z
kind: article
---

# How AI web search works

{body paragraph one…}

{body paragraph two…}

## Attachments

- [attachment.pdf](../_media/3f5a…b1.pdf)
```

- Filename: `2026-04-20-how-ai-web-search-works.md`. Slug from `title` lowercased, non-alphanumerics collapsed to `-`, trimmed, truncated to 60 chars.
- Frontmatter YAML is hand-composed (flat, six keys). Tags encoded as JSON-style array so commas-in-tag-names work.
- Body = card body exactly as the importer stored it (including any merged `note` annotation — the Phase 1 importer already merges `note` into `content`; no extra work here).
- Attachments section only present when `len(bundle.Media) > 0`.

### Media file content-addressed layout

`_media/3f/5a/3f5abc…b1.pdf` two-level fan-out by the first four hex chars of the sha, so a vault with 10k media assets stays indexable. Card markdown links out with `../_media/3f/5a/...pdf`. If the file already exists on disk and its sha matches, skip.

### Importer change: filename-stem ↔ mymind_id

Current `internal/importer/importer.go:152` inserts media with empty `card_id`. Replace with:

```go
stem := strings.TrimSuffix(filepath.Base(it.Path), filepath.Ext(it.Path))
cardID, ok := cardIDByMyMindID[stem]
if !ok {
    r.Warnings = append(r.Warnings, fmt.Sprintf("media %s: no matching card id", stem))
    continue
}
if _, err := tx.Exec(`INSERT INTO media(card_id, kind, path, sha256, mime) VALUES (?, ?, ?, ?, ?)`,
    cardID, mediaKind, it.Path, it.SHA256, it.Mime); err != nil { … }
```

`cardIDByMyMindID` is built by iterating `parsed` before the media loop.

### Schema migration 0002

```sql
-- 0002_media_fk_cleanup.sql
DELETE FROM media WHERE card_id = '' OR card_id IS NULL;
```

And `Open` gains one pragma: `PRAGMA foreign_keys = ON`. This only bites if someone tries to insert another empty `card_id` row — which Phase 2a's importer no longer does.

### `Source` interface extension

```go
// internal/source/source.go additions
type Media struct {
    Kind   string
    Path   string
    SHA256 string
    Mime   string
}

// added to Source interface:
MediaFor(cardID string) []Media
```

`FixtureSource.MediaFor` returns nil. `SQLiteSource.MediaFor` does a simple `SELECT kind, path, sha256, mime FROM media WHERE card_id = ?` with `ORDER BY path`.

Add a `cards.Media` type (`internal/cards/media.go`) so phoenix doesn't depend on `source`. `source.Media` is just a re-export or a thin alias. Cleaner: put `Media` in `internal/cards` and make both source and phoenix use it.

### Command wiring

`internal/commands/root.go` currently passes `fixture` to `newExportCmd`:

```go
newExportCmd(fixture),
```

Change to `newExportCmd(src)`. Because `src` is the real `SQLiteSource` when the DB exists, and `FixtureSource` when it doesn't (falls back to the Phase 0 fake output naturally, since no cards surface).

### Dry-run output preservation

The existing Phase 0 dry-run output (`"Mirroring N cards to ..."` plus a three-file sample) is friendly; keep its shape so the command's plain-text contract doesn't break discoverability. The new dry-run differs in one thing: instead of `"Phase 0: nothing written to disk."` it says `"Dry run. 0 files written. Remove --dry-run to write."`.

JSON and JSONL modes get real fields: `cards_written`, `cards_unchanged`, `media_written`, `media_skipped`, `warnings`, `path`.

---

## File structure

```
internal/
  cards/
    media.go                 new: Media value type
  phoenix/
    phoenix.go               new: package doc, Writer, CardBundle, WriteReport
    phoenix_test.go
    markdown.go              new: frontmatter + body composition
    markdown_test.go
    paths.go                 new: slug, dated filename, collision suffix, media path, relative link
    paths_test.go
    writer.go                new: Write implementation
    writer_test.go

  source/
    source.go                modify: add MediaFor; add Media re-export or alias

  storage/sqlite/
    sqlite.go                modify: add foreign_keys pragma, add MediaFor method
    sqlite_test.go           modify: assert MediaFor
    schema/
      0002_media_fk_cleanup.sql   new

  importer/
    importer.go              modify: filename-stem ↔ mymind_id lookup, skip orphans
    importer_test.go         modify: assert media has real card_id

  fixtures/
    source.go                modify: MediaFor returns nil

  commands/
    export.go                rewrite: call phoenix.Writer via src
    export_test.go           modify: golden files exercise real writer + temp vault
    testdata/golden/
      export_dryrun.txt      new
      export_real.txt        new
      export_empty.txt       new
    root.go                  modify: pass src (not fixture) to newExportCmd

docs/
  IMPORT_FORMAT.md           modify: document filename-stem → mymind_id linkage
```

---

## Conventions

- **TDD.** Start from the leaves: `slug`, `paths`, `markdown`, then `writer`, then the command wiring, then the importer change, then the migration. Every step is write-test-first.
- **Integration tests use `t.TempDir()` for the vault root** and `testdata/mymind_sample_export/` for the import fixture (Phase 1 already has this layout).
- **Golden files continue for command output.** `--to <path>` makes vault paths deterministic for goldens.
- **Commits per task.** No WIP commits. Use `git -c commit.gpgsign=false commit` everywhere.
- **No em-dashes. No emoji. Sentence case.** Spec kill-list stays in force.
- **Error messages: name the failed operation and the last successful state.** Spec §"Output contract".
- **No third-party YAML / markdown libs.** Hand-roll. Six frontmatter keys, zero templates.

---

## Task list

### Task 1: Add `cards.Media` type

**Files:** create `internal/cards/media.go`, `internal/cards/media_test.go`.

- [ ] **Step 1: Write the failing test.**

```go
// internal/cards/media_test.go
package cards

import "testing"

func TestMediaZeroValue(t *testing.T) {
	var m Media
	if m.Kind != "" || m.Path != "" || m.SHA256 != "" || m.Mime != "" {
		t.Fatalf("expected zero Media, got %+v", m)
	}
}

func TestMediaFieldsExported(t *testing.T) {
	m := Media{Kind: "document", Path: "x.pdf", SHA256: "abc", Mime: "application/pdf"}
	if m.Kind != "document" || m.Mime != "application/pdf" {
		t.Fatalf("fields did not round-trip")
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/cards/...`
Expected: FAIL — `undefined: Media`.

- [ ] **Step 3: Implement.**

```go
// internal/cards/media.go
package cards

// Media is an asset attached to a card. Paths are filesystem paths as seen at
// import time; callers normalise before display.
type Media struct {
	Kind   string // "image" | "video" | "document" | "other"
	Path   string
	SHA256 string
	Mime   string
}
```

- [ ] **Step 4: Run to verify pass.**

Run: `go test ./internal/cards/...`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
cd ~/cairn && git add internal/cards/media.go internal/cards/media_test.go && \
  git -c commit.gpgsign=false commit -m "feat(cards): add Media value type"
```

---

### Task 2: Extend `Source` interface with `MediaFor`

**Files:** modify `internal/source/source.go`; modify `internal/fixtures/source.go` (or wherever `FixtureSource` lives); modify `internal/storage/sqlite/sqlite.go`; update any other `Source` implementors; add/modify tests.

- [ ] **Step 1: Add a contract test.**

```go
// internal/source/source_contract_test.go
package source_test

import (
	"testing"

	"github.com/samay58/cairn/internal/source"
)

func TestFixtureSourceMediaForReturnsEmpty(t *testing.T) {
	src := source.NewFixtureSource()
	got := src.MediaFor("anything")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/source/...`
Expected: FAIL — `MediaFor undefined` or the fixture source doesn't implement the interface.

- [ ] **Step 3: Extend the interface and stub the fixture implementation.**

Edit `internal/source/source.go`, add to the `Source` interface:

```go
MediaFor(cardID string) []cards.Media
```

Edit `FixtureSource` (Phase 1 file):

```go
func (f *FixtureSource) MediaFor(cardID string) []cards.Media { return nil }
```

- [ ] **Step 4: Add the SQLite implementation.**

Edit `internal/storage/sqlite/sqlite.go`:

```go
func (s *SQLiteSource) MediaFor(cardID string) []cards.Media {
	rows, err := s.DB.Query(
		`SELECT kind, path, coalesce(sha256,''), coalesce(mime,'')
		 FROM media WHERE card_id = ? ORDER BY path`, cardID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []cards.Media
	for rows.Next() {
		var m cards.Media
		if err := rows.Scan(&m.Kind, &m.Path, &m.SHA256, &m.Mime); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}
```

- [ ] **Step 5: Run the whole suite to confirm interface satisfaction.**

Run: `go test ./...`
Expected: PASS (after fixture and sqlite both implement the new method; compile failure in any other implementor surfaces here).

- [ ] **Step 6: Commit.**

```bash
cd ~/cairn && git add internal/source/source.go internal/source/source_contract_test.go \
  internal/fixtures/source.go internal/storage/sqlite/sqlite.go && \
  git -c commit.gpgsign=false commit -m "feat(source): add MediaFor to Source interface"
```

---

### Task 3: Slug + dated filename helpers

**Files:** create `internal/phoenix/paths.go`, `internal/phoenix/paths_test.go`.

- [ ] **Step 1: Write the failing tests.**

```go
// internal/phoenix/paths_test.go
package phoenix

import (
	"testing"
	"time"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"How AI web search works":               "how-ai-web-search-works",
		"Claude & the meaning of life":          "claude-the-meaning-of-life",
		"   leading / trailing   ":              "leading-trailing",
		"éléphant":                              "elephant",
		"":                                      "untitled",
		"A/B Testing with 200% coverage":        "a-b-testing-with-200-coverage",
	}
	for in, want := range cases {
		got := Slug(in)
		if got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDailyFilename(t *testing.T) {
	ts := time.Date(2026, 4, 20, 20, 55, 33, 0, time.UTC)
	got := DailyFilename(ts, "How AI web search works")
	want := "2026-04-20-how-ai-web-search-works.md"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDailyFilenameTruncatesLongSlugs(t *testing.T) {
	ts := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	long := "A very long title that keeps on going and going past the truncation threshold we set"
	got := DailyFilename(ts, long)
	// date (10) + "-" (1) + slug (<=60) + ".md" (3) = <=74
	if len(got) > 74 {
		t.Fatalf("filename too long: %q (%d)", got, len(got))
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/phoenix/...`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement.**

```go
// internal/phoenix/paths.go
package phoenix

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const slugMaxLen = 60

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func Slug(title string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	ascii, _, _ := transform.String(t, title)
	s := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(ascii), "-"), "-")
	if s == "" {
		return "untitled"
	}
	if len(s) > slugMaxLen {
		s = strings.TrimRight(s[:slugMaxLen], "-")
	}
	return s
}

func DailyFilename(capturedAt time.Time, title string) string {
	return capturedAt.UTC().Format("2006-01-02") + "-" + Slug(title) + ".md"
}
```

- [ ] **Step 4: Add golang.org/x/text dep.**

Run:
```bash
cd ~/cairn && go get golang.org/x/text@latest && go mod tidy
```

- [ ] **Step 5: Run to verify pass.**

Run: `go test ./internal/phoenix/...`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
cd ~/cairn && git add go.mod go.sum internal/phoenix/paths.go internal/phoenix/paths_test.go && \
  git -c commit.gpgsign=false commit -m "feat(phoenix): slug and daily filename helpers"
```

---

### Task 4: Collision suffix + media path helpers

**Files:** extend `internal/phoenix/paths.go`, `internal/phoenix/paths_test.go`.

- [ ] **Step 1: Write the failing tests.**

```go
// append to paths_test.go
func TestCollisionSuffix(t *testing.T) {
	exists := map[string]bool{
		"2026-04-20-hello.md":   true,
		"2026-04-20-hello-2.md": true,
	}
	got := UniqueFilename("2026-04-20-hello.md", func(name string) bool { return exists[name] })
	want := "2026-04-20-hello-3.md"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMediaPath(t *testing.T) {
	got := MediaRelPath("3f5abc0000000000000000000000000000000000000000000000000000000000", "pdf")
	want := "_media/3f/5a/3f5abc0000000000000000000000000000000000000000000000000000000000.pdf"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRelMediaLinkFromCard(t *testing.T) {
	got := RelMediaLink("_media/3f/5a/3f5a.pdf")
	want := "_media/3f/5a/3f5a.pdf"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/phoenix/...`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement.**

```go
// append to paths.go
import "fmt"

func UniqueFilename(base string, exists func(string) bool) string {
	if !exists(base) {
		return base
	}
	stem := strings.TrimSuffix(base, ".md")
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s-%d.md", stem, i)
		if !exists(cand) {
			return cand
		}
	}
}

func MediaRelPath(sha string, ext string) string {
	if len(sha) < 4 {
		return fmt.Sprintf("_media/%s.%s", sha, ext)
	}
	return fmt.Sprintf("_media/%s/%s/%s.%s", sha[:2], sha[2:4], sha, ext)
}

func RelMediaLink(relPath string) string {
	// Card markdown lives at vault root (alongside _media/), so a relative link
	// is just the stored path.
	return relPath
}
```

- [ ] **Step 4: Run to verify pass.**

Run: `go test ./internal/phoenix/...`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
cd ~/cairn && git add internal/phoenix/paths.go internal/phoenix/paths_test.go && \
  git -c commit.gpgsign=false commit -m "feat(phoenix): collision suffix and media path helpers"
```

---

### Task 5: Markdown renderer (frontmatter + body)

**Files:** create `internal/phoenix/markdown.go`, `internal/phoenix/markdown_test.go`.

- [ ] **Step 1: Write the failing test.**

```go
// internal/phoenix/markdown_test.go
package phoenix

import (
	"strings"
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

func TestRenderMarkdownBasicArticle(t *testing.T) {
	c := cards.Card{
		ID:         "row1",
		MyMindID:   "MDE0O3xIKzJh4Y",
		Kind:       cards.KindArticle,
		Title:      "How AI web search works",
		URL:        "https://example.com/post",
		Body:       "First paragraph.\n\nSecond paragraph.",
		Tags:       []string{"ai", "search"},
		CapturedAt: time.Date(2026, 4, 20, 20, 55, 33, 0, time.UTC),
	}
	got := RenderMarkdown(c, nil)
	want := "---\n" +
		"mymind_id: MDE0O3xIKzJh4Y\n" +
		"url: https://example.com/post\n" +
		"tags: [\"ai\", \"search\"]\n" +
		"captured_at: 2026-04-20T20:55:33Z\n" +
		"kind: article\n" +
		"---\n\n" +
		"# How AI web search works\n\n" +
		"First paragraph.\n\n" +
		"Second paragraph.\n"
	if got != want {
		t.Fatalf("markdown mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderMarkdownWithMedia(t *testing.T) {
	c := cards.Card{
		ID:         "row1",
		MyMindID:   "MDE0O3xIKzJh4Y",
		Kind:       cards.KindArticle,
		Title:      "With attachment",
		Body:       "hello",
		CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
	}
	media := []MediaRef{
		{Filename: "MDE0O3xIKzJh4Y.pdf", RelPath: "_media/3f/5a/3f5a.pdf"},
	}
	got := RenderMarkdown(c, media)
	if !strings.Contains(got, "## Attachments") {
		t.Fatalf("missing Attachments section")
	}
	if !strings.Contains(got, "[MDE0O3xIKzJh4Y.pdf](_media/3f/5a/3f5a.pdf)") {
		t.Fatalf("missing attachment link")
	}
}

func TestRenderMarkdownEmptyBody(t *testing.T) {
	c := cards.Card{MyMindID: "a", Kind: cards.KindNote, Title: "t",
		CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)}
	got := RenderMarkdown(c, nil)
	if !strings.Contains(got, "# t\n") {
		t.Fatalf("expected title heading")
	}
	if strings.Contains(got, "\n\n\n") {
		t.Fatalf("unexpected triple newline in empty-body output:\n%s", got)
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/phoenix/...`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement.**

```go
// internal/phoenix/markdown.go
package phoenix

import (
	"fmt"
	"strings"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

type MediaRef struct {
	Filename string // human-readable filename, e.g. "MDE0O3xIKzJh4Y.pdf"
	RelPath  string // e.g. "_media/3f/5a/3f5a.pdf"
}

func RenderMarkdown(c cards.Card, media []MediaRef) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("mymind_id: ")
	b.WriteString(c.MyMindID)
	b.WriteString("\n")
	if c.URL != "" {
		b.WriteString("url: ")
		b.WriteString(c.URL)
		b.WriteString("\n")
	}
	b.WriteString("tags: ")
	b.WriteString(encodeTags(c.Tags))
	b.WriteString("\n")
	b.WriteString("captured_at: ")
	b.WriteString(c.CapturedAt.UTC().Format(time.RFC3339))
	b.WriteString("\n")
	b.WriteString("kind: ")
	b.WriteString(kindName(c.Kind))
	b.WriteString("\n---\n\n")

	b.WriteString("# ")
	b.WriteString(c.Title)
	b.WriteString("\n")

	body := strings.TrimSpace(c.Body)
	if body != "" {
		b.WriteString("\n")
		b.WriteString(body)
		b.WriteString("\n")
	}

	if len(media) > 0 {
		b.WriteString("\n## Attachments\n\n")
		for _, m := range media {
			fmt.Fprintf(&b, "- [%s](%s)\n", m.Filename, m.RelPath)
		}
	}
	return b.String()
}

func encodeTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	quoted := make([]string, len(tags))
	for i, t := range tags {
		quoted[i] = `"` + strings.ReplaceAll(t, `"`, `\"`) + `"`
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func kindName(k cards.Kind) string {
	switch k {
	case cards.KindArticle:
		return "article"
	case cards.KindImage:
		return "image"
	case cards.KindQuote:
		return "quote"
	case cards.KindNote:
		return "note"
	default:
		return string(k)
	}
}
```

- [ ] **Step 4: Run to verify pass.**

Run: `go test ./internal/phoenix/...`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
cd ~/cairn && git add internal/phoenix/markdown.go internal/phoenix/markdown_test.go && \
  git -c commit.gpgsign=false commit -m "feat(phoenix): render markdown with frontmatter and attachments"
```

---

### Task 6: Writer — dry run path

**Files:** create `internal/phoenix/phoenix.go`, `internal/phoenix/writer.go`, `internal/phoenix/writer_test.go`.

- [ ] **Step 1: Write the failing test.**

```go
// internal/phoenix/writer_test.go
package phoenix

import (
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

func TestWriterDryRunWritesNothing(t *testing.T) {
	tmp := t.TempDir()
	w := &Writer{Root: tmp, DryRun: true}
	bundles := []CardBundle{{
		Card: cards.Card{
			ID: "r1", MyMindID: "m1", Kind: cards.KindArticle,
			Title: "T", Body: "body",
			CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		},
	}}
	report, err := w.Write(bundles)
	if err != nil {
		t.Fatal(err)
	}
	if report.CardsWritten != 1 {
		t.Errorf("report.CardsWritten = %d, want 1", report.CardsWritten)
	}
	entries, _ := readDir(tmp)
	if len(entries) != 0 {
		t.Errorf("dry-run wrote %d entries, want 0", len(entries))
	}
}
```

(`readDir` is a tiny helper in the test file using `os.ReadDir`.)

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/phoenix/...`
Expected: FAIL — undefined `Writer`, `CardBundle`, etc.

- [ ] **Step 3: Implement the shell.**

```go
// internal/phoenix/phoenix.go
package phoenix

import "github.com/samay58/cairn/internal/cards"

type Writer struct {
	Root   string
	DryRun bool
}

type CardBundle struct {
	Card  cards.Card
	Media []cards.Media
}

type WriteReport struct {
	CardsWritten   int
	CardsUnchanged int
	MediaWritten   int
	MediaSkipped   int
	Warnings       []string
}
```

```go
// internal/phoenix/writer.go
package phoenix

import (
	"os"
	"path/filepath"
)

func (w *Writer) Write(bundles []CardBundle) (WriteReport, error) {
	var r WriteReport
	if !w.DryRun {
		if err := os.MkdirAll(w.Root, 0o755); err != nil {
			return r, err
		}
	}
	existing := make(map[string]bool)
	for _, b := range bundles {
		name := DailyFilename(b.Card.CapturedAt, b.Card.Title)
		name = UniqueFilename(name, func(n string) bool {
			if existing[n] {
				return true
			}
			_, err := os.Stat(filepath.Join(w.Root, n))
			return err == nil
		})
		existing[name] = true
		r.CardsWritten++
		if w.DryRun {
			continue
		}
		// real write path lands in Task 7; leave as dry-run-only for now.
	}
	return r, nil
}
```

- [ ] **Step 4: Run to verify pass.**

Run: `go test ./internal/phoenix/...`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
cd ~/cairn && git add internal/phoenix/phoenix.go internal/phoenix/writer.go internal/phoenix/writer_test.go && \
  git -c commit.gpgsign=false commit -m "feat(phoenix): Writer shell with dry-run accounting"
```

---

### Task 7: Writer — real card markdown write + idempotence

**Files:** extend `internal/phoenix/writer.go`, `internal/phoenix/writer_test.go`.

- [ ] **Step 1: Write the failing tests.**

```go
// append to writer_test.go
import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

func TestWriterWritesCardMarkdown(t *testing.T) {
	tmp := t.TempDir()
	w := &Writer{Root: tmp}
	bundles := []CardBundle{{
		Card: cards.Card{
			MyMindID: "m1", Kind: cards.KindArticle, Title: "Hello world",
			Body: "Para one.\n\nPara two.",
			CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		},
	}}
	r, err := w.Write(bundles)
	if err != nil {
		t.Fatal(err)
	}
	if r.CardsWritten != 1 {
		t.Fatalf("CardsWritten = %d", r.CardsWritten)
	}
	p := filepath.Join(tmp, "2026-04-20-hello-world.md")
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("# Hello world")) {
		t.Fatalf("missing title in %s", got)
	}
	if !bytes.Contains(got, []byte("mymind_id: m1")) {
		t.Fatalf("missing frontmatter id")
	}
}

func TestWriterIdempotent(t *testing.T) {
	tmp := t.TempDir()
	w := &Writer{Root: tmp}
	bundles := []CardBundle{{
		Card: cards.Card{MyMindID: "m1", Kind: cards.KindNote, Title: "T", Body: "x",
			CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)},
	}}
	if _, err := w.Write(bundles); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(tmp, "2026-04-20-t.md")
	firstStat, _ := os.Stat(p)
	// Second call with identical content.
	r, err := w.Write(bundles)
	if err != nil {
		t.Fatal(err)
	}
	if r.CardsUnchanged != 1 || r.CardsWritten != 0 {
		t.Fatalf("expected Unchanged=1 Written=0, got Written=%d Unchanged=%d", r.CardsWritten, r.CardsUnchanged)
	}
	secondStat, _ := os.Stat(p)
	if !secondStat.ModTime().Equal(firstStat.ModTime()) {
		t.Errorf("mtime changed on idempotent rewrite")
	}
}

func contentSHA(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/phoenix/...`
Expected: FAIL — writer currently emits nothing on disk.

- [ ] **Step 3: Extend `Writer.Write`.**

Replace the dry-run stub body with:

```go
for _, b := range bundles {
	name := DailyFilename(b.Card.CapturedAt, b.Card.Title)
	name = UniqueFilename(name, func(n string) bool {
		if existing[n] {
			return true
		}
		_, err := os.Stat(filepath.Join(w.Root, n))
		return err == nil
	})
	existing[name] = true

	// media refs for frontmatter/attachments — Task 8 fills these in.
	var refs []MediaRef
	content := RenderMarkdown(b.Card, refs)

	dest := filepath.Join(w.Root, name)
	if w.DryRun {
		r.CardsWritten++
		continue
	}
	if same, err := sameContent(dest, content); err == nil && same {
		r.CardsUnchanged++
		continue
	}
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return r, err
	}
	r.CardsWritten++
}
```

Helper:

```go
func sameContent(path, content string) (bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return string(existing) == content, nil
}
```

- [ ] **Step 4: Run to verify pass.**

Run: `go test ./internal/phoenix/...`
Expected: PASS (all tests including idempotence).

- [ ] **Step 5: Commit.**

```bash
cd ~/cairn && git add internal/phoenix/writer.go internal/phoenix/writer_test.go && \
  git -c commit.gpgsign=false commit -m "feat(phoenix): write card markdown with idempotence"
```

---

### Task 8: Writer — media copy with content-addressed paths

**Files:** extend `internal/phoenix/writer.go`, `internal/phoenix/writer_test.go`.

- [ ] **Step 1: Write the failing test.**

```go
// append to writer_test.go
func TestWriterCopiesMediaAndLinksFromCard(t *testing.T) {
	tmp := t.TempDir()
	src := t.TempDir()
	pdfPath := filepath.Join(src, "MDE0O3xIKzJh4Y.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4 fake pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha := contentSHA(t, pdfPath)

	w := &Writer{Root: tmp}
	bundles := []CardBundle{{
		Card: cards.Card{MyMindID: "MDE0O3xIKzJh4Y", Kind: cards.KindArticle, Title: "With PDF",
			CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)},
		Media: []cards.Media{{Kind: "document", Path: pdfPath, SHA256: sha, Mime: "application/pdf"}},
	}}
	r, err := w.Write(bundles)
	if err != nil {
		t.Fatal(err)
	}
	if r.MediaWritten != 1 {
		t.Fatalf("MediaWritten = %d, want 1", r.MediaWritten)
	}
	want := filepath.Join(tmp, "_media", sha[:2], sha[2:4], sha+".pdf")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected media at %s: %v", want, err)
	}
	card, _ := os.ReadFile(filepath.Join(tmp, "2026-04-20-with-pdf.md"))
	if !bytes.Contains(card, []byte("_media/"+sha[:2]+"/"+sha[2:4]+"/"+sha+".pdf")) {
		t.Fatalf("card missing relative media link:\n%s", card)
	}
}

func TestWriterMediaSkippedWhenAlreadyPresent(t *testing.T) {
	tmp := t.TempDir()
	src := t.TempDir()
	pdfPath := filepath.Join(src, "x.pdf")
	os.WriteFile(pdfPath, []byte("same bytes"), 0o644)
	sha := contentSHA(t, pdfPath)
	dest := filepath.Join(tmp, "_media", sha[:2], sha[2:4], sha+".pdf")
	os.MkdirAll(filepath.Dir(dest), 0o755)
	os.WriteFile(dest, []byte("same bytes"), 0o644)

	w := &Writer{Root: tmp}
	bundles := []CardBundle{{
		Card:  cards.Card{MyMindID: "m", Kind: cards.KindArticle, Title: "T", CapturedAt: time.Now()},
		Media: []cards.Media{{Path: pdfPath, SHA256: sha, Mime: "application/pdf"}},
	}}
	r, err := w.Write(bundles)
	if err != nil {
		t.Fatal(err)
	}
	if r.MediaWritten != 0 || r.MediaSkipped != 1 {
		t.Fatalf("got Written=%d Skipped=%d", r.MediaWritten, r.MediaSkipped)
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/phoenix/...`
Expected: FAIL — media copy not implemented.

- [ ] **Step 3: Implement media copy.**

In `writer.go`, before the markdown write, collect refs:

```go
refs := make([]MediaRef, 0, len(b.Media))
for _, m := range b.Media {
	rel := MediaRelPath(m.SHA256, extFromPath(m.Path))
	refs = append(refs, MediaRef{Filename: filepath.Base(m.Path), RelPath: rel})
	if w.DryRun {
		r.MediaWritten++
		continue
	}
	dest := filepath.Join(w.Root, rel)
	if existsSha(dest, m.SHA256) {
		r.MediaSkipped++
		continue
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return r, err
	}
	if err := copyFile(m.Path, dest); err != nil {
		r.Warnings = append(r.Warnings, "copy "+m.Path+": "+err.Error())
		continue
	}
	r.MediaWritten++
}
```

Plumb `refs` into `RenderMarkdown(b.Card, refs)`.

Helpers:

```go
func extFromPath(p string) string {
	e := filepath.Ext(p)
	if e == "" {
		return "bin"
	}
	return strings.TrimPrefix(e, ".")
}

func existsSha(dest, wantSha string) bool {
	f, err := os.Open(dest)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == wantSha
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil { return err }
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil { return err }
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
```

Imports needed: `crypto/sha256`, `encoding/hex`, `io`, `strings`.

- [ ] **Step 4: Run to verify pass.**

Run: `go test ./internal/phoenix/...`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
cd ~/cairn && git add internal/phoenix/writer.go internal/phoenix/writer_test.go && \
  git -c commit.gpgsign=false commit -m "feat(phoenix): copy media and link from card markdown"
```

---

### Task 9: Importer — link media to cards by filename-stem

**Files:** modify `internal/importer/importer.go`, `internal/importer/importer_test.go`.

- [ ] **Step 1: Write the failing test.**

```go
// in importer_test.go, add:
func TestImportLinksMediaByFilenameStem(t *testing.T) {
	dir := t.TempDir()
	csv := "\ufeffid,type,title,url,content,note,tags,created\n" +
		"abc,Article,With PDF,,body text,,ai,2026-04-20T00:00:00Z\n"
	if err := os.WriteFile(filepath.Join(dir, "cards.csv"), []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}
	// Attachment whose stem matches the mymind id.
	pdf := []byte("%PDF-1.4 fake")
	if err := os.WriteFile(filepath.Join(dir, "abc.pdf"), pdf, 0o644); err != nil {
		t.Fatal(err)
	}

	db := openTestDB(t)
	if _, err := importer.Import(db, dir); err != nil {
		t.Fatal(err)
	}
	var cnt int
	db.QueryRow(`SELECT count(*) FROM media WHERE card_id = (SELECT id FROM cards WHERE mymind_id='abc')`).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("expected 1 linked media row, got %d", cnt)
	}
	var empty int
	db.QueryRow(`SELECT count(*) FROM media WHERE card_id = ''`).Scan(&empty)
	if empty != 0 {
		t.Fatalf("expected 0 empty-card-id rows, got %d", empty)
	}
}

func TestImportWarnsOnOrphanMedia(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "cards.csv"),
		[]byte("\ufeffid,type,title,url,content,note,tags,created\n"+
			"abc,Article,T,,body,,,2026-04-20T00:00:00Z\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "orphan.pdf"), []byte("x"), 0o644)

	db := openTestDB(t)
	r, err := importer.Import(db, dir)
	if err != nil {
		t.Fatal(err)
	}
	var any bool
	for _, w := range r.Warnings {
		if strings.Contains(w, "orphan") {
			any = true
		}
	}
	if !any {
		t.Fatalf("expected warning about orphan.pdf, got %v", r.Warnings)
	}
}
```

(Use the existing test helper `openTestDB` if present; otherwise inline an `sql.Open("sqlite", ":memory:")` + `sqlite.Migrate`.)

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/importer/...`
Expected: FAIL — media still insert empty `card_id`, warning absent.

- [ ] **Step 3: Implement the linkage.**

In `internal/importer/importer.go`, before the media loop build the lookup map:

```go
cardByMyMind := make(map[string]string, len(parsed))
for _, c := range parsed {
	cardByMyMind[c.MyMindID] = c.ID
}
```

Replace the media insert block with:

```go
for _, it := range items {
	if filepath.Base(it.Path) == "cards.csv" {
		continue
	}
	mediaKind := "other"
	switch {
	case strings.HasPrefix(it.Mime, "image/"):
		mediaKind = "image"
	case strings.HasPrefix(it.Mime, "video/"):
		mediaKind = "video"
	case it.Mime == "application/pdf":
		mediaKind = "document"
	}
	stem := strings.TrimSuffix(filepath.Base(it.Path), filepath.Ext(it.Path))
	cardID, ok := cardByMyMind[stem]
	if !ok {
		r.Warnings = append(r.Warnings, fmt.Sprintf("media %s: no matching card id", filepath.Base(it.Path)))
		continue
	}
	if _, err := tx.Exec(`INSERT INTO media(card_id, kind, path, sha256, mime) VALUES (?, ?, ?, ?, ?)`,
		cardID, mediaKind, it.Path, it.SHA256, it.Mime); err != nil {
		r.Warnings = append(r.Warnings, fmt.Sprintf("media insert %s: %v", it.Path, err))
		continue
	}
	r.MediaCount++
}
```

Delete the old note comment (lines 262-266) that deferred the linkage; the plan does it now.

- [ ] **Step 4: Run to verify pass.**

Run: `go test ./internal/importer/...`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
cd ~/cairn && git add internal/importer/importer.go internal/importer/importer_test.go && \
  git -c commit.gpgsign=false commit -m "feat(importer): link media to cards by filename-stem"
```

---

### Task 10: Schema migration 0002 and FK pragma

**Files:** create `internal/storage/sqlite/schema/0002_media_fk_cleanup.sql`; modify `internal/storage/sqlite/sqlite.go`; add/modify migration tests.

- [ ] **Step 1: Write the failing test.**

```go
// internal/storage/sqlite/migrate_test.go (add case)
func TestMigration0002RemovesEmptyCardIDMedia(t *testing.T) {
	db := mustOpenMem(t)
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	// Insert a legacy row the Phase 1 importer would have produced.
	_, _ = db.Exec(`INSERT INTO cards(id, mymind_id, kind, title, captured_at) VALUES ('r1','m1','a','t','2026-04-20T00:00:00Z')`)
	_, _ = db.Exec(`INSERT INTO media(card_id, kind, path, sha256, mime) VALUES ('','other','/legacy','sha','x/y')`)
	// Re-run migrations — should be a no-op if 0002 already applied; this is
	// the idempotent re-entry test.
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM media WHERE card_id=''`).Scan(&n)
	if n != 0 {
		t.Fatalf("expected 0 empty card_id rows after 0002, got %d", n)
	}
}

func TestOpenEnablesForeignKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.db")
	s, err := Open(path)
	if err != nil { t.Fatal(err) }
	defer s.Close()
	var on int
	s.DB.QueryRow("PRAGMA foreign_keys").Scan(&on)
	if on != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d, want 1", on)
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/storage/sqlite/...`
Expected: FAIL — migration 0002 missing, FK pragma off.

- [ ] **Step 3: Add the migration SQL.**

```sql
-- internal/storage/sqlite/schema/0002_media_fk_cleanup.sql
DELETE FROM media WHERE card_id = '' OR card_id IS NULL;
```

Make sure it's picked up by the embed directive in `internal/storage/sqlite/schema.go` (Phase 1 already globbed the dir; verify no change needed).

- [ ] **Step 4: Enable foreign keys on `Open`.**

In `internal/storage/sqlite/sqlite.go`, extend the pragma list:

```go
for _, pragma := range []string{
	`PRAGMA journal_mode = WAL`,
	`PRAGMA synchronous = NORMAL`,
	`PRAGMA foreign_keys = ON`,
} {
```

- [ ] **Step 5: Run to verify pass.**

Run: `go test ./internal/storage/sqlite/...`
Expected: PASS.

Also run: `go test ./...`
Expected: PASS (importer tests from Task 9 should still pass under FK enforcement because we no longer insert empty `card_id`).

- [ ] **Step 6: Commit.**

```bash
cd ~/cairn && git add internal/storage/sqlite/schema/0002_media_fk_cleanup.sql \
  internal/storage/sqlite/sqlite.go internal/storage/sqlite/migrate_test.go && \
  git -c commit.gpgsign=false commit -m "feat(sqlite): migration 0002 cleans empty media card_id and enables FK enforcement"
```

---

### Task 11: Wire `cairn export` to real writer

**Files:** rewrite `internal/commands/export.go`, `internal/commands/export_test.go`; create goldens under `internal/commands/testdata/golden/`.

- [ ] **Step 1: Write failing golden-based test.**

```go
// internal/commands/export_test.go
func TestExportDryRunGolden(t *testing.T) {
	root, src := setupImportedSource(t) // helper that imports testdata and returns (commandRoot, source)
	vault := t.TempDir()
	got := runCmd(t, root, "export", "--dry-run", "--to", vault)
	assertGolden(t, "testdata/golden/export_dryrun.txt", got)
	// vault must be empty
	entries, _ := os.ReadDir(vault)
	if len(entries) != 0 {
		t.Fatalf("dry-run wrote %d files, want 0", len(entries))
	}
	_ = src
}

func TestExportRealWritesFilesAndGolden(t *testing.T) {
	root, _ := setupImportedSource(t)
	vault := t.TempDir()
	got := runCmd(t, root, "export", "--to", vault)
	assertGolden(t, "testdata/golden/export_real.txt", got)
	matches, _ := filepath.Glob(filepath.Join(vault, "*.md"))
	if len(matches) == 0 {
		t.Fatalf("expected markdown files in vault, got 0")
	}
}
```

(`setupImportedSource` imports `testdata/mymind_sample_export/` into a temp DB, returns a command tree built with that `SQLiteSource`. Phase 1's tests already have something close; extend.)

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/commands/ -run Export`
Expected: FAIL — current export is fake.

- [ ] **Step 3: Rewrite `export.go` around the phoenix writer.**

```go
// internal/commands/export.go
package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/phoenix"
	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

type exportView struct {
	CardsWritten   int      `json:"cards_written"`
	CardsUnchanged int      `json:"cards_unchanged"`
	MediaWritten   int      `json:"media_written"`
	MediaSkipped   int      `json:"media_skipped"`
	Warnings       []string `json:"warnings"`
	Path           string   `json:"path"`
	DryRun         bool     `json:"dry_run"`
}

func newExportCmd(src source.Source) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Mirror cards to the Phoenix vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil { return err }
			dry, _ := cmd.Flags().GetBool("dry-run")
			to, _ := cmd.Flags().GetString("to")
			if to == "" {
				to = defaultExportRoot()
			}

			bundles := collectBundles(src)
			w := &phoenix.Writer{Root: to, DryRun: dry}
			rep, werr := w.Write(bundles)
			if werr != nil {
				return fmt.Errorf("export to %s: %w", to, werr)
			}
			view := exportView{
				CardsWritten:   rep.CardsWritten,
				CardsUnchanged: rep.CardsUnchanged,
				MediaWritten:   rep.MediaWritten,
				MediaSkipped:   rep.MediaSkipped,
				Warnings:       rep.Warnings,
				Path:           to,
				DryRun:         dry,
			}
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(view))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL([]exportView{view}))
			default:
				err = writeExportPlain(out, view)
			}
			return err
		},
	}
	addListFlags(cmd)
	cmd.Flags().Bool("dry-run", false, "Preview without writing")
	cmd.Flags().String("to", "", "Vault root (defaults to ~/phoenix/Clippings/MyMind/)")
	return cmd
}

func collectBundles(src source.Source) []phoenix.CardBundle {
	all := src.All()
	out := make([]phoenix.CardBundle, 0, len(all))
	for _, c := range all {
		out = append(out, phoenix.CardBundle{Card: c, Media: convertMedia(src.MediaFor(c.ID))})
	}
	return out
}

func convertMedia(in []source.Media) []cards.Media {
	out := make([]cards.Media, len(in))
	for i, m := range in {
		out[i] = cards.Media{Kind: m.Kind, Path: m.Path, SHA256: m.SHA256, Mime: m.Mime}
	}
	return out
}

func defaultExportRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "phoenix", "Clippings", "MyMind")
}

func writeExportPlain(out io.Writer, v exportView) error {
	verb := "Wrote"
	if v.DryRun {
		verb = "Would write"
	}
	if _, err := fmt.Fprintf(out, "%s %d cards to %s\n", verb, v.CardsWritten, v.Path); err != nil {
		return err
	}
	if v.CardsUnchanged > 0 {
		fmt.Fprintf(out, "  %d cards unchanged\n", v.CardsUnchanged)
	}
	if v.MediaWritten > 0 || v.MediaSkipped > 0 {
		fmt.Fprintf(out, "  media: %d written, %d skipped\n", v.MediaWritten, v.MediaSkipped)
	}
	for _, wn := range v.Warnings {
		fmt.Fprintf(out, "  warning: %s\n", wn)
	}
	if v.DryRun {
		fmt.Fprintln(out, "Remove --dry-run to write.")
	}
	return nil
}
```

Remove `exportFile`, `exportLine`, `buildExportView`, `exportLines`, `slug`, `countKind`, `slugRe` (slug lives in phoenix now).

- [ ] **Step 4: Flip the wiring in `root.go`.**

```go
newExportCmd(src),  // was: newExportCmd(fixture)
```

- [ ] **Step 5: Add goldens.**

`internal/commands/testdata/golden/export_dryrun.txt`:

```
Would write 3 cards to <VAULT>
Remove --dry-run to write.
```

`internal/commands/testdata/golden/export_real.txt`:

```
Wrote 3 cards to <VAULT>
```

Use `<VAULT>` as a placeholder and have `assertGolden` substitute `t.TempDir()`-provided path before comparison (Phase 1 goldens already do this kind of substitution for timestamps; mirror that pattern).

- [ ] **Step 6: Run to verify pass.**

Run: `go test ./internal/commands/...`
Expected: PASS.

- [ ] **Step 7: Commit.**

```bash
cd ~/cairn && git add internal/commands/export.go internal/commands/export_test.go \
  internal/commands/root.go internal/commands/testdata/golden/export_dryrun.txt \
  internal/commands/testdata/golden/export_real.txt && \
  git -c commit.gpgsign=false commit -m "feat(export): mirror cards to Phoenix vault with real writer"
```

---

### Task 12: Document import format + add acceptance check

**Files:** modify `docs/IMPORT_FORMAT.md`; add acceptance steps to `README.md` or `PHASE-1-REPORT.md` successor.

- [ ] **Step 1: Update `docs/IMPORT_FORMAT.md`.**

Add a "Media linkage" section explaining: attachments are keyed by filename base matching `cards.csv` `id`. Example: `MDE0O3xIKzJh4Y.pdf` corresponds to card `id=MDE0O3xIKzJh4Y`. Orphan files are warned and skipped.

- [ ] **Step 2: Add acceptance check.**

Against the live export at `~/phoenix/Clippings/mymind/`:

```bash
cd ~/cairn && go build -o /tmp/cairn ./cmd/cairn
/tmp/cairn import ~/phoenix/Clippings/mymind/
/tmp/cairn export --dry-run --to /tmp/cairn-vault
/tmp/cairn export --to /tmp/cairn-vault
ls /tmp/cairn-vault | head
head -30 /tmp/cairn-vault/$(ls /tmp/cairn-vault | grep .md | head -1)
```

Expected: ~43 markdown files at the vault root, one `_media/<fan>/<out>/<sha>.pdf`, frontmatter readable, relative attachment link resolves inside Obsidian.

- [ ] **Step 3: Write the Phase 2a report.**

Create `PHASE-2A-REPORT.md` with the same skeleton as `PHASE-1-REPORT.md`: what shipped, acceptance check (live), surprising deltas uncovered, what's still fake, open items, gate status.

- [ ] **Step 4: Commit.**

```bash
cd ~/cairn && git add docs/IMPORT_FORMAT.md PHASE-2A-REPORT.md && \
  git -c commit.gpgsign=false commit -m "docs: Phase 2a report and media linkage spec"
```

---

## Self-review

**Spec coverage:**

- Phoenix bridge (`~/phoenix/Clippings/MyMind/{YYYY-MM-DD}-{slug}.md` + `_media/{sha}.{ext}`, frontmatter with `mymind_id`, `url`, `tags[]`, `captured_at`, `kind`): Tasks 3, 5, 7, 8.
- On-demand only, no watcher: explicit non-goal.
- Media content-addressed layout: Tasks 4, 8.
- Media-to-card linkage (Phase 1 open item): Tasks 9, 10.
- `cairn export` real implementation: Task 11.

**Placeholder scan:** no TBDs, all code blocks concrete, all commands executable. Each "write test" step shows the actual test; each "implement" step shows the actual implementation.

**Type consistency:**

- `phoenix.Writer`, `phoenix.CardBundle`, `phoenix.WriteReport`, `phoenix.MediaRef` used consistently across Tasks 5-8 and Task 11.
- `cards.Media` introduced in Task 1, used everywhere downstream.
- `Source.MediaFor(cardID string) []cards.Media` signature consistent between Tasks 2 and 11.
- `Slug`, `DailyFilename`, `UniqueFilename`, `MediaRelPath`, `RelMediaLink` consistent between Tasks 3-4 and Tasks 7-8.
- `WriteReport.CardsWritten` / `CardsUnchanged` / `MediaWritten` / `MediaSkipped` / `Warnings` consistent between Tasks 6-8 and Task 11's `exportView`.

**Scope check:** plan is one subsystem (export + its prerequisites). Under a day of focused work. Does not expand into packs, find, embeddings, or MCP — those are separate 2b/2c/2d plans.

---

## Verification (final)

After all tasks:

- `go test ./...` passes.
- `go vet ./...` passes.
- Live acceptance: import the real 43-card export, run `cairn export --to /tmp/cairn-vault`, open a couple of the generated `.md` files in Obsidian, confirm frontmatter parses and attachment links resolve.
- `git log --oneline` shows one commit per task, no WIP.

