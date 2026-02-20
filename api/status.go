package api

import (
	"net/http"
	"time"
)

type statusResponse struct {
	Status  string `json:"status"`
	Phone   string `json:"phone,omitempty"`
	Uptime  string `json:"uptime"`
	Version string `json:"version"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := string(s.Client.GetStatus())
	phone := s.Client.GetJID()
	uptime := time.Since(s.Client.GetStartTime()).Truncate(time.Second).String()

	writeJSON(w, http.StatusOK, statusResponse{
		Status:  status,
		Phone:   phone,
		Uptime:  uptime,
		Version: s.Version,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.Client.Logout(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}
