package export

import (
	"fmt"
	"sort"

	"metadata-service/internal/fetch"
	"metadata-service/internal/model"
	"metadata-service/internal/parse"
)

// fetchTorrentFileCRCs is fetch.FetchTorrentFileCRCs, swappable in tests.
var fetchTorrentFileCRCs = fetch.FetchTorrentFileCRCs

// maxConsecutiveTorrentErrors stops reading batch torrents for this run
// after this many failures in a row, so a blocked or down host can't stall
// the job on one timeout per batch. Unread batches are retried next run.
const maxConsecutiveTorrentErrors = 3

// fillBatchFileCRCs reads the file list of every batch release in the
// archive that hasn't been read yet, recording its episode CRCs and file
// count. Batches are visited in InfoHash order so runs are reproducible.
// Reports whether any archive entry changed.
func fillBatchFileCRCs(archive ReleasesArchive) bool {
	var pending []string
	for hash, r := range archive {
		if r.CRC32 == "" && r.FileCount == 0 && r.TorrentURL != "" {
			pending = append(pending, hash)
		}
	}
	sort.Strings(pending)

	changed := false
	consecutiveErrors := 0
	for _, hash := range pending {
		r := archive[hash]
		crcs, count, err := fetchTorrentFileCRCs(r.TorrentURL)
		if err != nil {
			fmt.Printf("Warning: read batch torrent %q: %v\n", r.Title, err)
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveTorrentErrors {
				fmt.Printf("Warning: %d batch torrent reads failed in a row; leaving the rest for the next run\n", consecutiveErrors)
				break
			}
			continue
		}
		consecutiveErrors = 0
		if count == 0 {
			continue // malformed/empty torrent: leave unread, retry next run
		}
		r.FileCRC32s = crcs
		r.FileCount = count
		archive[hash] = r
		changed = true
	}
	return changed
}

// releaseIndex finds the release that supplies download links for an
// episode file, by CRC32.
type releaseIndex struct {
	// single maps a CRC32 to its single-file release.
	single map[string]model.Release
	// batches maps a CRC32 to every batch release containing that file.
	batches map[string][]model.Release
}

func newReleaseIndex(archive ReleasesArchive) releaseIndex {
	idx := releaseIndex{
		single:  make(map[string]model.Release),
		batches: make(map[string][]model.Release),
	}
	for _, r := range archive {
		if r.MagnetURI == "" && r.TorrentURL == "" {
			continue // nothing to link to
		}
		if r.CRC32 != "" {
			if cur, ok := idx.single[r.CRC32]; !ok || newerRelease(r, cur) {
				idx.single[r.CRC32] = r
			}
			continue
		}
		for _, crc := range r.FileCRC32s {
			idx.batches[crc] = append(idx.batches[crc], r)
		}
	}
	return idx
}

// find returns the release to take crc's download links from. A
// single-file release always wins: it downloads exactly that episode. A
// batch is the fallback for episodes released only as part of a whole-arc
// torrent; among several batches containing the file, it prefers one of
// the same cut (version "normal"/"extended"), then the newest, so the
// choice is deterministic.
func (idx releaseIndex) find(crc, version string) (model.Release, bool) {
	if r, ok := idx.single[crc]; ok {
		return r, true
	}
	candidates := idx.batches[crc]
	if len(candidates) == 0 {
		return model.Release{}, false
	}
	best := candidates[0]
	for _, r := range candidates[1:] {
		rMatch := releaseVariant(r) == version
		bestMatch := releaseVariant(best) == version
		if rMatch != bestMatch {
			if rMatch {
				best = r
			}
			continue
		}
		if newerRelease(r, best) {
			best = r
		}
	}
	return best, true
}

// releaseVariant is the release's cut in EpisodeFile.Version's vocabulary.
// NormalizedVariant is missing on archive entries written before it existed.
func releaseVariant(r model.Release) string {
	if r.NormalizedVariant != "" {
		return r.NormalizedVariant
	}
	return parse.NormalizeVariant(r.Variant)
}

// newerRelease orders releases by publish time (RFC3339, so lexicographic
// is chronological), then InfoHash, giving a total order.
func newerRelease(a, b model.Release) bool {
	if a.PublishedAt != b.PublishedAt {
		return a.PublishedAt > b.PublishedAt
	}
	return a.InfoHash > b.InfoHash
}
