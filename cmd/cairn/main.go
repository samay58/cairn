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
