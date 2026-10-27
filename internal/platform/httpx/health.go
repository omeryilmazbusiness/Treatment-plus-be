package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Readiness is a dependency the process must reach before serving traffic.
type Readiness interface {
	Name() string
	Ready(ctx context.Context) error
}

type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func handleReadyz(checks []Readiness) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		type result struct {
			name string
			err  error
		}

		out := make(chan result, len(checks))
		var wg sync.WaitGroup
		for _, c := range checks {
			wg.Add(1)
			go func(c Readiness) {
				defer wg.Done()
				out <- result{name: c.Name(), err: c.Ready(ctx)}
			}(c)
		}
		go func() {
			wg.Wait()
			close(out)
		}()

		checksOut := make(map[string]string, len(checks))
		ok := true
		for res := range out {
			if res.err != nil {
				ok = false
				checksOut[res.name] = res.err.Error()
				continue
			}
			checksOut[res.name] = "ok"
		}

		status := http.StatusOK
		body := healthResponse{Status: "ok", Checks: checksOut}
		if !ok {
			status = http.StatusServiceUnavailable
			body.Status = "not_ready"
		}
		writeJSON(w, status, body)
	}
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	WriteJSON(w, status, v)
}
