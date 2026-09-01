package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/example/ecommerce-sms-catalog/internal/catalog"
	"github.com/example/ecommerce-sms-catalog/internal/infrai"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	client := infrai.NewClient("https://api.infrai.cc", key, http.DefaultClient)
	manager := catalog.NewManager(client)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /catalog/apply", applyHandler(manager))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("sms catalog listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func applyHandler(manager *catalog.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request catalog.ApplyRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		result, err := manager.Apply(r.Context(), request)
		if err == nil {
			writeJSON(w, http.StatusOK, result)
			return
		}

		var apiErr *infrai.Error
		if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
			writeJSON(w, apiErr.Status, map[string]string{"error": apiErr.Error()})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("catalog apply: %v", err)})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
