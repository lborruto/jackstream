const QUALITY_RULES = [
  [/\b(2160p|4k|uhd)\b/i, '4K'],
  [/\b1080p\b/i, '1080p'],
  [/\b720p\b/i, '720p'],
  [/\b480p\b/i, '480p'],
  [/\bcam\b/i, 'CAM'],
]

const SOURCE_RULES = [
  [/\bremux\b/i, 'Remux'],
  [/\b(bluray|bdrip|brrip)\b/i, 'BluRay'],
  [/\bweb[-.\s]?dl\b/i, 'WEB-DL'],
  [/\bwebrip\b/i, 'WEBRip'],
  [/\bhdtv\b/i, 'HDTV'],
  [/\bcam\b/i, 'CAM'],
]

const HDR_RULES = [
  [/\b(dv|dolby[.\s]?vision)\b/i, 'DV'],
  [/\bhdr10\+?\b/i, 'HDR10'],
  [/\bhdr\b/i, 'HDR'],
]

const CODEC_RULES = [
  [/\b(x265|hevc|h[.\s]?265)\b/i, 'x265'],
  [/\b(x264|h[.\s]?264)\b/i, 'x264'],
  [/\bav1\b/i, 'AV1'],
]

const AUDIO_RULES = [
  [/\bmulti\b/i, 'MULTI'],
  [/\b(truefrench|french|vff|vfq)\b/i, 'FRENCH'],
  [/\bvostfr\b/i, 'VOSTFR'],
  [/\b(english|eng)\b/i, 'ENG'],
]

function firstMatch(rules, title) {
  for (const [re, label] of rules) {
    if (re.test(title)) return label
  }
  return null
}

export function parseTorrentTitle(title) {
  const t = title || ''
  return {
    quality: firstMatch(QUALITY_RULES, t),
    source: firstMatch(SOURCE_RULES, t),
    hdr: firstMatch(HDR_RULES, t),
    codec: firstMatch(CODEC_RULES, t),
    audio: firstMatch(AUDIO_RULES, t),
  }
}
