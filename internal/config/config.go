package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type AgentsConfig struct {
	ClaudeRoot   string
	ActiveWindow time.Duration
	ToolTimeout  time.Duration
}

type GitConfig struct {
	MaxProcs int
	Timeout  time.Duration
}

type RefreshConfig struct {
	Tick time.Duration
}

type SortConfig struct {
	Default     string
	GroupByRepo bool
}

type InspectConfig struct {
	ChangesSort    string
	GraphMax       int
	FileLimit      int
	AttentionFirst bool
}

type ForgeConfig struct {
	Enabled string
	TTL     time.Duration
}

type DisplayConfig struct {
	ProjectFrom       string
	ProjectContainers []string
}

type FXConfig struct {
	Enabled   bool
	Bars      bool
	Pulse     bool
	Depth     bool
	Gradients bool
	BarWidth  int
	BarCap    int
}

type KeysConfig struct {
	Inspect    string
	Quit       []string
	Close      []string
	FocusNext  []string
	FocusPrev  []string
	PaneList   []string
	PaneChange []string
	PaneGraph  []string
	PanePR     []string
	Up         []string
	Down       []string
	PageUp     []string
	PageDown   []string
	Home       []string
	End        []string
	Pin        []string
	Group      []string
	Refresh    []string
	Diff       []string
	Commits    []string
	Status     []string
	AgentLog   []string
}

type Config struct {
	Roots    []string
	MaxDepth int
	Ignore   []string
	Agents   AgentsConfig
	Git      GitConfig
	Refresh  RefreshConfig
	Sort     SortConfig
	Inspect  InspectConfig
	Forge    ForgeConfig
	Display  DisplayConfig
	Keys     KeysConfig
	Theme    string
	FX       FXConfig
}

func Default() Config {
	return Config{
		MaxDepth: 4,
		Ignore:   []string{"node_modules", "vendor", "dist", ".next", "target", ".venv"},
		Agents: AgentsConfig{
			ClaudeRoot:   defaultClaudeRoot(),
			ActiveWindow: 10 * time.Minute,
			ToolTimeout:  30 * time.Second,
		},
		Git:     GitConfig{MaxProcs: 0, Timeout: 10 * time.Second},
		Refresh: RefreshConfig{Tick: 5 * time.Second},
		Sort:    SortConfig{Default: "activity", GroupByRepo: true},
		Inspect: InspectConfig{ChangesSort: "mtime", GraphMax: 200, FileLimit: 40, AttentionFirst: true},
		Forge:   ForgeConfig{Enabled: "auto", TTL: 60 * time.Second},
		Display: DisplayConfig{ProjectFrom: "parent", ProjectContainers: []string{"app", "src", "apps", "packages"}},
		Keys:    defaultKeys(),
		Theme:   "evangelion",
		FX:      FXConfig{Enabled: true, Bars: true, Pulse: true, Depth: true, Gradients: true, BarWidth: 7, BarCap: 20},
	}
}

func defaultClaudeRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

func defaultKeys() KeysConfig {
	return KeysConfig{
		Inspect:    "enter",
		Quit:       []string{"q", "ctrl+c"},
		Close:      []string{"esc"},
		FocusNext:  []string{"tab", "]"},
		FocusPrev:  []string{"shift+tab", "["},
		PaneList:   []string{"1"},
		PaneChange: []string{"2"},
		PaneGraph:  []string{"3"},
		PanePR:     []string{"4"},
		Up:         []string{"up", "k"},
		Down:       []string{"down", "j"},
		PageUp:     []string{"pgup"},
		PageDown:   []string{"pgdown"},
		Home:       []string{"g", "home"},
		End:        []string{"G", "end"},
		Pin:        []string{"space"},
		Group:      []string{"o"},
		Refresh:    []string{"r"},
		Diff:       []string{"d"},
		Commits:    []string{"c"},
		Status:     []string{"s"},
		AgentLog:   []string{"l"},
	}
}

type fileConfig struct {
	Roots    []string `yaml:"roots"`
	MaxDepth *int     `yaml:"max_depth"`
	Ignore   []string `yaml:"ignore"`
	Agents   struct {
		ClaudeRoot   *string `yaml:"claude_root"`
		ActiveWindow *string `yaml:"active_window"`
		ToolTimeout  *string `yaml:"tool_timeout"`
	} `yaml:"agents"`
	Git struct {
		MaxProcs *int    `yaml:"max_procs"`
		Timeout  *string `yaml:"timeout"`
	} `yaml:"git"`
	Refresh struct {
		Tick *string `yaml:"tick"`
	} `yaml:"refresh"`
	Sort struct {
		Default     *string `yaml:"default"`
		GroupByRepo *bool   `yaml:"group_by_repo"`
	} `yaml:"sort"`
	Inspect struct {
		ChangesSort    *string `yaml:"changes_sort"`
		GraphMax       *int    `yaml:"graph_max"`
		FileLimit      *int    `yaml:"file_limit"`
		AttentionFirst *bool   `yaml:"attention_first"`
	} `yaml:"inspect"`
	Forge struct {
		Enabled *string `yaml:"enabled"`
		TTL     *string `yaml:"ttl"`
	} `yaml:"forge"`
	Display struct {
		ProjectFrom       *string  `yaml:"project_from"`
		ProjectContainers []string `yaml:"project_containers"`
	} `yaml:"display"`
	Keys struct {
		Inspect    *string  `yaml:"inspect"`
		Quit       []string `yaml:"quit"`
		Close      []string `yaml:"close"`
		FocusNext  []string `yaml:"focus_next"`
		FocusPrev  []string `yaml:"focus_prev"`
		PaneList   []string `yaml:"pane_list"`
		PaneChange []string `yaml:"pane_changes"`
		PaneGraph  []string `yaml:"pane_graph"`
		PanePR     []string `yaml:"pane_prs"`
		Up         []string `yaml:"up"`
		Down       []string `yaml:"down"`
		PageUp     []string `yaml:"page_up"`
		PageDown   []string `yaml:"page_down"`
		Home       []string `yaml:"home"`
		End        []string `yaml:"end"`
		Pin        []string `yaml:"pin"`
		Group      []string `yaml:"group"`
		Refresh    []string `yaml:"refresh"`
		Diff       []string `yaml:"diff"`
		Commits    []string `yaml:"commits"`
		Status     []string `yaml:"status"`
		AgentLog   []string `yaml:"agent_log"`
	} `yaml:"keys"`
	Theme *string `yaml:"theme"`
	FX    struct {
		Enabled   *bool `yaml:"enabled"`
		Bars      *bool `yaml:"bars"`
		Pulse     *bool `yaml:"pulse"`
		Depth     *bool `yaml:"depth"`
		Gradients *bool `yaml:"gradients"`
		BarWidth  *int  `yaml:"bar_width"`
		BarCap    *int  `yaml:"bar_cap"`
	} `yaml:"fx"`
}

func Load(path string) (Config, error) {
	c := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.normalize()
			return c, nil
		}
		return c, err
	}
	var f fileConfig
	if err := yaml.Unmarshal(data, &f); err != nil {
		return c, err
	}
	f.applyTo(&c)
	c.normalize()
	return c, nil
}

func (c *Config) normalize() {
	for i, r := range c.Roots {
		c.Roots[i] = ExpandHome(r)
	}
	c.Agents.ClaudeRoot = ExpandHome(c.Agents.ClaudeRoot)
	if c.Git.MaxProcs < 0 {
		c.Git.MaxProcs = 0
	}
}

func (f fileConfig) applyTo(c *Config) {
	if len(f.Roots) > 0 {
		c.Roots = f.Roots
	}
	if f.MaxDepth != nil && *f.MaxDepth > 0 {
		c.MaxDepth = *f.MaxDepth
	}
	if len(f.Ignore) > 0 {
		c.Ignore = f.Ignore
	}
	if f.Agents.ClaudeRoot != nil {
		c.Agents.ClaudeRoot = *f.Agents.ClaudeRoot
	}
	if d, ok := parseDur(f.Agents.ActiveWindow); ok {
		c.Agents.ActiveWindow = d
	}
	if d, ok := parseDur(f.Agents.ToolTimeout); ok {
		c.Agents.ToolTimeout = d
	}
	if f.Git.MaxProcs != nil && *f.Git.MaxProcs >= 0 {
		c.Git.MaxProcs = *f.Git.MaxProcs
	}
	if d, ok := parseDur(f.Git.Timeout); ok {
		c.Git.Timeout = d
	}
	if d, ok := parseDur(f.Refresh.Tick); ok {
		c.Refresh.Tick = d
	}
	if f.Sort.Default != nil {
		c.Sort.Default = *f.Sort.Default
	}
	if f.Sort.GroupByRepo != nil {
		c.Sort.GroupByRepo = *f.Sort.GroupByRepo
	}
	if f.Inspect.ChangesSort != nil {
		c.Inspect.ChangesSort = *f.Inspect.ChangesSort
	}
	if f.Inspect.GraphMax != nil && *f.Inspect.GraphMax > 0 {
		c.Inspect.GraphMax = *f.Inspect.GraphMax
	}
	if f.Inspect.FileLimit != nil && *f.Inspect.FileLimit > 0 {
		c.Inspect.FileLimit = *f.Inspect.FileLimit
	}
	if f.Inspect.AttentionFirst != nil {
		c.Inspect.AttentionFirst = *f.Inspect.AttentionFirst
	}
	if f.Forge.Enabled != nil && *f.Forge.Enabled != "" {
		c.Forge.Enabled = *f.Forge.Enabled
	}
	if d, ok := parseDur(f.Forge.TTL); ok {
		c.Forge.TTL = d
	}
	if f.Display.ProjectFrom != nil && *f.Display.ProjectFrom != "" {
		c.Display.ProjectFrom = *f.Display.ProjectFrom
	}
	if len(f.Display.ProjectContainers) > 0 {
		c.Display.ProjectContainers = f.Display.ProjectContainers
	}
	f.applyKeys(&c.Keys)
	if f.Theme != nil && *f.Theme != "" {
		c.Theme = *f.Theme
	}
	f.applyFX(&c.FX)
}

func (f fileConfig) applyFX(fx *FXConfig) {
	if f.FX.Enabled != nil {
		fx.Enabled = *f.FX.Enabled
	}
	if f.FX.Bars != nil {
		fx.Bars = *f.FX.Bars
	}
	if f.FX.Pulse != nil {
		fx.Pulse = *f.FX.Pulse
	}
	if f.FX.Depth != nil {
		fx.Depth = *f.FX.Depth
	}
	if f.FX.Gradients != nil {
		fx.Gradients = *f.FX.Gradients
	}
	if f.FX.BarWidth != nil && *f.FX.BarWidth > 0 {
		fx.BarWidth = *f.FX.BarWidth
	}
	if f.FX.BarCap != nil && *f.FX.BarCap > 0 {
		fx.BarCap = *f.FX.BarCap
	}
}

func (f fileConfig) applyKeys(k *KeysConfig) {
	if f.Keys.Inspect != nil && *f.Keys.Inspect != "" {
		k.Inspect = *f.Keys.Inspect
	}
	overrides := []struct {
		dst *[]string
		src []string
	}{
		{&k.Quit, f.Keys.Quit},
		{&k.Close, f.Keys.Close},
		{&k.FocusNext, f.Keys.FocusNext},
		{&k.FocusPrev, f.Keys.FocusPrev},
		{&k.PaneList, f.Keys.PaneList},
		{&k.PaneChange, f.Keys.PaneChange},
		{&k.PaneGraph, f.Keys.PaneGraph},
		{&k.PanePR, f.Keys.PanePR},
		{&k.Up, f.Keys.Up},
		{&k.Down, f.Keys.Down},
		{&k.PageUp, f.Keys.PageUp},
		{&k.PageDown, f.Keys.PageDown},
		{&k.Home, f.Keys.Home},
		{&k.End, f.Keys.End},
		{&k.Pin, f.Keys.Pin},
		{&k.Group, f.Keys.Group},
		{&k.Refresh, f.Keys.Refresh},
		{&k.Diff, f.Keys.Diff},
		{&k.Commits, f.Keys.Commits},
		{&k.Status, f.Keys.Status},
		{&k.AgentLog, f.Keys.AgentLog},
	}
	for _, o := range overrides {
		if len(o.src) > 0 {
			*o.dst = o.src
		}
	}
}

func parseDur(s *string) (time.Duration, bool) {
	if s == nil || *s == "" {
		return 0, false
	}
	d, err := time.ParseDuration(*s)
	if err != nil || d <= 0 {
		return 0, false
	}
	return d, true
}

func ExpandHome(p string) string {
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func DefaultPath() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "yagura", "config.yaml")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "yagura", "config.yaml")
	}
	return "yagura.yaml"
}
