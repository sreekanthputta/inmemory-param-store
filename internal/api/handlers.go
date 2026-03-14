// Package api provides HTTP handlers for the parameter store.
package api

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"parameter-store/internal/models"
	"parameter-store/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

// getClientIP extracts the client IP for audit logging.
// Checks proxy headers first (X-Forwarded-For, X-Real-IP) before falling back to RemoteAddr.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs; first one is the original client
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Update handles POST /api/update
// Accepts batch updates - all changes are written atomically.
// Each record is tagged with operation type (insert/update/delete) and client IP.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.BatchUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.BatchUpdateResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate and set defaults
	for i, update := range req.Updates {
		if update.Key == "" {
			writeJSON(w, http.StatusBadRequest, models.BatchUpdateResponse{
				Success: false,
				Message: "Empty key in request",
			})
			return
		}
		if update.Type != models.TypeText && update.Type != models.TypePassword {
			req.Updates[i].Type = models.TypeText
		}
	}

	if err := h.store.BatchUpdate(req.Updates, getClientIP(r)); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.BatchUpdateResponse{
			Success: false,
			Message: "Update failed: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.BatchUpdateResponse{
		Success: true,
		Message: "OK",
		Count:   len(req.Updates),
	})
}

// List handles GET /api/list
// Returns all parameters with latest values. Passwords masked by default.
// Use ?unmask=true to reveal password values.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	params := h.store.List(r.URL.Query().Get("unmask") == "true")
	writeJSON(w, http.StatusOK, models.ListResponse{
		Parameters: params,
		Count:      len(params),
	})
}

// GetUnmasked handles GET /api/get?key=xxx
// Returns a single parameter with unmasked value. Used by UI to reveal passwords.
func (h *Handler) GetUnmasked(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key required"})
		return
	}

	param := h.store.Get(key)
	if param == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	writeJSON(w, http.StatusOK, models.ParameterView{
		Key:       param.Key,
		Value:     param.Value,
		Type:      param.Type,
		Timestamp: param.Timestamp,
		Masked:    false,
	})
}

// Health handles GET /api/health - simple health check for deployment.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetHistory handles GET /api/history?key=xxx
// Returns all log entries for a key showing full change history.
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key required"})
		return
	}

	history := h.store.GetHistory(key)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"key":     key,
		"history": history,
		"count":   len(history),
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
