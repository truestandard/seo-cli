# seo CLI — Specification

> Status: phases 0 to 6 built, 7 = GoReleaser on the public mirror · Target: `https://truestandard.agency` (hosting pending, dev at `http://localhost:3012`) · Authored 2026-08-28 from `/cli-builder`

## 1. Goal

Coding agents (Claude Code, Codex, Cursor, cron) and the people who run them drive TrueStandard Agency without opening the web app: log in with a key, pick a project, read ranks and AI answers, price a paid call, run it, log a shipped page, read the scoreboard. Machine-readable in and out, one exit-code table, and an MCP server that serves the same tools.

## 2. Identity, distribution, repo layout

| Property | Value |
|----------|-------|
| Primary binary | `seo` |
| Alias | none (`seo` is three letters) |
| Go module path | `github.com/truestandard/seo-cli` |
| Config dir | `~/.seo/` (`SEO_CONFIG_DIR` overrides) |
| Env: API base | `SEO_API_URL` |
| Env: token | `SEO_API_KEY` |
| Env: project | `SEO_PROJECT` |
| Token prefix | `seo_` (minted by the backend's `ApiKey.generate!`) |
| Loopback ports | none yet (see §6) |
| Default API base | `https://truestandard.agency` |
| Primary verbs | `ranks`, `scoreboard`, `estimate <paid command>` |

The CLI is the `cli/` directory of the private Rails repo `recmend/seo`. The public repo `truestandard/seo-cli` is a `git subtree` mirror of that directory (`bin/cli_publish`); the module path is the public repo root so `go install github.com/truestandard/seo-cli@latest` works. Extraction to its own source-of-truth repo is deferred.

```
cli/
  main.go
  cmd/            root, auth, projects, context, keywords, paid, ai, site, estimate, mcp, skills, version
  internal/api    client, errors
  internal/auth   login with a key, logout
  internal/config precedence, credentials
  internal/output JSON, exit codes
  internal/skills agent detection, install, doctor
  internal/table  --pretty renderer
  skill/          embedded SKILL.md body
  .goreleaser.yaml, .github/workflows/{ci,release}.yml
```

## 3. Configuration and precedence

1. flag (`--api-url`, `--token`, `--project`)
2. env (`SEO_API_URL`, `SEO_API_KEY`, `SEO_PROJECT`)
3. files: `~/.seo/credentials.json` `{token, api_url}` at 0600; `~/.seo/config.json` `{project, install_skills}`
4. default `https://truestandard.agency`

`SEO_API_KEY` always wins over the saved key; CI needs no file. The API URL is stored beside the token so a key minted against dev does not get sent to production.

## 4. Command reference

```
Auth      auth login --token | auth status | auth logout | whoami | login (hidden alias)
Project   projects | project show|create | use | context get|set|add-competitor|add-page|log
Keywords  keywords list|add|update|remove | research* | serp* | floor*
Ranks     track run*|status | ranks
AI        prompts list|add | ai run*|results|status | mentions*
Site      backlinks* | domain* | audit run*|show|list | gsc import|striking | ship | experiments list|add|outcome
Read      log | scoreboard | spend
Meta      estimate <paid> | mcp | skills install|list|doctor|uninstall | version
```

`*` paid. Global flags: `--project`, `--api-url`, `--token`, `--dry-run`, `--pretty`, `--json`, `--quiet`.

## 5. Output and error conventions

stdout: JSON always; `--pretty` swaps in tables. stderr: notices (`--quiet` silences). Success is the API body unchanged. Errors are `{"error": "<code>", "message": "<text>", ...extra}`. The backend's own envelope is `{error: <message>, code: <code>}`; the client normalizes both shapes.

| Code | Name | When |
|------|------|------|
| 0 | ok | |
| 1 | error | any API error not listed below (not_found, invalid, invalid_params, provider 4xx) |
| 2 | usage | bad flags, bad args, unknown command, `estimate` on a free command |
| 10 | needs_clarification | reserved |
| 11 | not_authenticated | no key, or the backend answers `unauthorized`/`forbidden` |
| 12 | insufficient_credits | reserved: the backend has no credits model yet |
| 13 | rate_limited | reserved |
| 14 | server_error | HTTP 5xx, which is how the backend reports a provider failure |

## 6. Authentication design

6.1 Browser loopback PKCE: not built. The backend has no users table (spec decision 9: single operator, keys only) and no Doorkeeper. When accounts land, add Doorkeeper with the loopback redirect set (`127.0.0.1:4123-4125`) and `auth login` without `--token` opens the browser; the callback belongs on 127.0.0.1 because the CLI is a process on the user's machine with no public URL.

6.2 Headless: `seo auth login --token seo_…` validates against `GET /api/v1/whoami` before saving. `SEO_API_KEY` alone authenticates any run.

6.3 Token model: `seo_` + 40 hex, stored as a SHA-256 digest server-side, revocable there. `auth logout` deletes the local file; there is no revoke endpoint yet.

## 7. Clarify handshake

None. No command asks questions before it runs.

## 8. Execution model

Synchronous. `track run --scheduled` and `audit run` enqueue server-side and return an id; `track status <id>` and `audit show <id>` read it. No `--async` or `--stream`.

## 9. Credits and billing

No credits model. Every paid call writes a `spend_entries` row on the backend; `seo spend` reads it. Every paid command takes `--dry-run` and `seo estimate <command>` wraps it. Exit 12 stays reserved.

## 10. Agent discovery

`auth login` installs `skill/body.md` into `.claude/skills/seo/SKILL.md` (when `.claude/` exists) and as a fenced block in `AGENTS.md` (when `AGENTS.md` or `.codex/` exists). `--no-skills` or `{"install_skills": false}` opts out. `skills doctor --fix` refreshes a stale copy; staleness is a byte compare against the embedded body.

## 11. Rails-side build

Nothing new. The CLI reuses `Api::V1::*` (Bearer, `ApiAuthentication`) and `POST /mcp` (`Mcp::Server`). The docs pages, `public/llms.txt`, `skills/README.md` and `docs/contracts.md` were updated for the Go surface.

## 12. Security

Credentials 0600 in a 0700 dir. The token appears in no log line and no error. `auth status` prints the URL and the identity, never the key. The MCP proxy binds nothing: it is stdio.

## 13. Build and sequencing

| Phase | Status |
|-------|--------|
| 0 auth seam | existing `ApiKey` + Bearer |
| 1 whoami | existing |
| 2 skeleton, config, output, exit codes, `auth login --token` | done |
| 3 first domain command (`projects`, `ranks`) | done |
| 4 full surface (25 commands, `estimate`) | done |
| 5 billing | n/a |
| 6 skills install | done |
| 7 distribution | GoReleaser on the public mirror; tag pending (SEO-25) |

## 14. Acceptance criteria

- `go install github.com/truestandard/seo-cli@latest` yields a working `seo`.
- `seo auth login --token` writes a 0600 credential and installs the skill unless `--no-skills`.
- `SEO_API_KEY` alone authenticates with no file present.
- `seo whoami`, `projects`, `ranks`, `scoreboard` return the API JSON; `--pretty` renders tables.
- Every paid command answers `--dry-run` with `estimate.cost` and spends nothing.
- A bad key exits 11; a bad flag exits 2; `estimate ranks` exits 2.
- `seo mcp` proxies the backend's tool list and fills in the default project.

## 15. Open questions

- ✅ PKCE: deferred until accounts exist (§6.1).
- ✅ Envelope: the backend keeps `{error, code}`; the CLI normalizes. Revisit when a second client appears.
- ✅ Distribution: GoReleaser on the public repo, not Rails-served `/install.sh`, because the repo is public and nothing is hosted yet.
- Revoke endpoint for `auth logout`: open.

## 16. Implementation log

- **2026-08-28 (phases 2 to 7):** Go rewrite of the TypeScript CLI, same command names, JSON-first stdout, exit-code table, `auth` group, `skills` group, dynamic MCP proxy over `github.com/modelcontextprotocol/go-sdk`. Verified live against `http://localhost:3012`: `auth login --token` (0600 file, skill installed), `whoami`, `auth status`, `projects`, `use precis`, `project`, `ranks --since 30d`, `scoreboard`, `keywords`, `context`, `log`, `spend`, `ai results`, `experiments`, `prompts`, `estimate` on all nine paid commands (`track run --live` = $0.84 for 42 keywords), exits 2/11/1 on bad flag/bad key/missing project, `mcp` handshake proxying 29 tools with `get_scoreboard` returning structured content; spend before and after identical ($0.25976). 24 Go tests green. Manual prerequisites: tag `v0.1.0` on the public repo for the first release archives; DNS for truestandard.agency (SEO-33).
