# Contributing

Thanks for helping. This file covers setup, tests, commits, and what a pull request needs.

## Setup

```
git clone git@github.com:truestandard/seo-cli.git
cd seo-cli
npm install
npm test
```

Node 20 or newer. `.nvmrc` pins 20.

Run a command from source without building:

```
npm run dev -- ranks --since 7d
```

Point it at a backend with `SEO_BASE_URL` and `SEO_TOKEN`, or run `seo login` once.

## Tests

```
npm test              vitest
npm run typecheck     tsc --noEmit
npm run build         tsc to dist/
```

Tests inject a fake `fetch` through `setFetch` in `src/client.ts`. No test talks to a network. A new command gets a test in `test/commands.test.ts` that asserts the request it sends and the table it prints.

## Code

- One file per command under `src/commands/`. Each maps to one API endpoint in `src/client.ts`.
- Code reads without comments. Pick a name that says what the thing does.
- Runtime dependencies stay at three: `commander`, `@modelcontextprotocol/sdk`, `zod`. Open an issue before adding one.
- The HTTP client is global `fetch`.
- Strict TypeScript. `npm run typecheck` runs with `noUncheckedIndexedAccess` and `exactOptionalPropertyTypes` on.

## Commits

Conventional commits, one change per commit:

```
feat: add seo audit list
fix: ranks prints unranked keywords
docs: cursor mcp block
chore: bump vitest
```

## Pull requests

Before you open one:

- [ ] `npm test`, `npm run typecheck`, and `npm run build` pass
- [ ] A new or changed command has a test
- [ ] README lists the command with its flags
- [ ] CHANGELOG has an entry under Unreleased
- [ ] No comments in code

CI runs the same three scripts on Node 20 and 22.

## Releases

Maintainers cut a release by bumping `version` in `package.json`, moving the Unreleased entries in CHANGELOG under the new version, and publishing a GitHub release with tag `v<version>`. The release workflow then runs `npm publish --provenance --access public`.

The workflow needs an `NPM_TOKEN` repository secret with publish rights on the `@truestandard` scope. Without it the publish step fails.
