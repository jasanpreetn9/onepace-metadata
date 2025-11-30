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

	err = export.ExportMetadata(arcs, "./data")
	if err != nil {
		panic(err)
	}

	fmt.Println("Metadata export complete.")
}
