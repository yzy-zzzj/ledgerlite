package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/yzy-zzzj/ledgerlite/internal/ledger"
	"github.com/yzy-zzzj/ledgerlite/internal/observability"
	"github.com/yzy-zzzj/ledgerlite/internal/types"
)

func Router(svc *ledger.Service, log *observability.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AccountID string `json:"account_id"`
			Currency  string `json:"currency"`
			Initial   int64  `json:"initial_cents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest); return
		}
		acc, err := svc.CreateAccount(r.Context(), req.AccountID, req.Currency, req.Initial)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest); return
		}
		writeJSON(w, http.StatusCreated, acc)
	})

	mux.HandleFunc("GET /v1/accounts/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/v1/accounts/"):]
		acc, err := svc.GetAccount(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound); return
		}
		writeJSON(w, http.StatusOK, acc)
	})

	mux.HandleFunc("POST /v1/transfer", func(w http.ResponseWriter, r *http.Request) {
		var req types.TransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest); return
		}
		res, err := svc.Transfer(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest); return
		}
		writeJSON(w, http.StatusOK, res)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
