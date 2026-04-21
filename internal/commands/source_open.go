package commands

import (
	"fmt"
	"os"

	"github.com/samay58/cairn/internal/source"
	"github.com/samay58/cairn/internal/storage/sqlite"
)

// openSource returns a Source reading from dbPath. If the file does not yet
// exist it returns a fixture-backed Source so Phase 0 commands continue to
// work before a real import has been run.
func openSource(dbPath string) (source.Source, error) {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return source.NewFixtureSource(), nil
		}
		return nil, fmt.Errorf("stat %s: %w", dbPath, err)
	}
	return sqlite.Open(dbPath)
}
