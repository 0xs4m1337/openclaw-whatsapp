package api

import (
	"net/http"

	"github.com/openclaw/whatsapp/store"
)

type contact struct {
	JID  string `json:"jid"`
	Name string `json:"name"`
}

func (s *Server) handleGetChats(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)

	chats, err := s.Store.GetChats(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chats == nil {
		chats = []store.Chat{}
	}

	writeJSON(w, http.StatusOK, chats)
}

func (s *Server) handleGetContacts(w http.ResponseWriter, r *http.Request) {
	wc := s.Client.GetClient()
	if wc == nil {
		writeError(w, http.StatusServiceUnavailable, "client not connected")
		return
	}

	contactStore := wc.Store.Contacts
	if contactStore == nil {
		writeJSON(w, http.StatusOK, []contact{})
		return
	}

	contacts, err := contactStore.GetAllContacts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]contact, 0, len(contacts))
	for jid, info := range contacts {
		name := info.PushName
		if name == "" {
			name = info.FullName
		}
		if name == "" {
			name = info.BusinessName
		}
		result = append(result, contact{
			JID:  jid.String(),
			Name: name,
		})
	}

	writeJSON(w, http.StatusOK, result)
}
