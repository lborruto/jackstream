package sorter

import (
	"reflect"
	"testing"

	"github.com/lborruto/jackstream/internal/parser"
)

type torrent struct {
	ID      string
	Parsed  parser.Parsed
	Seeders int
}

func (t torrent) ParsedTitle() parser.Parsed { return t.Parsed }
func (t torrent) SeederCount() int           { return t.Seeders }

func mk(id, quality, source, hdr, audio string, seeders int) torrent {
	return torrent{
		ID: id,
		Parsed: parser.Parsed{
			Quality: quality,
			Source:  source,
			HDR:     hdr,
			Codec:   "x264",
			Audio:   audio,
		},
		Seeders: seeders,
	}
}

func ids(ts []Sortable) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.(torrent).ID
	}
	return out
}

func TestQualityDominates(t *testing.T) {
	a := mk("a", "1080p", "WEB-DL", "", "", 5)
	b := mk("b", "4K", "WEB-DL", "", "", 1)
	out := Sort([]Sortable{a, b}, "")
	if out[0].(torrent).ID != "b" {
		t.Errorf("4K must rank above 1080p, got order %v", ids(out))
	}
}

func TestSourceBreaksTie(t *testing.T) {
	a := mk("a", "1080p", "WEBRip", "", "", 100)
	b := mk("b", "1080p", "BluRay", "", "", 1)
	if Sort([]Sortable{a, b}, "")[0].(torrent).ID != "b" {
		t.Error("BluRay must outrank WEBRip at same quality")
	}
}

func TestHDRBreaksTie(t *testing.T) {
	a := mk("a", "4K", "BluRay", "", "", 100)
	b := mk("b", "4K", "BluRay", "DV", "", 1)
	if Sort([]Sortable{a, b}, "")[0].(torrent).ID != "b" {
		t.Error("DV must outrank no-HDR at same quality+source")
	}
}

func TestSeedersBreakTie(t *testing.T) {
	a := mk("a", "1080p", "WEB-DL", "", "", 5)
	b := mk("b", "1080p", "WEB-DL", "", "", 50)
	if Sort([]Sortable{a, b}, "")[0].(torrent).ID != "b" {
		t.Error("higher seeders must win at equal quality/source/hdr")
	}
}

func TestCAMAlwaysLast(t *testing.T) {
	cam := mk("cam", "CAM", "", "", "", 10000)
	low := mk("low", "480p", "", "", "", 1)
	if Sort([]Sortable{cam, low}, "")[0].(torrent).ID != "low" {
		t.Error("CAM must sort below 480p even with more seeders")
	}
}

func TestUnknownBetweenKnownAndCAM(t *testing.T) {
	hd := mk("hd", "720p", "", "", "", 1)
	unk := mk("unk", "", "", "", "", 1000)
	cam := mk("cam", "CAM", "", "", "", 1000)
	out := Sort([]Sortable{unk, cam, hd}, "")
	got := ids(out)
	if !reflect.DeepEqual(got, []string{"hd", "unk", "cam"}) {
		t.Errorf("want [hd unk cam], got %v", got)
	}
}

func TestStableSort(t *testing.T) {
	a := mk("a", "1080p", "WEB-DL", "", "", 10)
	b := mk("b", "1080p", "WEB-DL", "", "", 10)
	c := mk("c", "1080p", "WEB-DL", "", "", 10)
	got := ids(Sort([]Sortable{a, b, c}, ""))
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("stable sort broken: %v", got)
	}
}

func TestDoesNotMutateInput(t *testing.T) {
	a := mk("a", "1080p", "", "", "", 10)
	b := mk("b", "4K", "", "", "", 10)
	in := []Sortable{a, b}
	_ = Sort(in, "")
	if in[0].(torrent).ID != "a" {
		t.Errorf("Sort mutated input")
	}
}

func TestLanguageBoostPromotesFrench(t *testing.T) {
	eng := mk("eng", "1080p", "", "", "ENG", 100)
	fr := mk("fr", "1080p", "", "", "FRENCH", 1)
	if Sort([]Sortable{eng, fr}, "FRENCH")[0].(torrent).ID != "fr" {
		t.Error("FRENCH preference must promote FRENCH over ENG at same quality")
	}
}

func TestLanguageBoostIncludesMulti(t *testing.T) {
	eng := mk("eng", "1080p", "", "", "ENG", 100)
	multi := mk("multi", "1080p", "", "", "MULTI", 1)
	if Sort([]Sortable{eng, multi}, "FRENCH")[0].(torrent).ID != "multi" {
		t.Error("FRENCH preference must also boost MULTI")
	}
}

func TestLanguageDoesNotOverrideQuality(t *testing.T) {
	engHigh := mk("engHigh", "4K", "", "", "ENG", 10)
	frLow := mk("frLow", "1080p", "", "", "FRENCH", 10)
	if Sort([]Sortable{engHigh, frLow}, "FRENCH")[0].(torrent).ID != "engHigh" {
		t.Error("4K ENG must still beat 1080p FRENCH")
	}
}
