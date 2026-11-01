// Package telemetry publica métricas agregadas y trazas locales sin datos de postulantes.
package telemetry

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type sample struct {
	Count   uint64
	Seconds float64
	Buckets [10]uint64
}
type Registry struct {
	mu     sync.Mutex
	values map[string]sample
}

var Default = &Registry{values: map[string]sample{}}

/** Observe sólo admite nombres cerrados de componentes/resultados; nunca IDs, rutas libres ni errores de proveedor. */
func (r *Registry) Observe(component, operation, result string, d time.Duration) {
	if !allowed(component, []string{"http", "otp", "outbox", "storage", "database", "cleanup", "migration", "exports", "rate_limit"}) || !allowed(result, []string{"ok", "error", "denied", "4xx", "5xx", "2xx", "3xx"}) {
		return
	}
	if !allowed(operation, []string{"request", "issue", "verify", "dispatch", "delivery", "save", "delete", "download", "transaction", "run", "sessions", "objects", "uploads", "expire_exams"}) {
		return
	}
	key := fmt.Sprintf(`component=%q,operation=%q,result=%q`, component, operation, result)
	r.mu.Lock()
	if r.values == nil {
		r.values = map[string]sample{}
	}
	v := r.values[key]
	v.Count++
	v.Seconds += d.Seconds()
	for i, limit := range []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10} {
		if d.Seconds() <= limit {
			v.Buckets[i]++
		}
	}
	v.Buckets[9]++
	r.values[key] = v
	r.mu.Unlock()
}
func allowed(v string, options []string) bool {
	for _, s := range options {
		if s == v {
			return true
		}
	}
	return false
}
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		r.mu.Lock()
		defer r.mu.Unlock()
		keys := make([]string, 0, len(r.values))
		for k := range r.values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		_, _ = fmt.Fprintln(w, `# TYPE backend_operations_total counter
# TYPE backend_operation_duration_seconds histogram`)
		for _, k := range keys {
			v := r.values[k]
			for i, limit := range []string{"0.01", "0.05", "0.1", "0.25", "0.5", "1", "2.5", "5", "10", "+Inf"} {
				_, _ = fmt.Fprintf(w, "backend_operation_duration_seconds_bucket{%s,le=%q} %d\n", k, limit, v.Buckets[i])
			}
			_, _ = fmt.Fprintf(w, "backend_operation_duration_seconds_count{%s} %d\n", k, v.Count)
			_, _ = fmt.Fprintf(w, "backend_operations_total{%s} %d\nbackend_operation_duration_seconds_sum{%s} %s\n", k, v.Count, k, strconv.FormatFloat(v.Seconds, 'g', -1, 64))
		}
	})
}

/** TraceParent genera un contexto W3C local nuevo; no copia baggage/URLs del cliente ni afirma exportación OTLP. */
func TraceParent() (string, string, string) {
	var trace [16]byte
	var span [8]byte
	if _, e := rand.Read(trace[:]); e != nil {
		return "", "", ""
	}
	if _, e := rand.Read(span[:]); e != nil {
		return "", "", ""
	}
	t, s := hex.EncodeToString(trace[:]), hex.EncodeToString(span[:])
	return "00-" + t + "-" + s + "-01", t, s
}
func HTTPClass(code int) string {
	if code < 200 {
		return "2xx"
	}
	if code < 300 {
		return "2xx"
	}
	if code < 400 {
		return "3xx"
	}
	if code < 500 {
		return "4xx"
	}
	return "5xx"
}
func Result(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}
func SafeRoute(route string) string {
	if !strings.HasPrefix(route, "/") {
		return "unmatched"
	}
	return route
}
