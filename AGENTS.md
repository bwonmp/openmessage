# AGENTS.md

OpenMessage is a local-first universal message database with a built-in MCP
server, ingesting SMS/RCS (Google Messages), Google Chat, iMessage, and
WhatsApp/Signal into one local inbox (Go CLI/daemon + embedded React web UI +
macOS app wrapper).

## Setup

```bash
go build -o openmessage .
```

Web UI e2e tests additionally need `npm ci` at the repo root.

## Test / validate

Primary command (from `CLAUDE.md`):

```bash
go test ./... -v
```

CI (`.github/workflows/test.yml`) additionally runs, on every PR:

```bash
go test ./... -coverprofile=coverage.out   # enforces a 40% coverage floor
go test -race ./...
npm ci && npx playwright install --with-deps chromium && npm run test:e2e
```

`site/` (the marketing site) and `macos/` (the Swift app) have their own
build/test steps in CI; touch those only if your change is in `site/` or
`macos/`.

## Conventions

- Conventional commits: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`.
- Never push directly to `main`. Always work on your own branch and open a
  PR.
- Keep PRs narrowly scoped to one change.
- If the task names a Linear issue, put its id (`BRY-NNN`) in the PR title.
- Never write credentials to files — pairing sessions and tokens live in the
  user's local data dir, never in the repo.
- Read `CLAUDE.md` for full conventions, including the live-install support
  traps and the dev-vs-release macOS bundle-id rules.
