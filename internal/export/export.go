package export

import (
	"encoding/json"
	"os"
	"time"

	"gopkg.in/yaml.v3"
	"metadata-service/internal/model"
	"metadata-service/internal/util"
)

type EpisodesArchive map[string]model.Episode // keyed by CRC32

// ExportMetadata writes:
//   arcs.json, arcs.yml
//   episodes.json, episodes.yml (append-only, never delete keys)
//   status.json
func ExportMetadata(arcs []model.Arc, outDir string) error {

	// Ensure directory exists
	if err := util.EnsureDir(outDir); err != nil {
		return err
	}

	// ========================================================
	// 1) EXPORT ARCS (ALWAYS FULL OVERWRITE)
	// ========================================================

	// ---- arcs.json ----
	arcsJSON, err := json.MarshalIndent(arcs, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/arcs.json", arcsJSON) {
		if err := os.WriteFile(outDir+"/arcs.json", arcsJSON, 0644); err != nil {
			return err
		}
	}

	// ---- arcs.yml ----
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
	// 2) BUILD / LOAD EPISODE ARCHIVE (APPEND-ONLY)
	// ========================================================

	existingEpisodes := EpisodesArchive{}

	episodesJSONPath := outDir + "/episodes.json"

	// If existing archive exists, load it
	if util.FileExists(episodesJSONPath) {
		raw, _ := os.ReadFile(episodesJSONPath)
		_ = json.Unmarshal(raw, &existingEpisodes)
	}

	// Merge new episodes — append only, never remove old keys
	for _, arc := range arcs {
		for _, ep := range arc.Episodes {

			for _, file := range ep.Files {

				key := file.CRC32
				if key == "" {
					continue
				}

				if _, exists := existingEpisodes[key]; !exists {
					// Add full episode metadata under CRC32
					existingEpisodes[key] = ep
				}
			}
		}
	}

	// ========================================================
	// 3) WRITE EPISODES ARCHIVE (append-only)
	// ========================================================

	// ---- episodes.json ----
	episodesJSON, err := json.MarshalIndent(existingEpisodes, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/episodes.json", episodesJSON) {
		if err := os.WriteFile(outDir+"/episodes.json", episodesJSON, 0644); err != nil {
			return err
		}
	}

	// ---- episodes.yml ----
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
	// 4) WRITE STATUS FILE
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
