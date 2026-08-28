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

## 0.1.0 (unpublished)

- TypeScript client for the API and a stdio MCP proxy. Replaced by the Go build before it shipped.
