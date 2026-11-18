package parse

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// filenameCRCRe matches the trailing CRC32 bracket of a One Pace release
// filename, e.g. "[One Pace][96-97] Loguetown 01 [1080p][ED297337].mkv".
var filenameCRCRe = regexp.MustCompile(`\[([0-9A-Fa-f]{8})\](?:\.\w+)?$`)

// FilenameCRC returns the upper-cased CRC32 embedded at the end of a One
// Pace filename, or "" if there isn't one (e.g. a batch folder name).
func FilenameCRC(name string) string {
	m := filenameCRCRe.FindStringSubmatch(strings.TrimSpace(name))
	if len(m) < 2 {
		return ""
	}
	return strings.ToUpper(m[1])
}

// TorrentFileCRCs returns the CRC32s of every file in a .torrent's metainfo
// that carries one in its name, in file order without duplicates. Batch
// releases (a whole arc in one torrent) have no CRC in the release name,
// but each episode file inside does, which is what lets them be matched to
// episodes.
func TorrentFileCRCs(metainfo []byte) ([]string, error) {
	names, err := TorrentFileNames(metainfo)
	if err != nil {
		return nil, err
	}
	return FileNameCRCs(names), nil
}

// FileNameCRCs returns the CRC32 of each filename that carries one, in
// order and without duplicates.
func FileNameCRCs(names []string) []string {
	var crcs []string
	seen := make(map[string]bool)
	for _, n := range names {
		if crc := FilenameCRC(n); crc != "" && !seen[crc] {
			seen[crc] = true
			crcs = append(crcs, crc)
		}
	}
	return crcs
}

// TorrentFileNames returns the base name of every file in a .torrent's
// metainfo: info.files[].path's last element for multi-file torrents, or
// info.name for single-file ones.
func TorrentFileNames(metainfo []byte) ([]string, error) {
	d := &bdecoder{b: metainfo}
	root, err := d.value()
	if err != nil {
		return nil, fmt.Errorf("decode torrent: %w", err)
	}
	top, ok := root.(map[string]any)
	if !ok {
		return nil, errors.New("decode torrent: top level is not a dictionary")
	}
	info, ok := top["info"].(map[string]any)
	if !ok {
		return nil, errors.New("decode torrent: missing info dictionary")
	}

	files, ok := info["files"].([]any)
	if !ok {
		name, ok := info["name"].(string)
		if !ok {
			return nil, errors.New("decode torrent: info has neither files nor name")
		}
		return []string{name}, nil
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		fd, ok := f.(map[string]any)
		if !ok {
			continue
		}
		path, ok := fd["path"].([]any)
		if !ok || len(path) == 0 {
			continue
		}
		if last, ok := path[len(path)-1].(string); ok {
			names = append(names, last)
		}
	}
	return names, nil
}

// bdecoder is a minimal bencode decoder: just enough to walk a torrent's
// metainfo. Byte strings decode to string, integers to int64, lists to
// []any and dictionaries to map[string]any.
type bdecoder struct {
	b []byte
	i int
}

const maxBencodeDepth = 64

func (d *bdecoder) value() (any, error) { return d.valueAt(0) }

func (d *bdecoder) valueAt(depth int) (any, error) {
	if depth > maxBencodeDepth {
		return nil, errors.New("bencode: nesting too deep")
	}
	if d.i >= len(d.b) {
		return nil, errors.New("bencode: unexpected end of input")
	}
	switch c := d.b[d.i]; {
	case c == 'i':
		end := d.indexFrom('e')
		if end < 0 {
			return nil, errors.New("bencode: unterminated integer")
		}
		n, err := strconv.ParseInt(string(d.b[d.i+1:end]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bencode: bad integer: %w", err)
		}
		d.i = end + 1
		return n, nil
	case c == 'l':
		d.i++
		var list []any
		for {
			if d.i >= len(d.b) {
				return nil, errors.New("bencode: unterminated list")
			}
			if d.b[d.i] == 'e' {
				d.i++
				return list, nil
			}
			v, err := d.valueAt(depth + 1)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
		}
	case c == 'd':
		d.i++
		dict := make(map[string]any)
		for {
			if d.i >= len(d.b) {
				return nil, errors.New("bencode: unterminated dictionary")
			}
			if d.b[d.i] == 'e' {
				d.i++
				return dict, nil
			}
			k, err := d.str()
			if err != nil {
				return nil, err
			}
			v, err := d.valueAt(depth + 1)
			if err != nil {
				return nil, err
			}
			dict[k] = v
		}
	case c >= '0' && c <= '9':
		return d.str()
	default:
		return nil, fmt.Errorf("bencode: unexpected byte %q at %d", c, d.i)
	}
}

func (d *bdecoder) str() (string, error) {
	colon := d.indexFrom(':')
	if colon < 0 {
		return "", errors.New("bencode: unterminated string length")
	}
	n, err := strconv.Atoi(string(d.b[d.i:colon]))
	if err != nil || n < 0 {
		return "", errors.New("bencode: bad string length")
	}
	start := colon + 1
	if start+n > len(d.b) {
		return "", errors.New("bencode: string runs past end of input")
	}
	d.i = start + n
	return string(d.b[start:d.i]), nil
}

func (d *bdecoder) indexFrom(c byte) int {
	for j := d.i; j < len(d.b); j++ {
		if d.b[j] == c {
			return j
		}
	}
	return -1
}
