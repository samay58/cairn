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
	once    sync.Once
	loaded  []cards.Card
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
