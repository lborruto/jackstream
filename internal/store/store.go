package store

import (
	"time"

	"github.com/lborruto/jackstream/internal/cache"
	"github.com/lborruto/jackstream/internal/parser"
)

type TorrentMeta struct {
	TorrentURL  string
	MagnetURI   string
	Size        int64
	Seeders     int
	InfoHash    string
	ParsedTitle parser.Parsed
}

const TTL = 2 * time.Hour

var store = cache.New[TorrentMeta](time.Now)

func Set(torrentID string, m TorrentMeta) {
	store.Set(torrentID, m, TTL)
}

func Get(torrentID string) (TorrentMeta, bool) {
	return store.Get(torrentID)
}

func Clear() { store.Clear() }
