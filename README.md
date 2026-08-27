# seo-cli

`seo` is the command-line client and stdio MCP server for the seo backend. It is a thin client: every command maps to one `/api/v1` endpoint, and `seo mcp` proxies the backend's MCP tools to Claude Code, Codex, and Cursor.

You need an API key from a running seo backend. The CLI does nothing without one.

## Install

```
npm i -g @recmend/seo-cli
```

From a checkout:

```
npm install
npm run build
npm link
```

Development runs straight from source with `npm run dev -- <command>` (tsx).

## Login

```
seo login
```

Prompts for the backend URL (default `http://localhost:3011`) and an API token (`seo_…`), verifies them with `GET /api/v1/whoami`, and writes `~/.config/seo/config.json`:

```json
{ "baseUrl": "http://localhost:3011", "token": "seo_…", "project": "my-site" }
```

Environment overrides: `SEO_BASE_URL`, `SEO_TOKEN`, `SEO_PROJECT`. Non-interactive login: `seo login --token seo_… --base-url http://host:3011`.

Pick a default project once:

```
seo projects
seo use my-site
```

## Global options

| Option | Effect |
|---|---|
| `--json` | print the raw API response |
| `--project <slug>` | override the default project |
| `--dry-run` | paid commands: print the cost estimate and spend nothing |
| `--base-url <url>` | override the backend URL |

Exit code is 1 on any API error; the message and code go to stderr.

## Commands

```
login                         verify and save URL + token
projects                      list projects
use <slug>                    set the default project
project [show [slug]|create]  show or create a project
context [get|set <key> <text>|add-competitor <domain>|add-page <path>|log <kind> <summary>]
keywords [list|add <kw...> --track --path --set --locked|update <id> --path --track --status|remove <id...>]
research <seeds...> --limit --max-kd [--keywords]        paid
serp <keyword> --depth                                    paid
track [run --live|--scheduled --set | status <id>]        paid
ranks --since 7d --set guarantee
prompts [list|add <text...> --set --locked --file]
ai [run --set --runs | results --since 30d --set | status <id>]   paid (run)
floor <keywords...>                                       paid
backlinks [domain] --history --months 12 --rows 200       paid
domain [domain] --limit 100 --force                       paid (cached 12h)
mentions --brand --competitor <name>... --force           paid (~$0.85, cached 24h)
audit [run --lighthouse --pages 20 --max-pages 500 | show <id> | list]   paid only with --lighthouse
gsc [import <csv> --dimension query|page --start --end | striking]
ship <url> --keyword --track --note
experiments [list|add <change> --page --hypothesis|outcome <id> <text>]
log --kind --days
scoreboard
spend --since 30d
estimate <command> [args...]  re-run a paid command with dry_run
mcp                           stdio MCP server proxying the backend's tools
```

Paid commands call DataForSEO or OpenRouter through the backend. Add `--dry-run` (or use `seo estimate …`) to see the cost first.

## Workflows

### 1. Weekly check

```
seo ranks --since 7d
seo scoreboard
seo spend --since 7d
```

`ranks` prints keyword, position, previous, delta, band change, and URL for the guarantee set, then one summary line (`top10= top20= top100= unranked= avg_position= floor_target= floor_met=`).

### 2. Add keywords, estimate, run

```
seo keywords add "seo cli" "rank tracker api" --track bofu --path /seo-cli --set guarantee --locked
seo estimate track run --live
seo track run --live
seo track status 42
```

### 3. AI visibility run

```
seo prompts add "best seo cli for coding agents" "how do I track rankings from a terminal" --set guarantee --locked
seo ai run --dry-run
seo ai run --runs 2
seo ai results --since 30d
```

### 4. Floor gate before writing a page

```
seo floor "seo cli" "rank tracker api" --dry-run
seo floor "seo cli" "rank tracker api"
```

Each keyword gets a verdict comparing the weakest page-1 domain's referring domains with your own.

### 5. Ship and verify

```
seo ship https://example.com/seo-cli --keyword "seo cli" --track bofu --note "new landing page"
seo experiments add "rewrote h1 to match query" --page /seo-cli --keyword "seo cli"
seo serp "seo cli" --depth 20
```

`ship` fetches the URL unauthenticated and records `http_status`, the `<h1>`, and `verified`.

### 6. Domain, backlinks, AI mentions

```
seo domain rival.example --limit 50
seo backlinks --history --rows 200
seo mentions --competitor "Rival One" --dry-run
seo mentions --competitor "Rival One"
```

`domain` prints organic traffic and keyword count, the top keywords by traffic with position, volume, CPC, and URL, then the top pages. The backend caches each domain for 12 hours; add `--force` to buy fresh data.

`backlinks --history` adds one row per month (backlinks, referring domains, new, lost); `--rows N` lists the top N referring domains with dofollow counts.

`mentions` reports how often the brand appears in ChatGPT and Google AI answers, the pages those answers cite, sample prompts, and share of voice against any `--competitor`. Cached 24 hours.

### 7. Site audit

```
seo audit run
seo audit run --lighthouse --pages 10
seo audit show 7
seo audit list
```

The crawl is the backend's own (sitemap index aware, key pages, retired paths must be 410) and costs nothing. It runs in the background: `run` returns the run id, `show` prints the summary, issues with severity and fix, and Lighthouse scores when `--lighthouse` was set.

### 8. MCP for coding agents

`seo mcp` fetches `tools/list` from the backend at startup and proxies every `tools/call`, so the tool set is whatever the backend serves. It uses the same config and env vars as the CLI and fills in the default project when a tool takes one and the caller omits it.

Claude Code:

```
claude mcp add seo -- seo mcp
```

Codex (`~/.codex/config.toml`):

```toml
[mcp_servers.seo]
command = "seo"
args = ["mcp"]
```

Cursor (`.cursor/mcp.json` or the global MCP settings):

```json
{
  "mcpServers": {
    "seo": {
      "command": "seo",
      "args": ["mcp"]
    }
  }
}
```

Set `SEO_TOKEN` / `SEO_BASE_URL` in the agent's environment if you do not want to rely on `~/.config/seo/config.json`.

## Development

```
npm test          vitest
npm run typecheck tsc --noEmit
npm run build     tsc to dist/
```

Runtime dependencies: `commander`, `@modelcontextprotocol/sdk`, `zod`. No other packages, and the HTTP client is global `fetch`.

## License

MIT
