# Public repo under @truestandard

Done = package renamed to `@truestandard/seo-cli`, README rewritten as public copy, community files and CI in place, CI green on origin main, gh metadata set, Nightshift site copy points at the new scope.

Assumptions: npm package not published yet (badge shows not found until the first release); no security@ address exists in the Rails docs, so security@truestandard.ai is the contact.

## Steps
1. [x] package.json: scope, author, repository, bugs, homepage, files, keywords
2. [x] README.md rewrite; slop.py clean
3. [x] LICENSE, CONTRIBUTING, CODE_OF_CONDUCT, SECURITY, CHANGELOG, .github/*, .editorconfig, .nvmrc
4. [x] test, typecheck, build green
5. [x] gh repo edit: description, homepage, topics, issues on, wiki and projects off
6. [x] push, confirm CI
7. [x] Rails copy: _install, llms.txt, marketing.en.yml (not committed)

## Decisions
- `seo --help` description now names Nightshift; the old text named "the seo backend", which no user sees.
- CHANGELOG ships in the npm tarball so `npm view` readers see what a version holds.
