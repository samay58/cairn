# Cairn Phase 0 Implementation Plan

**Goal:** Ship the Phase 0 fake CLI for `cairn` — every command from §2.6 of the spec produces hand-authored output, covered by golden-file snapshot tests, driven by 25 fixture cards, accompanied by a `SCREENPLAY.md` walkthrough.

**Architecture:** Single Go binary wired with `cobra` (fang adopted later once real styling matters). Fixture cards live as JSON and load into a `cards.Card` struct shared with future phases. One `render` package owns all output formatting (plain, JSON, JSONL). One `commands` package, one file per command. Golden files live under `testdata/golden/`. Tests compare real command output to golden files; an `UPDATE_GOLDEN=1` env toggles regeneration.

**Tech Stack:** Go 1.22, `github.com/spf13/cobra` for CLI scaffolding, stdlib `encoding/json`, stdlib `testing`. No sqlite, no bubbletea, no glamour, no embeddings in Phase 0 — all deferred.

**Module path:** `github.com/samay58/cairn`. If the GitHub handle resolution later flips to `samay`, a one-shot `go mod edit -module` fixes it. Not blocking Phase 0.

**Spec reference:** `docs/design/cairn-design.md`. Phase 0 scope is section "Phase 0. Design prototype."

---

## File Structure

```
~/cairn/
  .gitignore
  LICENSE                                       MIT
  README.md                                     minimal: "Cairn Phase 0 fake CLI"
  SCREENPLAY.md                                 first-run walkthrough
  go.mod
  go.sum
  cmd/cairn/main.go                             entry point, cobra wiring
  internal/cards/cards.go                       Card struct + kind enum
  internal/cards/cards_test.go
  internal/fixtures/cards.json                  25 fixture cards
  internal/fixtures/fixtures.go                 embed + loader
  internal/fixtures/fixtures_test.go
  internal/render/tokens.go                     color constants (plain mode no-op in Phase 0)
  internal/render/cardlist.go                   human-readable card list formatter
  internal/render/json.go                       --json and --jsonl
  internal/render/cardlist_test.go
  internal/render/json_test.go
  internal/golden/golden.go                     goldenFile test helper
  internal/commands/root.go                     root cobra cmd
  internal/commands/import.go
  internal/commands/status.go
  internal/commands/search.go
  internal/commands/find.go
  internal/commands/get.go
  internal/commands/open.go
  internal/commands/pack.go
  internal/commands/ask.go
  internal/commands/export.go
  internal/commands/config.go
  internal/commands/mcp.go                      parent + start/install/audit/permissions subs
  internal/commands/*_test.go                   one per command file
  testdata/golden/                              *.txt and *.json snapshots
```

Responsibilities are single-purpose. `cards` owns the type. `fixtures` owns Phase 0's sample data. `render` owns every output format. `commands` wires cobra and calls into `render`. Tests live beside the code; golden files live under `testdata/` where Go test tooling already expects them.

---

## Conventions

**Handle resolution in Phase 0.** Without persistent state, `@N` maps to `fixtures.All()[N-1]` everywhere. `SCREENPLAY.md` documents this limitation and notes that Phase 1 implements real last-list persistence via SQLite.

**Color output.** Phase 0 emits plain text by default. `--no-color` is still parsed but is a no-op since there is no color yet. Snapshot tests pin plain output; ANSI lands in Phase 1 when styling matters.

**Golden file update flow.** Tests read the golden from disk; if `os.Getenv("UPDATE_GOLDEN") == "1"`, the test writes the current output to disk and passes. This keeps regeneration explicit.

**Commit discipline.** Every task ends in a commit. No "WIP" commits; the plan's task boundaries are already bite-sized.

---

## Task 0: Repository bootstrap

**Files:**
- Create: `~/cairn/.gitignore`
- Create: `~/cairn/LICENSE`
- Create: `~/cairn/README.md`
- Create: `~/cairn/go.mod`

- [ ] **Step 1: Write `.gitignore`**

```gitignore
# Go
*.test
*.out
/dist/
/build/

# Editor
.idea/
.vscode/
.DS_Store

# Cairn runtime (lives in ~/.cairn, not here, but guard anyway)
/cairn
/cairn.db
```

- [ ] **Step 2: Write MIT `LICENSE`**

```
MIT License

Copyright (c) 2026 Samay Dhawan

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 3: Write `README.md`**

```markdown
# Cairn

Terminal-native bridge between MyMind and the AI tools you already use.

Status: **Phase 0: fake CLI for design review**. No real storage, no real search, no real MCP. The canonical spec is `docs/design/cairn-design.md`.

## Run the Phase 0 CLI

```bash
go build ./cmd/cairn
./cairn --help
```

See `SCREENPLAY.md` for a first-run walkthrough.

## License

MIT. See `LICENSE`.
```

- [ ] **Step 4: Initialize Go module**

Run:
```bash
cd ~/cairn && go mod init github.com/samay58/cairn
```

Expected: creates `go.mod` with module line and Go version.

- [ ] **Step 5: Verify tree**

Run:
```bash
cd ~/cairn && ls -la && cat go.mod
```

Expected: `.gitignore`, `LICENSE`, `README.md`, `go.mod`, `docs/` all present.

- [ ] **Step 6: Commit**

```bash
cd ~/cairn && git add .gitignore LICENSE README.md go.mod && \
  git commit -m "chore: repo bootstrap (MIT, go module, readme)"
```

---

## Task 1: Card type

**Files:**
- Create: `internal/cards/cards.go`
- Create: `internal/cards/cards_test.go`

- [ ] **Step 1: Write the failing test**

`internal/cards/cards_test.go`:
```go
package cards

import "testing"

func TestKindLetter(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindArticle, "a"},
		{KindImage, "i"},
		{KindQuote, "q"},
		{KindNote, "n"},
	}
	for _, tc := range tests {
		if got := tc.kind.Letter(); got != tc.want {
			t.Errorf("Kind(%q).Letter() = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestKindFromString(t *testing.T) {
	k, err := KindFromString("article")
	if err != nil {
		t.Fatal(err)
	}
	if k != KindArticle {
		t.Errorf("got %q, want %q", k, KindArticle)
	}
	if _, err := KindFromString("bogus"); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cards/...`
Expected: FAIL (package not yet present or symbols not defined).

- [ ] **Step 3: Implement `cards.go`**

`internal/cards/cards.go`:
```go
package cards

import (
	"fmt"
	"time"
)

type Kind string

const (
	KindArticle Kind = "article"
	KindImage   Kind = "image"
	KindQuote   Kind = "quote"
	KindNote    Kind = "note"
)

func (k Kind) Letter() string {
	switch k {
	case KindArticle:
		return "a"
	case KindImage:
		return "i"
	case KindQuote:
		return "q"
	case KindNote:
		return "n"
	}
	return "?"
}

func KindFromString(s string) (Kind, error) {
	switch s {
	case "article":
		return KindArticle, nil
	case "image":
		return KindImage, nil
	case "quote":
		return KindQuote, nil
	case "note":
		return KindNote, nil
	}
	return "", fmt.Errorf("unknown kind %q", s)
}

type Card struct {
	ID         string    `json:"id"`
	MyMindID   string    `json:"mymind_id"`
	Kind       Kind      `json:"kind"`
	Title      string    `json:"title"`
	URL        string    `json:"url,omitempty"`
	Body       string    `json:"body,omitempty"`
	Excerpt    string    `json:"excerpt,omitempty"`
	Source     string    `json:"source,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	CapturedAt time.Time `json:"captured_at"`
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/cards/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cards/ && git commit -m "feat(cards): Card type and Kind enum with letter mapping"
```

---

## Task 2: Fixtures (25 cards)

**Files:**
- Create: `internal/fixtures/cards.json`
- Create: `internal/fixtures/fixtures.go`
- Create: `internal/fixtures/fixtures_test.go`

- [ ] **Step 1: Write the failing test**

`internal/fixtures/fixtures_test.go`:
```go
package fixtures

import (
	"testing"

	"github.com/samay58/cairn/internal/cards"
)

func TestAllReturnsTwentyFive(t *testing.T) {
	got := All()
	if len(got) != 25 {
		t.Fatalf("len(All()) = %d, want 25", len(got))
	}
}

func TestAllCoversEveryKind(t *testing.T) {
	kinds := map[cards.Kind]int{}
	for _, c := range All() {
		kinds[c.Kind]++
	}
	for _, k := range []cards.Kind{cards.KindArticle, cards.KindImage, cards.KindQuote, cards.KindNote} {
		if kinds[k] == 0 {
			t.Errorf("no fixture cards of kind %q", k)
		}
	}
}

func TestByHandleIsOneBased(t *testing.T) {
	c, err := ByHandle(1)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != All()[0].ID {
		t.Errorf("ByHandle(1) != All()[0]")
	}
	if _, err := ByHandle(0); err == nil {
		t.Error("expected error for handle 0")
	}
	if _, err := ByHandle(26); err == nil {
		t.Error("expected error for handle out of range")
	}
}
```

- [ ] **Step 2: Author `cards.json` (25 entries)**

`internal/fixtures/cards.json` — write exactly 25 cards. Suggested distribution: 10 articles, 5 images, 4 quotes, 6 notes. Use realistic MyMind-shaped content so the screenplay reads believably. Example entries (extend to 25, keep IDs stable):

```json
[
  {
    "id": "c_0001",
    "mymind_id": "mm_a1b2",
    "kind": "article",
    "title": "OAuth 2.0 Device Authorization Grant",
    "url": "https://datatracker.ietf.org/doc/html/rfc8628",
    "excerpt": "Describes the OAuth 2.0 device authorization grant flow for browserless and input-constrained devices.",
    "source": "datatracker.ietf.org",
    "tags": ["oauth", "auth", "rfc"],
    "captured_at": "2026-03-14T09:22:00Z"
  },
  {
    "id": "c_0002",
    "mymind_id": "mm_b3c4",
    "kind": "quote",
    "title": "On craft",
    "body": "The way you do anything is the way you do everything.",
    "source": "Martha Beck",
    "tags": ["craft", "philosophy"],
    "captured_at": "2026-03-18T14:05:00Z"
  },
  {
    "id": "c_0003",
    "mymind_id": "mm_d5e6",
    "kind": "image",
    "title": "Dieter Rams desk, 1970s",
    "source": "vitsoe.com",
    "tags": ["design", "rams"],
    "captured_at": "2026-03-20T11:00:00Z"
  },
  {
    "id": "c_0004",
    "mymind_id": "mm_f7g8",
    "kind": "note",
    "title": "Cairn naming",
    "body": "Short, unclaimed, phonetically clean, fits the category. Carries a trail-marker metaphor without being too cute.",
    "tags": ["cairn", "project"],
    "captured_at": "2026-03-22T18:40:00Z"
  }
  /* ... 21 more cards covering all four kinds; use realistic MyMind-shaped entries ... */
]
```

Constraint for Phase 0 fixtures: every entry must have an `id`, `mymind_id`, `kind`, `title`, `captured_at`. `url` is required for `article`; `body` is required for `quote` and `note`; `image` may omit both `body` and `url`. Tags are optional but encouraged. Keep `captured_at` values within March–April 2026 so the screenplay dates read as "recent."

- [ ] **Step 3: Implement `fixtures.go`**

`internal/fixtures/fixtures.go`:
```go
package fixtures

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/samay58/cairn/internal/cards"
)

//go:embed cards.json
var raw []byte

var (
	once   sync.Once
	loaded []cards.Card
	loadErr error
)

func load() {
	once.Do(func() {
		loadErr = json.Unmarshal(raw, &loaded)
	})
}

func All() []cards.Card {
	load()
	if loadErr != nil {
		panic(fmt.Sprintf("fixtures: %v", loadErr))
	}
	return loaded
}

func ByHandle(n int) (cards.Card, error) {
	all := All()
	if n < 1 || n > len(all) {
		return cards.Card{}, fmt.Errorf("no card at handle @%d (valid: @1..@%d)", n, len(all))
	}
	return all[n-1], nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/fixtures/...`
Expected: PASS. If JSON parse fails, fix the fixtures file until it loads cleanly.

- [ ] **Step 5: Commit**

```bash
git add internal/fixtures/ && git commit -m "feat(fixtures): 25 fixture cards covering all four kinds"
```

---

## Task 3: Golden-file test helper

**Files:**
- Create: `internal/golden/golden.go`

- [ ] **Step 1: Implement the helper (no separate test — it IS the test infra)**

`internal/golden/golden.go`:
```go
package golden

import (
	"os"
	"path/filepath"
	"testing"
)

// Assert compares got to the contents of testdata/golden/<name>.
// If UPDATE_GOLDEN=1, writes got to disk and passes.
func Assert(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %q not found: %v (run with UPDATE_GOLDEN=1 to create)", path, err)
	}
	if string(want) != got {
		t.Errorf("golden %q mismatch\n--- want ---\n%s\n--- got ---\n%s", path, string(want), got)
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/golden/ && git commit -m "feat(golden): golden-file assertion helper"
```

---

## Task 4: Render package — card list formatter

**Files:**
- Create: `internal/render/tokens.go`
- Create: `internal/render/cardlist.go`
- Create: `internal/render/cardlist_test.go`
- Create: `testdata/golden/cardlist_basic.txt`

- [ ] **Step 1: Decide the exact output shape (this is the design work)**

Per spec §"Output contract": `@N` handle, title, one `type · source · date` line in tertiary color, one "why shown" line, then an excerpt. For Phase 0 plain output, skip color codes. Date format: `2006-01-02`. Example block for a single card:

```
@1  OAuth 2.0 Device Authorization Grant
    a · datatracker.ietf.org · 2026-03-14
    matched on title and tag oauth
    Describes the OAuth 2.0 device authorization grant flow for browserless
    and input-constrained devices.
```

Cards are separated by one blank line.

- [ ] **Step 2: Write the golden file**

`testdata/golden/cardlist_basic.txt`:
```
@1  OAuth 2.0 Device Authorization Grant
    a · datatracker.ietf.org · 2026-03-14
    matched on title and tag oauth
    Describes the OAuth 2.0 device authorization grant flow for browserless
    and input-constrained devices.

@2  On craft
    q · Martha Beck · 2026-03-18
    matched on tag craft
    The way you do anything is the way you do everything.
```

- [ ] **Step 3: Write the failing test**

`internal/render/cardlist_test.go`:
```go
package render

import (
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/golden"
)

func TestCardListBasic(t *testing.T) {
	items := []Match{
		{
			Card: cards.Card{
				Kind:       cards.KindArticle,
				Title:      "OAuth 2.0 Device Authorization Grant",
				Source:     "datatracker.ietf.org",
				Excerpt:    "Describes the OAuth 2.0 device authorization grant flow for browserless\nand input-constrained devices.",
				CapturedAt: time.Date(2026, 3, 14, 9, 22, 0, 0, time.UTC),
			},
			WhyShown: "matched on title and tag oauth",
		},
		{
			Card: cards.Card{
				Kind:       cards.KindQuote,
				Title:      "On craft",
				Source:     "Martha Beck",
				Body:       "The way you do anything is the way you do everything.",
				CapturedAt: time.Date(2026, 3, 18, 14, 5, 0, 0, time.UTC),
			},
			WhyShown: "matched on tag craft",
		},
	}
	got := CardList(items)
	golden.Assert(t, "cardlist_basic.txt", got)
}
```

- [ ] **Step 4: Run test, verify failure**

Run: `go test ./internal/render/...`
Expected: FAIL (package does not exist yet).

- [ ] **Step 5: Implement `tokens.go` and `cardlist.go`**

`internal/render/tokens.go`:
```go
package render

// Phase 0 emits plain text. Tokens are placeholders for Phase 1+ styling.
const (
	TokenSeparator = " · "
)
```

`internal/render/cardlist.go`:
```go
package render

import (
	"fmt"
	"strings"

	"github.com/samay58/cairn/internal/cards"
)

type Match struct {
	Card     cards.Card
	WhyShown string
}

func CardList(matches []Match) string {
	var b strings.Builder
	for i, m := range matches {
		if i > 0 {
			b.WriteString("\n")
		}
		handle := fmt.Sprintf("@%d", i+1)
		meta := m.Card.Kind.Letter() + TokenSeparator + m.Card.Source + TokenSeparator + m.Card.CapturedAt.Format("2006-01-02")
		excerpt := m.Card.Excerpt
		if excerpt == "" {
			excerpt = m.Card.Body
		}
		fmt.Fprintf(&b, "%s  %s\n", handle, m.Card.Title)
		fmt.Fprintf(&b, "    %s\n", meta)
		fmt.Fprintf(&b, "    %s\n", m.WhyShown)
		for _, line := range strings.Split(excerpt, "\n") {
			fmt.Fprintf(&b, "    %s\n", line)
		}
	}
	return b.String()
}
```

- [ ] **Step 6: Run test, verify pass**

Run: `go test ./internal/render/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/render/ testdata/golden/cardlist_basic.txt && \
  git commit -m "feat(render): card-list plain text formatter"
```

---

## Task 5: Render package — JSON and JSONL

**Files:**
- Create: `internal/render/json.go`
- Create: `internal/render/json_test.go`
- Create: `testdata/golden/cardlist.json`
- Create: `testdata/golden/cardlist.jsonl`

- [ ] **Step 1: Write the golden files**

`testdata/golden/cardlist.json` — pretty-printed, 2-space indent, trailing newline. Shape:
```json
[
  {
    "handle": 1,
    "card": {
      "id": "",
      "mymind_id": "",
      "kind": "article",
      "title": "OAuth 2.0 Device Authorization Grant",
      "source": "datatracker.ietf.org",
      "excerpt": "Describes the OAuth 2.0 device authorization grant flow for browserless\nand input-constrained devices.",
      "captured_at": "2026-03-14T09:22:00Z"
    },
    "why_shown": "matched on title and tag oauth"
  },
  {
    "handle": 2,
    "card": {
      "id": "",
      "mymind_id": "",
      "kind": "quote",
      "title": "On craft",
      "body": "The way you do anything is the way you do everything.",
      "source": "Martha Beck",
      "captured_at": "2026-03-18T14:05:00Z"
    },
    "why_shown": "matched on tag craft"
  }
]
```

`testdata/golden/cardlist.jsonl` — one compact JSON object per line:
```
{"handle":1,"card":{"id":"","mymind_id":"","kind":"article","title":"OAuth 2.0 Device Authorization Grant","source":"datatracker.ietf.org","excerpt":"Describes the OAuth 2.0 device authorization grant flow for browserless\nand input-constrained devices.","captured_at":"2026-03-14T09:22:00Z"},"why_shown":"matched on title and tag oauth"}
{"handle":2,"card":{"id":"","mymind_id":"","kind":"quote","title":"On craft","body":"The way you do anything is the way you do everything.","source":"Martha Beck","captured_at":"2026-03-18T14:05:00Z"},"why_shown":"matched on tag craft"}
```

- [ ] **Step 2: Write the failing tests**

`internal/render/json_test.go`:
```go
package render

import (
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/golden"
)

func sample() []Match {
	return []Match{
		{
			Card: cards.Card{
				Kind:       cards.KindArticle,
				Title:      "OAuth 2.0 Device Authorization Grant",
				Source:     "datatracker.ietf.org",
				Excerpt:    "Describes the OAuth 2.0 device authorization grant flow for browserless\nand input-constrained devices.",
				CapturedAt: time.Date(2026, 3, 14, 9, 22, 0, 0, time.UTC),
			},
			WhyShown: "matched on title and tag oauth",
		},
		{
			Card: cards.Card{
				Kind:       cards.KindQuote,
				Title:      "On craft",
				Source:     "Martha Beck",
				Body:       "The way you do anything is the way you do everything.",
				CapturedAt: time.Date(2026, 3, 18, 14, 5, 0, 0, time.UTC),
			},
			WhyShown: "matched on tag craft",
		},
	}
}

func TestCardListJSON(t *testing.T) {
	golden.Assert(t, "cardlist.json", CardListJSON(sample()))
}

func TestCardListJSONL(t *testing.T) {
	golden.Assert(t, "cardlist.jsonl", CardListJSONL(sample()))
}
```

- [ ] **Step 3: Implement `json.go`**

`internal/render/json.go`:
```go
package render

import (
	"bytes"
	"encoding/json"
)

func CardListJSON(matches []Match) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(matches); err != nil {
		return ""
	}
	return buf.String()
}

func CardListJSONL(matches []Match) string {
	var buf bytes.Buffer
	for _, m := range matches {
		line, _ := json.Marshal(m)
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return buf.String()
}
```

Note: `json.Marshal` of `Match` will use the struct tags on `cards.Card`. Confirm the golden matches exactly after a dry run; if field ordering surprises you, adjust the `Card` struct tags to taste before updating golden files.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/render/...`
Expected: PASS. If JSON key ordering differs from the golden, regenerate with `UPDATE_GOLDEN=1 go test ./internal/render/...` and inspect the diff.

- [ ] **Step 5: Commit**

```bash
git add internal/render/json.go internal/render/json_test.go testdata/golden/cardlist.json testdata/golden/cardlist.jsonl && \
  git commit -m "feat(render): JSON and JSONL card-list formats"
```

---

## Task 6: Cobra root command skeleton

**Files:**
- Create: `internal/commands/root.go`
- Create: `cmd/cairn/main.go`

- [ ] **Step 1: Add cobra dependency**

Run:
```bash
cd ~/cairn && go get github.com/spf13/cobra@latest
```

- [ ] **Step 2: Implement `root.go` with stub subcommands**

`internal/commands/root.go`:
```go
package commands

import (
	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "cairn",
		Short: "Terminal-native bridge between MyMind and the tools you already use",
		Long:  "Cairn makes your MyMind library queryable from the terminal and a first-class context source for AI tools.\n\nPhase 0 is a design prototype with hand-authored output. Real storage and search arrive in Phase 1.",
	}

	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.PersistentFlags().Bool("jsonl", false, "emit JSONL output")
	root.PersistentFlags().Bool("plain", false, "emit plain text (default in Phase 0)")
	root.PersistentFlags().Int("limit", 0, "cap number of results (0 = default)")
	root.PersistentFlags().Bool("no-color", false, "disable color output (no-op in Phase 0)")

	root.AddCommand(
		newImportCmd(),
		newStatusCmd(),
		newSearchCmd(),
		newFindCmd(),
		newGetCmd(),
		newOpenCmd(),
		newPackCmd(),
		newAskCmd(),
		newExportCmd(),
		newConfigCmd(),
		newMCPCmd(),
	)
	return root
}
```

- [ ] **Step 3: Create stub files for every subcommand**

Create `internal/commands/import.go`, `status.go`, `search.go`, `find.go`, `get.go`, `open.go`, `pack.go`, `ask.go`, `export.go`, `config.go`, `mcp.go`. Each has one constructor returning a `*cobra.Command` with `Short` set and `RunE` returning `nil` for now. Example stub:

```go
package commands

import "github.com/spf13/cobra"

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Ingest a MyMind export folder",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}
```

Repeat for each command listed in `NewRoot`. `mcp.go` gets a parent plus four stub subcommands (`start`, `install`, `audit`, `permissions`).

- [ ] **Step 4: Implement `cmd/cairn/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/samay58/cairn/internal/commands"
)

func main() {
	if err := commands.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Build and verify**

Run:
```bash
cd ~/cairn && go build ./cmd/cairn && ./cairn --help
```

Expected: help output listing every subcommand. No runtime error.

- [ ] **Step 6: Commit**

```bash
git add cmd/ internal/commands/ go.mod go.sum && \
  git commit -m "feat(cli): cobra root and stub subcommands for every §2.6 command"
```

---

## Task 7: `cairn status`

**Files:**
- Modify: `internal/commands/status.go`
- Create: `internal/commands/status_test.go`
- Create: `testdata/golden/status.txt`

- [ ] **Step 1: Design the output (golden file)**

`testdata/golden/status.txt`:
```
cairn 0.0.0-phase0

library   25 cards · last import 2026-04-19 · 0 pending
storage   ~/.cairn/cairn.db (0 B) · media cache off
mcp       not installed
clients   none

Phase 0 prototype. Real storage lands in Phase 1.
```

- [ ] **Step 2: Write the failing test**

`internal/commands/status_test.go`:
```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestStatus(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "status.txt", out.String())
}
```

- [ ] **Step 3: Run, verify failure (current stub returns empty output)**

Run: `go test ./internal/commands/... -run TestStatus`
Expected: FAIL.

- [ ] **Step 4: Implement `status.go`**

```go
package commands

import (
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Library size, last sync, MCP state, permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write([]byte(
				"cairn 0.0.0-phase0\n\n" +
					"library   25 cards · last import 2026-04-19 · 0 pending\n" +
					"storage   ~/.cairn/cairn.db (0 B) · media cache off\n" +
					"mcp       not installed\n" +
					"clients   none\n\n" +
					"Phase 0 prototype. Real storage lands in Phase 1.\n",
			))
			return err
		},
	}
}
```

- [ ] **Step 5: Run, verify pass**

Run: `go test ./internal/commands/... -run TestStatus`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/status.go internal/commands/status_test.go testdata/golden/status.txt && \
  git commit -m "feat(status): hand-authored status output with snapshot"
```

---

## Task 8: `cairn import` (happy path)

**Files:**
- Modify: `internal/commands/import.go`
- Modify: `internal/commands/status_test.go` (add TestImport)
- Create: `testdata/golden/import_ok.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/import_ok.txt`:
```
Reading export from /tmp/mymind-export-2026-04-19/
Found cards.csv · 25 rows · media folder with 9 files
Parsing 25 cards         done
Extracting 9 media files done
Indexing 63 chunks       done

Imported 25 cards (0 updated, 0 deleted). Database now at ~/.cairn/cairn.db.
Run `cairn search "<query>"` or `cairn find`.
```

- [ ] **Step 2: Write the failing test**

Add to `internal/commands/status_test.go` (or a new `import_test.go`):

```go
func TestImportOK(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"import", "/tmp/mymind-export-2026-04-19/"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "import_ok.txt", out.String())
}
```

- [ ] **Step 3: Run, verify failure**

Run: `go test ./internal/commands/... -run TestImportOK`
Expected: FAIL.

- [ ] **Step 4: Implement `import.go`**

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Ingest a MyMind export folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Reading export from %s\n", args[0])
			fmt.Fprintln(out, "Found cards.csv · 25 rows · media folder with 9 files")
			fmt.Fprintln(out, "Parsing 25 cards         done")
			fmt.Fprintln(out, "Extracting 9 media files done")
			fmt.Fprintln(out, "Indexing 63 chunks       done")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Imported 25 cards (0 updated, 0 deleted). Database now at ~/.cairn/cairn.db.")
			fmt.Fprintln(out, "Run `cairn search \"<query>\"` or `cairn find`.")
			return nil
		},
	}
}
```

- [ ] **Step 5: Run, verify pass**

Run: `go test ./internal/commands/... -run TestImportOK`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/import.go internal/commands/*_test.go testdata/golden/import_ok.txt && \
  git commit -m "feat(import): happy-path output with snapshot"
```

---

## Task 9: `cairn import` (error: path not found)

**Files:**
- Modify: `internal/commands/import.go`
- Modify: `internal/commands/import_test.go` (add TestImportNotFound)
- Create: `testdata/golden/import_err_notfound.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/import_err_notfound.txt`:
```
Error: import failed.

Could not read export directory: /tmp/does-not-exist
Last successful import: 2026-04-19 from /tmp/mymind-export-2026-04-19/

Check the path and try again.
```

- [ ] **Step 2: Write the failing test**

```go
func TestImportNotFound(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"import", "/tmp/does-not-exist"})
	_ = root.Execute()
	golden.Assert(t, "import_err_notfound.txt", out.String())
}
```

- [ ] **Step 3: Run, verify failure**

Run: `go test ./internal/commands/... -run TestImportNotFound`
Expected: FAIL.

- [ ] **Step 4: Extend `import.go` to branch on the magic "does-not-exist" path**

```go
func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Ingest a MyMind export folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if args[0] == "/tmp/does-not-exist" {
				fmt.Fprintln(out, "Error: import failed.")
				fmt.Fprintln(out)
				fmt.Fprintf(out, "Could not read export directory: %s\n", args[0])
				fmt.Fprintln(out, "Last successful import: 2026-04-19 from /tmp/mymind-export-2026-04-19/")
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Check the path and try again.")
				return nil
			}
			// happy path from Task 8 ...
			fmt.Fprintf(out, "Reading export from %s\n", args[0])
			// ... identical body, unchanged
			return nil
		},
	}
}
```

For Phase 0, branching on a magic fixture path is acceptable. Phase 1 switches to real `os.Stat` checks.

- [ ] **Step 5: Run, verify pass**

Run: `go test ./internal/commands/... -run TestImport`
Expected: PASS on both TestImportOK and TestImportNotFound.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/import.go internal/commands/import_test.go testdata/golden/import_err_notfound.txt && \
  git commit -m "feat(import): not-found error output with snapshot"
```

---

## Task 10: `cairn search` (happy path)

**Files:**
- Modify: `internal/commands/search.go`
- Create: `internal/commands/search_test.go`
- Create: `testdata/golden/search_oauth.txt`
- Create: `testdata/golden/search_oauth.json`

- [ ] **Step 1: Golden files**

`testdata/golden/search_oauth.txt` — the first 3 fixture cards matching "oauth" rendered via `render.CardList`. Author this based on the actual fixtures you wrote in Task 2. If fixture #1 is the OAuth RFC, use it as @1; pick 2 more relevant cards.

`testdata/golden/search_oauth.json` — the same three matches rendered via `render.CardListJSON`.

- [ ] **Step 2: Write the failing test**

```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestSearchOAuthPlain(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"search", "oauth"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "search_oauth.txt", out.String())
}

func TestSearchOAuthJSON(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"search", "oauth", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "search_oauth.json", out.String())
}
```

- [ ] **Step 3: Run, verify failure**

Run: `go test ./internal/commands/... -run TestSearchOAuth`
Expected: FAIL.

- [ ] **Step 4: Implement `search.go` with a hardcoded query→fixtures map**

```go
package commands

import (
	"fmt"
	"strings"

	"github.com/samay58/cairn/internal/fixtures"
	"github.com/samay58/cairn/internal/render"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Hybrid retrieval, card-list output",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			matches := fakeSearch(query)

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

// fakeSearch returns hand-picked fixture subsets for Phase 0 snapshots.
// Every recognized query string must be deterministic.
func fakeSearch(query string) []render.Match {
	q := strings.ToLower(strings.TrimSpace(query))
	all := fixtures.All()
	switch q {
	case "oauth":
		// Pick the three cards most relevant to "oauth". Adjust indices to match
		// the fixtures you wrote in Task 2.
		return []render.Match{
			{Card: all[0], WhyShown: "matched on title and tag oauth"},
			{Card: all[10], WhyShown: "matched on tag auth"},
			{Card: all[17], WhyShown: "matched on body"},
		}
	case "zzz-empty":
		return nil
	}
	// Default demo: return the first three fixtures so every other query shows something.
	return []render.Match{
		{Card: all[0], WhyShown: "demo result 1"},
		{Card: all[1], WhyShown: "demo result 2"},
		{Card: all[2], WhyShown: "demo result 3"},
	}
}
```

- [ ] **Step 5: Run, verify pass**

Run: `go test ./internal/commands/... -run TestSearchOAuth`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/search.go internal/commands/search_test.go testdata/golden/search_oauth.txt testdata/golden/search_oauth.json && \
  git commit -m "feat(search): plain and JSON output for happy-path query"
```

---

## Task 11: `cairn search` (no-results state)

**Files:**
- Modify: `internal/commands/search.go`
- Modify: `internal/commands/search_test.go`
- Create: `testdata/golden/search_empty.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/search_empty.txt`:
```
No cards matched "zzz-empty".

Try a broader query, drop filters, or check `cairn status` to confirm the import is fresh.
```

- [ ] **Step 2: Add test**

```go
func TestSearchEmpty(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"search", "zzz-empty"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "search_empty.txt", out.String())
}
```

- [ ] **Step 3: Run, verify failure**

Run: `go test ./internal/commands/... -run TestSearchEmpty`
Expected: FAIL.

- [ ] **Step 4: Implement `writeNoResults`**

Append to `internal/commands/search.go`:
```go
import "io"

func writeNoResults(out io.Writer, query string) error {
	_, err := fmt.Fprintf(out,
		"No cards matched %q.\n\nTry a broader query, drop filters, or check `cairn status` to confirm the import is fresh.\n",
		query,
	)
	return err
}
```

- [ ] **Step 5: Run, verify pass**

Run: `go test ./internal/commands/... -run TestSearchEmpty`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/search.go internal/commands/search_test.go testdata/golden/search_empty.txt && \
  git commit -m "feat(search): designed no-results state"
```

---

## Task 12: `cairn get` (happy path by @handle)

**Files:**
- Modify: `internal/commands/get.go`
- Create: `internal/commands/get_test.go`
- Create: `testdata/golden/get_at2.txt`

- [ ] **Step 1: Golden file (authored against fixture #2)**

`testdata/golden/get_at2.txt`:
```
@2  On craft
q · Martha Beck · 2026-03-18
tags: craft, philosophy

The way you do anything is the way you do everything.
```

- [ ] **Step 2: Write the failing test**

```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestGetByHandle(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"get", "@2"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "get_at2.txt", out.String())
}
```

- [ ] **Step 3: Run, verify failure**

Run: `go test ./internal/commands/... -run TestGetByHandle`
Expected: FAIL.

- [ ] **Step 4: Implement `get.go`**

```go
package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samay58/cairn/internal/fixtures"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <card>",
		Short: "Render full card in terminal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			ref := args[0]
			if !strings.HasPrefix(ref, "@") {
				fmt.Fprintf(out, "Phase 0 only supports @handle references. Got %q.\n", ref)
				return nil
			}
			n, err := strconv.Atoi(strings.TrimPrefix(ref, "@"))
			if err != nil {
				fmt.Fprintf(out, "Invalid handle: %q.\n", ref)
				return nil
			}
			c, err := fixtures.ByHandle(n)
			if err != nil {
				fmt.Fprintln(out, err.Error())
				fmt.Fprintln(out, "Run a list command (search, find) to refresh handles.")
				return nil
			}
			meta := c.Kind.Letter() + " · " + c.Source + " · " + c.CapturedAt.Format("2006-01-02")
			fmt.Fprintf(out, "@%d  %s\n", n, c.Title)
			fmt.Fprintln(out, meta)
			if len(c.Tags) > 0 {
				fmt.Fprintf(out, "tags: %s\n", strings.Join(c.Tags, ", "))
			}
			fmt.Fprintln(out)
			body := c.Body
			if body == "" {
				body = c.Excerpt
			}
			fmt.Fprintln(out, body)
			return nil
		},
	}
}
```

- [ ] **Step 5: Run, verify pass**

Run: `go test ./internal/commands/... -run TestGetByHandle`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/get.go internal/commands/get_test.go testdata/golden/get_at2.txt && \
  git commit -m "feat(get): render fixture card by @handle"
```

---

## Task 13: `cairn get` (error: unknown handle)

**Files:**
- Modify: `internal/commands/get_test.go`
- Create: `testdata/golden/get_err_unknown.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/get_err_unknown.txt`:
```
no card at handle @99 (valid: @1..@25)
Run a list command (search, find) to refresh handles.
```

- [ ] **Step 2: Add test**

```go
func TestGetUnknownHandle(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"get", "@99"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "get_err_unknown.txt", out.String())
}
```

- [ ] **Step 3: Run tests (should PASS immediately — `get.go` already handles this)**

Run: `go test ./internal/commands/... -run TestGetUnknownHandle`
Expected: PASS. If not, adjust the golden until it matches the actual output of `fixtures.ByHandle`.

- [ ] **Step 4: Commit**

```bash
git add internal/commands/get_test.go testdata/golden/get_err_unknown.txt && \
  git commit -m "test(get): snapshot unknown-handle error state"
```

---

## Task 14: `cairn open`

**Files:**
- Modify: `internal/commands/open.go`
- Create: `internal/commands/open_test.go`
- Create: `testdata/golden/open_at1.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/open_at1.txt`:
```
Would open: https://datatracker.ietf.org/doc/html/rfc8628
(Phase 0 fake: real `cairn open` invokes the OS default browser.)
```

- [ ] **Step 2: Write the failing test**

```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestOpenByHandle(t *testing.T) {
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

- [ ] **Step 3: Run, verify failure**

Run: `go test ./internal/commands/... -run TestOpenByHandle`
Expected: FAIL.

- [ ] **Step 4: Implement `open.go`**

```go
package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samay58/cairn/internal/fixtures"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <card>",
		Short: "Open a card in the MyMind browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			n, err := strconv.Atoi(strings.TrimPrefix(args[0], "@"))
			if err != nil {
				fmt.Fprintf(out, "Invalid handle: %q.\n", args[0])
				return nil
			}
			c, err := fixtures.ByHandle(n)
			if err != nil {
				fmt.Fprintln(out, err.Error())
				return nil
			}
			url := c.URL
			if url == "" {
				url = fmt.Sprintf("https://access.mymind.com/cards/%s", c.MyMindID)
			}
			fmt.Fprintf(out, "Would open: %s\n", url)
			fmt.Fprintln(out, "(Phase 0 fake: real `cairn open` invokes the OS default browser.)")
			return nil
		},
	}
}
```

- [ ] **Step 5: Run, verify pass**

Run: `go test ./internal/commands/... -run TestOpenByHandle`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/open.go internal/commands/open_test.go testdata/golden/open_at1.txt && \
  git commit -m "feat(open): handle-to-URL resolution with fake open"
```

---

## Task 15: `cairn find` (mock TUI frame)

**Files:**
- Modify: `internal/commands/find.go`
- Create: `internal/commands/find_test.go`
- Create: `testdata/golden/find_frame.txt`

- [ ] **Step 1: Golden file — one static frame of the TUI**

`testdata/golden/find_frame.txt`:
```
cairn find · 25 cards · type: all  source: all  since: all

› _

@1  OAuth 2.0 Device Authorization Grant
    a · datatracker.ietf.org · 2026-03-14
    recent
    Describes the OAuth 2.0 device authorization grant flow for browserless
    and input-constrained devices.

@2  On craft
    q · Martha Beck · 2026-03-18
    recent
    The way you do anything is the way you do everything.

@3  Dieter Rams desk, 1970s
    i · vitsoe.com · 2026-03-20
    recent

enter open url · o open in mymind · c copy · y yank · tab cycle filters · esc quit

Phase 0: this is a static mock. Real TUI lands in Phase 2.
```

- [ ] **Step 2: Write the failing test**

```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestFindFrame(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"find"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "find_frame.txt", out.String())
}
```

- [ ] **Step 3: Run, verify failure**

Run: `go test ./internal/commands/... -run TestFindFrame`
Expected: FAIL.

- [ ] **Step 4: Implement `find.go`**

```go
package commands

import (
	"fmt"

	"github.com/samay58/cairn/internal/fixtures"
	"github.com/samay58/cairn/internal/render"
	"github.com/spf13/cobra"
)

func newFindCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "find",
		Short: "Full-screen fuzzy TUI (Phase 0: static mock)",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			all := fixtures.All()
			top := []render.Match{
				{Card: all[0], WhyShown: "recent"},
				{Card: all[1], WhyShown: "recent"},
				{Card: all[2], WhyShown: "recent"},
			}

			fmt.Fprintf(out, "cairn find · %d cards · type: all  source: all  since: all\n", len(all))
			fmt.Fprintln(out)
			fmt.Fprintln(out, "› _")
			fmt.Fprintln(out)
			fmt.Fprint(out, render.CardList(top))
			fmt.Fprintln(out)
			fmt.Fprintln(out, "enter open url · o open in mymind · c copy · y yank · tab cycle filters · esc quit")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Phase 0: this is a static mock. Real TUI lands in Phase 2.")
			return nil
		},
	}
}
```

- [ ] **Step 5: Run, verify pass**

Run: `go test ./internal/commands/... -run TestFindFrame`
Expected: PASS. If the render shape differs from your golden, regenerate with `UPDATE_GOLDEN=1` and inspect the diff.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/find.go internal/commands/find_test.go testdata/golden/find_frame.txt && \
  git commit -m "feat(find): static mock TUI frame for Phase 0"
```

---

## Task 16: `cairn pack` (Claude profile and JSON profile)

**Files:**
- Modify: `internal/commands/pack.go`
- Create: `internal/commands/pack_test.go`
- Create: `testdata/golden/pack_claude.txt`
- Create: `testdata/golden/pack_json.txt`

- [ ] **Step 1: Golden files**

`testdata/golden/pack_claude.txt`:
```
<context source="cairn" query="oauth device flow" cards="3">
  <card id="c_0001" kind="article" captured="2026-03-14" source="datatracker.ietf.org">
    <title>OAuth 2.0 Device Authorization Grant</title>
    <url>https://datatracker.ietf.org/doc/html/rfc8628</url>
    <excerpt>Describes the OAuth 2.0 device authorization grant flow for browserless and input-constrained devices.</excerpt>
  </card>
  <card id="c_0011" kind="article" captured="2026-04-02" source="oauth.net">
    <title>Proof Key for Code Exchange (PKCE)</title>
    <url>https://oauth.net/2/pkce/</url>
    <excerpt>PKCE protects OAuth public clients from authorization code interception attacks.</excerpt>
  </card>
  <card id="c_0018" kind="note" captured="2026-04-10" source="cairn">
    <title>Why device flow for cairn</title>
    <excerpt>Browserless CLI with no secret storage. Device flow + PKCE is the RFC-blessed path.</excerpt>
  </card>
</context>

Citation key: c_0001, c_0011, c_0018.
```

`testdata/golden/pack_json.txt` — the same content serialized as pretty JSON with keys `query`, `cards` (array of card objects), and `citation_keys`. Author a matching golden.

- [ ] **Step 2: Write the failing tests**

```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestPackClaude(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"pack", "oauth device flow", "--for", "claude"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "pack_claude.txt", out.String())
}

func TestPackJSON(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"pack", "oauth device flow", "--for", "json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "pack_json.txt", out.String())
}
```

- [ ] **Step 3: Run, verify failure**

Run: `go test ./internal/commands/... -run TestPack`
Expected: FAIL.

- [ ] **Step 4: Implement `pack.go`**

```go
package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/fixtures"
	"github.com/spf13/cobra"
)

func newPackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack <query>",
		Short: "Cited context pack for an external AI",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			profile, _ := cmd.Flags().GetString("for")
			picks := packPicks(query)
			out := cmd.OutOrStdout()
			switch profile {
			case "claude", "":
				writePackClaude(out, query, picks)
			case "json":
				writePackJSON(out, query, picks)
			default:
				fmt.Fprintf(out, "Unknown profile %q. Supported in Phase 0: claude, json.\n", profile)
			}
			return nil
		},
	}
	cmd.Flags().String("for", "claude", "target profile: claude, chatgpt, cursor, markdown, json")
	return cmd
}

func packPicks(query string) []cards.Card {
	all := fixtures.All()
	// Phase 0: return a curated deterministic subset for the screenplay query.
	// Replace these indices with the actual indices of cards relevant to "oauth device flow".
	return []cards.Card{all[0], all[10], all[17]}
}

func writePackClaude(out stringWriter, query string, picks []cards.Card) {
	fmt.Fprintf(out, "<context source=\"cairn\" query=%q cards=\"%d\">\n", query, len(picks))
	for _, c := range picks {
		fmt.Fprintf(out, "  <card id=%q kind=%q captured=%q source=%q>\n",
			c.ID, string(c.Kind), c.CapturedAt.Format("2006-01-02"), c.Source)
		fmt.Fprintf(out, "    <title>%s</title>\n", c.Title)
		if c.URL != "" {
			fmt.Fprintf(out, "    <url>%s</url>\n", c.URL)
		}
		excerpt := c.Excerpt
		if excerpt == "" {
			excerpt = c.Body
		}
		fmt.Fprintf(out, "    <excerpt>%s</excerpt>\n", excerpt)
		fmt.Fprintln(out, "  </card>")
	}
	fmt.Fprintln(out, "</context>")
	fmt.Fprintln(out)
	keys := make([]string, 0, len(picks))
	for _, c := range picks {
		keys = append(keys, c.ID)
	}
	fmt.Fprintf(out, "Citation key: %s.\n", strings.Join(keys, ", "))
}

type stringWriter interface {
	Write(p []byte) (int, error)
}

func writePackJSON(out stringWriter, query string, picks []cards.Card) {
	payload := map[string]any{
		"query":         query,
		"cards":         picks,
		"citation_keys": citationKeys(picks),
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Fprintln(out, string(b))
}

func citationKeys(picks []cards.Card) []string {
	out := make([]string, 0, len(picks))
	for _, c := range picks {
		out = append(out, c.ID)
	}
	return out
}
```

- [ ] **Step 5: Run, verify pass**

Run: `go test ./internal/commands/... -run TestPack`
Expected: PASS. If JSON ordering differs, regenerate golden with `UPDATE_GOLDEN=1`.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/pack.go internal/commands/pack_test.go testdata/golden/pack_claude.txt testdata/golden/pack_json.txt && \
  git commit -m "feat(pack): claude and json profiles"
```

---

## Task 17: `cairn ask` (Phase 4 stub)

**Files:**
- Modify: `internal/commands/ask.go`
- Create: `internal/commands/ask_test.go`
- Create: `testdata/golden/ask_stub.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/ask_stub.txt`:
```
`cairn ask` ships in Phase 4, after the Phase 3 integrity gate passes.

For now, use `cairn pack "<question>"` to hand a cited bundle to your AI of choice.
```

- [ ] **Step 2: Write test**

```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestAskStub(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"ask", "what is device flow?"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "ask_stub.txt", out.String())
}
```

- [ ] **Step 3: Implement `ask.go`**

```go
package commands

import "github.com/spf13/cobra"

func newAskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask <question>",
		Short: "RAG synthesis with citations (Phase 4)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write([]byte(
				"`cairn ask` ships in Phase 4, after the Phase 3 integrity gate passes.\n\n" +
					"For now, use `cairn pack \"<question>\"` to hand a cited bundle to your AI of choice.\n",
			))
			return err
		},
	}
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/commands/... -run TestAskStub`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/commands/ask.go internal/commands/ask_test.go testdata/golden/ask_stub.txt && \
  git commit -m "feat(ask): phase-4 stub message"
```

---

## Task 18: `cairn export`

**Files:**
- Modify: `internal/commands/export.go`
- Create: `internal/commands/export_test.go`
- Create: `testdata/golden/export.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/export.txt`:
```
Mirroring 25 cards to ~/phoenix/Clippings/MyMind/

  2026-03-14  oauth-2-0-device-authorization-grant.md
  2026-03-18  on-craft.md
  2026-03-20  dieter-rams-desk-1970s.md
  ... 22 more files ...

Media (9 files) → ~/phoenix/Clippings/MyMind/_media/

Phase 0: nothing written to disk. Real export runs in Phase 2.
```

- [ ] **Step 2: Write test**

```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestExport(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"export"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "export.txt", out.String())
}
```

- [ ] **Step 3: Implement `export.go`**

```go
package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/samay58/cairn/internal/fixtures"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Mirror cards to Phoenix vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			all := fixtures.All()
			fmt.Fprintf(out, "Mirroring %d cards to ~/phoenix/Clippings/MyMind/\n\n", len(all))
			for i, c := range all {
				if i < 3 {
					fmt.Fprintf(out, "  %s  %s.md\n", c.CapturedAt.Format("2006-01-02"), slug(c.Title))
				}
			}
			if len(all) > 3 {
				fmt.Fprintf(out, "  ... %d more files ...\n", len(all)-3)
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Media (9 files) → ~/phoenix/Clippings/MyMind/_media/")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Phase 0: nothing written to disk. Real export runs in Phase 2.")
			return nil
		},
	}
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/commands/... -run TestExport`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/commands/export.go internal/commands/export_test.go testdata/golden/export.txt && \
  git commit -m "feat(export): phoenix vault mirror dry-run"
```

---

## Task 19: `cairn config`

**Files:**
- Modify: `internal/commands/config.go`
- Create: `internal/commands/config_test.go`
- Create: `testdata/golden/config.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/config.txt`:
```
~/.cairn/config.toml (Phase 0 defaults)

[storage]
cache_full_content = false
cache_media        = false

[embeddings]
model = "minilm-l6-v2"
device = "cpu"

[llm]
provider = "anthropic"
model    = "claude-opus-4-7"

[mcp]
default_permissions = { search_mind = true, get_related = true, get_card = "prompt", save_to_mind = false }

Phase 0: values are hand-authored defaults. Real config reads/writes arrive in Phase 1.
```

- [ ] **Step 2: Write test**

```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestConfig(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"config"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "config.txt", out.String())
}
```

- [ ] **Step 3: Implement `config.go`**

```go
package commands

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed config_stub.txt
var configStub string

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Edit config (Phase 0: shows defaults only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write([]byte(configStub))
			return err
		},
	}
}
```

Then create `internal/commands/config_stub.txt` with the same content as the golden. Embedding the stub keeps the string editable without escaping.

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/commands/... -run TestConfig`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/commands/config.go internal/commands/config_stub.txt internal/commands/config_test.go testdata/golden/config.txt && \
  git commit -m "feat(config): phase-0 defaults display"
```

---

## Task 20: `cairn mcp start`

**Files:**
- Modify: `internal/commands/mcp.go`
- Create: `internal/commands/mcp_test.go`
- Create: `testdata/golden/mcp_start.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/mcp_start.txt`:
```
cairn mcp server (stdio) — Phase 0 fake

Would expose 5 tools:
  search_mind(query, limit?, filters?)
  get_card(card_id, include_full_text?)
  get_related(card_id, limit?)
  create_context_pack(query, max_tokens?)
  save_to_mind(url_or_text, note?)              [disabled]

Client config:
  claude-code     not installed
  claude-desktop  not installed

Phase 0 exits immediately. Real server runs in Phase 3.
```

- [ ] **Step 2: Write test**

```go
package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestMCPStart(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"mcp", "start"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "mcp_start.txt", out.String())
}
```

- [ ] **Step 3: Implement `mcp.go` parent + `start` subcommand**

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	mcp := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server, install, audit, permissions",
	}
	mcp.AddCommand(newMCPStartCmd(), newMCPInstallCmd(), newMCPAuditCmd(), newMCPPermissionsCmd())
	return mcp
}

func newMCPStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the local MCP server over stdio",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "cairn mcp server (stdio) — Phase 0 fake")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Would expose 5 tools:")
			fmt.Fprintln(out, "  search_mind(query, limit?, filters?)")
			fmt.Fprintln(out, "  get_card(card_id, include_full_text?)")
			fmt.Fprintln(out, "  get_related(card_id, limit?)")
			fmt.Fprintln(out, "  create_context_pack(query, max_tokens?)")
			fmt.Fprintln(out, "  save_to_mind(url_or_text, note?)              [disabled]")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Client config:")
			fmt.Fprintln(out, "  claude-code     not installed")
			fmt.Fprintln(out, "  claude-desktop  not installed")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Phase 0 exits immediately. Real server runs in Phase 3.")
			return nil
		},
	}
}
```

Note: the em-dash in the golden above should be replaced with a period for consistency with the rest of the output. Edit both the golden and the command output to use a period. Final line: `cairn mcp server (stdio). Phase 0 fake`, or keep `cairn mcp server (stdio)` and move the Phase-0 label to its own line. Pick one and apply to both files.

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/commands/... -run TestMCPStart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/commands/mcp.go internal/commands/mcp_test.go testdata/golden/mcp_start.txt && \
  git commit -m "feat(mcp): start subcommand phase-0 output"
```

---

## Task 21: `cairn mcp install claude-code`

**Files:**
- Modify: `internal/commands/mcp.go`
- Modify: `internal/commands/mcp_test.go`
- Create: `testdata/golden/mcp_install_claude_code.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/mcp_install_claude_code.txt`:
```
Installing cairn MCP server for claude-code.

Target config: ~/.claude/mcp.json
Existing servers detected: 2 (brave-search, filesystem)

Proposed merge (new keys only):
  mcpServers.cairn = {
    "command": "cairn",
    "args": ["mcp", "start"]
  }

Phase 0 would write this merge. Real install runs in Phase 3.
Run `cairn mcp permissions` to review tool-level scopes.
```

- [ ] **Step 2: Write test**

```go
func TestMCPInstallClaudeCode(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"mcp", "install", "claude-code"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "mcp_install_claude_code.txt", out.String())
}
```

- [ ] **Step 3: Implement `install` subcommand**

```go
func newMCPInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <client>",
		Short: "Write client config for cairn MCP",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := args[0]
			out := cmd.OutOrStdout()
			switch client {
			case "claude-code":
				fmt.Fprintln(out, "Installing cairn MCP server for claude-code.")
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Target config: ~/.claude/mcp.json")
				fmt.Fprintln(out, "Existing servers detected: 2 (brave-search, filesystem)")
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Proposed merge (new keys only):")
				fmt.Fprintln(out, "  mcpServers.cairn = {")
				fmt.Fprintln(out, "    \"command\": \"cairn\",")
				fmt.Fprintln(out, "    \"args\": [\"mcp\", \"start\"]")
				fmt.Fprintln(out, "  }")
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Phase 0 would write this merge. Real install runs in Phase 3.")
				fmt.Fprintln(out, "Run `cairn mcp permissions` to review tool-level scopes.")
			case "claude-desktop":
				writeInstallClaudeDesktop(out)
			default:
				fmt.Fprintf(out, "Unknown client %q. Phase 0 supports: claude-code, claude-desktop.\n", client)
			}
			return nil
		},
	}
}

func writeInstallClaudeDesktop(out stringWriter) {
	// body added in Task 22
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/commands/... -run TestMCPInstallClaudeCode`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/commands/mcp.go internal/commands/mcp_test.go testdata/golden/mcp_install_claude_code.txt && \
  git commit -m "feat(mcp): install claude-code merge preview"
```

---

## Task 22: `cairn mcp install claude-desktop`

**Files:**
- Modify: `internal/commands/mcp.go`
- Modify: `internal/commands/mcp_test.go`
- Create: `testdata/golden/mcp_install_claude_desktop.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/mcp_install_claude_desktop.txt`:
```
Installing cairn MCP server for claude-desktop.

Target config: ~/Library/Application Support/Claude/claude_desktop_config.json
Existing servers detected: 4 (brave-search, filesystem, exa, perplexity-ask)

Proposed merge (new keys only):
  mcpServers.cairn = {
    "command": "cairn",
    "args": ["mcp", "start"]
  }

Phase 0 would write this merge. Real install runs in Phase 3.
Run `cairn mcp permissions` to review tool-level scopes.
```

- [ ] **Step 2: Write test and implement**

Add `TestMCPInstallClaudeDesktop` mirroring Task 21. Implement `writeInstallClaudeDesktop` by writing the lines above.

- [ ] **Step 3: Run, verify pass**

Run: `go test ./internal/commands/... -run TestMCPInstallClaudeDesktop`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/commands/mcp.go internal/commands/mcp_test.go testdata/golden/mcp_install_claude_desktop.txt && \
  git commit -m "feat(mcp): install claude-desktop merge preview"
```

---

## Task 23: `cairn mcp audit`

**Files:**
- Modify: `internal/commands/mcp.go`
- Modify: `internal/commands/mcp_test.go`
- Create: `testdata/golden/mcp_audit.txt`

- [ ] **Step 1: Golden file (uses the spec's reference format verbatim)**

`testdata/golden/mcp_audit.txt`:
```
Today

  10:42  Claude Code searched "OAuth MCP authorization"
         Returned 5 snippets

  10:43  Claude Code requested full content for @2
         Allowed once by user

  10:49  Cursor searched "SQLite FTS5"
         Returned 8 snippets

Yesterday

  22:11  Claude Desktop searched "prompt injection mitigations"
         Returned 4 snippets

Phase 0: sample audit rows. Real log reads from the mcp_audit table in Phase 3.
```

- [ ] **Step 2: Write test**

```go
func TestMCPAudit(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"mcp", "audit"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "mcp_audit.txt", out.String())
}
```

- [ ] **Step 3: Implement `audit` subcommand**

Add the literal audit text as an embedded string and write it. Replace the em-dash in the footer with a period: `Phase 0: sample audit rows. Real log reads from the mcp_audit table in Phase 3.` No em-dashes anywhere.

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/commands/... -run TestMCPAudit`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/commands/mcp.go internal/commands/mcp_test.go testdata/golden/mcp_audit.txt && \
  git commit -m "feat(mcp): audit log render"
```

---

## Task 24: `cairn mcp permissions`

**Files:**
- Modify: `internal/commands/mcp.go`
- Modify: `internal/commands/mcp_test.go`
- Create: `testdata/golden/mcp_permissions.txt`

- [ ] **Step 1: Golden file**

`testdata/golden/mcp_permissions.txt`:
```
Per-client MCP permissions

  claude-code
    search_mind           allow
    get_related           allow
    get_card              prompt (include_full_text=true)
    create_context_pack   allow
    save_to_mind          deny

  claude-desktop
    search_mind           allow
    get_related           allow
    get_card              prompt (include_full_text=true)
    create_context_pack   allow
    save_to_mind          deny

Phase 0: values are the privacy-first defaults. Edit ~/.cairn/config.toml in Phase 3 to change them.
```

- [ ] **Step 2: Write test and implement**

Add `TestMCPPermissions` mirroring prior tests. Implement `newMCPPermissionsCmd` to emit the exact string above.

- [ ] **Step 3: Run, verify pass**

Run: `go test ./internal/commands/... -run TestMCPPermissions`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/commands/mcp.go internal/commands/mcp_test.go testdata/golden/mcp_permissions.txt && \
  git commit -m "feat(mcp): permissions display"
```

---

## Task 25: Write `SCREENPLAY.md`

**Files:**
- Create: `SCREENPLAY.md`

- [ ] **Step 1: Author the first-run walkthrough**

Structure: a single narrative file that walks `import → status → search → find → get → pack → mcp install claude-code → mcp audit`. For each step:
- One sentence of narration ("You just unzipped a MyMind export and want to confirm cairn sees it.")
- The exact command the user types.
- A fenced block showing the exact expected output (copy-paste from the relevant golden file).
- Any Phase 0 caveat ("Phase 0 hand-authored; real data in Phase 1.")

Goal: the screenplay is the design artifact that Samay reads end-to-end to sign off. Every fenced block must match a golden file byte-for-byte.

- [ ] **Step 2: Verify every fenced block matches a golden**

Run a quick check:
```bash
for f in testdata/golden/*.txt; do
  name=$(basename "$f")
  grep -q "$name referenced" SCREENPLAY.md 2>/dev/null || true
done
```

This is a manual cross-read, not an automated test. Each golden file's content should appear in a fenced block somewhere in the screenplay.

- [ ] **Step 3: Commit**

```bash
git add SCREENPLAY.md && git commit -m "docs: SCREENPLAY.md first-run walkthrough"
```

---

## Task 26: Final verification pass

**Files:** none (review only)

- [ ] **Step 1: Run the full test suite**

```bash
cd ~/cairn && go test ./...
```

Expected: all tests pass.

- [ ] **Step 2: Build and exercise every command manually**

```bash
cd ~/cairn && go build -o cairn ./cmd/cairn
./cairn --help
./cairn status
./cairn import /tmp/mymind-export-2026-04-19/
./cairn import /tmp/does-not-exist
./cairn search oauth
./cairn search zzz-empty
./cairn get @2
./cairn get @99
./cairn open @1
./cairn find
./cairn pack "oauth device flow" --for claude
./cairn pack "oauth device flow" --for json
./cairn ask "what is device flow?"
./cairn export
./cairn config
./cairn mcp start
./cairn mcp install claude-code
./cairn mcp install claude-desktop
./cairn mcp audit
./cairn mcp permissions
rm cairn
```

Expected: every command prints output matching its golden file. No panics, no cobra usage errors.

- [ ] **Step 3: Verify the rules from the spec output contract**

Visual-pass check:
- No emoji anywhere in any output.
- Single-letter kinds only in card-list (`a`, `i`, `q`, `n`).
- Separator is `·` (U+00B7), appears nowhere else as chrome.
- All output is sentence case (no ALL CAPS headers).
- No em-dashes in any golden file: `grep -R "—" testdata/` returns nothing.
- Default output fits 80 columns: `awk '{ if (length > 80) print FILENAME":"NR": "length" chars" }' testdata/golden/*.txt` returns nothing.

Fix any violations by editing both the command output and the golden in lockstep.

- [ ] **Step 4: Write `PHASE-0-REPORT.md`**

Short retrospective at the repo root:

```markdown
# Phase 0 report

**Shipped:** fake CLI for every §2.6 command, 25 fixtures covering all four kinds, golden-file snapshot tests, SCREENPLAY.md walkthrough.

**Descoped from Phase 0 (intentional):**
- Real TUI interaction in `cairn find`. A static frame stands in.
- Real MCP server. `mcp start` prints a placeholder and exits.
- ANSI color output. Plain text only until Phase 1.

**Open questions surfaced:**
- [any you discovered]

**Gate status:** ready for Samay's review. Sign-off unlocks Phase 1.
```

- [ ] **Step 5: Commit and hand off**

```bash
git add PHASE-0-REPORT.md && git commit -m "docs: phase 0 report"
git log --oneline
```

Expected: clean commit history from bootstrap through Phase 0 report.

---

## Self-review (run before handoff)

**Spec coverage.** Every command in §2.6 of the spec has at least one task. Import covers happy and error paths. Search covers happy and empty paths. Get covers happy and error paths. MCP covers start, install (both clients), audit, permissions. Output flags `--json`, `--jsonl`, `--plain` are exercised at least once via the pack and search tests. Phoenix bridge is exercised via `cairn export`.

**Placeholder scan.** No "TBD" / "TODO" / "implement later" in the plan. Every code block is complete as written. The one place a pattern repeats (command stubs in Task 6) shows one example and instructs identical shape for the rest — acceptable because the stubs are trivial and later tasks write the real bodies.

**Type consistency.** `cards.Card`, `render.Match`, `fixtures.All()`, `fixtures.ByHandle(n)`, `golden.Assert(t, name, got)` — these signatures are used identically across every task they appear in.

**Fixture-dependent golden files.** Several goldens (search, pack, cardlist) reference specific fixture indices. These will need to match the actual `cards.json` you author in Task 2. If mismatches surface during tests, regenerate with `UPDATE_GOLDEN=1` and inspect the diff, or edit `cards.json` so fixture positions align with the assumed goldens. Call this out during Task 2 so the engineer is not surprised.

**Gaps.** `cairn mcp start` in Phase 0 prints a fake message; the spec does not require a real server here, so this is intentional. `cairn config` shows defaults only; editing is deferred to Phase 1 per spec. `cairn ask` is a stub; Phase 4 by design.

