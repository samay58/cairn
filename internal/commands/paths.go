package commands

import (
	"os"
	"path/filepath"
)

// cairnDBPath resolves the SQLite database location. CAIRN_HOME overrides the
// default ~/.cairn/ directory so tests can isolate state.
func cairnDBPath() string {
	if v := os.Getenv("CAIRN_HOME"); v != "" {
		return filepath.Join(v, "cairn.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".cairn", "cairn.db")
	}
	return filepath.Join(home, ".cairn", "cairn.db")
}
