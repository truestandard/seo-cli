`seo` is the command line for TrueStandard Agency, the SEO operator that runs the loop for every project: find the terms you can win, check the target, ship the page, track the rank and the AI answers each week. Every command maps to one API call and prints JSON.

## Setup

- Log in once: `seo auth login --token seo_…` (the key arrives by email at signup). CI and headless runs set `SEO_API_KEY=seo_…` instead; it wins over the saved key.
- The default backend is `https://truestandard.agency`. Point at another one with `--api-url` or `SEO_API_URL`.
- Confirm with `seo whoami`.
- Pick a project once: `seo projects` then `seo use <slug>`. Override per call with `--project <slug>` or `SEO_PROJECT`.

## Read stdout, never stderr

stdout is JSON on success and on failure. Parse it. stderr carries notices for people. `--pretty` switches stdout to tables for a human; never pass it when you parse.

## Commands

```
seo projects                                  list projects
seo project show [slug] | create <slug> --name --domain [--repo-path --location-code --language-code]
seo use <slug>                                set the default project
seo context [get] | set <key> [text] [--file] | add-competitor <domain> | add-page <path> | log <kind> <summary>
seo keywords [list --set] | add <kw...> [--track --path --set --locked --volume --kd] | update <id> [--path --track --status] | remove <id...>
seo research <seeds...> [--limit --max-kd --keywords]            paid
seo serp <keyword> [--depth]                                     paid
seo track [run --live|--scheduled --set] | status <id>           paid
seo ranks [--since 7d] [--set guarantee]
seo prompts [list --set] | add <text...> [--set --locked --file]
seo ai [run --set --runs] | results [--since 30d --set] | status <id>   paid (run)
seo floor <keywords...>                                          paid
seo backlinks [domain] [--history --months --rows]               paid
seo domain [domain] [--limit --force]                            paid, cached 12h
seo mentions [--brand --competitor ... --force]                  paid, cached 24h
seo audit [run --lighthouse --pages --max-pages] | show <id> | list [--limit]   paid with --lighthouse
seo gsc import <csv> --dimension query|page --start --end | striking
seo ship <url> [--keyword --track --note]
seo experiments [list] | add <change...> [--page --hypothesis --keyword --shipped-on] | outcome <id> <text...>
seo log [--kind --days]
seo scoreboard
seo spend [--since 30d]
seo estimate <paid command...>                price it, spend nothing
seo mcp                                       stdio MCP server that proxies the backend tools
seo skills install | list | doctor [--fix] | uninstall
```

Global flags: `--project`, `--api-url`, `--token`, `--dry-run`, `--pretty`, `--json`, `--quiet`.

## Money

Paid commands buy data from providers through the backend. Run `seo estimate <command>` or add `--dry-run` first: the response carries `estimate.cost` and nothing is spent. Every paid call is logged; `seo spend --since 7d` shows it.

Locked keyword and prompt sets are measurement contracts. Never edit them. Add a new set instead.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | ok |
| 1 | error (the JSON on stdout says which) |
| 2 | bad flags or arguments |
| 11 | not authenticated: run `seo auth login --token …` or set `SEO_API_KEY` |
| 14 | the backend or a provider failed; retry later |

The error envelope is `{"error": "<code>", "message": "<text>"}` plus any context fields.

## Weekly loop

```
seo ranks --since 7d
seo ai results --since 30d
seo scoreboard
seo spend --since 7d
```

## MCP

When the agent supports MCP, prefer `seo mcp`: `claude mcp add seo -- seo mcp`. It serves the same tools as the backend and fills in the default project.
