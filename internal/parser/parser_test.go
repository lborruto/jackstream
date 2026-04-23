package parser

import "testing"

func TestQuality(t *testing.T) {
	cases := []struct {
		title, want string
	}{
		{"Movie.2024.2160p.BluRay.x265", "4K"},
		{"Movie.2024.4K.HDR", "4K"},
		{"Movie.2024.1080p.WEB-DL", "1080p"},
		{"Movie.2024.720p.HDTV", "720p"},
		{"Movie.2024.480p.DVDRip", "480p"},
		{"Movie.2024.CAM.NEW", "CAM"},
		{"Movie 2024", ""},
	}
	for _, c := range cases {
		if got := Parse(c.title).Quality; got != c.want {
			t.Errorf("%s → quality=%q want %q", c.title, got, c.want)
		}
	}
}

func TestSource(t *testing.T) {
	cases := []struct{ title, want string }{
		{"Movie.2024.1080p.BluRay.REMUX.HEVC", "Remux"},
		{"Movie.2024.1080p.BluRay.x264", "BluRay"},
		{"Movie.2024.1080p.BDRip.x264", "BluRay"},
		{"Movie.2024.1080p.WEB-DL.x264", "WEB-DL"},
		{"Movie.2024.1080p.WEB.DL.x264", "WEB-DL"},
		{"Movie.2024.1080p.WEBRip.x264", "WEBRip"},
		{"Movie.2024.720p.HDTV.x264", "HDTV"},
		{"Movie.2024.CAM", "CAM"},
		{"Movie.2024", ""},
	}
	for _, c := range cases {
		if got := Parse(c.title).Source; got != c.want {
			t.Errorf("%s → source=%q want %q", c.title, got, c.want)
		}
	}
}

func TestHDR(t *testing.T) {
	cases := []struct{ title, want string }{
		{"Movie.2024.2160p.DV.HDR10.HEVC", "DV"},
		{"Movie.2024.2160p.Dolby.Vision.HEVC", "DV"},
		{"Movie.2024.2160p.HDR10.HEVC", "HDR10"},
		{"Movie.2024.2160p.HDR10+.HEVC", "HDR10"},
		{"Movie.2024.2160p.HDR.HEVC", "HDR"},
		{"Movie.2024.1080p.SDR", ""},
	}
	for _, c := range cases {
		if got := Parse(c.title).HDR; got != c.want {
			t.Errorf("%s → hdr=%q want %q", c.title, got, c.want)
		}
	}
}

func TestCodec(t *testing.T) {
	cases := []struct{ title, want string }{
		{"Movie.2024.1080p.x265", "x265"},
		{"Movie.2024.1080p.HEVC", "x265"},
		{"Movie.2024.1080p.H.265", "x265"},
		{"Movie.2024.1080p.x264", "x264"},
		{"Movie.2024.1080p.H264", "x264"},
		{"Movie.2024.1080p.AV1", "AV1"},
		{"Movie.2024", ""},
	}
	for _, c := range cases {
		if got := Parse(c.title).Codec; got != c.want {
			t.Errorf("%s → codec=%q want %q", c.title, got, c.want)
		}
	}
}

func TestAudioPriority(t *testing.T) {
	cases := []struct{ title, want string }{
		{"Movie.2024.MULTI.FRENCH", "MULTI"},
		{"Movie.2024.FRENCH.VOSTFR", "FRENCH"},
		{"Movie.2024.TRUEFRENCH", "FRENCH"},
		{"Movie.2024.VOSTFR.ENG", "VOSTFR"},
		{"Movie.2024.ENGLISH", "ENG"},
		{"Movie.2024", ""},
	}
	for _, c := range cases {
		if got := Parse(c.title).Audio; got != c.want {
			t.Errorf("%s → audio=%q want %q", c.title, got, c.want)
		}
	}
}

func TestComposite(t *testing.T) {
	got := Parse("The.Substance.2024.2160p.UHD.BluRay.REMUX.HDR10.HEVC.MULTI-GROUP")
	want := Parsed{
		Quality: "4K",
		Source:  "Remux",
		HDR:     "HDR10",
		Codec:   "x265",
		Audio:   "MULTI",
	}
	if got != want {
		t.Errorf("composite: got %#v, want %#v", got, want)
	}
}
