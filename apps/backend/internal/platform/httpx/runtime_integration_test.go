package httpx_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Sólo ejecuta tráfico contra una API y Mailpit locales de prueba explícitos.
func TestRuntimeCrossCuttingSessionCSRFAndSMTP(t *testing.T) {
	base := os.Getenv("RUNTIME_API_URL")
	mailpit := os.Getenv("RUNTIME_MAILPIT_URL")
	if base == "" || mailpit == "" {
		t.Skip("defina RUNTIME_API_URL y RUNTIME_MAILPIT_URL para contrato HTTP vivo")
	}
	for _, endpoint := range []string{base, mailpit} {
		u, e := url.Parse(endpoint)
		if e != nil || u.Scheme != "http" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") {
			t.Fatal("la prueba sólo admite servicios locales")
		}
	}
	origin := os.Getenv("RUNTIME_ORIGIN")
	if origin == "" {
		origin = "http://localhost:18081"
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 8 * time.Second}
	call := func(c *http.Client, method, path, origin string, input interface{}, want int) []byte {
		t.Helper()
		var body io.Reader
		if input != nil {
			raw, e := json.Marshal(input)
			if e != nil {
				t.Fatal(e)
			}
			body = bytes.NewReader(raw)
		}
		r, e := http.NewRequest(method, base+path, body)
		if e != nil {
			t.Fatal(e)
		}
		r.Header.Set("Content-Type", "application/json")
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		response, e := c.Do(r)
		if e != nil {
			t.Fatal("API de prueba no disponible")
		}
		defer response.Body.Close()
		raw, e := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		if e != nil || response.StatusCode != want {
			t.Fatalf("contrato %s: esperado %d, recibido %d", path, want, response.StatusCode)
		}
		return raw
	}
	var ready struct{ Checks map[string]string }
	if json.Unmarshal(call(client, "GET", "/readyz", "", nil, 200), &ready) != nil {
		t.Fatal("readiness inválido")
	}
	for _, name := range []string{"database", "redis", "storage", "outbox", "smtp", "downloads"} {
		if ready.Checks[name] != "ok" {
			t.Fatal("dependencia de integración no comprobada", name)
		}
	}
	csrfJar, _ := cookiejar.New(nil)
	targetURL, _ := url.Parse(base)
	csrfJar.SetCookies(targetURL, []*http.Cookie{{Name: "session", Value: "probe"}})
	csrfClient := &http.Client{Jar: csrfJar, Timeout: 8 * time.Second}
	call(csrfClient, "POST", "/api/v1/auth/register/request", "", map[string]string{}, 403)
	call(csrfClient, "POST", "/api/v1/auth/register/request", "https://evil.invalid", map[string]string{}, 403)
	random := uuid.New()
	n := int(random[0])<<16 | int(random[1])<<8 | int(random[2])
	cedula := fmt.Sprintf("240%06d", n%900000+100000)
	sum := 0
	for i, c := range cedula {
		d := int(c - '0')
		if i%2 == 0 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	cedula += fmt.Sprint((10 - sum%10) % 10)
	email := "runtime-" + uuid.NewString() + "@synthetic.invalid"
	pass := "Synthetic!2026" + uuid.NewString()
	issued := time.Now().Add(-time.Second)
	var challenge struct {
		ID string `json:"desafio_id"`
	}
	if json.Unmarshal(call(client, "POST", "/api/v1/auth/register/request", origin, map[string]string{"cedula": cedula, "correo": email}, 200), &challenge) != nil || challenge.ID == "" {
		t.Fatal("sin desafío de registro")
	}
	code := ""
	for n := 0; n < 40 && code == ""; n++ {
		response, e := client.Get(mailpit + "/api/v1/messages?limit=200")
		if e != nil {
			t.Fatal("Mailpit de prueba no disponible")
		}
		var listing struct {
			Messages []struct {
				ID      string
				Created time.Time
				To      []struct{ Address string }
			}
		}
		e = json.NewDecoder(response.Body).Decode(&listing)
		response.Body.Close()
		if e != nil {
			t.Fatal("listado Mailpit inválido")
		}
		for _, m := range listing.Messages {
			if m.Created.Before(issued) {
				continue
			}
			for _, to := range m.To {
				if to.Address != email {
					continue
				}
				r, e := client.Get(mailpit + "/api/v1/message/" + m.ID)
				if e != nil {
					t.Fatal("mensaje no disponible")
				}
				var message struct{ Text string }
				e = json.NewDecoder(r.Body).Decode(&message)
				r.Body.Close()
				if e != nil {
					t.Fatal("mensaje inválido")
				}
				code = regexp.MustCompile(`\b[0-9]{6}\b`).FindString(message.Text)
			}
		}
		if code == "" {
			time.Sleep(time.Second)
		}
	}
	if code == "" {
		t.Fatal("OTP no se recibió en Mailpit")
	}
	call(client, "POST", "/api/v1/auth/register/verify", origin, map[string]string{"cedula": cedula, "correo": email, "codigo_otp": code, "desafio_id": challenge.ID, "nombres": "SINTETICO", "apellidos": "NO VALIDO", "password": pass}, 201)
	u, _ := url.Parse(base)
	cookies := jar.Cookies(u)
	if len(cookies) == 0 {
		t.Fatal("registro sin cookie de sesión")
	}
	old := cookies[0]
	stale := &http.Client{Timeout: 5 * time.Second}
	call(client, "POST", "/api/v1/auth/login", origin, map[string]string{"cedula": cedula, "password": pass}, 200)
	r, _ := http.NewRequest("GET", base+"/api/v1/auth/session", nil)
	r.AddCookie(old)
	response, e := stale.Do(r)
	if e != nil {
		t.Fatal("sesión no comprobada")
	}
	response.Body.Close()
	if response.StatusCode != 401 {
		t.Fatal("sesión anterior al login sigue válida")
	}
	call(client, "POST", "/api/v1/auth/logout", "", map[string]string{}, 403)
	call(client, "GET", "/api/v1/auth/session", "", nil, 200)
	call(client, "POST", "/api/v1/auth/logout", origin, map[string]string{}, 200)
	call(client, "GET", "/api/v1/auth/session", "", nil, 401)
	if strings.Contains(base, "token=") {
		t.Fatal("token en URL")
	}
}
