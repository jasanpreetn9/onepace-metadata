package fetch

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"metadata-service/internal/config"
	"metadata-service/internal/model"
)

//
// Episode description CSV row (gid=0)
// arc_title, arc_part, title_en, description_en
//

func FetchEpisodeDescriptions() (map[string]map[int]model.EpisodeMeta, error) {

	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=0",
		config.OnePaceEpisodeDescID,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch episode descriptions CSV: %w", err)
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	reader.FieldsPerRecord = -1

	// Skip header row
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	// map["Romance Dawn"][1] = EpisodeMeta{Title, Description}
	result := make(map[string]map[int]model.EpisodeMeta)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csv read: %w", err)
		}

		if len(row) < 4 {
			continue
		}

		arcTitle := strings.TrimSpace(row[0])
		epStr := strings.TrimSpace(row[1])
		title := strings.TrimSpace(row[2])
		desc := strings.TrimSpace(row[3])

		if arcTitle == "" || epStr == "" || title == "" {
			continue
		}

		epNum, err := strconv.Atoi(epStr)
		if err != nil {
			continue
		}

		arcTitle = cleanTitle(arcTitle)

		if _, ok := result[arcTitle]; !ok {
			result[arcTitle] = make(map[int]model.EpisodeMeta)
		}

		result[arcTitle][epNum] = model.EpisodeMeta{
			Title:       title,
			Description: desc,
		}
	}

	return result, nil
}

func cleanTitle(s string) string {
	s = strings.ReplaceAll(s, "(WIP)", "")
	s = strings.ReplaceAll(s, "(TBR)", "")
	return strings.TrimSpace(s)
}
