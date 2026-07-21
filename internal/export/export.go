package export

import (
	"encoding/json"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"metadata-service/internal/fetch"
	"metadata-service/internal/model"
	"metadata-service/internal/util"
)

// Archive stored as:
// { "<CRC32>": EpisodeArchiveEntry }
type EpisodesArchive map[string]model.EpisodeArchiveEntry

// Archive stored as:
// { "<InfoHash>": Release }
type ReleasesArchive map[string]model.Release

func ExportMetadata(arcs []model.Arc, releases []model.Release, outDir string) error {

	// Ensure output directory exists
	if err := util.EnsureDir(outDir); err != nil {
		return err
	}
	metadataChanged := false

	// Index releases by CRC32 so the episode archive can be enriched with
	// magnet/torrent links without a per-CRC Nyaa search.
	releasesByCRC := make(map[string]model.Release, len(releases))
	for _, r := range releases {
		if r.CRC32 != "" {
			releasesByCRC[r.CRC32] = r
		}
	}
	// ========================================================
	// 1) EXPORT ARCS (modern structure)
	// ========================================================

	// --- arcs.json ---
	arcsJSON, err := json.MarshalIndent(arcs, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/arcs.json", arcsJSON) {
		if err := os.WriteFile(outDir+"/arcs.json", arcsJSON, 0644); err != nil {
			return err
		}
		metadataChanged = true
	}

	// --- arcs.yml ---
	arcsYAML, err := yaml.Marshal(arcs)
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/arcs.yml", arcsYAML) {
		if err := os.WriteFile(outDir+"/arcs.yml", arcsYAML, 0644); err != nil {
			return err
		}
		metadataChanged = true
	}

	// ========================================================
	// 2) LOAD EXISTING EPISODE ARCHIVE (append-only)
	// ========================================================

	archive := EpisodesArchive{}
	archivePath := outDir + "/episodes.json"

	if util.FileExists(archivePath) {
		raw, _ := os.ReadFile(archivePath)
		_ = json.Unmarshal(raw, &archive)
	}

	// ========================================================
	// 3) MERGE NEW EPISODES — ALWAYS APPEND, NEVER REMOVE
	// ========================================================

	for _, arc := range arcs {
		for _, ep := range arc.Episodes {

			// ---- NORMAL FILE ----
			if ep.Files.Normal != nil {
				key := ep.Files.Normal.CRC32
				if key != "" {

					if _, exists := archive[key]; !exists {

						file := *ep.Files.Normal
						enrichFileFromRelease(&file, releasesByCRC)

						archive[key] = model.EpisodeArchiveEntry{
							Arc:         ep.Arc,
							Episode:     ep.Episode,
							Title:       ep.Title,
							Description: ep.Description,
							Chapters:    ep.Chapters,
							AnimeEps:    ep.AnimeEps,
							Released:    ep.Released,
							File:        file,
						}
						metadataChanged = true
					}
				}
			}

			// ---- EXTENDED FILE ----
			if ep.Files.Extended != nil {
				key := ep.Files.Extended.CRC32
				if key != "" {

					if _, exists := archive[key]; !exists {

						file := *ep.Files.Extended
						enrichFileFromRelease(&file, releasesByCRC)

						archive[key] = model.EpisodeArchiveEntry{
							Arc:         ep.Arc,
							Episode:     ep.Episode,
							Title:       ep.Title,
							Description: ep.Description,
							Chapters:    ep.Chapters,
							AnimeEps:    ep.AnimeEps,
							Released:    ep.Released,
							File:        file,
						}
						metadataChanged = true
					}
				}
			}
		}
	}

	// ========================================================
	// 4) WRITE EPISODE ARCHIVE (legacy format)
	// ========================================================

	// --- episodes.json ---
	archiveJSON, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(archivePath, archiveJSON) {
		if err := os.WriteFile(archivePath, archiveJSON, 0644); err != nil {
			return err
		}
	}

	// --- episodes.yml ---
	archiveYAML, err := yaml.Marshal(archive)
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/episodes.yml", archiveYAML) {
		if err := os.WriteFile(outDir+"/episodes.yml", archiveYAML, 0644); err != nil {
			return err
		}
	}

	// ========================================================
	// 5) LOAD + MERGE + WRITE RELEASES ARCHIVE (append-only)
	// ========================================================

	releasesArchive := ReleasesArchive{}
	releasesPath := outDir + "/releases.json"

	if util.FileExists(releasesPath) {
		raw, _ := os.ReadFile(releasesPath)
		_ = json.Unmarshal(raw, &releasesArchive)
	}

	for _, r := range releases {
		if r.InfoHash == "" {
			continue
		}
		if _, exists := releasesArchive[r.InfoHash]; !exists {
			releasesArchive[r.InfoHash] = r
			metadataChanged = true
		}
	}

	releasesJSON, err := json.MarshalIndent(releasesArchive, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(releasesPath, releasesJSON) {
		if err := os.WriteFile(releasesPath, releasesJSON, 0644); err != nil {
			return err
		}
	}

	releasesYAML, err := yaml.Marshal(releasesArchive)
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/releases.yml", releasesYAML) {
		if err := os.WriteFile(outDir+"/releases.yml", releasesYAML, 0644); err != nil {
			return err
		}
	}

	// ========================================================
	// 6) WRITE STATUS FILE
	// ========================================================

	if metadataChanged {
		status := map[string]any{
			"updated_at": time.Now().UTC().Format(time.RFC3339),
			"arcs":       len(arcs),
			"episodes":   len(archive),
			"releases":   len(releasesArchive),
		}

		statusJSON, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(outDir+"/status.json", statusJSON, 0644); err != nil {
			return err
		}
	}

	return nil
}

// enrichFileFromRelease fills in an episode file's download links, preferring
// the onepace.net releases feed (exact CRC match, no network round-trip)
// over the Nyaa RSS search fallback used when a CRC isn't in the feed.
func enrichFileFromRelease(file *model.EpisodeFile, releasesByCRC map[string]model.Release) {
	if release, ok := releasesByCRC[file.CRC32]; ok {
		file.URL = release.NyaaURL
		file.MagnetURI = release.MagnetURI
		file.TorrentURL = release.TorrentURL
		return
	}

	if file.URL == "" {
		file.URL = fetch.ResolveNyaaURL(file.CRC32)
	}
}
