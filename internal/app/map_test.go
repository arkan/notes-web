package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMapNote(t *testing.T, v *Vault, rel, body string) {
	t.Helper()
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseNotesMapBlockConfig(t *testing.T) {
	raw := "from: \"Areas/Amsterdam/Crèches\"\nwhere:\n  type: creche\nlat: latitude\nlon: longitude\ntitle: name\nsubtitle: status\ncolor: status\n"
	cfg, err := parseNotesMapConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.From != "Areas/Amsterdam/Crèches" || cfg.Where["type"] != "creche" || cfg.Lat != "latitude" || cfg.Lon != "longitude" || cfg.Title != "name" || cfg.Subtitle != "status" || cfg.Color != "status" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestBuildNotesMapPointsFiltersFolderFrontmatterAndCoordinates(t *testing.T) {
	v := makeVault(t)
	writeMapNote(t, v, "Areas/Amsterdam/Crèches/Appelboom.md", "---\ntype: creche\nname: Appelboom\nstatus: en attente\nlatitude: 52.3905\nlongitude: 4.9451\naddress: Purmerweg 86\nmap_url: https://maps.example/appelboom\ndistance_home_min: 8-11 vélo\n---\n# Appelboom\n")
	writeMapNote(t, v, "Areas/Amsterdam/Crèches/MissingCoords.md", "---\ntype: creche\nname: Missing\nstatus: en attente\n---\n# Missing\n")
	writeMapNote(t, v, "Areas/Amsterdam/Crèches/OtherType.md", "---\ntype: school\nname: School\nlatitude: 52.1\nlongitude: 4.1\n---\n# School\n")
	writeMapNote(t, v, "Areas/Elsewhere/Creche.md", "---\ntype: creche\nname: Elsewhere\nlatitude: 52.2\nlongitude: 4.2\n---\n# Elsewhere\n")

	cfg := notesMapConfig{From: "Areas/Amsterdam/Crèches", Where: map[string]string{"type": "creche"}, Lat: "latitude", Lon: "longitude", Title: "name", Subtitle: "status", Color: "status"}
	data := buildNotesMapData(v, cfg)
	if len(data.Points) != 1 {
		t.Fatalf("points len=%d want 1: %+v", len(data.Points), data.Points)
	}
	p := data.Points[0]
	if p.Title != "Appelboom" || p.Subtitle != "en attente" || p.ColorValue != "en attente" || p.URL != "/Areas/Amsterdam/Cr%C3%A8ches/Appelboom.md" {
		t.Fatalf("unexpected point: %+v", p)
	}
	if p.Lat != 52.3905 || p.Lon != 4.9451 {
		t.Fatalf("unexpected coordinates: %+v", p)
	}
	if p.Address != "Purmerweg 86" || p.MapURL != "https://maps.example/appelboom" || p.DistanceHome != "8-11 vélo" {
		t.Fatalf("missing popup fields: %+v", p)
	}
	if data.SkippedMissingCoords != 1 {
		t.Fatalf("skipped missing coords=%d want 1", data.SkippedMissingCoords)
	}
}

func TestRendererReplacesNotesMapFenceWithMapContainer(t *testing.T) {
	v := makeVault(t)
	writeMapNote(t, v, "Areas/Amsterdam/Crèches/Appelboom.md", "---\ntype: creche\nname: Appelboom\nstatus: en attente\nlatitude: 52.3905\nlongitude: 4.9451\n---\n# Appelboom\n")
	note := Note{RelPath: "Areas/Amsterdam/Crèches/Crèches.md", Body: "# Crèches\n\n```notes-map\nfrom: \"Areas/Amsterdam/Crèches\"\nwhere:\n  type: creche\nlat: latitude\nlon: longitude\ntitle: name\nsubtitle: status\ncolor: status\n```\n"}
	doc := NewRenderer(v).Render(note)
	for _, want := range []string{`class="notes-map"`, `data-notes-map=`, `Appelboom`, `/Areas/Amsterdam/Cr%C3%A8ches/Appelboom.md`, `52.3905`} {
		if !strings.Contains(doc.HTML, want) {
			t.Fatalf("rendered map missing %q in:\n%s", want, doc.HTML)
		}
	}
	if strings.Contains(doc.HTML, "```notes-map") {
		t.Fatalf("raw notes-map fence leaked into HTML:\n%s", doc.HTML)
	}
}

func TestNotesMapInvalidConfigRendersDiagnostic(t *testing.T) {
	v := makeVault(t)
	note := Note{RelPath: "Map.md", Body: "```notes-map\nwhere:\n  type: creche\n```\n"}
	doc := NewRenderer(v).Render(note)
	if !strings.Contains(doc.HTML, `class="notes-map-error"`) || !strings.Contains(doc.HTML, "from is required") {
		t.Fatalf("missing diagnostic in:\n%s", doc.HTML)
	}
}

func TestNotesMapStaticAssetsExposeRenderer(t *testing.T) {
	for _, want := range []string{"initNotesMaps", "data-notes-map", "openstreetmap.org", "notes-map-marker"} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing map JS %q", want)
		}
	}
	for _, want := range []string{".notes-map", ".notes-map-tile", ".notes-map-marker", ".notes-map-popup"} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing map CSS %q", want)
		}
	}
}
