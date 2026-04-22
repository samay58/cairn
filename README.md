<p align="center">
  <img src="docs/assets/cairn-cover.png" alt="Cairn: a stack of stone-bound volumes marking the trail, with more cairns receding along the ridge" width="800">
</p>

# Cairn

A terminal-native bridge between MyMind and the tools I actually work in.

## The problem

MyMind is the best capture tool I have used. One click in the browser and it pulls the article, the image, the quote, the tab, auto-tags it, OCRs it, and tucks it away. My library grows on its own.

Getting things back out is a different story. MyMind has no public API. Their search works fine inside the MyMind app and is invisible to everything else. So the cards I saved sit in a place that Claude Code cannot see, Obsidian cannot see, and my terminal cannot see. The library compounds. The leverage does not.

Cairn is the egress path.

## What it does

Cairn reads a MyMind export into a local SQLite database, builds a full-text search index over it, and mirrors every card into an Obsidian vault as a markdown file with frontmatter and content-addressed attachments.

Once a card is in the vault:

- `cairn search "<something>"` finds it from the terminal in a few milliseconds.
- Obsidian treats it like any other note. Wikilinks, backlinks, and the graph view all work.
- Future MCP support will expose the library to Claude Code and Claude Desktop as tool calls with per-client permissions and an audit trail. That is the point: the things I save should be usable by the AI tools that actually write with me.

Single Go binary. Pure-Go SQLite, no CGo. Local by default. No daemon, no watcher, nothing running in the background.

## What it is not

Not a MyMind clone. Not a second home for cards. Not a capture tool. Not a dashboard. Not a browser. MyMind keeps the capture surface; Cairn plumbs the egress.

## Status

Import, search, and the Phoenix vault mirror are real. Context packs, the fuzzy TUI, MCP, and RAG are designed but each phase has to earn its ship: if I do not reach for the previous phase in real work within a week, the next one does not land. Per-phase honest accounting lives in `PHASE-0-REPORT.md`, `PHASE-1-REPORT.md`, and `PHASE-2A-REPORT.md`.

The full design is at `docs/design/cairn-design.md`. A first-run walkthrough is at `SCREENPLAY.md`.

## Try it

```bash
go build -o cairn ./cmd/cairn
./cairn import /path/to/your/mymind/export
./cairn status
./cairn search "something you remember saving"
./cairn export --to ~/phoenix/Clippings/MyMind
```

`CAIRN_HOME` overrides the default `~/.cairn/` state directory if you want to sandbox a run.

## License

MIT. See `LICENSE`.
