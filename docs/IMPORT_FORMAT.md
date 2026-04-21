# MyMind export format (observed)

Based on a real export to `~/phoenix/Clippings/mymind/` on 2026-04-21.

## Files

- `cards.csv` at the export root.
- Attachments (PDFs, images) at the export root alongside `cards.csv`. No `media/` subdirectory.

## cards.csv

UTF-8 with a BOM on the header line. Columns observed:

| column    | notes                                                                                                                                          |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`      | MyMind card identifier. Used as both `id` and `mymind_id` in cairn's schema.                                                                   |
| `type`    | Capitalized: `Article`, `Document`, `Embed`, `Note`, `WebPage`, `YouTubeVideo`, `Image` (not observed in this export but expected), `Quote` (not observed). |
| `title`   | Human title. May be empty (e.g. for raw note cards).                                                                                           |
| `url`     | Source URL. Empty for notes and many documents.                                                                                                |
| `content` | Body text. May be multi-line with embedded newlines inside CSV quotes.                                                                         |
| `note`    | User-written annotation, separate from `content`. When `content` is empty, cairn uses `note` as the body. When both are populated, cairn appends the note to the body labeled `Note:`. |
| `tags`    | Comma-separated tag list. May contain duplicates in the source data; cairn deduplicates before insert.                                         |
| `created` | ISO-8601 timestamp (RFC 3339).                                                                                                                 |

## Type mapping (cairn Phase 1)

Phase 1 preserves the four-letter kind display (`a/i/q/n`) and maps MyMind types into those buckets:

- `Article`, `WebPage`, `Document`, `Embed`, `YouTubeVideo` → `article` (`a`)
- `Image`, `Photo` → `image` (`i`)
- `Quote` → `quote` (`q`)
- `Note` → `note` (`n`)

Phase 2 may introduce finer-grained kinds (e.g. `document`, `video`) once the UI knows how to render them distinctly.

## Parser behavior

- BOM is stripped from the header.
- Column names are matched case-insensitively.
- Synonyms accepted: `body` ≡ `text` ≡ `content`; `created` ≡ `created_at` ≡ `captured_at` ≡ `date`; `id` ≡ `mymind_id` ≡ `card_id`; `source` ≡ `domain`; `url` ≡ `link`.
- Unknown columns are silently ignored.
- Rows missing `id` or `type` produce a warning and are skipped; the rest of the import continues.
- Empty `title` is allowed (note cards in MyMind commonly have no title).
- Unknown type values produce a warning and skip the row.
- Duplicate tags within a card are deduplicated before insert.

## Media

The scanner walks the export root if no `media/` subfolder exists. It skips `cards.csv`, computes SHA-256 + MIME for every remaining file, and stores the result in the `media` table with kind derived from MIME (`image/*` → image, `video/*` → video, `application/pdf` → document, otherwise `other`). Phase 1 does not link media rows back to specific cards; that mapping lands in Phase 2 once the export format exposes it.
