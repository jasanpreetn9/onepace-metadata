package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"metadata-service/internal/model"
)

// TestExportMetadata_IsCurrentAndBackfill exercises the parts of the export
// pipeline that only show up across multiple runs against a persisted
// archive: marking the newest CRC per (episode, variant) as current, and
// backfilling stable IDs onto archive entries that predate them.
func TestExportMetadata_IsCurrentAndBackfill(t *testing.T) {
	dir := t.TempDir()

	// Seed an existing archive as if written by a previous run: one entry
	// with no ArcID/EpisodeID (pre-migration) for arc 1 episode 1, and one
	// unrelated entry for arc 1 episode 2 that's already current and
	// shouldn't be touched.
	seed := EpisodesArchive{
		"AAAAAAAA": model.EpisodeArchiveEntry{
			Arc:     1,
			Episode: 1,
			Title:   "Old Title",
			File: model.EpisodeFile{
				Version: "normal",
				CRC32:   "AAAAAAAA",
			},
			Released: "2024-01-01",
		},
		"CCCCCCCC": model.EpisodeArchiveEntry{
			ArcID:     "arc1",
			EpisodeID: "arc1-002",
			Arc:       1,
			Episode:   2,
			Title:     "Unrelated",
			File: model.EpisodeFile{
				Version: "normal",
				CRC32:   "CCCCCCCC",
			},
			Released:  "2023-01-01",
			IsCurrent: true,
		},
	}
	seedJSON, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "episodes.json"), seedJSON, 0644); err != nil {
		t.Fatal(err)
	}

	// This run's scrape: episode 1 was re-released under a new CRC with a
	// newer date; episode 2 is unchanged (same CRC as the seed, so it's
	// skipped by the "already exists" merge check).
	arcs := []model.Arc{
		{
			ID:  "arc1",
			Arc: 1,
			Episodes: []model.Episode{
				{
					ID:       "arc1-001",
					Arc:      1,
					Episode:  1,
					Title:    "New Title",
					Released: "2025-01-01",
					Files: model.EpisodeFileVariants{
						Normal: &model.EpisodeFile{Version: "normal", CRC32: "BBBBBBBB"},
					},
				},
				{
					ID:       "arc1-002",
					Arc:      1,
					Episode:  2,
					Title:    "Unrelated",
					Released: "2023-01-01",
					Files: model.EpisodeFileVariants{
						Normal: &model.EpisodeFile{Version: "normal", CRC32: "CCCCCCCC"},
					},
				},
			},
		},
	}

	if err := ExportMetadata(arcs, nil, dir); err != nil {
		t.Fatalf("ExportMetadata: %v", err)
	}

	archiveRaw, err := os.ReadFile(filepath.Join(dir, "episodes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var archive EpisodesArchive
	if err := json.Unmarshal(archiveRaw, &archive); err != nil {
		t.Fatal(err)
	}

	// The old entry should be backfilled with stable IDs...
	old, ok := archive["AAAAAAAA"]
	if !ok {
		t.Fatal("expected AAAAAAAA to still be present (archive is append-only)")
	}
	if old.ArcID != "arc1" || old.EpisodeID != "arc1-001" {
		t.Errorf("AAAAAAAA not backfilled: ArcID=%q EpisodeID=%q", old.ArcID, old.EpisodeID)
	}
	// ...and demoted to not-current now that BBBBBBBB is newer.
	if old.IsCurrent {
		t.Error("AAAAAAAA should no longer be current")
	}

	newer, ok := archive["BBBBBBBB"]
	if !ok {
		t.Fatal("expected BBBBBBBB to have been merged in")
	}
	if !newer.IsCurrent {
		t.Error("BBBBBBBB should be current (newest Released date)")
	}

	unrelated, ok := archive["CCCCCCCC"]
	if !ok || !unrelated.IsCurrent {
		t.Error("CCCCCCCC should be untouched and still current")
	}

	// episodes-current.json should have exactly one file per episode,
	// pointing at the current CRC.
	currentRaw, err := os.ReadFile(filepath.Join(dir, "episodes-current.json"))
	if err != nil {
		t.Fatal(err)
	}
	var current map[string]model.CurrentEpisode
	if err := json.Unmarshal(currentRaw, &current); err != nil {
		t.Fatal(err)
	}
	if len(current) != 2 {
		t.Fatalf("expected 2 current episodes, got %d", len(current))
	}
	ep1, ok := current["arc1-001"]
	if !ok || ep1.Files.Normal == nil || ep1.Files.Normal.CRC32 != "BBBBBBBB" {
		t.Errorf("arc1-001 current file = %+v, want CRC32 BBBBBBBB", ep1.Files.Normal)
	}
	ep2, ok := current["arc1-002"]
	if !ok || ep2.Files.Normal == nil || ep2.Files.Normal.CRC32 != "CCCCCCCC" {
		t.Errorf("arc1-002 current file = %+v, want CRC32 CCCCCCCC", ep2.Files.Normal)
	}
}

func TestNewerArchiveEntry_TieBreaks(t *testing.T) {
	entry := func(crc, released string) model.EpisodeArchiveEntry {
		return model.EpisodeArchiveEntry{Released: released, File: model.EpisodeFile{CRC32: crc}}
	}
	none := map[string]bool{}
	noPub := map[string]string{}

	tests := []struct {
		name   string
		a, b   model.EpisodeArchiveEntry
		listed map[string]bool
		pub    map[string]string
		want   bool
	}{
		{"newer released wins over listed", entry("AAAAAAAA", "2025-02-01"), entry("BBBBBBBB", "2025-01-01"), map[string]bool{"BBBBBBBB": true}, noPub, true},
		{"older released loses", entry("AAAAAAAA", "2025-01-01"), entry("BBBBBBBB", "2025-02-01"), none, noPub, false},
		{"tie: sheet-listed wins", entry("AAAAAAAA", "2025-01-11"), entry("BBBBBBBB", "2025-01-11"), map[string]bool{"AAAAAAAA": true}, map[string]string{"BBBBBBBB": "2025-10-27T12:00:00.000Z"}, true},
		{"tie: newer publish wins", entry("AAAAAAAA", "2025-01-11"), entry("BBBBBBBB", "2025-01-11"), none, map[string]string{"AAAAAAAA": "2025-10-27T12:00:00.000Z", "BBBBBBBB": "2025-01-10T12:00:00.000Z"}, true},
		{"tie: published beats unpublished", entry("AAAAAAAA", "2025-01-11"), entry("BBBBBBBB", "2025-01-11"), none, map[string]string{"AAAAAAAA": "2025-01-10T12:00:00.000Z"}, true},
		{"tie: falls back to CRC order", entry("AAAAAAAA", "2025-01-11"), entry("BBBBBBBB", "2025-01-11"), none, noPub, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newerArchiveEntry(tt.a, tt.b, tt.listed, tt.pub); got != tt.want {
				t.Errorf("newerArchiveEntry(%s, %s) = %v, want %v", tt.a.File.CRC32, tt.b.File.CRC32, got, tt.want)
			}
			// Antisymmetry: swapping the arguments must flip the answer for
			// distinct CRCs, otherwise the winner depends on scan order.
			if got := newerArchiveEntry(tt.b, tt.a, tt.listed, tt.pub); got == tt.want {
				t.Errorf("newerArchiveEntry not antisymmetric for %s/%s", tt.a.File.CRC32, tt.b.File.CRC32)
			}
		})
	}
}

// TestExportMetadata_IsCurrentStableOnTie reproduces the flip-flop seen in
// the auto-update commits: two CRCs for the same episode share a Released
// date. The CRC the sheet currently lists must win, on every run.
func TestExportMetadata_IsCurrentStableOnTie(t *testing.T) {
	dir := t.TempDir()

	seed := EpisodesArchive{
		"C060DA73": model.EpisodeArchiveEntry{
			ArcID: "arc1", EpisodeID: "arc1-003", Arc: 1, Episode: 3,
			Released:  "2025-01-11",
			File:      model.EpisodeFile{Version: "normal", CRC32: "C060DA73"},
			IsCurrent: true,
		},
	}
	seedJSON, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "episodes.json"), seedJSON, 0644); err != nil {
		t.Fatal(err)
	}

	arcs := []model.Arc{{
		ID: "arc1", Arc: 1,
		Episodes: []model.Episode{{
			ID: "arc1-003", Arc: 1, Episode: 3, Released: "2025-01-11",
			Files: model.EpisodeFileVariants{
				Normal: &model.EpisodeFile{Version: "normal", CRC32: "CC4900AE", URL: "https://example.invalid"},
			},
		}},
	}}

	for run := 0; run < 20; run++ {
		if err := ExportMetadata(arcs, nil, dir); err != nil {
			t.Fatalf("run %d: ExportMetadata: %v", run, err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, "episodes.json"))
		if err != nil {
			t.Fatal(err)
		}
		var archive EpisodesArchive
		if err := json.Unmarshal(raw, &archive); err != nil {
			t.Fatal(err)
		}
		if !archive["CC4900AE"].IsCurrent || archive["C060DA73"].IsCurrent {
			t.Fatalf("run %d: want CC4900AE current, got CC4900AE=%v C060DA73=%v",
				run, archive["CC4900AE"].IsCurrent, archive["C060DA73"].IsCurrent)
		}
	}
}
