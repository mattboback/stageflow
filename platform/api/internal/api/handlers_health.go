package api

import (
	"net/http"

	"github.com/mattboback/stageflow/packages/shared-go/httputil"
)

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httputil.RespondOK(w, map[string]string{
		"status": "healthy",
	})
}
