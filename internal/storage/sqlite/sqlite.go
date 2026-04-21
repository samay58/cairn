package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
	_ "modernc.org/sqlite"
)

type SQLiteSource struct {
	DB *sql.DB
}

// Open opens the SQLite database at path, running any pending migrations.
// Callers are responsible for calling Close.
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
	out, err := s.loadCards(`SELECT id, mymind_id, kind, title, coalesce(url,''), coalesce(body,''),
		coalesce(excerpt,''), coalesce(source,''), captured_at FROM cards
		WHERE deleted_at IS NULL ORDER BY captured_at DESC`)
	if err != nil {
		return nil
	}
	s.hydrateTags(out)
	return out
}

func (s *SQLiteSource) ByHandle(n int) (cards.Card, error) {
	row := s.DB.QueryRow(`SELECT cards.id, cards.mymind_id, cards.kind, cards.title, coalesce(cards.url,''),
		coalesce(cards.body,''), coalesce(cards.excerpt,''), coalesce(cards.source,''), cards.captured_at
		FROM handles JOIN cards ON cards.id = handles.card_id
		WHERE handles.position = ? AND cards.deleted_at IS NULL`, n)
	c, err := scanCard(row)
	if err != nil {
		return cards.Card{}, fmt.Errorf("no card at handle @%d (run `cairn search` or `cairn find` to refresh handles)", n)
	}
	s.hydrateTags([]cards.Card{c})
	return c, nil
}

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
	coalesce(cards.body,''), coalesce(cards.excerpt,''), coalesce(cards.source,''), cards.captured_at
	FROM cards ` + join + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY cards.captured_at DESC`

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
		c, err := scanCardRow(rows)
		if err != nil {
			continue
		}
		matches = append(matches, render.Match{Card: c, WhyShown: whyShownFTS(c, ftsQ)})
	}
	if len(matches) > 0 {
		list := make([]cards.Card, len(matches))
		for i := range matches {
			list[i] = matches[i].Card
		}
		s.hydrateTags(list)
		for i := range matches {
			matches[i].Card = list[i]
		}
	}
	return matches
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
		if m.Card.ID == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO handles(position, card_id, created_at) VALUES (?, ?, ?)`, i+1, m.Card.ID, now); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// --- helpers ---

func (s *SQLiteSource) loadCards(query string, args ...any) ([]cards.Card, error) {
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []cards.Card
	for rows.Next() {
		c, err := scanCardRow(rows)
		if err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *SQLiteSource) hydrateTags(list []cards.Card) {
	for i, c := range list {
		trows, err := s.DB.Query(`SELECT tag FROM tags WHERE card_id = ? ORDER BY tag`, c.ID)
		if err != nil {
			continue
		}
		var tags []string
		for trows.Next() {
			var t string
			if err := trows.Scan(&t); err == nil {
				tags = append(tags, t)
			}
		}
		trows.Close()
		list[i].Tags = tags
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCardRow(r rowScanner) (cards.Card, error) {
	return scanCard(r)
}

func scanCard(r rowScanner) (cards.Card, error) {
	var c cards.Card
	var captured string
	if err := r.Scan(&c.ID, &c.MyMindID, &c.Kind, &c.Title, &c.URL, &c.Body, &c.Excerpt, &c.Source, &captured); err != nil {
		return cards.Card{}, err
	}
	c.CapturedAt, _ = time.Parse(time.RFC3339, captured)
	return c, nil
}

// parseQuery splits a raw search string into FTS terms plus Filters, respecting
// any pre-populated Filters passed in (command-provided filters win).
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

func escapeFTSTerm(t string) string {
	t = strings.ReplaceAll(t, `"`, `""`)
	return `"` + t + `"`
}

func whyShownFTS(c cards.Card, q string) string {
	if q == "" {
		return "Recent."
	}
	lo := strings.ToLower(q)
	if strings.Contains(strings.ToLower(c.Title), lo) {
		return "Matched on title."
	}
	return "Matched on body."
}
