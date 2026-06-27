<!--
Title must follow Conventional Commits in English, e.g. "feat(tui): add PR pane".
Keep the change focused and atomic — one concern per PR.
-->

## What

Briefly, what does this change do?

## Why

The motivation or the problem it solves. Link issues with `Closes #N`.

## How to verify

Steps to see it working (commands, what to look for in the TUI).

## Checklist

- [ ] `make lint` passes (gofmt, vet, staticcheck)
- [ ] `make test` passes (`go test -race`)
- [ ] `make audit` passes (govulncheck)
- [ ] No hardcoded values; new behavior is configurable where it should be
- [ ] No dead code or unnecessary comments
