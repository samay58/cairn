package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

type outputMode string

const (
	outputPlain outputMode = "plain"
	outputJSON  outputMode = "json"
	outputJSONL outputMode = "jsonl"
)

func addOutputFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "emit JSON output")
	cmd.Flags().Bool("jsonl", false, "emit JSONL output")
	cmd.Flags().Bool("plain", false, "emit plain text")
	cmd.Flags().Bool("no-color", false, "disable color output (no-op in Phase 0)")
}

func addListFlags(cmd *cobra.Command) {
	addOutputFlags(cmd)
	cmd.Flags().Int("limit", 0, "cap number of results (0 = default)")
}

func selectedOutputMode(cmd *cobra.Command) (outputMode, error) {
	asJSON, err := cmd.Flags().GetBool("json")
	if err != nil {
		return "", err
	}
	asJSONL, err := cmd.Flags().GetBool("jsonl")
	if err != nil {
		return "", err
	}
	asPlain, err := cmd.Flags().GetBool("plain")
	if err != nil {
		return "", err
	}

	selected := 0
	mode := outputPlain
	if asJSON {
		selected++
		mode = outputJSON
	}
	if asJSONL {
		selected++
		mode = outputJSONL
	}
	if asPlain {
		selected++
		mode = outputPlain
	}
	if selected > 1 {
		return "", fmt.Errorf("choose at most one of --json, --jsonl, or --plain")
	}
	return mode, nil
}

func limitValue(cmd *cobra.Command) (int, error) {
	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		return 0, err
	}
	if limit < 0 {
		return 0, fmt.Errorf("--limit must be non-negative")
	}
	return limit, nil
}

func applyLimit[T any](items []T, limit int) []T {
	if limit <= 0 || limit >= len(items) {
		return items
	}
	return items[:limit]
}
