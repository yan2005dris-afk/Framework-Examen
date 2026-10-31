// Package mail es el adaptador de salida para enviar correos. En desarrollo
// apunta a Mailpit; en producción a un servidor SMTP real.
package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"net"
	netmail "net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

/**
 * Errores del envío de correo.
 */
var (
	ErrEmptyRecipient = errors.New("el destinatario del correo está vacío")
	ErrEmptySender    = errors.New("el remitente del correo está vacío")
	ErrSMTPDisabled   = errors.New("el envío de correo no está configurado")
)

/**
 * Message es un correo saliente.
 */
type Message struct {
	Attachments   []Attachment
	RenderPayload []byte
	ID            string
	Suppressed    bool
	Kind          string
	ResourceID    string
	ExpiresAt     time.Time
	ChallengeKey  string
	ChallengeID   string
	To            string
	Subject       string
	Body          string
}

/**
 * Sender es el puerto de salida para enviar correos.
 */
type Sender interface {
	Send(ctx context.Context, message Message) error
}

/**
 * Config son los parámetros de conexión al servidor SMTP.
 */
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	UseTLS   bool
	Timeout  time.Duration
}

/**
 * SMTPSender envía correos hablando SMTP con un servidor.
 */
type SMTPSender struct {
	config Config
}

/**
 * NewSMTPSender crea el remitente con la configuración indicada.
 *
 * @param {Config} config - Parámetros de conexión al servidor SMTP.
 * @returns {SMTPSender} Remitente listo para usar.
 */
func NewSMTPSender(config Config) *SMTPSender {
	if config.Port == 0 {
		config.Port = 25
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}

	return &SMTPSender{config: config}
}

/**
 * Send entrega un correo al servidor SMTP.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {Message} message - Correo a enviar.
 * @returns {void}
 * @throws {Error} Si el correo no tiene destinatario o el servidor SMTP falla.
 */
func (s *SMTPSender) Send(ctx context.Context, message Message) error {
	if strings.TrimSpace(message.To) == "" {
		return ErrEmptyRecipient
	}
	if strings.TrimSpace(s.config.From) == "" {
		return ErrEmptySender
	}
	if s.config.Host == "" {
		return ErrSMTPDisabled
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	body, err := composeMessage(s.config.From, message)
	if err != nil {
		return err
	}
	address := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))

	dialer := net.Dialer{Timeout: s.config.Timeout}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("no se pudo conectar al servidor SMTP: %w", err)
	}
	defer connection.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	} else {
		_ = connection.SetDeadline(time.Now().Add(s.config.Timeout))
	}

	client, err := smtp.NewClient(connection, s.config.Host)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar la sesión SMTP: %w", err)
	}
	defer client.Close()

	if s.config.UseTLS {
		if err := client.StartTLS(&tls.Config{ServerName: s.config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("no se pudo iniciar TLS con el servidor SMTP: %w", err)
		}
	}

	if s.config.User != "" {
		auth := smtp.PlainAuth("", s.config.User, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("no se pudo autenticar en el servidor SMTP: %w", err)
		}
	}

	envelopeFrom, _ := netmail.ParseAddress(s.config.From)
	envelopeTo, _ := netmail.ParseAddress(message.To)
	if err := client.Mail(envelopeFrom.Address); err != nil {
		return fmt.Errorf("no se pudo indicar el remitente: %w", err)
	}

	if err := client.Rcpt(envelopeTo.Address); err != nil {
		return fmt.Errorf("no se pudo indicar el destinatario: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("no se pudo iniciar el cuerpo del correo: %w", err)
	}

	if _, err := writer.Write([]byte(body)); err != nil {
		return fmt.Errorf("no se pudo escribir el cuerpo del correo: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("no se pudo cerrar el cuerpo del correo: %w", err)
	}

	// DATA ya fue aceptado. Un fallo de QUIT no debe causar otro envío.
	_ = client.Quit()

	return nil
}

/**
 * buildMessage arma el correo en formato RFC 5322.
 *
 * @param {string} from - Remitente del correo.
 * @param {Message} message - Contenido del correo.
 * @returns {string} Mensaje listo para enviar.
 */
func buildMessage(from string, message Message) string {
	body, _ := composeMessage(from, message)
	return body
}
