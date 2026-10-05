# Cairn Phase 2b: One-Step Sync

**Goal:** A fresh MyMind export reaches the Phoenix vault with one command, and search says something useful about cards that carry no text. Then, once a key exists, read MyMind's API instead of the export file.

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

## Part 2: APISource (gated on a key)

Facts checked against MyMind's announcement and the published OpenAPI description (`0.4.0-alpha`, transcribed in `nawwwal/mymind`):

- Base URL `https://api.mymind.com`. Each request carries a fresh HS256 JWT: header `kid` is the key id, claims are `method`, `path` (no query string), `iat`, `exp` (`iat + 300`), signed with the base64-decoded key secret.
- `GET /objects?limit=10000` lists every object; without `q` it costs about one credit. Object fields: `id`, `title`, `summary` or `ai.summary`, `content`, `notes[]`, `tags[]`, `spaces[]`, `source.url`, `created`, `modified`, `deleted`.
- `GET /objects/{id}/content` with `Accept: text/markdown` returns the full text for one credit.
- Keys are read-only or full access. Cairn needs read-only.

Plan:

1. `internal/mymindapi`: signer plus a client for the two calls above, with `RateLimit` headers surfaced in the receipt.
2. Map an Object to `cards.Card`: `summary` to `Excerpt`, content to `Body`, notes appended as the CSV path does (`Note: ...`), `source.url` to `URL`, `created` to `CapturedAt`.
3. Split `importer.Import` at the point where parsed cards enter the transaction (`ImportCards(db, sourceLabel, cards, mediaDir)`), so CSV and API share upsert, tombstone, chunk, and FTS logic.
4. `cairn sync --api` lists objects, fetches content only for cards whose `modified` moved or whose body is empty, capped by `--max-content` (default 50 per run), and records the source as `api` in `sync_log`.
5. Credentials from `CAIRN_MYMIND_KID` / `CAIRN_MYMIND_SECRET`, else the macOS Keychain items `mymind-api-kid` / `mymind-api-secret`.

Acceptance: one live run against the full library returns the same 964 ids as the CSV import, fills summaries for every card, fetches content within the cap, and a second run spends no content credits. Until that run happens the code stays off `main`; a client built only against a transcribed spec is a guess.

Non-goals: writing to MyMind, a daemon, MCP (MyMind ships its own server).
