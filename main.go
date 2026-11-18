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
		// The run continues on the archived releases, but say so loudly:
		// "::warning::" surfaces as an annotation on the GitHub Actions run
		// summary, where a plain log line went unnoticed for months.
		fmt.Println("::warning title=Releases feed unavailable::failed to fetch releases feed:", err)
	}

	err = export.ExportMetadata(arcs, releases, "./data")
	if err != nil {
		panic(err)
	}

	fmt.Println("Metadata export complete.")
}
