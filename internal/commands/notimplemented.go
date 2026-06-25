package commands

import (
	"fmt"
	"io"
)

func writeNotImplemented(out io.Writer, message, next string) error {
	if _, err := fmt.Fprintln(out, message); err != nil {
		return err
	}
	if next == "" {
		return nil
	}
	_, err := fmt.Fprintln(out, next)
	return err
}
