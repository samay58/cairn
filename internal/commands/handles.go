package commands

import (
	"fmt"
	"strconv"
	"strings"
)

func parseHandle(ref string) (int, error) {
	if !strings.HasPrefix(ref, "@") {
		return 0, fmt.Errorf("Only @handle references are supported. Got %q.", ref)
	}

	n, err := strconv.Atoi(strings.TrimPrefix(ref, "@"))
	if err != nil {
		return 0, fmt.Errorf("Invalid handle: %q.", ref)
	}
	return n, nil
}
