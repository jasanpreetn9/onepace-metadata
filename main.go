package main

import (
	"fmt"
	"metadata-service/internal/export"
	"metadata-service/internal/fetch"
)

func main() {
	arcs, err := fetch.FetchEpisodeGuideHome()
	if err != nil {
		panic(err)
	}

	releases, err := fetch.FetchReleases()
	if err != nil {
		fmt.Println("Warning: failed to fetch releases feed:", err)
	}

	err = export.ExportMetadata(arcs, releases, "./data")
	if err != nil {
		panic(err)
	}

	fmt.Println("Metadata export complete.")
}
