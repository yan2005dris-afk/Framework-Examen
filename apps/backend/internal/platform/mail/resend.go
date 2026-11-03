package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

/**
 * ResendConfig son los parámetros para el cliente de la API REST de Resend.
 */
type ResendConfig struct {
	APIKey  string
	From    string
	Timeout time.Duration
	BaseURL string
}

/**
 * ResendSender entrega correos invocando la API REST de Resend vía HTTPS (puerto 443).
 * Es ideal para entornos como Render Free donde los puertos SMTP tradicionales (25, 465, 587)
 * están bloqueados por firewall.
 */
type ResendSender struct {
	config ResendConfig
	client *http.Client
}

/**
 * NewResendSender crea un nuevo remitente que utiliza la API de Resend.
 *
 * @param {ResendConfig} config - Configuración de Resend (API Key, remitente por defecto).
 * @returns {*ResendSender} Remitente listo para despachar correos.
 */
func NewResendSender(config ResendConfig) *ResendSender {
	if config.Timeout <= 0 {
		config.Timeout = 15 * time.Second
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.resend.com/emails"
	}
	return &ResendSender{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

/** resendAttachment define el formato esperado por Resend para adjuntos. */
type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // Base64 encoded
}

/** resendPayload define el cuerpo JSON requerido por POST /emails. */
type resendPayload struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	Text        string             `json:"text,omitempty"`
	HTML        string             `json:"html,omitempty"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

/**
 * Ready valida que la API Key esté configurada y el remitente sea válido.
 *
 * @param {context.Context} ctx - Contexto de cancelación.
 * @returns {error} Error si no está configurado.
 */
func (s *ResendSender) Ready(ctx context.Context) error {
	if s.config.APIKey == "" {
		return errors.New("resend: API key no configurada")
	}
	if s.config.From == "" {
		return errors.New("resend: remitente no configurado")
	}
	return nil
}

/**
 * Send entrega un correo a través del endpoint HTTPS de Resend.
 *
 * @param {context.Context} ctx - Contexto de la operación.
 * @param {Message} message - Mensaje a despachar.
 * @returns {error} Error si la entrega falló.
 */
func (s *ResendSender) Send(ctx context.Context, message Message) error {
	if message.To == "" {
		return ErrEmptyRecipient
	}
	from := s.config.From
	if from == "" {
		return ErrEmptySender
	}

	payload := resendPayload{
		From:    from,
		To:      []string{message.To},
		Subject: message.Subject,
		Text:    message.Body,
	}

	if len(message.Attachments) > 0 {
		payload.Attachments = make([]resendAttachment, 0, len(message.Attachments))
		for _, att := range message.Attachments {
			payload.Attachments = append(payload.Attachments, resendAttachment{
				Filename: att.Name,
				Content:  base64.StdEncoding.EncodeToString(att.Data),
			})
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("resend: error al serializar payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.BaseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("resend: error al crear request HTTP: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(s.config.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Framework-Examen-API/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: fallo de red al despachar correo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Message string `json:"message"`
			Name    string `json:"name"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Message != "" {
			return fmt.Errorf("resend api error (status %d): %s (%s)", resp.StatusCode, errResp.Message, errResp.Name)
		}
		return fmt.Errorf("resend api error: status HTTP %d", resp.StatusCode)
	}

	return nil
}
