package fetch

import (
	"context"
	"fmt"
	"metadata-service/internal/config"
	"metadata-service/internal/model"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

//
// ===== PUBLIC ENTRY =====
//

// FetchEpisodeGuideHome parses the main arc list (HTML) + all arc CSVs.
func FetchEpisodeGuideHome() ([]model.Arc, error) {

	arcs, err := fetchArcList(config.OnePaceEpisodeGuide)
	if err != nil {
		return nil, fmt.Errorf("fetchArcList: %w", err)
	}

	arcs = normalizeArcIDs(arcs)

	for i := range arcs {
		if arcs[i].GID == "" {
			continue
		}
		fmt.Printf("Fetching - %d - %s.\n", arcs[i].Arc, arcs[i].Title)
		episodes, err := fetchArcEpisodes(config.OnePaceEpisodeGuide, arcs[i].GID)
		if err != nil {
			fmt.Printf("Warning: failed to fetch episodes for arc %d: %v\n", arcs[i].Arc, err)
			continue
		}

		for idx := range episodes {
			episodes[idx].Arc = arcs[i].Arc
		}

		arcs[i].Episodes = append(arcs[i].Episodes, episodes...)

		sort.Slice(arcs[i].Episodes, func(a, b int) bool {
			return arcs[i].Episodes[a].Episode < arcs[i].Episodes[b].Episode
		})
	}

	// Merge descriptions
	desc, err := FetchEpisodeDescriptions()
	if err == nil {
		for i := range arcs {
			set, ok := desc[arcs[i].Title]
			if !ok {
				continue
			}
			for idx := range arcs[i].Episodes {
				ep := &arcs[i].Episodes[idx]
				if meta, ok := set[ep.Episode]; ok {
					ep.Title = meta.Title
					ep.Description = meta.Description
				}
			}
		}
	}

	return arcs, nil
}

// fetchArcList reads the main Google Sheet HTML arc list.
func fetchArcList(spreadsheetID string) ([]model.Arc, error) {
	url := fmt.Sprintf("https://docs.google.com/spreadsheets/u/0/d/%s/htmlview/sheet?headers=true&gid=0", spreadsheetID)

	fmt.Println("Launching Chrome...")

	ctx, cancel := chromedp.NewContext(
		context.Background(),
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	var html string

	fmt.Println("Navigating to:", url)

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),

		// Wait until table loads
		chromedp.WaitVisible(`table.waffle`, chromedp.ByQuery),
		chromedp.WaitReady(`table.waffle`, chromedp.ByQuery),

		// Google Sheets still loads slowly, give it a little extra
		chromedp.Sleep(2*time.Second),

		// Dump entire HTML
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp: %w", err)
	}

	// Parse using goquery (use a new reader)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("goquery: %w", err)
	}

	var arcs []model.Arc

	// This selector works for the sheet HTML you posted
	rows := doc.Find("table.waffle tr")

	if rows.Length() == 0 {
		// fallback selectors
		rows = doc.Find("table.ritz tr")
	}
	if rows.Length() == 0 {
		rows = doc.Find("table.grid-container tr")
	}
	if rows.Length() == 0 {
		return nil, fmt.Errorf("no rows found in sheet HTML")
	}

	rows.Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		rawArc := strings.TrimSpace(cells.Eq(0).Text())
		if rawArc == "" || rawArc == "No." {
			return
		}
		arcFloat, err := strconv.ParseFloat(rawArc, 64)

		title := strings.TrimSpace(cells.Eq(1).Text())
		if title == "" || title == "Arcs" {
			return
		}

		// Extract GID from <a href="#gid=1122135437">
		gid := ""
		cells.Eq(1).Find("a").Each(func(_ int, a *goquery.Selection) {
			if href, ok := a.Attr("href"); ok && strings.Contains(href, "gid=") {
				parts := strings.Split(href, "gid=")
				gid = parts[len(parts)-1]
			}
		})

		if err != nil {
			return
		}

		// Detect WIP / TBR tags in the title
		status := ""
		cleanTitle := title

		upper := strings.ToUpper(title)
		if strings.Contains(upper, "(WIP)") {
			status = "WIP"
			cleanTitle = strings.TrimSpace(strings.ReplaceAll(title, "(WIP)", ""))
		} else if strings.Contains(upper, "(TBR)") {
			status = "TBR"
			cleanTitle = strings.TrimSpace(strings.ReplaceAll(title, "(TBR)", ""))
		}
		mangaChapters := strings.TrimSpace(cells.Eq(3).Text())
		numberofChapters := strings.TrimSpace(cells.Eq(4).Text())
		animeEpisodes := strings.TrimSpace(cells.Eq(5).Text())
		episodesAdapted := strings.TrimSpace(cells.Eq(6).Text())
		fillerEpisodes := strings.TrimSpace(cells.Eq(7).Text())
		timeSavedMins := strings.TrimSpace(cells.Eq(11).Text())
		timeSavedPercent := strings.TrimSpace(cells.Eq(12).Text())
		audioLanguages := strings.TrimSpace(cells.Eq(13).Text())
		subtitleLanguages := strings.TrimSpace(cells.Eq(14).Text())
		resolution := strings.TrimSpace(cells.Eq(16).Text())

		arcs = append(arcs, model.Arc{
			Arc:               int(arcFloat * 10),
			Title:             cleanTitle,
			Status:            status,
			AudioLanguages:    audioLanguages,
			SubtitleLanguages: subtitleLanguages,
			MangaChapters:     mangaChapters,
			NumberOfChapters:  numberofChapters,
			EpisodesAdapted:   episodesAdapted,
			FillerEpisodes:    fillerEpisodes,
			TimeSavedMins:     timeSavedMins,
			TimeSavedPercent:  timeSavedPercent,
			AnimeEpisodes:     animeEpisodes,
			Resolution:        resolution,
			GID:               gid,
		})

	})
	return arcs, nil
}

// normalizeArcIDs fixes fractional arc numbers (6.5 → 7)
// AND guarantees unique, sequential arc IDs.
func normalizeArcIDs(arcs []model.Arc) []model.Arc {
	if len(arcs) == 0 {
		return arcs
	}

	normalized := make([]model.Arc, len(arcs))

	nextID := 1
	lastAssigned := 0

	for i, arc := range arcs {

		// Convert fractional decimals
		raw := float64(arc.Arc) / 10.0

		// Round fractional arc numbers UP (6.5 → 7)
		var rounded int
		if raw == float64(int(raw)) {
			rounded = int(raw)
		} else {
			rounded = int(raw) + 1
		}

		// Ensure no duplicates:
		// If rounded ≤ lastAssigned, bump it
		if rounded <= lastAssigned {
			rounded = lastAssigned + 1
		}

		// Assign sequential ID
		normalized[i] = arc
		normalized[i].Arc = nextID

		lastAssigned = rounded
		nextID++
	}

	return normalized
}

// Removes odd spaces and trims
func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\u00A0", " ")
	return strings.TrimSpace(s)
}

// Extract trailing episode number from "Romance Dawn 03"
func extractEpisodeNumber(s string) int {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return 0
	}

	last := parts[len(parts)-1]

	// Remove leading zeros
	last = strings.TrimLeft(last, "0")
	if last == "" {
		last = "0"
	}

	n, _ := strconv.Atoi(last)
	return n
}

// Convert "2025.05.03" → "2025-05-03"
func convertDate(s string) string {
	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		if len(parts) == 3 {
			return fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
		}
	}
	return strings.TrimSpace(s)
}

// Extracts the real URL from a Google redirect href.
// Example:
// https://www.google.com/url?q=https://nyaa.si/view/2004229&...  → "https://nyaa.si/view/2004229"
func extractURLFromHref(href string) string {
	if href == "" {
		return ""
	}

	// Google redirect links contain: ?q=<real URL>
	idx := strings.Index(href, "q=")
	if idx != -1 {
		real := href[idx+2:] // trim "?q="
		if amp := strings.Index(real, "&"); amp != -1 {
			real = real[:amp]
		}

		unescaped, err := url.QueryUnescape(real)
		if err == nil && unescaped != "" {
			return unescaped
		}

		return real
	}

	return href
}

// fetchArcEpisodes downloads & parses each arc’s episode guide.
// fetchArcEpisodes parses the episode table for a specific arc.
func fetchArcEpisodes(spreadsheetID, gid string) ([]model.Episode, error) {

	sheetURL := fmt.Sprintf(
		"https://docs.google.com/spreadsheets/u/0/d/%s/htmlview/sheet?headers=true&gid=%s",
		spreadsheetID, gid,
	)

	fmt.Println("Fetching arc episodes:", sheetURL)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	var html string

	err := chromedp.Run(ctx,
		chromedp.Navigate(sheetURL),
		chromedp.WaitVisible(`table.waffle`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("goquery: %w", err)
	}

	rows := doc.Find("table.waffle tr")
	if rows.Length() == 0 {
		rows = doc.Find("table.ritz tr")
	}
	if rows.Length() == 0 {
		return nil, fmt.Errorf("no rows found in sheet")
	}

	var episodes []model.Episode

	rows.Each(func(i int, row *goquery.Selection) {

		if i < 2 {
			return
		}

		cells := row.Find("td")
		if cells.Length() < 7 {
			return
		}

		epName := cleanText(cells.Eq(1).Text())
		if epName == "" {
			return
		}

		epNum := extractEpisodeNumber(epName)

		chapters := cleanText(cells.Eq(2).Text())
		animeEps := cleanText(cells.Eq(3).Text())
		releaseDate := convertDate(cleanText(cells.Eq(4).Text()))
		length := cleanText(cells.Eq(5).Text())

		var files model.EpisodeFileVariants
		var hasExtended bool

		// ─────────────────────────────────────
		// NORMAL VERSION
		// ─────────────────────────────────────
		var crc32 string
		var url string

		cells.Eq(6).Find("a").Each(func(_ int, a *goquery.Selection) {
			crc32 = cleanText(a.Text())
			if href, ok := a.Attr("href"); ok {
				url = extractURLFromHref(href)
			}
		})

		if crc32 != "" {
			files.Normal = &model.EpisodeFile{
				Version: "normal",
				CRC32:   crc32,
				Length:  length,
				URL:     url,
			}
		}

		// ─────────────────────────────────────
		// EXTENDED VERSION
		// ─────────────────────────────────────
		if cells.Length() >= 8 {

			var crcExt, urlExt string
			extLength := ""

			cells.Eq(7).Find("a").Each(func(_ int, a *goquery.Selection) {
				crcExt = cleanText(a.Text())
				if href, ok := a.Attr("href"); ok {
					urlExt = extractURLFromHref(href)
				}
			})

			if cells.Length() >= 9 {
				extLength = cleanText(cells.Eq(8).Text())
			}

			if crcExt != "" {
				hasExtended = true
				files.Extended = &model.EpisodeFile{
					Version: "extended",
					CRC32:   crcExt,
					Length:  extLength,
					URL:     urlExt,
				}
			}
		}

		episodes = append(episodes, model.Episode{
			Episode:     epNum,
			Title:       epName,
			Chapters:    chapters,
			AnimeEps:    animeEps,
			Released:    releaseDate,
			HasExtended: hasExtended,
			Files:       files,
		})
	})

	return episodes, nil
}
