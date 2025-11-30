package export

import (
	"encoding/json"
	"os"
	"sort"

	"metadata-service/internal/model"

	"gopkg.in/yaml.v3"
)

func Export(arcs []model.Arc, outDir string) error {

	history := make(map[string]model.Episode)

	// Build final arcs
	for arcIndex := range arcs {
		arc := &arcs[arcIndex]

		// Sort episodes
		sort.Slice(arc.Episodes, func(a, b int) bool {
			return arc.Episodes[a].Episode < arc.Episodes[b].Episode
		})

		for i := range arc.Episodes {
			ep := arc.Episodes[i]

			// Add history entries
			for _, f := range ep.Files {
				if f.CRC32 != "" {
					history[f.CRC32] = ep
				}
				if f.CRC32Extended != "" {
					history[f.CRC32Extended] = ep
				}
			}

			// Keep only newest file version in arcs
			if len(ep.Files) > 0 {
				latest := ep.Files[len(ep.Files)-1]
				ep.Files = []model.EpisodeFile{latest}
			}

			arc.Episodes[i] = ep
		}
	}

	// Write arcs.yml
	if err := writeYAML(outDir+"/arcs.yml", arcs); err != nil {
		return err
	}

	// Write episodes.yml (history)
	if err := writeYAML(outDir+"/episodes.yml", history); err != nil {
		return err
	}

	// Write arcs.json
	if err := writeJSON(outDir+"/arcs.json", arcs); err != nil {
		return err
	}

	// Write episodes.json
	if err := writeJSON(outDir+"/episodes.json", history); err != nil {
		return err
	}

	return nil
}

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
