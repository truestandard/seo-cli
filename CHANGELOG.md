# Changelog

## Unreleased

- Rewritten in Go as one static binary. No Node runtime.
- stdout is JSON on success and on failure. `--pretty` prints tables.
- Exit codes: 0 ok, 1 error, 2 usage, 11 not authenticated, 14 server or provider failure.
- `seo auth login --token`, `seo auth status`, `seo auth logout`, `seo whoami`.
- `SEO_API_KEY`, `SEO_API_URL`, `SEO_PROJECT` win over the saved files.
- `seo skills install` writes an agent skill into `.claude/skills/seo/` or a managed block in `AGENTS.md`. Login does it once.
- New commands: `domain`, `mentions`, `audit`.
- Default backend is https://truestandard.agency.
- New commands: `triage` (who is searching each keyword, and which keywords one page answers), `links` (audited pages ranked as internal-link sources for a live url or a draft), `fit` (does each tracked keyword's page answer it up front, in its words, in the right form).
- `research` prints the triage of what it bought; `ship` prints ranked link sources and whether each links to the new page yet.
- New command: `competitors watch`. It reads the weekly watch: new pages on each rival's site, the searches they gained and lost, and the searches worth a page from you. `--run` takes a snapshot now.
- `ai results` shows how each answer treats the brand, and counts rivals the answers recommend apart from rivals they only name.

## 0.1.0 (unpublished)

- TypeScript client for the API and a stdio MCP proxy. Replaced by the Go build before it shipped.
