# OpenMessage

Local-first universal message database with built-in MCP server. Ingests messages from SMS/RCS (Google Messages), Google Chat, iMessage, and WhatsApp.

## Architecture

```
├── cmd/              Go CLI commands (pair, serve, send, read, status, import)
├── internal/
│   ├── app/          Bootstrap, data dir, backfill
│   ├── client/       libgm Google Messages protocol
│   ├── db/           SQLite store (conversations, messages, contacts, unified_contacts, drafts)
│   ├── importer/     Multi-platform import adapters (gchat, imessage, whatsapp)
│   ├── story/        Stats computation + narrative story generation
│   ├── tools/        MCP tools (see internal/tools/tools.go Register for the authoritative list)
│   ├── viz/          Relationship visualization renderer (self-contained HTML)
│   └── web/          HTTP API + embedded React UI
├── macos/            Swift macOS app wrapper
├── site/             Static website (deployed to openmessage.ai)
└── vercel.json       Vercel config (root — NOT site/vercel.json)
```

Full MCP tool list, HTTP API reference, v2-cutover detail, `generate_viz` internals, and extended file pointers: **[docs/agent-notes.md](docs/agent-notes.md)**.

## Supporting a live install (READ FIRST for support/debug tasks)

If you are debugging a real user's install — sends failing, re-pairing, reading
their actual messages — read **[docs/agent-runbook.md](docs/agent-runbook.md)**
before touching anything. The traps that cost the most:

- **Two data dirs, not one.** The macOS app's live store is
  `~/Library/Application Support/OpenMessage/` (`OPENMESSAGES_DATA_DIR`). The
  CLI default (`~/.local/share/openmessage/`) is a **separate, usually stale**
  store — point CLI tools at the app dir for live data.
- **Read live messages via the HTTP API** (`/api/conversations/<id>/messages`,
  `/api/search`, `/api/status`), not `sqlite3` directly — the app holds the
  WAL'd DB and a direct reader hits "unable to open database file (14)".
- **Re-pairing Google Messages:** QR is dead; use Google Account cookie
  pairing; clear `session.json` from **both** data dirs; don't over-reconnect
  (it throttles the account). Full recipe in the runbook.

## Local CLI (read-only, no transports)

```bash
openmessage read "<query>" [--limit N] [--phone NUMBER] [--since YYYY-MM-DD] [--until YYYY-MM-DD] [--json]
openmessage status [--json]   # per-platform counts + sync freshness
openmessage import {gchat|gchat-conversation|imessage|whatsapp} <path> [flags]
```

`status` flags any platform whose latest message trails the newest overall by
≥3 days ("Nd behind") — a stale row means the daemon isn't syncing that
platform and searches over that window will miss messages. `imessage` import
needs Full Disk Access.

## MCP serving — exactly one process may own live transports

`serve --mcp-stdio` (the shape MCP hosts spawn per session) is a
**transportless client**: zero transport supervisors/sync loops — the app
daemon owns all live connections (two processes sharing WhatsApp/signal-cli
credentials log each other out). Sends, reactions, and `get_status` route
through the running daemon's HTTP API; reads come from the local store
directly. See [docs/agent-runbook.md](docs/agent-runbook.md) ("MCP serving")
for the failure mode this prevents and the `~/.mcp.json` recipe.

## Vercel deployment (openmessage.ai)

- **Always deploy from the repo root** — `.vercel/project.json` pins the project/scope.
- **Config is root `vercel.json`**, not `site/vercel.json`.
- **Scope: `max-ghenis-projects`** (personal account, NOT PolicyEngine).

```bash
cd /Users/maxghenis/openmessages && vercel --prod
curl -s -o /dev/null -w "%{http_code}" https://openmessage.ai   # always verify after deploy
```

Domains: `openmessage.ai` + `openmessages.ai` alias, both Cloudflare DNS → 76.76.21.21.

## Building the macOS app

```bash
./macos/build.sh           # dev build: com.openmessage.app.dev / "OpenMessage (dev)" — never shadows the installed app
RELEASE=1 ./macos/build.sh # required for anything installable/shippable
```

Dev builds get a distinct bundle identity **on purpose** — a stale dev build
shadowing the installed app in LaunchServices caused two live outages (see
[docs/agent-runbook.md](docs/agent-runbook.md) "Bundle-id shadowing"). A nested
`.claude/worktrees/*` checkout also needs `GOWORK=off` (Go otherwise resolves
the main module via the parent's `go.work`). Install/release commands:
[docs/agent-notes.md](docs/agent-notes.md).

## Testing

```bash
go test ./cmd/ -v   # unit + integration
go test ./... -v    # everything
```

## Agentic story generation

`/generate-story <Name>` produces a fact-grounded relationship visualization by
exploring conversations agentically rather than a single hallucination-prone
API call. Full recipe: `.claude/commands/generate-story.md`. `generate_viz` /
`render_story` section layout and params: [docs/agent-notes.md](docs/agent-notes.md).

## Key files

- `internal/app/app.go` — data dir resolution; **macOS app overrides the CLI default** (see "Two data dirs" above)
- `macos/OpenMessage/Sources/BackendManager.swift` — launches Go backend, manages app state
