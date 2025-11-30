package main

import (
	"fmt"
	"log"
	"metadata-service/internal/export"
	"metadata-service/internal/fetch"
	"os"
)

func main() {

	fmt.Println("▶ Fetching One Pace metadata...")

	arcs, err := fetch.FetchEpisodeGuideHome()
	if err != nil {
		log.Fatalf("Failed to fetch metadata: %v", err)
	}

	fmt.Printf("✓ Loaded %d arcs\n", len(arcs))

	outDir := "./data"

	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("Failed to create output dir: %v", err)
	}

	// EXPORT ALL FILES
	fmt.Println("▶ Exporting metadata...")

	err = export.Export(arcs, outDir)
	if err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	// WRITE STATUS.json
	err = export.WriteStatus(outDir)
	if err != nil {
		log.Fatalf("Failed to write status.json: %v", err)
	}

	fmt.Println("🎉 Export completed! Files written to /data/")
}
