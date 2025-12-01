package export

import (
	"encoding/json"
	"os"
	"time"

	"metadata-service/internal/model"
	"metadata-service/internal/util"

	"gopkg.in/yaml.v3"
)

// Archive entry — ONE entry per CRC32

type EpisodesArchive map[string]model.EpisodeArchiveEntry // keyed by CRC32

// ExportMetadata writes:
// arcs.json, arcs.yml
// episodes.json, episodes.yml (append-only, never delete keys)
// status.json
func ExportMetadata(arcs []model.Arc, outDir string) error {

	// Ensure directory exists
	if err := util.EnsureDir(outDir); err != nil {
		return err
	}

	// ========================================================
	// 1) EXPORT ARCS (FULL OVERWRITE)
	// ========================================================

	arcsJSON, err := json.MarshalIndent(arcs, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/arcs.json", arcsJSON) {
		if err := os.WriteFile(outDir+"/arcs.json", arcsJSON, 0644); err != nil {
			return err
		}
	}

	arcsYAML, err := yaml.Marshal(arcs)
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/arcs.yml", arcsYAML) {
		if err := os.WriteFile(outDir+"/arcs.yml", arcsYAML, 0644); err != nil {
			return err
		}
	}

	// ========================================================
	// 2) LOAD EXISTING EPISODE ARCHIVE (append-only)
	// ========================================================

	episodesPath := outDir + "/episodes.json"
	existingEpisodes := EpisodesArchive{}

	if util.FileExists(episodesPath) {
		raw, _ := os.ReadFile(episodesPath)
		_ = json.Unmarshal(raw, &existingEpisodes)
	}

	// ========================================================
	// 3) MERGE NEW EPISODE DATA (append-only)
	// ========================================================

	for _, arc := range arcs {
		for _, ep := range arc.Episodes {

			for _, file := range ep.Files {

				key := file.CRC32
				if key == "" {
					continue
				}

				// Skip if already archived
				if _, exists := existingEpisodes[key]; exists {
					continue
				}

				// Convert into archive entry containing ONLY this file variant
				entry := model.EpisodeArchiveEntry{
					Arc:         ep.Arc,
					Episode:     ep.Episode,
					Title:       ep.Title,
					Description: ep.Description,
					Chapters:    ep.Chapters,
					AnimeEps:    ep.AnimeEps,
					Released:    ep.Released,
					File:        file, // only this specific variant
				}

				existingEpisodes[key] = entry
			}
		}
	}

	// ========================================================
	// 4) WRITE EPISODES ARCHIVE
	// ========================================================

	episodesJSON, err := json.MarshalIndent(existingEpisodes, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/episodes.json", episodesJSON) {
		if err := os.WriteFile(outDir+"/episodes.json", episodesJSON, 0644); err != nil {
			return err
		}
	}

	episodesYAML, err := yaml.Marshal(existingEpisodes)
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/episodes.yml", episodesYAML) {
		if err := os.WriteFile(outDir+"/episodes.yml", episodesYAML, 0644); err != nil {
			return err
		}
	}

	// ========================================================
	// 5) WRITE STATUS FILE
	// ========================================================

	status := map[string]any{
		"updated_at": time.Now().UTC().Format(time.RFC3339),
		"arcs":       len(arcs),
		"episodes":   len(existingEpisodes),
	}

	statusJSON, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outDir+"/status.json", statusJSON, 0644); err != nil {
		return err
	}

	return nil
}
