# Contributing

Thanks for looking. This repo is the client. The backend it talks to is private, so a contribution here changes how a command reads, prints or exits, not what the API returns.

## Setup

Go 1.22 or newer.

```
git clone git@github.com:truestandard/seo-cli.git
cd seo-cli
go build -o seo ./cmd/seo
./seo version
```

Point it at a backend with `--api-url` or `SEO_API_URL`. The key comes from `seo auth login --token` or `SEO_API_KEY`.

## Before you open a pull request

```
gofmt -l .
go vet ./...
go test ./...
```

CI runs the same three steps and a build.

## Rules that keep the CLI usable by agents

- stdout is JSON. Success and errors both. `--pretty` is the only thing that changes it.
- Notices for people go to stderr.
- Exit codes are a contract. Do not add or move one without a note in the README table and in `skill/body.md`.
- A paid command must accept `--dry-run` and print the estimate without spending.
- No code comments. Name things so the code reads on its own.

## Layout

```
cmd/seo/main.go
internal/cli    one file per command group; root.go owns the tree
internal/api    HTTP client and the error envelope
internal/auth   login with a key, logout
internal/config flag > env > file > default
internal/output JSON to stdout, exit codes
internal/skills detect agents, install the skill
internal/table  the --pretty renderer
skill/          the embedded agent skill
```

## Releases

Maintainers tag `vX.Y.Z` on `main`. GoReleaser builds darwin and linux archives for amd64 and arm64 and attaches them to the GitHub release with a checksum file.
