package app

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DailyGlob      string         `yaml:"daily_glob"`
	DailyNotesGlob string         `yaml:"daily_notes_glob"`
	Hidden         []string       `yaml:"hidden"`
	HiddenBlocks   []string       `yaml:"hidden_blocks"`
	FolderSort     string         `yaml:"folder_sort"`
	Sidebar        SidebarConfig  `yaml:"sidebar"`
	Homepage       HomepageConfig `yaml:"homepage"`
	Editing        EditingConfig  `yaml:"editing"`
	Todo           TodoConfig     `yaml:"todo"`
}

type TodoConfig struct {
	TodoFile string `yaml:"todo_file"`
}

type EditingConfig struct {
	Enabled       bool   `yaml:"enabled"`
	TrashPath     string `yaml:"trash_path"`
	TemplateName  string `yaml:"template_name"`
	HideTemplates bool   `yaml:"hide_templates"`
	SlugMode      string `yaml:"slug"`
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
	Order  []string             `yaml:"order"`
	Blocks HomepageBlocksConfig `yaml:"blocks"`
}

type HomepageBlocksConfig struct {
	Today          VisibilityConfig   `yaml:"today"`
	QuickJump      QuickJumpConfig    `yaml:"quick_jump"`
	Todos          VisibilityConfig   `yaml:"todos"`
	ActiveProjects HomepageListConfig `yaml:"active_projects"`
	Calendar       VisibilityConfig   `yaml:"calendar"`
	SelectedDay    VisibilityConfig   `yaml:"selected_day"`
	RecentNotes    HomepageListConfig `yaml:"recent_notes"`
	Diagnostics    VisibilityConfig   `yaml:"diagnostics"`
}

type HomepageListConfig struct {
	Visible *bool `yaml:"visible"`
	Limit   int   `yaml:"limit"`
}

type QuickJumpConfig struct {
	Visible *bool            `yaml:"visible"`
	Items   *[]QuickJumpItem `yaml:"items"`
}

type QuickJumpItem struct {
	Label string `yaml:"label"`
	Path  string `yaml:"path"`
}

func (h HomepageListConfig) Hidden() bool {
	return h.Visible != nil && !*h.Visible
}

func (h HomepageListConfig) LimitOrDefault(def int) int {
	if h.Limit <= 0 {
		return def
	}
	return h.Limit
}

func (q QuickJumpConfig) Hidden() bool {
	return q.Visible != nil && !*q.Visible
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
		if fav.Path == "" || fav.Label == "" || v.isExcludedFromEnumeration(fav.Path, cfg) || (cfg.HideBlock("todo") && fav.Path == "_todo") {
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
		HideHomepageTodos:    cfg.HideBlock("todo") || cfg.HideBlock("todos") || cfg.Homepage.Blocks.Todos.Hidden(),
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

// Homepage block IDs
const (
	BlockToday          = "today"
	BlockQuickJump      = "quick_jump"
	BlockTodos          = "todos"
	BlockActiveProjects = "active_projects"
	BlockCalendar       = "calendar"
	BlockSelectedDay    = "selected_day"
	BlockRecentNotes    = "recent_notes"
	BlockDiagnostics    = "diagnostics"
)

var defaultHomepageOrder = []string{
	BlockToday, BlockCalendar, BlockTodos, BlockActiveProjects,
	BlockSelectedDay, BlockQuickJump, BlockRecentNotes, BlockDiagnostics,
}

var validBlockIDs = map[string]bool{
	BlockToday: true, BlockQuickJump: true, BlockTodos: true, BlockActiveProjects: true,
	BlockCalendar: true, BlockSelectedDay: true, BlockRecentNotes: true, BlockDiagnostics: true,
}

var mainColumnIDs = map[string]bool{
	BlockToday: true, BlockTodos: true, BlockActiveProjects: true, BlockRecentNotes: true,
}

func homeBlockColumn(id string) string {
	if mainColumnIDs[id] {
		return "main"
	}
	return "side"
}

// OrderedVisibleBlocks returns the list of visible homepage blocks in their
// configured order, appending any missing valid blocks at the end in default
// order and ignoring unknown IDs.
func (cfg Config) OrderedVisibleBlocks() []HomepageBlock {
	order := cfg.Homepage.Order
	seen := map[string]bool{}
	var blocks []HomepageBlock

	for _, id := range order {
		if !validBlockIDs[id] {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		if cfg.homeBlockHidden(id) {
			continue
		}
		blocks = append(blocks, HomepageBlock{ID: id, Column: homeBlockColumn(id)})
	}
	for _, id := range defaultHomepageOrder {
		if seen[id] {
			continue
		}
		seen[id] = true
		if cfg.homeBlockHidden(id) {
			continue
		}
		blocks = append(blocks, HomepageBlock{ID: id, Column: homeBlockColumn(id)})
	}
	return blocks
}

func (cfg Config) homeBlockHidden(id string) bool {
	switch id {
	case BlockToday:
		return cfg.Homepage.Blocks.Today.Hidden()
	case BlockQuickJump:
		// Quick-jump is homepage-owned. Only its own visible flag controls visibility.
		return cfg.Homepage.Blocks.QuickJump.Hidden()
	case BlockTodos:
		return cfg.Homepage.Blocks.Todos.Hidden() || cfg.HideBlock("todo") || cfg.HideBlock("todos")
	case BlockActiveProjects:
		return cfg.Homepage.Blocks.ActiveProjects.Hidden()
	case BlockCalendar:
		return cfg.Homepage.Blocks.Calendar.Hidden() || cfg.HideBlock("calendar")
	case BlockSelectedDay:
		return cfg.Homepage.Blocks.SelectedDay.Hidden()
	case BlockRecentNotes:
		return cfg.Homepage.Blocks.RecentNotes.Hidden()
	case BlockDiagnostics:
		return cfg.Homepage.Blocks.Diagnostics.Hidden()
	}
	return false
}

var defaultQuickJumpItems = []QuickJumpItem{
	{Label: "Today", Path: "/"},
	{Label: "TODO", Path: "/_todo"},
	{Label: "Search", Path: "/_search"},
	{Label: "Daily Briefings", Path: "Areas/Daily Briefings"},
}

// QuickJumpItems returns the resolved quick-jump items for the homepage.
// Defaults are used when items is nil; an explicit empty list yields no links.
// All items go through the same resolution pipeline: empty label/path skip,
// hidden path filtering, TODO skip when hidden_blocks hides todo.
func (v *Vault) QuickJumpItems() []HomepageLink {
	cfg := v.LoadConfig()
	if cfg.Homepage.Blocks.QuickJump.Hidden() {
		return nil
	}

	raw := cfg.Homepage.Blocks.QuickJump.Items
	if raw != nil && len(*raw) == 0 {
		return []HomepageLink{}
	}

	var sources []QuickJumpItem
	if raw == nil {
		sources = defaultQuickJumpItems
	} else {
		sources = *raw
	}

	return v.resolveQuickJumpItems(sources, cfg)
}

// resolveQuickJumpItems processes raw QuickJumpItem entries through
// validation, hidden-path filtering, and URL resolution.
func (v *Vault) resolveQuickJumpItems(items []QuickJumpItem, cfg Config) []HomepageLink {
	var out []HomepageLink
	for _, item := range items {
		label := strings.TrimSpace(item.Label)
		path := strings.TrimSpace(item.Path)
		if label == "" || path == "" {
			continue
		}

		url := v.resolveQuickJumpPath(path, cfg)
		if url == "" {
			continue
		}

		// Skip TODO links when hidden_blocks hides todo
		isTodo := path == "_todo" || path == "/_todo" || url == "/_todo"
		if isTodo && (cfg.HideBlock("todo") || cfg.HideBlock("todos")) {
			continue
		}

		out = append(out, HomepageLink{Label: label, Path: path, URL: url})
	}
	return out
}

// resolveQuickJumpPath resolves a quick-jump path string to a URL.
// Returns "" if the path should be skipped (hidden vault path).
func (v *Vault) resolveQuickJumpPath(path string, cfg Config) string {
	if isQuickJumpInternalRoute(path) {
		// Known internal app route — always safe, use as-is.
		return path
	}
	if strings.HasPrefix(path, "/") {
		// Slash-prefixed vault path — strip leading / and apply hidden filtering.
		rel := strings.TrimPrefix(path, "/")
		if v.isExcludedFromEnumeration(rel, cfg) {
			return ""
		}
		return v.URLForRel(rel)
	}
	if strings.HasPrefix(path, "_") {
		candidate := "/" + strings.TrimLeft(path, "/")
		if isQuickJumpInternalRoute(candidate) {
			return candidate
		}
	}
	// Bare vault path — apply hidden filtering.
	if v.isExcludedFromEnumeration(path, cfg) {
		return ""
	}
	return v.URLForRel(path)
}

func isQuickJumpInternalRoute(path string) bool {
	switch path {
	case "/", "/_search", "/_tags", "/_todo", "/_broken-links", "/_orphans", "/_dataview":
		return true
	}
	return strings.HasPrefix(path, "/_tags/")
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
	cfg := defaultConfig()
	b, err := os.ReadFile(filepath.Join(v.Root, ".notes-web.yaml"))
	if err != nil {
		return cfg
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg
	}
	return cfg
}

func defaultConfig() Config {
	return Config{
		DailyGlob:      "Areas/Daily Briefings/*-briefing.md",
		DailyNotesGlob: "Daily Notes/*/*/*.md",
		FolderSort:     "name_asc",
		Editing: EditingConfig{
			TrashPath:     "_trash",
			TemplateName:  "_template.md",
			HideTemplates: true,
			SlugMode:      "kebab_lowercase",
		},
	}
}
