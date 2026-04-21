# Cairn first-run walkthrough

This document traces a single session from a fresh mymind export through MCP installation. Every command here runs against Phase 0 fixture data; the outputs below match the golden files exactly.

---

## Importing your mymind export

You have just downloaded your mymind CSV export to `/tmp/mymind-export-2026-04-19/`. Run `cairn import` to pull the cards in.

```console
$ cairn import /tmp/mymind-export-2026-04-19/
```

```
Reading export from /tmp/mymind-export-2026-04-19/
Found cards.csv · 25 rows · media folder with 9 files
Parsing 25 cards         done
Extracting 9 media files done
Indexing 63 chunks       done

Imported 25 cards (0 updated, 0 deleted). Database now at ~/.cairn/cairn.db.
Run `cairn search "<query>"` or `cairn find`.
```

If the path does not exist, cairn tells you immediately and shows the last successful import so you have a reference point.

```console
$ cairn import /tmp/does-not-exist
```

```
Error: import failed.

Could not read export directory: /tmp/does-not-exist
Last successful import: 2026-04-19 from /tmp/mymind-export-2026-04-19/

Check the path and try again.
```

---

## Checking the library state

`cairn status` gives a one-screen overview of what is loaded, where data lives, and whether the MCP server is wired up.

```console
$ cairn status
```

```
cairn 0.0.0-phase0

library   25 cards · last import 2026-04-19 · 0 pending
storage   ~/.cairn/cairn.db (0 B) · media cache off
mcp       not installed
clients   none

Phase 0 prototype. Real storage lands in Phase 1.
```

The `mcp not installed` line is the cue to come back to after the walkthrough ends.

---

## Searching for a topic

You saved a handful of cards about OAuth. Searching by keyword returns ranked results with a short reason for each match.

```console
$ cairn search oauth
```

```
@1  OAuth 2.0 Device Authorization Grant
    a · datatracker.ietf.org · 2026-03-14
    matched on title and tag oauth
    Describes the OAuth 2.0 device authorization grant flow for browserless and input-constrained devices.

@2  Proof Key for Code Exchange (PKCE)
    a · oauth.net · 2026-03-27
    matched on tag auth
    PKCE protects public OAuth 2.0 clients from authorization code interception attacks by binding the token request to the original authorization request via a code verifier and challenge.

@3  Why device flow for cairn
    n · 2026-04-03
    matched on body
    Cairn will use device flow for its initial OAuth handshake because the CLI has no embedded browser. The device flow sends the user to a URL on their phone or laptop browser while the CLI polls for token delivery. PKCE layered on top ensures the token exchange can't be intercepted even if the authorization code leaks.
```

The `@N` handles refer to positions in this result list. They carry over to the next `get`, `open`, or `pack` call in the same session.

---

## Browsing the full library

`cairn find` opens a static mock of the TUI browser. Phase 2 makes this interactive.

```console
$ cairn find
```

```
cairn find · 25 cards · type: all  source: all  since: all

› _

@1  OAuth 2.0 Device Authorization Grant
    a · datatracker.ietf.org · 2026-03-14
    recent
    Describes the OAuth 2.0 device authorization grant flow for browserless and input-constrained devices.

@2  On craft
    q · Martha Beck · 2026-03-18
    recent
    The way you do anything is the way you do everything.

@3  Dieter Rams desk, 1970s
    i · vitsoe.com · 2026-03-20
    recent

enter open url · o open in mymind · c copy · y yank · tab cycle filters · esc quit

Phase 0: this is a static mock. Real TUI lands in Phase 2.
```

---

## Reading a card in full

After the search above, `@2` refers to the PKCE card. `cairn get @2` prints the full record.

```console
$ cairn get @2
```

```
@2  On craft
q · Martha Beck · 2026-03-18
tags: craft, philosophy

The way you do anything is the way you do everything.
```

Note: handle resolution in Phase 0 maps `@N` to position `N` in the fixture list, not the search result list. Phase 1 implements real last-list persistence via SQLite.

---

## Packing cards for an AI context window

`cairn pack` takes a query and a target profile, then formats the matching cards as a ready-to-paste context block. The `--for claude` profile emits XML that Claude reads natively.

```console
$ cairn pack "oauth device flow" --for claude
```

```
<context source="cairn" query="oauth device flow" cards="3">
  <card id="c_0001" kind="article" captured="2026-03-14" source="datatracker.ietf.org">
    <title>OAuth 2.0 Device Authorization Grant</title>
    <url>https://datatracker.ietf.org/doc/html/rfc8628</url>
    <excerpt>Describes the OAuth 2.0 device authorization grant flow for browserless and input-constrained devices.</excerpt>
  </card>
  <card id="c_0011" kind="article" captured="2026-03-27" source="oauth.net">
    <title>Proof Key for Code Exchange (PKCE)</title>
    <url>https://oauth.net/2/pkce/</url>
    <excerpt>PKCE protects public OAuth 2.0 clients from authorization code interception attacks by binding the token request to the original authorization request via a code verifier and challenge.</excerpt>
  </card>
  <card id="c_0018" kind="note" captured="2026-04-03" source="">
    <title>Why device flow for cairn</title>
    <excerpt>Cairn will use device flow for its initial OAuth handshake because the CLI has no embedded browser. The device flow sends the user to a URL on their phone or laptop browser while the CLI polls for token delivery. PKCE layered on top ensures the token exchange can't be intercepted even if the authorization code leaks.</excerpt>
  </card>
</context>

Citation key: c_0001, c_0011, c_0018.
```

Paste the output directly into a Claude conversation as context before asking your question.

---

## Installing the MCP server in Claude Code

`cairn mcp install claude-code` previews the config merge without touching the file. Phase 3 performs the real write.

```console
$ cairn mcp install claude-code
```

```
Installing cairn MCP server for claude-code.

Target config: ~/.claude/mcp.json
Existing servers detected: 2 (brave-search, filesystem)

Proposed merge (new keys only):
  mcpServers.cairn = {
    "command": "cairn",
    "args": ["mcp", "start"]
  }

Phase 0 would write this merge. Real install runs in Phase 3.
Run `cairn mcp permissions` to review tool-level scopes.
```

---

## Auditing MCP access

After install, `cairn mcp audit` shows a chronological log of which AI clients searched your library and what they retrieved.

```console
$ cairn mcp audit
```

```
Today

  10:42  Claude Code searched "OAuth MCP authorization"
         Returned 5 snippets

  10:43  Claude Code requested full content for @2
         Allowed once by user

  10:49  Cursor searched "SQLite FTS5"
         Returned 8 snippets

Yesterday

  22:11  Claude Desktop searched "prompt injection mitigations"
         Returned 4 snippets

Phase 0: sample audit rows. Real log reads from the mcp_audit table in Phase 3.
```

The audit log is the primary tool for reviewing what context your AI clients have been pulling from your library.

---

## Phase 0 limits

The following capabilities are stubbed or absent in Phase 0. Each has a planned phase noted.

- No real storage. The database file `~/.cairn/cairn.db` is not written. All data comes from 25 hard-coded fixture cards in `internal/fixtures/cards.json`.
- No interactive TUI. `cairn find` prints a static frame. Real keyboard navigation lands in Phase 2.
- No real MCP server. `cairn mcp start` prints a stub. The actual JSON-RPC server lands in Phase 3.
- No real MCP install. `cairn mcp install` previews the config merge but does not write it.
- No color output. ANSI styling is deferred to Phase 1 when the render pipeline is stable.
- Handle resolution is positional. `@N` maps to fixture card N, not a persisted last-result list. Phase 1 fixes this via SQLite.
- `cairn open` prints a URL but does not invoke the system browser in Phase 0.
- `cairn ask` is a stub. Natural language question answering over the library lands in Phase 4.
- `cairn export` is a dry-run. The real phoenix vault mirror writes in Phase 2.
- The `source=""` attribute on the note card in pack XML output is correct for notes with no source URL. Phase 1 fixture imports will carry real source values.
