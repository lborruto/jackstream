package server

import (
	"net/http"

	"github.com/lborruto/jackstream/internal/config"
)

func (s *Server) handleStreamFile(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("config")
	tid := r.PathValue("torrentID")
	c, err := config.Decode(raw)
	if err == nil {
		_, err = config.Validate(c)
	}
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid_config", "detail": err.Error()})
		return
	}
	if err := s.bt.StreamFile(w, r, tid); err != nil {
		_ = err
	}
}
