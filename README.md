# One Pace Metadata

The **One Pace Metadata** fetches, parses, normalizes, and exports metadata from the official **One Pace Episode Guide Google Sheets**.  
It automatically builds a complete structured dataset of arcs, episodes, and file versions.

All metadata is exported in both **JSON** and **YAML**, ready to be used by:
- Media servers (Jellyfin / Plex)
- Automation tools
- Episode indexing
- Archival purposes


---

## Acknowledgements

Thanks to `ladyisatis/one-pace-metadata` for the original implementation and inspiration. This repository is a complete rewrite focused on maintainability and new features; the original project made this work possible. Contributions are welcome — please fork the repo and open a pull request. This was an early Go project and may contain rough edges; issues, feedback, and fixes are appreciated.

## Features

### Arc List Parsing
- Fetches the main arc list from the Episode Guide
- Extracts:
  - Arc number
  - Arc title
  - Status (WIP / TBR)
  - Audio language list
  - Subtitle language list
  - Resolution
  - GID (sheet ID for episode list)
- Handles fractional arc numbers (e.g., `6.5 → 7`)
- Ensures arcs are ordered sequentially with unique IDs

### Episode Parsing (per arc)
- Loads each arc's sheet with **headless Chrome**
- Extracts:
  - Episode number
  - Title (temporary from sheet, replaced later)
  - Manga chapters
  - Anime episode references
  - Release date
  - Standard CRC32 version
  - Extended CRC32 version (if available)
  - File URLs (decoded from Google redirect links)
  - Length and extended length
- Supports **multiple files per episode** via `[]EpisodeFile`

### Episode Descriptions
- Fetches a CSV containing:
  - Saga title (Arc name)
  - Part number (Episode number)
  - English episode title
  - English episode description
- Injects the proper title + description into each episode after parsing

### ✔ Export System
Exports two datasets:

#### `/data/arcs.json` and `/data/arcs.yml`
Structured by arcs:
- Each arc contains:
  - Arc metadata
  - Sorted list of episodes
  - Each episode contains full file versions

#### `/data/episodes.json` and `/data/episodes.yml`
Indexed by CRC32:
- Each CRC32 key points to episode metadata
- Keeps all historical CRC32 entries
- Ensures old versions remain available even after One Pace updates files

---


## Requirements

- Go 1.21+
- chromedp
- Chrome or Chromium installed

---

## Running

```
go run main.go

or build:

go build -o metadata-service .
./metadata-service
```
---

## 📤 Output

The following files will be created in `/data`:

```
arcs.json
arcs.yml
episodes.json
episodes.yml
```

---

## 📄 License

GNU GPLv3 License
