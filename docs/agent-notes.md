# Agent notes: MCP tools, HTTP API, and viz internals

Reference detail trimmed out of `CLAUDE.md` to keep that file cheap to load
every session. Nothing here is more current than `internal/tools/tools.go`
`Register` — treat that as the authoritative tool list if this drifts.

## MCP tools

- `get_messages`, `get_conversation`, `search_messages` — cross-platform by default
- `list_conversations` — optional `source_platform` filter (sms, gchat, imessage, whatsapp)
- `get_person_messages` — all messages with a person across all platforms
- `get_person_messages_range` — date-filtered version of get_person_messages (for deep-diving into specific periods)
- `import_messages` — import from any supported source
- `conversation_stats` — volume, heatmap, phrases, response times, gaps (single conversation)
- `generate_story` — narrative chapters with optional Claude API enhancement (single conversation)
- `person_stats` — cross-platform stats for all 1:1 messages with a person (merges + deduplicates)
- `generate_person_story` — cross-platform narrative story for a person (merges + deduplicates)
- `generate_viz` — self-contained HTML visualization combining data dashboards + narrative (see "Relationship visualization" below)
- `render_story` — render a pre-built Story JSON into HTML viz; supports `photo_paths` (curated list) or `photos_dir`
- `send_message`, `draft_message`, `download_media`, `list_contacts`, `get_status`

### v2 cutover: read-path caps and frozen tools

On a v2-primary install the message-read tools (including `get_person_messages`
and `get_person_messages_range`) serve the v2 store through the canonical read
seam; on v2 their `limit` is capped (500 and 2,000, and the output says when
the cap applied) because the v2 batch path fans out per matching conversation,
while the legacy single-query path keeps the caller's value above a floor of 1
(a negative SQLite LIMIT means no limit). Date arguments are YYYY-MM-DD in
local time on every surface (CLI, HTTP, MCP) via `db.ParseDayBound`, and the
range tool labels rows in that same local time. The stats/story/viz tools
(`conversation_stats`, `generate_story`, `person_stats`, `generate_person_story`,
`generate_viz`, `render_story`) still load full histories from the legacy
store, which froze at cutover, so they return an error naming the working read
tools instead.

## HTTP API

- `GET /api/stats/{conversation_id}` — conversation statistics JSON
- `GET /api/story/{conversation_id}?style=intimate&api_key=...` — generated story JSON
- `GET /api/conversations?limit=50` — list all conversations (all platforms)
- `GET /api/search?q=...` — **conversation-level** search: one row per matching
  conversation (`ConversationID`/`Name`/`Participants`/`preview`), matched by
  message text and by conversation name/participants. This feeds the web UI's
  search box — it does not return message rows.
- `GET /api/search/messages?q=...` — **message-level** search: raw message DTOs
  (`MessageID`/`Body`/`TimestampMS`…, same shape as
  `/api/conversations/<id>/messages`), the HTTP twin of the `search_messages`
  MCP tool. Optional `phone`, `conversation_id`, `since`/`until` (YYYY-MM-DD,
  local time, `until` inclusive to end of day), `limit` (default 50, max 500).

## Schema

Messages and conversations have `source_platform` (sms/gchat/imessage/whatsapp/signal/telegram) and messages have `source_id` for dedup. Unified contacts table maps people across platforms.

## Relationship visualization (`generate_viz`)

Generates a self-contained HTML file combining data dashboards with narrative chapters. Output is deployable to Vercel or viewable locally.

**Sections**: password gate, hero, timeline nav, narrative chapters (early/middle/late), monthly volume chart (Chart.js), sender split donut, response times, hour-of-week heatmap, phrase cloud (colored by sender ratio), longest gap callout, interspersed photo breaks (chronologically aligned), interludes, closing.

**Key parameters**: `name` (person to search), `output_path` (relative to `OPENMESSAGES_EXPORT_DIR`, default `~/Documents/OpenMessage`, unless `OPENMESSAGES_ALLOW_ANY_EXPORT_PATH=1` is set), `timezone` (default ET), `password`, `api_key` (for Claude-generated narrative), colors (`primary_color`, `secondary_color`, etc.).

**Architecture**:
- `internal/viz/config.go` — `VizConfig` struct, section ordering, color theming
- `internal/viz/render.go` — `RenderHTML()` orchestrator, Chart.js data building
- `internal/viz/template.go` — Go html/template with all CSS/JS inline (except CDN fonts + Chart.js)
- `internal/viz/photos.go` — `Photo` struct, `EncodePhotosFromDir/Paths()`, date parsing from filenames, chronological sorting
- `internal/tools/viz.go` — MCP tool handler

**Stats engine extensions** (`internal/story/stats.go`):
- `PhraseCount.BySender` — per-sender phrase counts for colored word cloud
- `ComputeStats(messages, tz)` — timezone parameter for TZ-shifted heatmap

## Agentic story generation (`/generate-story`)

Claude Code slash command that produces fact-grounded relationship visualizations. Instead of a single-pass API call that hallucinates, the agent explores conversations agentically:

1. `person_stats` → identify 4-8 pivotal periods from volume patterns
2. `get_person_messages_range` → deep-dive into each period's actual messages
2.5. Photo curation → visually inspect candidate photos, select best 15-25
3. Write chapters grounded in real quotes and events
4. `render_story` → combine narrative with data dashboards into HTML

**Usage:** `/generate-story Jenn` from Claude Code in this project.

**Key tools:**
- `get_person_messages_range` — date-filtered cross-platform messages for deep-dives
- `render_story` — accepts pre-built Story JSON + person name, computes stats, renders HTML

**Command file:** `.claude/commands/generate-story.md` (full phase-by-phase recipe and factual-grounding rules).

## macOS build & release commands

```bash
cp -R macos/build/OpenMessage.app /Applications/ && xattr -cr /Applications/OpenMessage.app   # install locally (needs RELEASE=1 build)
gh release upload v0.1.0 macos/build/OpenMessage.dmg --repo MaxGhenis/openmessage --clobber    # update GitHub release
```

## Key files (extended)

- `internal/db/db.go` — schema, structs, migration
- `internal/importer/` — gchat.go, imessage.go, whatsapp.go
- `internal/story/stats.go` — conversation statistics computation (with timezone + per-sender phrases)
- `internal/story/generate.go` — narrative story generation (local or Claude API)
- `internal/viz/` — relationship visualization renderer (config, template, render, photos)
- `internal/client/events.go` — handles Google Messages protocol events
