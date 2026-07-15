package fetch

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// nyaaRSS models the subset of the Nyaa RSS feed we care about.
type nyaaRSS struct {
	Channel struct {
		Items []struct {
			Title string `xml:"title"`
			GUID  string `xml:"guid"`
		} `xml:"item"`
	} `xml:"channel"`
}

var nyaaHTTP = &http.Client{Timeout: 15 * time.Second}

// ResolveNyaaURL looks up a One Pace release on Nyaa by its CRC32 and returns
// the torrent view URL, or "" if it can't be found. The episode guide sheet
// used to hyperlink every CRC to its Nyaa page but no longer does, so this
// recovers the download URL for newly released episodes.
func ResolveNyaaURL(crc32 string) string {
	q := url.QueryEscape(`"One Pace" ` + crc32)
	resp, err := nyaaHTTP.Get("https://nyaa.si/?page=rss&q=" + q)
	if err != nil {
		fmt.Printf("Warning: nyaa lookup for %s failed: %v\n", crc32, err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Warning: nyaa lookup for %s: status %d\n", crc32, resp.StatusCode)
		return ""
	}

	var feed nyaaRSS
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		fmt.Printf("Warning: nyaa lookup for %s: parse: %v\n", crc32, err)
		return ""
	}

	// Require the CRC to appear in the release title so a fuzzy search
	// match can't attach the wrong torrent.
	for _, item := range feed.Channel.Items {
		if strings.Contains(strings.ToUpper(item.Title), strings.ToUpper(crc32)) {
			return item.GUID
		}
	}
	return ""
}
