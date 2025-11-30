package export

import (
	"encoding/json"
	"os"
	"time"
)

type Status struct {
	LastUpdate    string  `json:"last_update"`
	LastUpdateTS  float64 `json:"last_update_ts"`
	Version       string  `json:"version"`
	BaseURL       string  `json:"base_url"`
}

func WriteStatus(path string) error {

	now := time.Now().UTC()

	status := Status{
		LastUpdate:   now.Format(time.RFC3339),
		LastUpdateTS: float64(now.Unix()),
		Version:      now.Format("20060102.15"),
		BaseURL:      "",
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path+"/status.json", data, 0644)
}
