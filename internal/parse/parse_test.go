package parse

import (
	"testing"

	"metadata-service/internal/model"
)

func TestChapterRange(t *testing.T) {
	cases := []struct {
		in   string
		want *model.ChapterRange
	}{
		{"1 - 7", &model.ChapterRange{Start: 1, End: 7}},
		{"129-132", &model.ChapterRange{Start: 129, End: 132}},
		{"8-21", &model.ChapterRange{Start: 8, End: 21}},
		{"Ch. 353-355", &model.ChapterRange{Start: 353, End: 355}},
		{"Ep. 248-249", &model.ChapterRange{Start: 248, End: 249}},
		{"1 - 4, 19", nil},
		{"42,22", nil},
		{"Episode of East Blue, Ep. 312 (Intro)", nil},
		{"", nil},
		{"Ch. 1", nil},
	}

	for _, c := range cases {
		got := ChapterRange(c.in)
		if c.want == nil {
			if got != nil {
				t.Errorf("ChapterRange(%q) = %+v, want nil", c.in, got)
			}
			continue
		}
		if got == nil || *got != *c.want {
			t.Errorf("ChapterRange(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestLengthSeconds(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"17:57", 1077},
		{"27:03", 1623},
		{"1:23:45", 5025},
		{"", 0},
		{"garbage", 0},
	}

	for _, c := range cases {
		if got := LengthSeconds(c.in); got != c.want {
			t.Errorf("LengthSeconds(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestNormalizeVariant(t *testing.T) {
	cases := []struct{ in, want string }{
		{"regular", "normal"},
		{"extended", "extended"},
		{"normal", "normal"},
		{"Regular", "normal"},
		{"", ""},
		{"weird", "weird"},
	}

	for _, c := range cases {
		if got := NormalizeVariant(c.in); got != c.want {
			t.Errorf("NormalizeVariant(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPercent(t *testing.T) {
	if got := Percent("27.00%"); got == nil || *got != 27.0 {
		t.Errorf("Percent(27.00%%) = %v, want 27.0", got)
	}
	if got := Percent(""); got != nil {
		t.Errorf("Percent(\"\") = %v, want nil", got)
	}
	if got := Percent("garbage"); got != nil {
		t.Errorf("Percent(garbage) = %v, want nil", got)
	}
}

func TestIntVal(t *testing.T) {
	if got := IntVal("28"); got == nil || *got != 28 {
		t.Errorf("IntVal(28) = %v, want 28", got)
	}
	if got := IntVal(""); got != nil {
		t.Errorf("IntVal(\"\") = %v, want nil", got)
	}
	if got := IntVal("garbage"); got != nil {
		t.Errorf("IntVal(garbage) = %v, want nil", got)
	}
}
