package app

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Favorites  []Favorite
	DailyGlob  string
	Hidden     []string
	FolderSort string
}

type Favorite struct {
	Path  string
	Label string
	URL   string
}

func (v *Vault) Favorites() []Favorite {
	cfg := v.LoadConfig()
	if len(cfg.Favorites) == 0 {
		cfg.Favorites = []Favorite{
			{Path: "Areas/Daily Briefings", Label: "Daily Briefings"},
			{Path: "_todo", Label: "Todos"},
			{Path: "Projects", Label: "Projects"},
		}
	}
	var out []Favorite
	for _, fav := range cfg.Favorites {
		fav.Path = strings.Trim(fav.Path, " /")
		fav.Label = strings.TrimSpace(fav.Label)
		if fav.Path == "" || fav.Label == "" || v.isHiddenRel(fav.Path, cfg.Hidden) {
			continue
		}
		fav.URL = v.URLForRel(fav.Path)
		out = append(out, fav)
	}
	return out
}

func (v *Vault) LoadConfig() Config {
	cfg := Config{DailyGlob: "Areas/Daily Briefings/*-briefing.md", FolderSort: "name_asc"}
	b, err := os.ReadFile(filepath.Join(v.Root, ".notes-web.yaml"))
	if err != nil {
		return cfg
	}
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	section := ""
	currentFavorite := -1
	for scanner.Scan() {
		tr := strings.TrimSpace(scanner.Text())
		if tr == "" || strings.HasPrefix(tr, "#") {
			continue
		}
		if strings.HasPrefix(tr, "daily_glob:") {
			cfg.DailyGlob = yamlScalar(strings.TrimPrefix(tr, "daily_glob:"))
			section = ""
			currentFavorite = -1
			continue
		}
		if strings.HasPrefix(tr, "folder_sort:") {
			cfg.FolderSort = yamlScalar(strings.TrimPrefix(tr, "folder_sort:"))
			section = ""
			currentFavorite = -1
			continue
		}
		if strings.HasPrefix(tr, "favorites:") {
			section = "favorites"
			currentFavorite = -1
			continue
		}
		if strings.HasPrefix(tr, "hidden:") {
			section = "hidden"
			currentFavorite = -1
			continue
		}
		if strings.HasPrefix(tr, "-") {
			value := strings.TrimSpace(strings.TrimPrefix(tr, "-"))
			switch section {
			case "favorites":
				currentFavorite = -1
				if strings.HasPrefix(value, "path:") {
					cfg.Favorites = append(cfg.Favorites, Favorite{Path: yamlScalar(strings.TrimPrefix(value, "path:"))})
					currentFavorite = len(cfg.Favorites) - 1
				}
			case "hidden":
				cfg.Hidden = append(cfg.Hidden, yamlScalar(value))
			}
			continue
		}
		if section == "favorites" && currentFavorite >= 0 {
			if strings.HasPrefix(tr, "path:") {
				cfg.Favorites[currentFavorite].Path = yamlScalar(strings.TrimPrefix(tr, "path:"))
				continue
			}
			if strings.HasPrefix(tr, "label:") {
				cfg.Favorites[currentFavorite].Label = yamlScalar(strings.TrimPrefix(tr, "label:"))
				continue
			}
		}
		section = ""
		currentFavorite = -1
	}
	return cfg
}

func yamlScalar(value string) string {
	return strings.Trim(strings.TrimSpace(value), "'\"")
}
