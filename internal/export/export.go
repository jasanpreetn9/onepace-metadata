package export

import (
	"encoding/json"
	"fmt"
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
		raw, err := os.ReadFile(archivePath)
		if err != nil {
			return fmt.Errorf("read existing episode archive %s: %w", archivePath, err)
		}
		if err := json.Unmarshal(raw, &archive); err != nil {
			return fmt.Errorf("parse existing episode archive %s: %w", archivePath, err)
		}
	}

	// ========================================================
	// 2b) LOAD + MERGE RELEASES ARCHIVE (append-only)
	// ========================================================
	// Loaded before the episode steps so 3d can use the full release
	// history (not just this run's feed) when breaking is_current ties.

	releasesArchive := ReleasesArchive{}
	releasesPath := outDir + "/releases.json"

	if util.FileExists(releasesPath) {
		raw, err := os.ReadFile(releasesPath)
		if err != nil {
			return fmt.Errorf("read existing releases archive %s: %w", releasesPath, err)
		}
		if err := json.Unmarshal(raw, &releasesArchive); err != nil {
			return fmt.Errorf("parse existing releases archive %s: %w", releasesPath, err)
		}
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
							ArcID:       arc.ID,
							EpisodeID:   ep.ID,
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
							ArcID:       arc.ID,
							EpisodeID:   ep.ID,
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
	// 3b) BACKFILL EXISTING ENTRIES WITH MAGNET/TORRENT LINKS
	// ========================================================
	// The releases feed was added after most of the archive already existed,
	// so entries created before this feature won't have magnet_uri/torrent_url
	// from the "new episode" path above. Fill in only what's missing —
	// additive, never overwrites an entry's existing data.
	for crc, entry := range archive {
		if entry.File.MagnetURI != "" && entry.File.TorrentURL != "" {
			continue
		}
		release, ok := releasesByCRC[crc]
		if !ok {
			continue
		}
		changed := false
		if entry.File.MagnetURI == "" && release.MagnetURI != "" {
			entry.File.MagnetURI = release.MagnetURI
			changed = true
		}
		if entry.File.TorrentURL == "" && release.TorrentURL != "" {
			entry.File.TorrentURL = release.TorrentURL
			changed = true
		}
		if entry.File.URL == "" && release.NyaaURL != "" {
			entry.File.URL = release.NyaaURL
			changed = true
		}
		if entry.File.ReleaseInfoHash == "" && release.InfoHash != "" {
			entry.File.ReleaseInfoHash = release.InfoHash
			changed = true
		}
		if changed {
			archive[crc] = entry
			metadataChanged = true
		}
	}

	// ========================================================
	// 3c) BACKFILL STABLE ARC/EPISODE IDS ONTO OLDER ENTRIES
	// ========================================================
	// ArcID/EpisodeID were added after most of the archive already existed
	// (same situation as 3b above). Backfill them from this run's arcs by
	// matching on the (Arc, Episode) numbers the entry was recorded with.
	type arcEpisodeKey struct {
		Arc     int
		Episode int
	}
	type stableIDs struct{ ArcID, EpisodeID string }
	idLookup := make(map[arcEpisodeKey]stableIDs)
	for _, arc := range arcs {
		for _, ep := range arc.Episodes {
			idLookup[arcEpisodeKey{Arc: ep.Arc, Episode: ep.Episode}] = stableIDs{ArcID: arc.ID, EpisodeID: ep.ID}
		}
	}
	for crc, entry := range archive {
		if entry.ArcID != "" && entry.EpisodeID != "" {
			continue
		}
		ids, ok := idLookup[arcEpisodeKey{Arc: entry.Arc, Episode: entry.Episode}]
		if !ok {
			continue
		}
		entry.ArcID = ids.ArcID
		entry.EpisodeID = ids.EpisodeID
		archive[crc] = entry
		metadataChanged = true
	}

	// ========================================================
	// 3d) COMPUTE IS_CURRENT PER (EPISODE, VARIANT)
	// ========================================================
	// Episodes get re-released under new CRC32s over time; mark the entry
	// with the newest Released date (ISO YYYY-MM-DD, so lexicographic
	// comparison is chronological) as current within its group, so a
	// consumer can find "the" download link without scanning every
	// historical CRC itself. Groups by EpisodeID when known, falling back
	// to the raw (Arc, Episode) numbers for any entry the backfill above
	// couldn't resolve.
	//
	// Released only has day precision, and a re-release can land on the same
	// date as the CRC it replaces, so ties are broken deterministically (see
	// newerArchiveEntry). Without this the winner depended on map iteration
	// order and flipped between runs.
	listedCRCs := make(map[string]bool)
	for _, arc := range arcs {
		for _, ep := range arc.Episodes {
			if ep.Files.Normal != nil && ep.Files.Normal.CRC32 != "" {
				listedCRCs[ep.Files.Normal.CRC32] = true
			}
			if ep.Files.Extended != nil && ep.Files.Extended.CRC32 != "" {
				listedCRCs[ep.Files.Extended.CRC32] = true
			}
		}
	}
	publishedAt := make(map[string]string)
	for _, r := range releasesArchive {
		if r.CRC32 != "" && r.PublishedAt > publishedAt[r.CRC32] {
			publishedAt[r.CRC32] = r.PublishedAt
		}
	}

	type versionKey struct {
		Episode string
		Variant string
	}
	groups := make(map[versionKey][]string)
	for crc, entry := range archive {
		epKey := entry.EpisodeID
		if epKey == "" {
			epKey = fmt.Sprintf("%d-%d", entry.Arc, entry.Episode)
		}
		k := versionKey{Episode: epKey, Variant: entry.File.Version}
		groups[k] = append(groups[k], crc)
	}
	for _, crcs := range groups {
		latest := crcs[0]
		for _, crc := range crcs[1:] {
			if newerArchiveEntry(archive[crc], archive[latest], listedCRCs, publishedAt) {
				latest = crc
			}
		}
		for _, crc := range crcs {
			want := crc == latest
			if entry := archive[crc]; entry.IsCurrent != want {
				entry.IsCurrent = want
				archive[crc] = entry
				metadataChanged = true
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
	// 4b) BUILD + WRITE DERIVED "CURRENT" VIEW
	// ========================================================
	// One entry per episode, keyed by EpisodeID, holding only the archive
	// entries marked IsCurrent per variant — see model.CurrentEpisode.
	currentEpisodes := make(map[string]model.CurrentEpisode)
	for _, entry := range archive {
		if !entry.IsCurrent {
			continue
		}
		epKey := entry.EpisodeID
		if epKey == "" {
			epKey = fmt.Sprintf("%d-%d", entry.Arc, entry.Episode)
		}

		ce := currentEpisodes[epKey]
		ce.ArcID = entry.ArcID
		ce.EpisodeID = entry.EpisodeID
		ce.Arc = entry.Arc
		ce.Episode = entry.Episode
		ce.Title = entry.Title
		ce.Description = entry.Description
		ce.Chapters = entry.Chapters
		ce.AnimeEps = entry.AnimeEps
		ce.Released = entry.Released

		file := entry.File
		if file.Version == "extended" {
			ce.Files.Extended = &file
		} else {
			ce.Files.Normal = &file
		}
		currentEpisodes[epKey] = ce
	}

	currentPath := outDir + "/episodes-current.json"
	currentJSON, err := json.MarshalIndent(currentEpisodes, "", "  ")
	if err != nil {
		return err
	}
	if !util.FileUnchanged(currentPath, currentJSON) {
		if err := os.WriteFile(currentPath, currentJSON, 0644); err != nil {
			return err
		}
	}

	currentYAML, err := yaml.Marshal(currentEpisodes)
	if err != nil {
		return err
	}
	if !util.FileUnchanged(outDir+"/episodes-current.yml", currentYAML) {
		if err := os.WriteFile(outDir+"/episodes-current.yml", currentYAML, 0644); err != nil {
			return err
		}
	}

	// ========================================================
	// 5) WRITE RELEASES ARCHIVE (loaded + merged in step 2b)
	// ========================================================

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

// newerArchiveEntry reports whether a should replace b as the current entry
// of an (episode, variant) group. The order is a total order on distinct
// CRCs, so the result never depends on map iteration order:
//  1. newer Released date (ISO YYYY-MM-DD, lexicographic == chronological)
//  2. the CRC the episode guide sheet lists right now
//  3. newer release publish time from the releases feed (RFC3339)
//  4. greater CRC32 string — arbitrary, but stable across runs
func newerArchiveEntry(a, b model.EpisodeArchiveEntry, listedCRCs map[string]bool, publishedAt map[string]string) bool {
	if a.Released != b.Released {
		return a.Released > b.Released
	}
	aCRC, bCRC := a.File.CRC32, b.File.CRC32
	if listedCRCs[aCRC] != listedCRCs[bCRC] {
		return listedCRCs[aCRC]
	}
	if pa, pb := publishedAt[aCRC], publishedAt[bCRC]; pa != pb {
		return pa > pb
	}
	return aCRC > bCRC
}

// enrichFileFromRelease fills in an episode file's download links, preferring
// the onepace.net releases feed (exact CRC match, no network round-trip)
// over the Nyaa RSS search fallback used when a CRC isn't in the feed.
func enrichFileFromRelease(file *model.EpisodeFile, releasesByCRC map[string]model.Release) {
	if release, ok := releasesByCRC[file.CRC32]; ok {
		file.URL = release.NyaaURL
		file.MagnetURI = release.MagnetURI
		file.TorrentURL = release.TorrentURL
		file.ReleaseInfoHash = release.InfoHash
		return
	}

	if file.URL == "" {
		file.URL = fetch.ResolveNyaaURL(file.CRC32)
	}
}
