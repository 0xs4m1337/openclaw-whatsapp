package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/openclaw/whatsapp/store"
)

type sendTextRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

func (s *Server) handleSendText(w http.ResponseWriter, r *http.Request) {
	var req sendTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.To == "" || req.Message == "" {
		writeError(w, http.StatusBadRequest, "to and message are required")
		return
	}

	if err := s.Client.SendText(r.Context(), req.To, req.Message); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) handleSendFile(w http.ResponseWriter, r *http.Request) {
	// 50 MB max
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	to := r.FormValue("to")
	caption := r.FormValue("caption")
	if to == "" {
		writeError(w, http.StatusBadRequest, "to is required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	mimetype := http.DetectContentType(data)
	filename := header.Filename

	if err := s.Client.SendFile(r.Context(), to, data, mimetype, filename, caption); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	chatJID := r.URL.Query().Get("chat")
	if chatJID == "" {
		writeError(w, http.StatusBadRequest, "chat query parameter is required")
		return
	}

	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	msgs, err := s.Store.GetMessages(chatJID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if msgs == nil {
		msgs = []store.Message{}
	}

	writeJSON(w, http.StatusOK, msgs)
}

func (s *Server) handleSearchMessages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q query parameter is required")
		return
	}

	limit := queryInt(r, "limit", 20)

	msgs, err := s.Store.SearchMessages(q, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if msgs == nil {
		msgs = []store.Message{}
	}

	writeJSON(w, http.StatusOK, msgs)
}

func (s *Server) handleGetChatMessages(w http.ResponseWriter, r *http.Request) {
	jid := chi.URLParam(r, "jid")
	if jid == "" {
		writeError(w, http.StatusBadRequest, "jid path parameter is required")
		return
	}

	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	msgs, err := s.Store.GetMessages(jid, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if msgs == nil {
		msgs = []store.Message{}
	}

	writeJSON(w, http.StatusOK, msgs)
}

type replyRequest struct {
	To             string `json:"to"`
	Message        string `json:"message"`
	QuoteMessageID string `json:"quote_message_id,omitempty"`
}

func (s *Server) handleReply(w http.ResponseWriter, r *http.Request) {
	var req replyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.To == "" || req.Message == "" {
		writeError(w, http.StatusBadRequest, "to and message are required")
		return
	}

	if err := s.Client.SendText(r.Context(), req.To, req.Message); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}
