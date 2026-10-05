# Cairn Phase 2b: One-Step Sync

**Goal:** The MyMind library reaches the Phoenix vault with one command, read through MyMind's API, and search says something useful about every card.

**Why now:** The Phoenix refresh took three manual steps: `cairn import`, `cairn export --to <path>`, and a shell loop that re-linked raw attachments by sha256. The export default pointed at `research-archive/mymind-cards`, a folder Phoenix never used, so every run needed `--to`. Every card saved since April 2026 arrives with an empty body, because MyMind's CSV now ships title, URL, tags, and notes only; search printed "No excerpt available." for all of them. And MyMind opened a public-beta API in June 2026, which removes the design's binding constraint.

---

## Part 1: shipped

| Change | Where |
|---|---|
| `cairn sync [export-dir] [--to vault]`: import, export, re-link in one process. Defaults: `~/phoenix/Clippings/mymind` and `~/phoenix/04-knowledge-base/mymind-cards`. Plain, `--json`, and `--jsonl` receipts. Refuses a vault path equal to the export folder. | `internal/commands/sync.go` |
| `phoenix.RelinkRaw`: each regular attachment in the export folder whose sha256 matches a `_media/` copy becomes a relative symlink, written beside the file and renamed over it. Files with no vault copy stay and are reported. | `internal/phoenix/relink.go` |
| Export default corrected to `~/phoenix/04-knowledge-base/mymind-cards`. | `internal/commands/export.go` |
| Import and export plumbing extracted (`runImport`, `mirrorToVault`) so `sync` shares the exact code paths of the separate commands. | `import.go`, `export.go` |
| Cards with no text show their first eight tags as the excerpt in `search`, `get`, and `pack`. | `internal/render/cardlist.go` |
| `Book`, `SoftwareApplication`, and `Video` map to article, ending six repeat warnings per run on the real library. | `internal/importer/csv.go` |
| Docs: README Phoenix workflow, design-doc constraint, screenplay paths. | |

**Acceptance, met 2026-10-05:** `go vet` and `go test ./...` pass, with new tests for re-link (link, orphan, relative target, idempotence) and for sync (first run writes and links, second run changes nothing, vault-equals-export refused). On the real library, `cairn sync` reported 964 unchanged, 0 written, 17 already linked, 0 warnings, and the raw folder holds only `cards.csv` plus 17 links.

## Part 2: APISource, shipped the same day

`cairn sync` reads the MyMind API whenever a key is configured (`internal/mymindapi`), and falls back to the export folder without one or with `--from-export`.

What the live API showed, which the transcribed spec did not:

- `GET /objects?limit=10000` returns the whole library, 965 objects in 2.5 MB, for 1 credit. Fetching by `id` also costs 1. Credits are a plan quota (100,000 per 30 days on Mastermind), not a charge; past the quota the API returns 429 until the window resets.
- API ids are the CSV ids with eight leading zeros. `CardID` strips them, so vault filenames and Phoenix's triage state stay keyed the same way.
- `GET /objects/{id}/content` answers 422 "object does not have content" for saved web pages. Only MyMind-native text (notes) has content, and it arrives inline in the list. For everything else the text is MyMind's AI `summary`, present on 838 of 965 objects, with `[[entity]]` markup that cairn strips.
- Notes arrive as ProseMirror JSON (`application/prose+json`); `prose.Text` flattens them to paragraphs. The API held notes on 18 cards.
- `GET /objects/{id}/blob` returns `application/octet-stream`, so the attachment's extension comes from the object's declared `blob.type`.

Bugs the live rehearsal caught before the vault was touched:

- Four cards untitled in the CSV gained titles from the API, and the writer forked them into new files beside the old ones. The writer now indexes the vault by `mymind_id`: a card keeps the file it owns, and a placeholder `...-untitled.md` file is renamed once a title exists.
- The first test run reached the live API through the Keychain. Tests now pin `--from-export` or point `CAIRN_MYMIND_API_URL` at a local server that verifies every request's JWT.

**Acceptance, met 2026-10-05:** on copies of the database and mirror, then on the real vault, the first API sync pulled 965 objects for 1 credit, imported 1 new and 890 updated cards, rewrote 844 mirror files with summaries, renamed 4 placeholders, and left 0 duplicate ids. The second run reported 0 new, 0 updated, 0 written, 0 renamed. A snapshot under half the live library is refused before any tombstone.

Non-goals: writing to MyMind, a daemon, MCP (MyMind ships its own server).
