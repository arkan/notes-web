package app

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DailyGlob    string         `yaml:"daily_glob"`
	Hidden       []string       `yaml:"hidden"`
	HiddenBlocks []string       `yaml:"hidden_blocks"`
	FolderSort   string         `yaml:"folder_sort"`
	Sidebar      SidebarConfig  `yaml:"sidebar"`
	Homepage     HomepageConfig `yaml:"homepage"`
}

type UIConfig struct {
	HideHomepageCalendar bool
	HideHomepageTodos    bool
	HideSidebarExplore   bool
	HideSidebarTodo      bool
	HideSidebarFavorites bool
}

type SidebarConfig struct {
	Explore   VisibilityConfig       `yaml:"explore"`
	Favorites SidebarFavoritesConfig `yaml:"favorites"`
}

type SidebarFavoritesConfig struct {
	Visible *bool      `yaml:"visible"`
	Items   []Favorite `yaml:"items"`
}

func (f SidebarFavoritesConfig) Hidden() bool {
	return f.Visible != nil && !*f.Visible
}

type HomepageConfig struct {
	Blocks HomepageBlocksConfig `yaml:"blocks"`
}

type HomepageBlocksConfig struct {
	Calendar VisibilityConfig `yaml:"calendar"`
	Todos    VisibilityConfig `yaml:"todos"`
	Todo     VisibilityConfig `yaml:"todo"`
}

type VisibilityConfig struct {
	Visible *bool `yaml:"visible"`
}

type Favorite struct {
	Path  string `yaml:"path"`
	Label string `yaml:"label"`
	URL   string `yaml:"-"`
}

func (v *Vault) Favorites() []Favorite {
	cfg := v.LoadConfig()
	items := cfg.Sidebar.Favorites.Items
	if len(items) == 0 {
		items = []Favorite{
			{Path: "Areas/Daily Briefings", Label: "Daily Briefings"},
			{Path: "_todo", Label: "Todos"},
			{Path: "Projects", Label: "Projects"},
		}
	}
	if cfg.UI().HideSidebarFavorites {
		return nil
	}
	var out []Favorite
	for _, fav := range items {
		fav.Path = strings.Trim(fav.Path, " /")
		fav.Label = strings.TrimSpace(fav.Label)
		if fav.Path == "" || fav.Label == "" || v.isHiddenRel(fav.Path, cfg.Hidden) || (cfg.HideBlock("todo") && fav.Path == "_todo") {
			continue
		}
		fav.URL = v.URLForRel(fav.Path)
		out = append(out, fav)
	}
	return out
}

func (cfg Config) UI() UIConfig {
	return UIConfig{
		HideHomepageCalendar: cfg.HideBlock("calendar") || cfg.Homepage.Blocks.Calendar.Hidden(),
		HideHomepageTodos:    cfg.HideBlock("todo") || cfg.Homepage.Blocks.Todos.Hidden() || cfg.Homepage.Blocks.Todo.Hidden(),
		HideSidebarExplore:   cfg.HideBlock("explore") || cfg.Sidebar.Explore.Hidden(),
		HideSidebarTodo:      cfg.HideBlock("todo"),
		HideSidebarFavorites: cfg.HideBlock("favorites") || cfg.Sidebar.Favorites.Hidden(),
	}
}

func (cfg Config) HideBlock(name string) bool {
	name = normalizeBlockName(name)
	for _, block := range cfg.HiddenBlocks {
		if normalizeBlockName(block) == name {
			return true
		}
	}
	return false
}

func normalizeBlockName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.Trim(name, "_- ")
	if name == "todos" {
		return "todo"
	}
	return name
}

func (v VisibilityConfig) Hidden() bool {
	return v.Visible != nil && !*v.Visible
}

func (v *Vault) LoadConfig() Config {
	cfg := Config{DailyGlob: "Areas/Daily Briefings/*-briefing.md", FolderSort: "name_asc"}
	b, err := os.ReadFile(filepath.Join(v.Root, ".notes-web.yaml"))
	if err != nil {
		return cfg
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg
	}
	return cfg
}
