package export

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"metadata-service/internal/model"
)

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}

// stubTorrents replaces the .torrent fetcher for the test, serving file
// CRC lists by torrent URL and counting calls.
func stubTorrents(t *testing.T, files map[string][]string) *int {
	t.Helper()
	calls := 0
	orig := fetchTorrentFileCRCs
	fetchTorrentFileCRCs = func(url string) ([]string, int, error) {
		calls++
		crcs, ok := files[url]
		if !ok {
			return nil, 0, errors.New("not found")
		}
		return crcs, len(crcs) + 1, nil // +1: a non-episode file (e.g. .txt)
	}
	t.Cleanup(func() { fetchTorrentFileCRCs = orig })
	return &calls
}

// Reproduces logpose issue #20: the 2026-10-26 Loguetown re-release was
// published only as whole-arc batch torrents, so its episodes never got
// download links.
func TestExportMetadata_LinksFromBatchTorrents(t *testing.T) {
	dir := t.TempDir()

	entry := func(crc, version string) model.EpisodeArchiveEntry {
		return model.EpisodeArchiveEntry{
			ArcID: "logue", EpisodeID: "logue-" + crc, Arc: 8, Episode: 1,
			Released: "2026-10-26",
			File:     model.EpisodeFile{Version: version, CRC32: crc},
		}
	}
	writeJSON(t, filepath.Join(dir, "episodes.json"), EpisodesArchive{
		"ED297337": entry("ED297337", "normal"),   // in both regular batches
		"7191C79A": entry("7191C79A", "extended"), // only in the extended batch
		"BE8D8724": entry("BE8D8724", "normal"),   // in all three batches
		"AAAA0001": entry("AAAA0001", "normal"),   // has its own single-file release
		"DEADBEEF": entry("DEADBEEF", "normal"),   // in no release at all
	})

	writeJSON(t, filepath.Join(dir, "releases.json"), ReleasesArchive{
		"aaaa": {Title: "[One Pace][96-100] Loguetown [1080p]", Variant: "regular", InfoHash: "aaaa",
			PublishedAt: "2026-10-26T12:00:00.000Z", MagnetURI: "magnet:?xt=urn:btih:aaaa", TorrentURL: "https://t/aaaa.torrent"},
		"bbbb": {Title: "[One Pace][96-100] Loguetown [1080p]", Variant: "regular", InfoHash: "bbbb",
			PublishedAt: "2026-10-26T12:00:00.000Z", MagnetURI: "magnet:?xt=urn:btih:bbbb", TorrentURL: "https://t/bbbb.torrent"},
		"eeee": {Title: "[One Pace][96-100] Loguetown - Extended Cut [1080p]", Variant: "extended", InfoHash: "eeee",
			PublishedAt: "2026-10-26T13:00:00.000Z", MagnetURI: "magnet:?xt=urn:btih:eeee", TorrentURL: "https://t/eeee.torrent"},
		"ffff": {Title: "[One Pace][1] Single [1080p][AAAA0001].mkv", Variant: "regular", InfoHash: "ffff", CRC32: "AAAA0001",
			PublishedAt: "2026-01-01T00:00:00.000Z", MagnetURI: "magnet:?xt=urn:btih:ffff", TorrentURL: "https://t/ffff.torrent",
			NyaaURL: "https://nyaa.si/view/1"},
		"9999": {Title: "Broken batch", Variant: "regular", InfoHash: "9999",
			PublishedAt: "2026-10-26T12:00:00.000Z", MagnetURI: "magnet:?xt=urn:btih:9999", TorrentURL: "https://t/missing.torrent"},
	})

	calls := stubTorrents(t, map[string][]string{
		"https://t/aaaa.torrent": {"ED297337", "BE8D8724"},
		"https://t/bbbb.torrent": {"BE8D8724", "ED297337", "A6C3657B"},
		"https://t/eeee.torrent": {"BE8D8724", "7191C79A"},
		"https://t/ffff.torrent": {"AAAA0001"}, // must never be fetched: not a batch
	})

	if err := ExportMetadata(nil, nil, dir); err != nil {
		t.Fatalf("ExportMetadata: %v", err)
	}

	var archive EpisodesArchive
	readJSON(t, filepath.Join(dir, "episodes.json"), &archive)

	wantHash := map[string]string{
		"ED297337": "bbbb", // regular batches tie on date → greater InfoHash
		"7191C79A": "eeee",
		"BE8D8724": "bbbb", // same-cut batch beats the newer extended one
		"AAAA0001": "ffff", // single-file release wins over any batch
	}
	for crc, hash := range wantHash {
		f := archive[crc].File
		if f.ReleaseInfoHash != hash || f.MagnetURI != "magnet:?xt=urn:btih:"+hash || f.TorrentURL == "" {
			t.Errorf("%s: got hash=%q magnet=%q torrent=%q, want links from %s",
				crc, f.ReleaseInfoHash, f.MagnetURI, f.TorrentURL, hash)
		}
	}
	if got := archive["AAAA0001"].File.URL; got != "https://nyaa.si/view/1" {
		t.Errorf("AAAA0001 URL = %q, want the release's Nyaa URL", got)
	}
	if f := archive["DEADBEEF"].File; f.MagnetURI != "" || f.TorrentURL != "" {
		t.Errorf("DEADBEEF should stay unlinked, got %+v", f)
	}

	var releases ReleasesArchive
	readJSON(t, filepath.Join(dir, "releases.json"), &releases)
	if r := releases["bbbb"]; r.FileCount != 4 || len(r.FileCRC32s) != 3 {
		t.Errorf("bbbb: FileCount=%d FileCRC32s=%v, want 4 files / 3 CRCs recorded", r.FileCount, r.FileCRC32s)
	}
	if r := releases["9999"]; r.FileCount != 0 {
		t.Errorf("failed batch read should stay unread for retry, got FileCount=%d", r.FileCount)
	}
	if *calls != 4 { // aaaa, bbbb, eeee, 9999
		t.Errorf("torrent fetches = %d, want 4 (batches only)", *calls)
	}

	// Second run: read batches are never fetched again; only the failed
	// one is retried.
	*calls = 0
	if err := ExportMetadata(nil, nil, dir); err != nil {
		t.Fatalf("second ExportMetadata: %v", err)
	}
	if *calls != 1 {
		t.Errorf("second run torrent fetches = %d, want 1 (only the failed batch)", *calls)
	}
}

func TestFillBatchFileCRCs_StopsAfterConsecutiveErrors(t *testing.T) {
	archive := ReleasesArchive{}
	for _, h := range []string{"a1", "a2", "a3", "a4", "a5"} {
		archive[h] = model.Release{InfoHash: h, TorrentURL: "https://t/" + h}
	}
	calls := stubTorrents(t, nil) // every fetch fails

	if fillBatchFileCRCs(archive) {
		t.Error("nothing should have changed")
	}
	if *calls != maxConsecutiveTorrentErrors {
		t.Errorf("fetches = %d, want %d before giving up", *calls, maxConsecutiveTorrentErrors)
	}
}
