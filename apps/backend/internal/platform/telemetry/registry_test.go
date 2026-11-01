package telemetry

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMetricsRejectSensitiveLabelsAndAreConcurrentSafe(t *testing.T) {
	r := &Registry{}
	r.Observe("http", "request", "2xx", time.Second)
	r.Observe("persona@example.test", "request", "ok", time.Second)
	r.Observe("http", "request/persona@example.test", "ok", time.Second)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(w.Body.String(), `component="http"`) || strings.Contains(w.Body.String(), "example.test") || strings.Contains(w.Body.String(), `\n`) {
		t.Fatal(w.Body.String())
	}
	parent, trace, span := TraceParent()
	if len(parent) != 55 || len(trace) != 32 || len(span) != 16 {
		t.Fatal("contexto inválido")
	}
}

func TestHistogramIsCumulativeUnderConcurrentWrites(t *testing.T) {
	r := &Registry{}
	var wg sync.WaitGroup
	for n := 0; n < 100; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); r.Observe("http", "request", "2xx", 100*time.Millisecond) }()
	}
	wg.Wait()
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	body := w.Body.String()
	for _, line := range []string{`backend_operations_total{component="http",operation="request",result="2xx"} 100`, `le="0.05"} 0`, `le="0.1"} 100`, `le="+Inf"} 100`} {
		if !strings.Contains(body, line) {
			t.Fatal("histograma incorrecto", line)
		}
	}
}
