package fetch

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"metadata-service/internal/parse"
)

// userAgent identifies this scraper to the sites it reads. onepace.net sits
// behind Cloudflare, which challenges requests carrying a generic library
// User-Agent (Go's default "Go-http-client/1.1", or none) from GitHub
// Actions with a 403. A descriptive UA naming the project is accepted.
const userAgent = "onepace-metadata/1.0 (+https://github.com/jasanpreetn9/onepace-metadata)"

// maxTorrentBytes caps a single .torrent download. Real One Pace batch
// torrents are ~50 KB; this only guards against a misbehaving server.
const maxTorrentBytes = 4 << 20

var httpClient = &http.Client{Timeout: 30 * time.Second}

// get performs a GET with the project User-Agent and returns the response
// if it is 200 OK. Callers must close the body.
func get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return resp, nil
}

// FetchTorrentFileCRCs downloads a .torrent and returns the CRC32s of the
// episode files inside it, plus its total file count. Used for batch
// releases, whose release name has no CRC, so they can be matched to the
// episodes they contain.
func FetchTorrentFileCRCs(torrentURL string) (crcs []string, fileCount int, err error) {
	resp, err := get(torrentURL)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTorrentBytes+1))
	if err != nil {
		return nil, 0, fmt.Errorf("read %s: %w", torrentURL, err)
	}
	if len(body) > maxTorrentBytes {
		return nil, 0, fmt.Errorf("%s: torrent larger than %d bytes", torrentURL, maxTorrentBytes)
	}
	names, err := parse.TorrentFileNames(body)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", torrentURL, err)
	}
	return parse.FileNameCRCs(names), len(names), nil
}
