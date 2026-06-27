# Contributing to yagura

Thanks for your interest in improving yagura. This document covers everything you need
to make a change land cleanly.

## Development setup

yagura is a single Go module. You need **Go 1.25+** and `git`; `gh` is optional (it
powers the PR badges). Everything else is wired through the `Makefile`.

```sh
make build    # compile to ./bin/yagura
make test     # go test -race ./...
make lint     # gofmt + go vet + staticcheck
make audit    # govulncheck
```

`staticcheck` and `govulncheck` are pinned as module tools, so they use the versions the
project expects — no global install required. CI runs gofmt, vet, staticcheck, the race
detector and govulncheck on every push.

## Architecture

The codebase is layered, and the logic layers never depend on the TUI or the CLI:

```
cmd/yagura/   entrypoint
internal/
  model/      domain types
  git/        git exec with a concurrency semaphore and per-command timeout
  discovery/  recursive repository discovery
  worktree/   porcelain parsing + enrichment
  agents/     agent sources (Claude transcript tail parsing, mtime cache)
  activity/   activity scoring and ordering
  config/     YAML loading, defaults, path normalization
  theme/      themes and color roles
  scan/       snapshot orchestration
  cli/        headless mode
  tui/        Bubble Tea v2 Model-Update-View
```

See [`CONVENTIONS.md`](CONVENTIONS.md) for the full code rules. The essentials:

- **No work inside `Update` or `View`** — subprocesses and file reads run in a `tea.Cmd`
  and return as typed messages. A blocking scan freezes the UI.
- **Cell-width-aware layout** — padding and truncation go through `lipgloss.Width`; never
  `len()`, or wide glyphs break the grid.
- **Nothing hardcoded** — config drives roots, depth, timeouts, theme and every keybinding.
- **No dead code, no unnecessary comments** — a comment documents only a non-derivable why.

## Tests

yagura is built test-first; the logic layers carry the coverage. Add a failing test, make
it pass with the smallest change, keep `go test -race` green. UI rendering is covered by
golden-style view tests under `internal/tui`.

## Submitting a change

1. Branch off `main` with a descriptive name: `feat/...`, `fix/...`, `refactor/...`,
   `docs/...`, `chore/...`.
2. Keep the change focused and atomic — one concern per pull request.
3. Use [Conventional Commits](https://www.conventionalcommits.org/) for commit and PR
   titles, in English: `type(scope): summary`. The PR title feeds the changelog.
4. Make sure `make lint`, `make test` and `make audit` pass.
5. Open a pull request describing **what** changed, **why**, and **how to verify it**.

`main` is always releasable; changes land via squash merge.

## Reporting bugs and proposing features

Open an issue using the templates. For bugs, include your OS, yagura version
(`yagura --version`), and the smallest steps that reproduce the problem. For security
issues, follow [`SECURITY.md`](SECURITY.md) instead of opening a public issue.
