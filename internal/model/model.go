package model

//
// ===============================
//         ARC STRUCTS
// ===============================
//

// Arc represents a One Pace story arc.
type Arc struct {
	Arc         int    `json:"arc" yaml:"arc"`
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Poster      string `json:"poster,omitempty" yaml:"poster,omitempty"`

	AudioLanguages    string `json:"audio_languages" yaml:"audio_languages"`
	SubtitleLanguages string `json:"subtitle_languages" yaml:"subtitle_languages"`
	Resolution        string `json:"resolution" yaml:"resolution"`

	Status string `json:"status" yaml:"status"` // WIP, TBR, or empty

	// Episodes keyed by episode number (1, 2, 3…)
	Episodes []Episode `json:"episodes" yaml:"episodes"`

	// Internal use only — Google Sheet GID for fetching episode CSV
	GID string `json:"gid,omitempty" yaml:"gid,omitempty"`
}

//
// ===============================
//       EPISODE STRUCTS
// ===============================
//

// Episode represents a single One Pace episode's metadata.
type Episode struct {
	Arc     int `json:"arc" yaml:"arc"`
	Episode int `json:"episode" yaml:"episode"`

	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`

	Chapters string `json:"chapters" yaml:"chapters"`
	AnimeEps string `json:"episodes" yaml:"episodes"`

	Released string `json:"released" yaml:"released"`

	// List of file versions for this episode
	Files []EpisodeFile `json:"files" yaml:"files"`

	// Optional: reference pointer to another episode YAML
	Reference string `json:"reference,omitempty" yaml:"reference,omitempty"`
}

//
// ===============================
//   FILE VARIANT (CRC32 SETS)
// ===============================
//

// EpisodeFile represents one video release of an episode:
// - Original version
// - Extended version
// - Newer re-release
type EpisodeFile struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	CRC32   string `json:"crc32" yaml:"crc32"`

	Length string `json:"length,omitempty" yaml:"length,omitempty"`

	URL string `json:"url,omitempty" yaml:"url,omitempty"`
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
