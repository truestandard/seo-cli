# seo-cli initial build

Done = `npm test`, `npm run build`, `npm run typecheck` all green; every command in contracts.md CLI list registered; `seo mcp` proxies tools/list + tools/call to the backend; README with install, login, 6 workflows; initial commit.

Assumptions: backend not running, so tests use an injected fetch; response envelopes accept both bare arrays and `{<resource>: [...]}` since contracts.md does not pin them.

## Steps
1. [x] Read contracts.md, spec.md, schema migration for field names
2. [x] package.json, tsconfig, deps (commander, @modelcontextprotocol/sdk, zod; dev typescript, vitest, @types/node, tsx)
3. [x] src/config.ts, src/client.ts (one fn per endpoint, since helper, fetch setter), src/output.ts
4. [x] src/commands/*.ts + src/program.ts + src/index.ts
5. [x] tests: config, client, since, output, ranks, research --dry-run
6. [x] README.md, LICENSE, .gitignore
7. [x] verify: test/build/typecheck, smoke `npx tsx src/index.ts --help`
8. [ ] commit

## Decisions
- tsx added as a devDependency so `npm run dev` never hits an npx install prompt.
- Low-level `Server` from the MCP SDK (not `McpServer`) because it passes backend JSON schemas through untouched.
- MCP proxy fills `project`/`slug` args from config when the tool schema declares them and the caller omits them.
- Coordinator changed `frozen` to `locked` mid-build (API field + CLI flag); applied to keywords add, prompts add, and list columns.
- MCP proxy verified over real stdio against a fake /mcp backend in the scratchpad (tools/list, tools/call, default project fill, 401 path).
