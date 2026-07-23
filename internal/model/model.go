package model

//
// ===============================
//         ARC STRUCTS
// ===============================
//

type Arc struct {
	// ID is a stable identifier derived from GID (falls back to a
	// slugified Title for arcs with no sheet yet). Unlike Arc, it is never
	// reassigned by normalizeArcIDs, so it's safe to use as a join key
	// across scrapes.
	ID  string `json:"id,omitempty" yaml:"id,omitempty"`
	Arc int    `json:"arc" yaml:"arc"`

	Title string `json:"title" yaml:"title"`

	AudioLanguages    string `json:"audio_languages" yaml:"audio_languages"`
	SubtitleLanguages string `json:"subtitle_languages" yaml:"subtitle_languages"`
	Resolution        string `json:"resolution" yaml:"resolution"`

	MangaChapters         string        `json:"manga_chapters" yaml:"manga_chapters"`
	MangaChapterRange     *ChapterRange `json:"manga_chapter_range,omitempty" yaml:"manga_chapter_range,omitempty"`
	NumberOfChapters      string        `json:"number_of_chapters" yaml:"number_of_chapters"`
	AnimeEpisodes         string        `json:"anime_episodes" yaml:"anime_episodes"`
	AnimeEpisodeRange     *ChapterRange `json:"anime_episode_range,omitempty" yaml:"anime_episode_range,omitempty"`
	EpisodesAdapted       string        `json:"episodes_adapted" yaml:"episodes_adapted"`
	FillerEpisodes        string        `json:"filler_episodes" yaml:"filler_episodes"`
	TimeSavedMins         string        `json:"time_saved_mins" yaml:"time_saved_mins"`
	TimeSavedMinsValue    *int          `json:"time_saved_mins_value,omitempty" yaml:"time_saved_mins_value,omitempty"`
	TimeSavedPercent      string        `json:"time_saved_percent" yaml:"time_saved_percent"`
	TimeSavedPercentValue *float64      `json:"time_saved_percent_value,omitempty" yaml:"time_saved_percent_value,omitempty"`

	Status string `json:"status" yaml:"status"` // WIP / TBR / ""

	Episodes []Episode `json:"episodes" yaml:"episodes"`

	GID string `json:"gid,omitempty" yaml:"gid,omitempty"`
}

// ChapterRange is a best-effort parse of a "start-end" range string. It's
// only populated when the source text is an unambiguous two-number range —
// left nil for non-contiguous ("1 - 4, 19") or prose-style values rather
// than guessing. See internal/parse.ChapterRange.
type ChapterRange struct {
	Start int `json:"start" yaml:"start"`
	End   int `json:"end" yaml:"end"`
}

//
// ===============================
//       EPISODE STRUCTS
// ===============================
//

type Episode struct {
	// ID is a stable identifier: Arc.ID + zero-padded episode number.
	// Assumes episode numbers within an arc are never renumbered by the
	// source sheet — the best available natural key given the data source.
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
	Arc     int    `json:"arc" yaml:"arc"`
	Episode int    `json:"episode" yaml:"episode"`

	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`

	Chapters     string        `json:"chapters" yaml:"chapters"`
	ChapterRange *ChapterRange `json:"chapter_range,omitempty" yaml:"chapter_range,omitempty"`
	AnimeEps     string        `json:"episodes" yaml:"episodes"`

	Released string `json:"released" yaml:"released"`

	HasExtended bool                `json:"has_extended" yaml:"has_extended"`
	Files       EpisodeFileVariants `json:"files" yaml:"files"`
}

//
// ===============================
//   FILE VARIANTS (NORMAL/EXTENDED)
// ===============================
//

type EpisodeFileVariants struct {
	Normal   *EpisodeFile `json:"normal,omitempty" yaml:"normal,omitempty"`
	Extended *EpisodeFile `json:"extended,omitempty" yaml:"extended,omitempty"`
}

type EpisodeFile struct {
	Version string `json:"version" yaml:"version"` // "normal" | "extended" | etc

	CRC32         string `json:"crc32" yaml:"crc32"`
	Length        string `json:"length,omitempty" yaml:"length,omitempty"`
	LengthSeconds int    `json:"length_seconds,omitempty" yaml:"length_seconds,omitempty"`

	URL        string `json:"url,omitempty" yaml:"url,omitempty"`
	MagnetURI  string `json:"magnet_uri,omitempty" yaml:"magnet_uri,omitempty"`
	TorrentURL string `json:"torrent_url,omitempty" yaml:"torrent_url,omitempty"`

	// ReleaseInfoHash links back to the Release (by InfoHash) that supplied
	// this file's magnet/torrent links, when known.
	ReleaseInfoHash string `json:"release_info_hash,omitempty" yaml:"release_info_hash,omitempty"`
}

// CurrentEpisode is the "current" view of a single episode derived from the
// historical archive: only the file(s) marked IsCurrent, keyed by
// Episode.ID. Exists because the flat, CRC32-keyed archive alone can't
// answer "what's the current download link for this episode" without a
// consumer re-deriving it themselves.
type CurrentEpisode struct {
	ArcID     string `json:"arc_id,omitempty" yaml:"arc_id,omitempty"`
	EpisodeID string `json:"episode_id,omitempty" yaml:"episode_id,omitempty"`

	Arc         int    `json:"arc" yaml:"arc"`
	Episode     int    `json:"episode" yaml:"episode"`
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Chapters    string `json:"chapters" yaml:"chapters"`
	AnimeEps    string `json:"episodes" yaml:"episodes"`
	Released    string `json:"released" yaml:"released"`

	Files EpisodeFileVariants `json:"files" yaml:"files"`
}

type EpisodeMeta struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}

type EpisodeArchiveEntry struct {
	// ArcID/EpisodeID are the stable identifiers (see Arc.ID/Episode.ID)
	// this archive entry belongs to, so it can be joined back reliably
	// instead of only via the Arc/Episode numbers below.
	ArcID     string `json:"arc_id,omitempty" yaml:"arc_id,omitempty"`
	EpisodeID string `json:"episode_id,omitempty" yaml:"episode_id,omitempty"`

	Arc         int    `json:"arc" yaml:"arc"`
	Episode     int    `json:"episode" yaml:"episode"`
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Chapters    string `json:"chapters" yaml:"chapters"`
	AnimeEps    string `json:"episodes" yaml:"episodes"`
	Released    string `json:"released" yaml:"released"`

	// Only the single file variant for this CRC
	File EpisodeFile `json:"file" yaml:"file"`

	// IsCurrent is true when no other archive entry sharing this entry's
	// (ArcID, EpisodeID, File.Version) has a newer Released date.
	IsCurrent bool `json:"is_current" yaml:"is_current"`
}

//
// ===============================
//   RELEASES (onepace.net/en/releases feed)
// ===============================
//

type Release struct {
	Title         string   `json:"title" yaml:"title"`
	Variant       string   `json:"variant" yaml:"variant"` // "regular" | "extended"
	CRC32         string   `json:"crc32,omitempty" yaml:"crc32,omitempty"`
	PublishedAt   string   `json:"published_at" yaml:"published_at"` // RFC3339
	MangaChapters string   `json:"manga_chapters,omitempty" yaml:"manga_chapters,omitempty"`
	AnimeEpisodes string   `json:"anime_episodes,omitempty" yaml:"anime_episodes,omitempty"`
	Changelog     []string `json:"changelog,omitempty" yaml:"changelog,omitempty"`

	InfoHash   string `json:"info_hash" yaml:"info_hash"`
	NyaaURL    string `json:"nyaa_url,omitempty" yaml:"nyaa_url,omitempty"`
	TorrentURL string `json:"torrent_url,omitempty" yaml:"torrent_url,omitempty"`
	MagnetURI  string `json:"magnet_uri,omitempty" yaml:"magnet_uri,omitempty"`

	// NormalizedVariant re-expresses Variant in EpisodeFile.Version's
	// vocabulary ("normal"/"extended") so the two can be joined/compared
	// directly. See internal/parse.NormalizeVariant.
	NormalizedVariant string        `json:"normalized_variant,omitempty" yaml:"normalized_variant,omitempty"`
	MangaChapterRange *ChapterRange `json:"manga_chapter_range,omitempty" yaml:"manga_chapter_range,omitempty"`
	AnimeEpisodeRange *ChapterRange `json:"anime_episode_range,omitempty" yaml:"anime_episode_range,omitempty"`
}
