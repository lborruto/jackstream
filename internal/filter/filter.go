package filter

import (
	"regexp"
	"strings"

	"github.com/lborruto/jackstream/internal/config"
	"github.com/lborruto/jackstream/internal/parser"
)

type Filterable interface {
	Title() string
	ParsedTitle() parser.Parsed
	Size() int64
}

// Quality rank: CAM=0, unknown=1, 480p=2, 720p=3, 1080p=4, 4K=5.
var qualityRank = map[string]int{
	"CAM": 0, "": 1, "480p": 2, "720p": 3, "1080p": 4, "4K": 5,
}

func rankOf(q string) int {
	if v, ok := qualityRank[q]; ok {
		return v
	}
	return 1 // unknown label treated as null
}

func buildBlacklistRegex(kw []string) *regexp.Regexp {
	if len(kw) == 0 {
		return nil
	}
	parts := make([]string, 0, len(kw))
	for _, k := range kw {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		parts = append(parts, regexp.QuoteMeta(k))
	}
	if len(parts) == 0 {
		return nil
	}
	return regexp.MustCompile(`(?i)\b(?:` + strings.Join(parts, "|") + `)\b`)
}

func Filter(items []Filterable, c config.Config) []Filterable {
	maxQ, minQ := c.MaxQuality, c.MinQuality
	var maxBytes, minBytes int64
	if c.MaxSizeGb > 0 {
		maxBytes = int64(c.MaxSizeGb * float64(1<<30))
	}
	if c.MinSizeMb > 0 {
		minBytes = int64(c.MinSizeMb * float64(1<<20))
	}
	blRe := buildBlacklistRegex(c.BlacklistKeywords)

	out := make([]Filterable, 0, len(items))
	for _, it := range items {
		q := it.ParsedTitle().Quality
		if maxQ != "" && rankOf(q) > rankOf(maxQ) {
			continue
		}
		if minQ != "" && rankOf(q) < rankOf(minQ) {
			continue
		}
		size := it.Size()
		if maxBytes > 0 && size > maxBytes {
			continue
		}
		if minBytes > 0 && size < minBytes {
			continue
		}
		if blRe != nil && blRe.MatchString(it.Title()) {
			continue
		}
		out = append(out, it)
	}
	return out
}
