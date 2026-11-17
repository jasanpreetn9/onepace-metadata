package parse

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// bstr bencodes a byte string.
func bstr(s string) string { return fmt.Sprintf("%d:%s", len(s), s) }

// fileEntry bencodes one entry of info.files.
func fileEntry(path ...string) string {
	out := "d" + bstr("length") + "i1e" + bstr("path") + "l"
	for _, p := range path {
		out += bstr(p)
	}
	return out + "ee"
}

// Shaped like the real 2026-10-26 Loguetown batch: a multi-file torrent
// whose own name has no CRC but whose files each do, plus a nested path, a
// non-episode file and a duplicate.
var batchTorrent = "d" + bstr("announce") + bstr("http://tracker/announce") +
	bstr("info") + "d" +
	bstr("files") + "l" +
	fileEntry("[One Pace][96-97] Loguetown 01 [1080p][ED297337].mkv") +
	fileEntry("sub", "[One Pace][98-99] Loguetown 02 [1080p][1ebfb55b].mkv") +
	fileEntry("notes.txt") +
	fileEntry("[One Pace][96-97] Loguetown 01 [1080p][ED297337].mkv") +
	"e" +
	bstr("name") + bstr("[One Pace][96-100] Loguetown [1080p]") +
	bstr("piece length") + "i16384e" +
	"ee"

func TestTorrentFileCRCs_Batch(t *testing.T) {
	got, err := TorrentFileCRCs([]byte(batchTorrent))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ED297337", "1EBFB55B"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TorrentFileCRCs = %v, want %v", got, want)
	}
}

func TestTorrentFileNames_SingleFile(t *testing.T) {
	single := "d" + bstr("info") + "d" + bstr("length") + "i1e" + bstr("name") + bstr("[One Pace][100] Loguetown 03 [1080p][BE8D8724].mkv") + "ee"
	got, err := TorrentFileNames([]byte(single))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || FilenameCRC(got[0]) != "BE8D8724" {
		t.Errorf("TorrentFileNames = %v, want one file with CRC BE8D8724", got)
	}
}

func TestTorrentFileNames_Malformed(t *testing.T) {
	for _, in := range []string{
		"",
		"d4:info",               // truncated
		"d4:infoi1ee",           // info not a dict
		"le",                    // top level not a dict
		"d4:infod4:name9:short", // string past end
		"d4:infod5:filesli-ee",  // bad integer
	} {
		if _, err := TorrentFileNames([]byte(in)); err == nil {
			t.Errorf("TorrentFileNames(%q) = nil error, want error", in)
		}
	}
}

func TestFilenameCRC(t *testing.T) {
	cases := map[string]string{
		"[One Pace][96-97] Loguetown 01 [1080p][ED297337].mkv": "ED297337",
		"[One Pace][96-100] Loguetown [1080p]":                 "",
		"Loguetown":                                            "",
		"[One Pace][1] Romance Dawn 01 [480p][abcdef12]":       "ABCDEF12",
	}
	for in, want := range cases {
		if got := FilenameCRC(in); got != want {
			t.Errorf("FilenameCRC(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestTorrentFileCRCs_RealFiles runs against real .torrent files when
// ONEPACE_TORRENT_DIR points at a directory of them; skipped otherwise.
func TestTorrentFileCRCs_RealFiles(t *testing.T) {
	dir := os.Getenv("ONEPACE_TORRENT_DIR")
	if dir == "" {
		t.Skip("ONEPACE_TORRENT_DIR not set")
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.torrent"))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		crcs, err := TorrentFileCRCs(b)
		if err != nil {
			t.Errorf("%s: %v", filepath.Base(p), err)
			continue
		}
		t.Logf("%s: %v", filepath.Base(p), crcs)
	}
}
