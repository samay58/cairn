// Package phoenix writes Cairn cards and their media into a Phoenix vault
// (~/phoenix/04-knowledge-base/mymind-cards/) as markdown files with
// content-addressed attachments.
package phoenix

import "github.com/samay58/cairn/internal/cards"

type Writer struct {
	Root   string
	DryRun bool
}

type CardBundle struct {
	Card  cards.Card
	Media []cards.Media
}

type WriteReport struct {
	CardsWritten   int
	CardsUnchanged int
	CardsRenamed   int
	MediaWritten   int
	MediaSkipped   int
	Warnings       []string
}
