package consent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type ConsentStore interface {
	Grant(context.Context, string, string, string, string) error
	Revoke(context.Context, string, string, string) error
	Check(context.Context, string, string) (bool, error)
}

type Service struct {
	consents ConsentStore
}

func NewService(consents ConsentStore) http.Handler {
	return &Service{consents: consents}
}

type consentChange struct {
	Event          OrderEvent `json:"event"`
	Action         string     `json:"action"`
	Source         string     `json:"source"`
	IdempotencyKey string     `json:"idempotency_key"`
}

type dispatchRequest struct {
	UserID  string     `json:"user_id"`
	OrderID string     `json:"order_id"`
	Event   OrderEvent `json:"event"`
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/users/") && strings.HasSuffix(r.URL.Path, "/consent"):
		s.changeConsent(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/orders/dispatch":
		s.decideDispatch(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Service) changeConsent(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/users/"), "/consent")
	if userID == "" || strings.Contains(userID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user path"})
		return
	}
	var input consentChange
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	category, err := CategoryFor(input.Event)
	if err != nil || input.IdempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "event and idempotency_key are required"})
		return
	}

	switch input.Action {
	case "grant":
		err = s.consents.Grant(r.Context(), userID, category, input.Source, input.IdempotencyKey)
	case "revoke":
		err = s.consents.Revoke(r.Context(), userID, category, input.IdempotencyKey)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action must be grant or revoke"})
		return
	}
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": userID, "category": category, "state": input.Action + "ed"})
}

func (s *Service) decideDispatch(w http.ResponseWriter, r *http.Request) {
	var input dispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.UserID == "" || input.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id, order_id, and event are required"})
		return
	}
	category, err := CategoryFor(input.Event)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	granted, err := s.consents.Check(r.Context(), input.UserID, category)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	decision, _ := DecideDispatch(input.OrderID, input.Event, granted)
	writeJSON(w, http.StatusOK, decision)
}

func writeUpstreamError(w http.ResponseWriter, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		writeJSON(w, apiErr.StatusCode, map[string]string{"error": apiErr.Code, "message": apiErr.Message})
		return
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream request failed"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
