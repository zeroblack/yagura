package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	ucli "github.com/urfave/cli/v3"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/scan"
	"github.com/zeroblack/yagura/internal/version"
)

type worktreeOut struct {
	Branch     string `json:"branch"`
	Path       string `json:"path"`
	LiveAgents int    `json:"live_agents"`
}

type repoOut struct {
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	Worktrees []worktreeOut `json:"worktrees"`
}

func renderJSON(snap scan.Snapshot) (string, error) {
	out := []repoOut{}
	for _, r := range snap.Repos {
		ro := repoOut{Name: r.Name, Path: r.Path}
		for _, w := range r.Worktrees {
			ro.Worktrees = append(ro.Worktrees, worktreeOut{Branch: w.Branch, Path: w.Path, LiveAgents: w.LiveAgents()})
		}
		out = append(out, ro)
	}
	b, err := json.MarshalIndent(out, "", "  ")
	return string(b), err
}

func renderTable(snap scan.Snapshot) string {
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "REPO\tBRANCH\tPATH\tLIVE")
	for _, r := range snap.Repos {
		for _, w := range r.Worktrees {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%d\n", r.Name, w.Branch, w.Path, w.LiveAgents())
		}
	}
	tw.Flush()
	return strings.TrimRight(b.String(), "\n")
}

func doctor(cfg config.Config, w io.Writer) error {
	gitPath, gitErr := exec.LookPath("git")
	if gitErr != nil {
		fmt.Fprintln(w, "fail  git    not found in PATH")
	} else {
		fmt.Fprintf(w, "ok    git    %s\n", gitPath)
	}
	if ghPath, err := exec.LookPath("gh"); err != nil {
		fmt.Fprintln(w, "warn  gh     not found (PR badges disabled)")
	} else {
		fmt.Fprintf(w, "ok    gh     %s\n", ghPath)
	}
	if len(cfg.Roots) == 0 {
		fmt.Fprintln(w, "ok    roots  none configured (current directory is used)")
	}
	for _, r := range cfg.Roots {
		if fi, err := os.Stat(r); err == nil && fi.IsDir() {
			fmt.Fprintf(w, "ok    root   %s\n", r)
		} else {
			fmt.Fprintf(w, "warn  root   %s is not an accessible directory\n", r)
		}
	}
	if root := cfg.Agents.ClaudeRoot; root != "" {
		if _, err := os.Stat(root); err == nil {
			fmt.Fprintf(w, "ok    agents %s\n", root)
		} else {
			fmt.Fprintf(w, "warn  agents %s not found (no agent sessions yet?)\n", root)
		}
	}
	fmt.Fprintf(w, "ok    theme  %s\n", cfg.Theme)
	if gitErr != nil {
		return errors.New("git is required")
	}
	return nil
}

func App(cfg config.Config, runTUI func(context.Context, config.Config) error) *ucli.Command {
	return &ucli.Command{
		Name:    "yagura",
		Usage:   "agent-first multi-repo git worktree cockpit",
		Version: version.Version,
		Action: func(ctx context.Context, c *ucli.Command) error {
			return runTUI(ctx, cfg)
		},
		Commands: []*ucli.Command{
			{
				Name:  "list",
				Usage: "list worktrees and agents across repos",
				Flags: []ucli.Flag{&ucli.BoolFlag{Name: "json", Usage: "machine-readable JSON output"}},
				Action: func(ctx context.Context, c *ucli.Command) error {
					snap, err := scan.NewScanner(cfg).Take(ctx)
					if err != nil {
						return err
					}
					out := renderTable(snap)
					if c.Bool("json") {
						if out, err = renderJSON(snap); err != nil {
							return err
						}
					}
					fmt.Println(out)
					return nil
				},
			},
			{
				Name:  "doctor",
				Usage: "validate toolchain and config",
				Action: func(ctx context.Context, c *ucli.Command) error {
					return doctor(cfg, os.Stdout)
				},
			},
		},
	}
}
