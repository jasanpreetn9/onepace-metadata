package export

import (
	"encoding/json"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"metadata-service/internal/model"
	"metadata-service/internal/util"
)

// Archive stored as:
// { "<CRC32>": EpisodeArchiveEntry }
type EpisodesArchive map[string]model.EpisodeArchiveEntry

func ExportMetadata(arcs []model.Arc, outDir string) error {

	// Ensure output directory exists
	if err := util.EnsureDir(outDir); err != nil {
		return err
	}
	metadataChanged := false
	// ========================================================
	// 1) EXPORT ARCS (modern structure)
	// ========================================================

	// --- arcs.json ---
	arcsJSON, err := json.MarshalIndent(arcs, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/arcs.json", arcsJSON) {
		if err := os.WriteFile(outDir+"/arcs.json", arcsJSON, 0644); err != nil {
			return err
		}
		metadataChanged = true
	}

	// --- arcs.yml ---
	arcsYAML, err := yaml.Marshal(arcs)
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/arcs.yml", arcsYAML) {
		if err := os.WriteFile(outDir+"/arcs.yml", arcsYAML, 0644); err != nil {
			return err
		}
		metadataChanged = true
	}

	// ========================================================
	// 2) LOAD EXISTING EPISODE ARCHIVE (append-only)
	// ========================================================

	archive := EpisodesArchive{}
	archivePath := outDir + "/episodes.json"

	if util.FileExists(archivePath) {
		raw, _ := os.ReadFile(archivePath)
		_ = json.Unmarshal(raw, &archive)
	}

	// ========================================================
	// 3) MERGE NEW EPISODES — ALWAYS APPEND, NEVER REMOVE
	// ========================================================

	for _, arc := range arcs {
		for _, ep := range arc.Episodes {

			// ---- NORMAL FILE ----
			if ep.Files.Normal != nil {
				key := ep.Files.Normal.CRC32
				if key != "" {

					if _, exists := archive[key]; !exists {

						archive[key] = model.EpisodeArchiveEntry{
							Arc:         ep.Arc,
							Episode:     ep.Episode,
							Title:       ep.Title,
							Description: ep.Description,
							Chapters:    ep.Chapters,
							AnimeEps:    ep.AnimeEps,
							Released:    ep.Released,
							File:        *ep.Files.Normal,
						}
						metadataChanged = true
					}
				}
			}

			// ---- EXTENDED FILE ----
			if ep.Files.Extended != nil {
				key := ep.Files.Extended.CRC32
				if key != "" {

					if _, exists := archive[key]; !exists {

						archive[key] = model.EpisodeArchiveEntry{
							Arc:         ep.Arc,
							Episode:     ep.Episode,
							Title:       ep.Title,
							Description: ep.Description,
							Chapters:    ep.Chapters,
							AnimeEps:    ep.AnimeEps,
							Released:    ep.Released,
							File:        *ep.Files.Extended,
						}
						metadataChanged = true
					}
				}
			}
		}
	}

	// ========================================================
	// 4) WRITE EPISODE ARCHIVE (legacy format)
	// ========================================================

	// --- episodes.json ---
	archiveJSON, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(archivePath, archiveJSON) {
		if err := os.WriteFile(archivePath, archiveJSON, 0644); err != nil {
			return err
		}
	}

	// --- episodes.yml ---
	archiveYAML, err := yaml.Marshal(archive)
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/episodes.yml", archiveYAML) {
		if err := os.WriteFile(outDir+"/episodes.yml", archiveYAML, 0644); err != nil {
			return err
		}
	}

	// ========================================================
	// 5) WRITE STATUS FILE
	// ========================================================

	if metadataChanged {
		status := map[string]any{
			"updated_at": time.Now().UTC().Format(time.RFC3339),
			"arcs":       len(arcs),
			"episodes":   len(archive),
		}

		statusJSON, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(outDir+"/status.json", statusJSON, 0644); err != nil {
			return err
		}
	}

	return nil
}
