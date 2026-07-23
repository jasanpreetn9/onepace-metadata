// Package parse holds small, dependency-free parsers that turn the raw
// strings scraped from Google Sheets / the releases feed into typed data.
// Every function here is best-effort: it returns nil/zero on anything that
// doesn't cleanly match, rather than guessing.
package parse

import (
	"strconv"
	"strings"

	"metadata-service/internal/model"
)

// ChapterRange parses a "start-end" style range such as "1 - 7", "129-132",
// or "Ch. 353-355" into a model.ChapterRange. It returns nil for anything
// that isn't a clean two-number range, including non-contiguous lists
// ("1 - 4, 19"), reversed pairs ("42,22"), and prose ("Episode of East
// Blue, Ep. 312 (Intro)") — those are never guessed at.
func ChapterRange(s string) *model.ChapterRange {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"Ch.", "Ep."} {
		if strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)) {
			s = strings.TrimSpace(s[len(prefix):])
			break
		}
	}

	dash := strings.Index(s, "-")
	if dash < 0 {
		return nil
	}

	startStr := strings.TrimSpace(s[:dash])
	endStr := strings.TrimSpace(s[dash+1:])

	start, err := strconv.Atoi(startStr)
	if err != nil {
		return nil
	}
	end, err := strconv.Atoi(endStr)
	if err != nil {
		return nil
	}
	if start > end {
		return nil
	}

	return &model.ChapterRange{Start: start, End: end}
}

// LengthSeconds parses a "mm:ss" or "h:mm:ss" duration string into total
// seconds. Returns 0 (the JSON-omitted zero value) if the string doesn't
// parse.
func LengthSeconds(s string) int {
	parts := strings.Split(strings.TrimSpace(s), ":")

	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return 0
		}
		nums = append(nums, n)
	}

	switch len(nums) {
	case 2:
		return nums[0]*60 + nums[1]
	case 3:
		return nums[0]*3600 + nums[1]*60 + nums[2]
	default:
		return 0
	}
}

// NormalizeVariant maps the releases feed's "regular"/"extended" vocabulary
// onto the episode file's "normal"/"extended" vocabulary so the two can be
// compared/joined directly. Unrecognized values pass through unchanged.
func NormalizeVariant(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "regular":
		return "normal"
	case "normal", "extended":
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return s
	}
}

// Percent parses a "27.00%" style string into a float64 (27.0). Returns nil
// if the string is empty or doesn't parse.
func Percent(s string) *float64 {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%"))
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// IntVal parses a plain integer string. Returns nil if the string is empty
// or doesn't parse.
func IntVal(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}
