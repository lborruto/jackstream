package parser

import "regexp"

type Parsed struct {
	Quality string
	Source  string
	HDR     string
	Codec   string
	Audio   string
}

type rule struct {
	re    *regexp.Regexp
	label string
}

var qualityRules = []rule{
	{regexp.MustCompile(`(?i)\b(2160p|4k|uhd)\b`), "4K"},
	{regexp.MustCompile(`(?i)\b1080p\b`), "1080p"},
	{regexp.MustCompile(`(?i)\b720p\b`), "720p"},
	{regexp.MustCompile(`(?i)\b480p\b`), "480p"},
	{regexp.MustCompile(`(?i)\bcam\b`), "CAM"},
}

var sourceRules = []rule{
	{regexp.MustCompile(`(?i)\bremux\b`), "Remux"},
	{regexp.MustCompile(`(?i)\b(bluray|bdrip|brrip)\b`), "BluRay"},
	{regexp.MustCompile(`(?i)\bweb[-.\s]?dl\b`), "WEB-DL"},
	{regexp.MustCompile(`(?i)\bwebrip\b`), "WEBRip"},
	{regexp.MustCompile(`(?i)\bhdtv\b`), "HDTV"},
	{regexp.MustCompile(`(?i)\bcam\b`), "CAM"},
}

var hdrRules = []rule{
	{regexp.MustCompile(`(?i)\b(dv|dolby[.\s]?vision)\b`), "DV"},
	{regexp.MustCompile(`(?i)\bhdr10\+?\b`), "HDR10"},
	{regexp.MustCompile(`(?i)\bhdr\b`), "HDR"},
}

var codecRules = []rule{
	{regexp.MustCompile(`(?i)\b(x265|hevc|h[.\s]?265)\b`), "x265"},
	{regexp.MustCompile(`(?i)\b(x264|h[.\s]?264)\b`), "x264"},
	{regexp.MustCompile(`(?i)\bav1\b`), "AV1"},
}

var audioRules = []rule{
	{regexp.MustCompile(`(?i)\bmulti\b`), "MULTI"},
	{regexp.MustCompile(`(?i)\b(truefrench|french|vff|vfq)\b`), "FRENCH"},
	{regexp.MustCompile(`(?i)\bvostfr\b`), "VOSTFR"},
	{regexp.MustCompile(`(?i)\b(english|eng)\b`), "ENG"},
}

func firstMatch(rules []rule, title string) string {
	for _, r := range rules {
		if r.re.MatchString(title) {
			return r.label
		}
	}
	return ""
}

func Parse(title string) Parsed {
	return Parsed{
		Quality: firstMatch(qualityRules, title),
		Source:  firstMatch(sourceRules, title),
		HDR:     firstMatch(hdrRules, title),
		Codec:   firstMatch(codecRules, title),
		Audio:   firstMatch(audioRules, title),
	}
}
