package app

import (
	"encoding/json"
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type notesMapConfig struct {
	From     string            `yaml:"from"`
	Where    map[string]string `yaml:"where"`
	Lat      string            `yaml:"lat"`
	Lon      string            `yaml:"lon"`
	Title    string            `yaml:"title"`
	Subtitle string            `yaml:"subtitle"`
	Color    string            `yaml:"color"`
}

type notesMapData struct {
	Config               notesMapConfig  `json:"config"`
	Points               []notesMapPoint `json:"points"`
	SkippedMissingCoords int             `json:"skippedMissingCoords"`
}

type notesMapPoint struct {
	Title               string  `json:"title"`
	Subtitle            string  `json:"subtitle,omitempty"`
	ColorValue          string  `json:"colorValue,omitempty"`
	Lat                 float64 `json:"lat"`
	Lon                 float64 `json:"lon"`
	URL                 string  `json:"url"`
	RelPath             string  `json:"relPath"`
	Address             string  `json:"address,omitempty"`
	MapURL              string  `json:"mapUrl,omitempty"`
	Website             string  `json:"website,omitempty"`
	DistanceHome        string  `json:"distanceHome,omitempty"`
	DistanceNoorderpark string  `json:"distanceNoorderpark,omitempty"`
	Status              string  `json:"status,omitempty"`
}

func parseNotesMapConfig(raw string) (notesMapConfig, error) {
	cfg := notesMapConfig{
		Where: map[string]string{},
		Lat:   "latitude",
		Lon:   "longitude",
		Title: "title",
		Color: "status",
	}
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, err
	}
	cfg.From = filepath.ToSlash(strings.Trim(strings.TrimSpace(cfg.From), "/"))
	cfg.Lat = strings.TrimSpace(cfg.Lat)
	cfg.Lon = strings.TrimSpace(cfg.Lon)
	cfg.Title = strings.TrimSpace(cfg.Title)
	cfg.Subtitle = strings.TrimSpace(cfg.Subtitle)
	cfg.Color = strings.TrimSpace(cfg.Color)
	if cfg.Where == nil {
		cfg.Where = map[string]string{}
	}
	if cfg.From == "" {
		return cfg, fmt.Errorf("from is required")
	}
	if cfg.Lat == "" || cfg.Lon == "" {
		return cfg, fmt.Errorf("lat and lon fields are required")
	}
	if cfg.Title == "" {
		cfg.Title = "title"
	}
	return cfg, nil
}

func buildNotesMapData(v *Vault, cfg notesMapConfig) notesMapData {
	data := notesMapData{Config: cfg}
	prefix := filepath.ToSlash(strings.Trim(cfg.From, "/"))
	for _, path := range v.MarkdownFiles() {
		note, err := v.ReadNote(path)
		if err != nil {
			continue
		}
		rel := filepath.ToSlash(note.RelPath)
		if rel == prefix || !strings.HasPrefix(rel, prefix+"/") {
			continue
		}
		if !frontmatterMatches(note.Frontmatter, cfg.Where) {
			continue
		}
		lat, okLat := frontmatterFloat(note.Frontmatter, cfg.Lat)
		lon, okLon := frontmatterFloat(note.Frontmatter, cfg.Lon)
		if !okLat || !okLon {
			data.SkippedMissingCoords++
			continue
		}
		title := frontmatterString(note.Frontmatter, cfg.Title)
		if title == "" {
			title = v.Title(note)
		}
		point := notesMapPoint{
			Title:               title,
			Subtitle:            frontmatterString(note.Frontmatter, cfg.Subtitle),
			ColorValue:          frontmatterString(note.Frontmatter, cfg.Color),
			Lat:                 lat,
			Lon:                 lon,
			URL:                 v.URLForRel(rel),
			RelPath:             rel,
			Address:             frontmatterString(note.Frontmatter, "address"),
			MapURL:              frontmatterString(note.Frontmatter, "map_url"),
			Website:             frontmatterString(note.Frontmatter, "website"),
			DistanceHome:        frontmatterString(note.Frontmatter, "distance_home_min"),
			DistanceNoorderpark: frontmatterString(note.Frontmatter, "distance_noorderpark_min"),
			Status:              frontmatterString(note.Frontmatter, "status"),
		}
		data.Points = append(data.Points, point)
	}
	sort.SliceStable(data.Points, func(i, j int) bool {
		return strings.ToLower(data.Points[i].Title) < strings.ToLower(data.Points[j].Title)
	})
	return data
}

func frontmatterMatches(fm map[string]any, where map[string]string) bool {
	for key, want := range where {
		if frontmatterString(fm, key) != want {
			return false
		}
	}
	return true
}

func frontmatterString(fm map[string]any, key string) string {
	key = strings.TrimSpace(key)
	if key == "" || fm == nil {
		return ""
	}
	v, ok := fm[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case fmt.Stringer:
		return strings.TrimSpace(x.String())
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func frontmatterFloat(fm map[string]any, key string) (float64, bool) {
	v, ok := fm[key]
	if !ok || v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint64:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	default:
		f, err := strconv.ParseFloat(fmt.Sprint(x), 64)
		return f, err == nil
	}
}

func preprocessNotesMapBlocks(s string, v *Vault) string {
	re := notesMapFenceRe()
	return re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		cfg, err := parseNotesMapConfig(parts[1])
		if err != nil {
			return renderNotesMapError(err)
		}
		return renderNotesMapBlock(buildNotesMapData(v, cfg))
	})
}

func notesMapFenceRe() *regexp.Regexp {
	return regexp.MustCompile("(?s)```notes-map\\s*\\n(.*?)\\n```")
}

func renderNotesMapError(err error) string {
	return `<div class="notes-map-error" role="note">Map configuration error: ` + html.EscapeString(err.Error()) + `</div>`
}

func renderNotesMapBlock(data notesMapData) string {
	payload, err := json.Marshal(data)
	if err != nil {
		return renderNotesMapError(err)
	}
	if len(data.Points) == 0 {
		return `<div class="notes-map notes-map-empty" data-notes-map="` + html.EscapeString(string(payload)) + `">No map points found.</div>`
	}
	return `<div class="notes-map" data-notes-map="` + html.EscapeString(string(payload)) + `"><noscript>Enable JavaScript to view the map.</noscript></div>`
}
