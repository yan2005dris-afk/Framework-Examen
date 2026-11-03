package mail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResendSender_Send_Success(t *testing.T) {
	var receivedPayload resendPayload

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("metodo esperado POST, recibido %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer re_test_123" {
			t.Errorf("header Authorization incorrecto: %s", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("error decodificando json: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"msg_123"}`))
	}))
	defer ts.Close()

	sender := NewResendSender(ResendConfig{
		APIKey:  "re_test_123",
		From:    "onboarding@resend.dev",
		BaseURL: ts.URL,
		Timeout: 5 * time.Second,
	})

	err := sender.Send(context.Background(), Message{
		To:      "usuario@example.com",
		Subject: "Código OTP",
		Body:    "Tu código es: 123456",
		Attachments: []Attachment{
			{Name: "test.pdf", MIME: "application/pdf", Data: []byte("%PDF-test")},
		},
	})

	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if len(receivedPayload.To) != 1 || receivedPayload.To[0] != "usuario@example.com" {
		t.Errorf("destinatario incorrecto: %v", receivedPayload.To)
	}
	if receivedPayload.Subject != "Código OTP" {
		t.Errorf("asunto incorrecto: %s", receivedPayload.Subject)
	}
	if len(receivedPayload.Attachments) != 1 || receivedPayload.Attachments[0].Filename != "test.pdf" {
		t.Errorf("adjuntos incorrectos: %v", receivedPayload.Attachments)
	}
}

func TestResendSender_Send_ErrorFromAPI(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Domain not verified","name":"validation_error"}`))
	}))
	defer ts.Close()

	sender := NewResendSender(ResendConfig{
		APIKey:  "re_bad_key",
		From:    "onboarding@resend.dev",
		BaseURL: ts.URL,
	})

	err := sender.Send(context.Background(), Message{
		To:      "usuario@example.com",
		Subject: "Test",
		Body:    "Hola",
	})

	if err == nil {
		t.Fatal("se esperaba error y no ocurrió")
	}
}
