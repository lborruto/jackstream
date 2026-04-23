package filter

import (
	"testing"

	"github.com/lborruto/jackstream/internal/config"
	"github.com/lborruto/jackstream/internal/parser"
)

type tItem struct {
	T       string
	Parsed  parser.Parsed
	SizeVal int64
}

func (t tItem) Title() string              { return t.T }
func (t tItem) ParsedTitle() parser.Parsed { return t.Parsed }
func (t tItem) Size() int64                { return t.SizeVal }

func mk(title, quality string, sizeBytes int64) tItem {
	return tItem{T: title, Parsed: parser.Parsed{Quality: quality}, SizeVal: sizeBytes}
}

func titles(fs []Filterable) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Title()
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestEmptyConfigPassesEverything(t *testing.T) {
	in := []Filterable{mk("a", "1080p", 1<<30), mk("b", "720p", 500<<20), mk("c", "CAM", 1<<29)}
	got := Filter(in, config.Config{})
	if len(got) != 3 {
		t.Errorf("want 3, got %d", len(got))
	}
}

func TestMaxQualityDropsHigher(t *testing.T) {
	in := []Filterable{
		mk("4k", "4K", 1<<30),
		mk("hd", "1080p", 1<<30),
		mk("sd", "720p", 1<<30),
		mk("unk", "", 1<<30),
	}
	got := Filter(in, config.Config{MaxQuality: "720p"})
	want := []string{"sd", "unk"}
	if !eq(titles(got), want) {
		t.Errorf("max=720p want %v, got %v", want, titles(got))
	}
}

func TestMinQualityDropsLower(t *testing.T) {
	in := []Filterable{
		mk("4k", "4K", 1<<30),
		mk("hd", "1080p", 1<<30),
		mk("sd", "720p", 1<<30),
		mk("vhs", "480p", 1<<30),
		mk("unk", "", 1<<30),
	}
	got := Filter(in, config.Config{MinQuality: "720p"})
	want := []string{"4k", "hd", "sd"}
	if !eq(titles(got), want) {
		t.Errorf("min=720p want %v, got %v", want, titles(got))
	}
}

func TestMaxSizeGbDropsLarger(t *testing.T) {
	in := []Filterable{mk("ok", "1080p", 5<<30), mk("big", "1080p", 20<<30)}
	got := Filter(in, config.Config{MaxSizeGb: 10})
	if len(got) != 1 || got[0].Title() != "ok" {
		t.Errorf("want [ok], got %v", titles(got))
	}
}

func TestMinSizeMbDropsSmaller(t *testing.T) {
	in := []Filterable{mk("tiny", "1080p", 100<<20), mk("ok", "1080p", 2<<30)}
	got := Filter(in, config.Config{MinSizeMb: 500})
	if len(got) != 1 || got[0].Title() != "ok" {
		t.Errorf("want [ok], got %v", titles(got))
	}
}

func TestBlacklistWordBoundary(t *testing.T) {
	in := []Filterable{
		mk("Movie.2024.1080p.CAM.x264", "1080p", 1<<30),
		mk("Scam.Likely.2024.1080p.WEB-DL", "1080p", 1<<30),
	}
	got := Filter(in, config.Config{BlacklistKeywords: []string{"CAM"}})
	if len(got) != 1 || got[0].Title() != "Scam.Likely.2024.1080p.WEB-DL" {
		t.Errorf("blacklist dropped wrong item: %v", titles(got))
	}
}

func TestBlacklistMultipleKeywords(t *testing.T) {
	in := []Filterable{
		mk("Cam.2024", "CAM", 1<<30),
		mk("HDCAM.2024", "CAM", 1<<30),
		mk("Clean.2024", "1080p", 1<<30),
	}
	got := Filter(in, config.Config{BlacklistKeywords: []string{"CAM", "HDCAM"}})
	if len(got) != 1 || got[0].Title() != "Clean.2024" {
		t.Errorf("want [Clean.2024], got %v", titles(got))
	}
}

func TestBlacklistRegexInjectionEscaped(t *testing.T) {
	in := []Filterable{mk("Movie.2024.1080p", "1080p", 1<<30)}
	got := Filter(in, config.Config{BlacklistKeywords: []string{".*"}})
	if len(got) != 1 {
		t.Errorf("`.*` must be escaped, should not match everything")
	}
}

func TestBlacklistCaseInsensitive(t *testing.T) {
	in := []Filterable{mk("Movie.2024.HdCaM.x264", "", 1<<30)}
	got := Filter(in, config.Config{BlacklistKeywords: []string{"hdcam"}})
	if len(got) != 0 {
		t.Errorf("blacklist must be case-insensitive, got %v", titles(got))
	}
}
