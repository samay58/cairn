# Cairn first-run walkthrough

Phase 1 ships real import, status, search, get, and open backed by SQLite. Find, pack, ask, export, mcp, and config are still Phase 0 fakes; each section below that covers them notes which phase makes it real.

Every fenced output block in the import through open sections was captured by running the binary against `testdata/mymind_sample_export/` with a fresh `CAIRN_HOME`. If you clone the repo and run the same commands you will see the same output.

---

## Importing your mymind export

You have just downloaded your mymind export. Run `cairn import` and point it at the folder.

```console
$ cairn import testdata/mymind_sample_export
```

```
Reading export from testdata/mymind_sample_export
Parsed 4 cards; 4 inserted, 0 updated, 0 tombstoned.
Media: 1 files. Chunks: 3.

Database at ~/.cairn/cairn.db.
Run `cairn search "<query>"` or `cairn find`.
```

Cards are inserted once per import. On subsequent runs the importer matches by mymind ID and emits updated or tombstoned counts instead of inserted. Media files land next to the database; the chunk count reflects FTS5 index rows, not file count.

---

## Checking the library state

`cairn status` gives a one-screen overview of what is loaded, where data lives, and whether the MCP server is wired up.

```console
$ cairn status
```

```
cairn 0.1.0-phase1

library   4 cards · last import 2026-04-21T16:22:11Z · 0 pending
storage   ~/.cairn/cairn.db (96.0 KB) · media cache off
mcp       not installed
clients   none

Phase 1. Import-backed search. Other commands ship later.
```

The timestamp is ISO 8601 UTC. The storage line shows actual file size; the 96 KB for four sample cards gives a sense of overhead before the library grows.

---

## Searching for a topic

You saved a quote about craft and want to find it again. Search by keyword.

```console
$ cairn search "craft"
```

```
@1  On craft
    q · Martha Beck · 2026-03-18
    Matched on title.
    The way you do anything is the way you do everything.
```

The `@N` handle refers to position in this result list and persists across commands. A subsequent `get @1` or `open @1` in the same session resolves to this card via the `handles` table in SQLite; Phase 0 mapped handles positionally from a fixture list, so the cross-command link is one of the things Phase 1 actually ships.

Search accepts filter prefixes. To restrict to notes only:

```console
$ cairn search "type:note"
```

```
@1  Cairn naming
    n · 2026-03-22
    Recent.
    Short. Unclaimed. Fits the category.
```

---

## Reading a card in full

After the search above, `@1` refers to the craft quote. `cairn get @1` prints the full record.

```console
$ cairn get @1
```

```
@1  On craft
q · Martha Beck · 2026-03-18

The way you do anything is the way you do everything.
```

---

## Opening a card in the browser

`cairn open @1` launches the URL in your default OS browser. To preview the URL without opening a browser, set `CAIRN_DRY_OPEN=1`.

```console
$ CAIRN_DRY_OPEN=1 cairn open @1
```

```
Would open: https://access.mymind.com/cards/mm_2
```

In normal use, omit the env var; the binary calls the OS `open` command (macOS) or `xdg-open` (Linux) and returns immediately.

---

## The rest of Phase 0

The following commands shipped in Phase 0 as fakes. They remain unchanged below while their respective phases complete.

- `cairn find`: static TUI mock. Real keyboard-navigable browser ships in Phase 2.
- `cairn pack` and `cairn export`: dry-run output. Real context packing and phoenix vault mirror ship in Phase 2.
- `cairn mcp install`, `cairn mcp start`, `cairn mcp audit`, `cairn mcp permissions`: stub and preview output. Real JSON-RPC MCP server ships in Phase 3.
- `cairn ask`: stub redirecting to `pack`. Natural language question answering over the library ships in Phase 4.
- `cairn config`: shows defaults. No Phase assigned; behavior may stay as-is.

### Browsing the full library

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

### Packing cards for an AI context window

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

### Installing the MCP server in Claude Code

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

### Auditing MCP access

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
