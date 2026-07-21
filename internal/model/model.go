package model

//
// ===============================
//         ARC STRUCTS
// ===============================
//

type Arc struct {
	Arc         int    `json:"arc" yaml:"arc"`
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Poster      string `json:"poster,omitempty" yaml:"poster,omitempty"`

	AudioLanguages    string `json:"audio_languages" yaml:"audio_languages"`
	SubtitleLanguages string `json:"subtitle_languages" yaml:"subtitle_languages"`
	Resolution        string `json:"resolution" yaml:"resolution"`

	MangaChapters    string `json:"manga_chapters" yaml:"manga_chapters"`
	NumberOfChapters string `json:"number_of_chapters" yaml:"number_of_chapters"`
	AnimeEpisodes    string `json:"anime_episodes" yaml:"anime_episodes"`
	EpisodesAdapted  string `json:"episodes_adapted" yaml:"episodes_adapted"`
	FillerEpisodes   string `json:"filler_episodes" yaml:"filler_episodes"`
	TimeSavedMins    string `json:"time_saved_mins" yaml:"time_saved_mins"`
	TimeSavedPercent string `json:"time_saved_percent" yaml:"time_saved_percent"`

	Status string `json:"status" yaml:"status"` // WIP / TBR / ""

	Episodes []Episode `json:"episodes" yaml:"episodes"`

	GID string `json:"gid,omitempty" yaml:"gid,omitempty"`
}

//
// ===============================
//       EPISODE STRUCTS
// ===============================
//

type Episode struct {
	Arc     int `json:"arc" yaml:"arc"`
	Episode int `json:"episode" yaml:"episode"`

	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`

	Chapters string `json:"chapters" yaml:"chapters"`
	AnimeEps string `json:"episodes" yaml:"episodes"`

	Released string `json:"released" yaml:"released"`

	HasExtended bool                `json:"has_extended" yaml:"has_extended"`
	Files       EpisodeFileVariants `json:"files" yaml:"files"`

	Reference string `json:"reference,omitempty" yaml:"reference,omitempty"`
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

	CRC32  string `json:"crc32" yaml:"crc32"`
	Length string `json:"length,omitempty" yaml:"length,omitempty"`

	URL        string `json:"url,omitempty" yaml:"url,omitempty"`
	MagnetURI  string `json:"magnet_uri,omitempty" yaml:"magnet_uri,omitempty"`
	TorrentURL string `json:"torrent_url,omitempty" yaml:"torrent_url,omitempty"`
}

type EpisodeMeta struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}

type EpisodeArchiveEntry struct {
	Arc         int    `json:"arc" yaml:"arc"`
	Episode     int    `json:"episode" yaml:"episode"`
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Chapters    string `json:"chapters" yaml:"chapters"`
	AnimeEps    string `json:"episodes" yaml:"episodes"`
	Released    string `json:"released" yaml:"released"`

	// Only the single file variant for this CRC
	File EpisodeFile `json:"file" yaml:"file"`
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
}
