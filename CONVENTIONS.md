# Conventions — yagura

## Layout

```
cmd/yagura/          entrypoint
internal/
  model/             domain types (Repo, Worktree, AgentSession, enums)
  git/               git exec with a concurrency semaphore and per-command timeout
  discovery/         recursive repository discovery
  worktree/          porcelain parsing + enrichment (last commit, file activity)
  agents/            AgentSource + ClaudeSource (transcript tail parsing, mtime cache)
  activity/          activity scoring and ordering
  config/            YAML loading, defaults, path normalization
  theme/             themes (evangelion) and color roles
  scan/              snapshot orchestrator (Scanner)
  cli/               headless mode (urfave/cli v3)
  tui/               Bubble Tea v2 Model-Update-View
```

Each package has a single responsibility and is tested in isolation. The logic
layers never depend on the TUI or the CLI.

## Code rules

- **Zero unnecessary comments.** A comment only documents a non-derivable why
  (hidden invariant, workaround). Descriptive names do the rest.
- **TDD**: failing test → minimal implementation → green. Coverage target is
  ≥70% on the logic layers (discovery, worktree, agents, activity).
- `gofmt` and `go vet` stay clean at all times; CI also runs the race detector.
- **Nothing hardcoded**: roots, depth, agent windows, intervals, git limits,
  theme and keybindings come from config with sensible defaults.
- **No blocking work inside `Update` or `View`**: subprocesses and file reads
  run inside `tea.Cmd` and return as typed messages.
- **Cell-width aware layout**: padding and truncation go through
  `lipgloss.Width` (wide glyphs span two cells) — never `len()` for layout.
- **No dead code**: replaced code is deleted, not kept "just in case".
- **DRY without overengineering**: extract on the second occurrence, not the
  hypothetical one.

## Stack

Go 1.25+, Charm v2 (`charm.land/{bubbletea,lipgloss}/v2`), `urfave/cli/v3`,
`yaml.v3`, `testify`. Build, test and lint through the `Makefile`.

## Git

GitHub Flow: one short-lived branch per change (`type/short-desc`), titles in
Conventional Commits, squash merge. `main` is always releasable.

## License

Apache-2.0. Pieces adapted from `chmouel/lazyworktree` keep their attribution
in `NOTICE`.
