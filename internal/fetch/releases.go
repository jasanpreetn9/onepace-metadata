package fetch

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"metadata-service/internal/model"
)

// onePaceReleasesFeed is the official releases feed for onepace.net. It's a
// plain XML endpoint (no headless Chrome needed) that covers the full
// release history, keyed by BitTorrent infoHash.
const onePaceReleasesFeed = "https://onepace.net/en/releases/atom.xml"

// dnCRCRe pulls the trailing CRC32 bracket out of a release filename, e.g.
// "[One Pace][1011-1012] Wano 61 [1080p][1EF3F26C].mkv" → "1EF3F26C".
var dnCRCRe = regexp.MustCompile(`\[([0-9A-Fa-f]{8})\](?:\.\w+)?$`)

//
// ===== ATOM FEED SHAPE =====
//

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID        string       `xml:"id"`
	Title     string       `xml:"title"`
	Published string       `xml:"published"`
	Category  atomCategory `xml:"category"`
	Links     []atomLink   `xml:"link"`
	Content   atomContent  `xml:"content"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type atomContent struct {
	Inner string `xml:",innerxml"`
}

//
// ===== PUBLIC ENTRY =====
//

// FetchReleases downloads and parses the onepace.net releases feed.
func FetchReleases() ([]model.Release, error) {
	resp, err := http.Get(onePaceReleasesFeed)
	if err != nil {
		return nil, fmt.Errorf("fetch releases feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch releases feed: status %d", resp.StatusCode)
	}

	var feed atomFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("decode releases feed: %w", err)
	}

	releases := make([]model.Release, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		releases = append(releases, parseAtomEntry(entry))
	}

	return releases, nil
}

func parseAtomEntry(entry atomEntry) model.Release {
	release := model.Release{
		Title:       strings.TrimSpace(entry.Title),
		Variant:     entry.Category.Term,
		PublishedAt: strings.TrimSpace(entry.Published),
		InfoHash:    strings.TrimPrefix(entry.ID, "urn:btih:"),
	}
	if release.Variant == "" {
		release.Variant = "regular"
	}

	for _, link := range entry.Links {
		switch {
		case strings.HasPrefix(link.Href, "magnet:"):
			release.MagnetURI = link.Href
			release.CRC32 = crcFromMagnet(link.Href)
		case link.Rel == "enclosure":
			release.TorrentURL = link.Href
		case link.Rel == "related":
			release.NyaaURL = link.Href
		}
	}

	if doc, err := goquery.NewDocumentFromReader(strings.NewReader(entry.Content.Inner)); err == nil {
		doc.Find("dl dt").Each(func(_ int, s *goquery.Selection) {
			label := strings.ToLower(strings.TrimSpace(s.Text()))
			val := strings.TrimSpace(s.Next().Text())
			switch label {
			case "manga chapters":
				release.MangaChapters = val
			case "anime episodes":
				release.AnimeEpisodes = val
			}
		})

		doc.Find("details").Each(func(_ int, s *goquery.Selection) {
			if !strings.Contains(strings.ToLower(s.Find("summary").Text()), "changelog") {
				return
			}
			s.Find("li").Each(func(_ int, li *goquery.Selection) {
				if text := strings.TrimSpace(li.Text()); text != "" {
					release.Changelog = append(release.Changelog, text)
				}
			})
		})
	}

	return release
}

// crcFromMagnet extracts the CRC32 embedded in a magnet URI's "dn" filename.
func crcFromMagnet(magnetURI string) string {
	raw := strings.TrimPrefix(magnetURI, "magnet:?")
	values, err := url.ParseQuery(raw)
	if err != nil {
		return ""
	}

	dn := values.Get("dn")
	if dn == "" {
		return ""
	}

	match := dnCRCRe.FindStringSubmatch(dn)
	if len(match) < 2 {
		return ""
	}

	return strings.ToUpper(match[1])
}
