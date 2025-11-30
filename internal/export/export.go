package export

import (
	"encoding/json"
	"os"

	"gopkg.in/yaml.v3"
	"metadata-service/internal/model"
	"metadata-service/internal/util"
)

func ExportMetadata(arcs []model.Arc, outDir string) error {

	// JSON file
	jsonPath := outDir + "/data.json"
	jsonBytes, err := json.MarshalIndent(arcs, "", "  ")
	if err != nil {
		return err
	}

	if !util.FileUnchanged(jsonPath, jsonBytes) {
		err = os.WriteFile(jsonPath, jsonBytes, 0644)
		if err != nil {
			return err
		}
	}

	// YAML file
	ymlPath := outDir + "/data.yml"
	ymlBytes, err := yaml.Marshal(arcs)
	if err != nil {
		return err
	}

	if !util.FileUnchanged(ymlPath, ymlBytes) {
		err = os.WriteFile(ymlPath, ymlBytes, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}
