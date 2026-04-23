package sorter

import (
	"sort"

	"github.com/lborruto/jackstream/internal/parser"
)

// Sortable is the minimal surface the sorter needs from a torrent-like value:
// a parsed title (quality/source/hdr/audio) and a seeder count.
type Sortable interface {
	ParsedTitle() parser.Parsed
	SeederCount() int
}

var qualityScore = map[string]int{
	"4K": 4, "1080p": 3, "720p": 2, "480p": 1,
	"CAM": -1,
}

var sourceScore = map[string]int{
	"Remux": 5, "BluRay": 4, "WEB-DL": 3, "WEBRip": 2, "HDTV": 1,
	"CAM": -1,
}

var hdrScore = map[string]int{
	"DV": 3, "HDR10": 2, "HDR": 1,
}

// langIncludes maps a preferred-language key to the set of audio labels that
// should receive the language boost. MULTI is considered a match for most
// specific languages because multi-track releases include the preferred track.
var langIncludes = map[string]map[string]struct{}{
	"FRENCH": {"FRENCH": {}, "MULTI": {}},
	"ENG":    {"ENG": {}, "MULTI": {}},
	"VOSTFR": {"VOSTFR": {}},
	"MULTI":  {"MULTI": {}},
}

func score(m map[string]int, key string) int {
	if v, ok := m[key]; ok {
		return v
	}
	return 0
}

func languageBoost(audio, preferred string) int {
	if preferred == "" {
		return 0
	}
	set, ok := langIncludes[preferred]
	if !ok {
		return 0
	}
	if _, hit := set[audio]; hit {
		return 1
	}
	return 0
}

type key struct {
	q, s, h, boost, seed, i int
}

type entry struct {
	v Sortable
	k key
}

// Sort returns a new slice ordered by a cascade of keys:
// quality > source > hdr > language boost > seeders, with a stable tie-break
// on original input position. The input slice is not mutated.
func Sort(ts []Sortable, preferredLanguage string) []Sortable {
	out := make([]Sortable, len(ts))
	copy(out, ts)

	entries := make([]entry, len(out))
	for i, t := range out {
		p := t.ParsedTitle()
		entries[i] = entry{
			v: t,
			k: key{
				q:     score(qualityScore, p.Quality),
				s:     score(sourceScore, p.Source),
				h:     score(hdrScore, p.HDR),
				boost: languageBoost(p.Audio, preferredLanguage),
				seed:  t.SeederCount(),
				i:     i,
			},
		}
	}

	sort.SliceStable(entries, func(a, b int) bool {
		ka, kb := entries[a].k, entries[b].k
		if ka.q != kb.q {
			return ka.q > kb.q
		}
		if ka.s != kb.s {
			return ka.s > kb.s
		}
		if ka.h != kb.h {
			return ka.h > kb.h
		}
		if ka.boost != kb.boost {
			return ka.boost > kb.boost
		}
		if ka.seed != kb.seed {
			return ka.seed > kb.seed
		}
		return ka.i < kb.i
	})

	for i := range entries {
		out[i] = entries[i].v
	}
	return out
}
