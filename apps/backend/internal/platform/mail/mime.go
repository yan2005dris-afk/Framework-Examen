package mail

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/google/uuid"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
)

var ErrInvalidMessage = errors.New("el contenido del correo no es válido")
var ErrUnpreparedMessage = errors.New("el documento del correo aún no fue preparado")

/** Attachment contiene bytes del servidor; nunca una URL arbitraria suministrada por el cliente. */
type Attachment struct {
	Name string
	MIME string
	Data []byte
}

func composeMessage(from string, message Message) (string, error) {
	if len(message.RenderPayload) > 0 {
		return "", ErrUnpreparedMessage
	}
	if strings.ContainsAny(from+message.To, "\r\n") {
		return "", ErrInvalidMessage
	}
	sender, err := mail.ParseAddress(from)
	if err != nil {
		return "", ErrInvalidMessage
	}
	recipient, err := mail.ParseAddress(message.To)
	if err != nil {
		return "", ErrInvalidMessage
	}
	var out bytes.Buffer
	out.WriteString("From: " + sender.String() + "\r\nTo: " + recipient.String() + "\r\nSubject: " + mime.QEncoding.Encode("UTF-8", strings.Join(strings.Fields(message.Subject), " ")) + "\r\nMIME-Version: 1.0\r\n")
	if id, err := uuid.Parse(message.ID); err == nil {
		out.WriteString("Message-ID: <" + id.String() + "@notificaciones.biess.local>\r\n")
	}
	if len(message.Attachments) == 0 {
		out.WriteString("Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n")
		encoded := quotedprintable.NewWriter(&out)
		if _, err = encoded.Write([]byte(message.Body)); err != nil {
			return "", err
		}
		if err = encoded.Close(); err != nil {
			return "", err
		}
		out.WriteString("\r\n")
		return out.String(), nil
	}
	if len(message.Attachments) > 5 {
		return "", ErrInvalidMessage
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	sum := sha256.Sum256([]byte(message.ID + "|" + message.Subject + "|" + message.Body))
	boundary := "ef12-" + hex.EncodeToString(sum[:16])
	if err = writer.SetBoundary(boundary); err != nil {
		return "", err
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Type", "text/plain; charset=UTF-8")
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(h)
	if err != nil {
		return "", err
	}
	textEncoder := quotedprintable.NewWriter(part)
	if _, err = textEncoder.Write([]byte(message.Body)); err != nil {
		return "", err
	}
	if err = textEncoder.Close(); err != nil {
		return "", err
	}
	total := 0
	for _, attachment := range message.Attachments {
		total += len(attachment.Data)
		kind, params, err := mime.ParseMediaType(attachment.MIME)
		if err != nil || len(params) > 0 || kind != "application/pdf" || len(attachment.Data) == 0 || total > 20<<20 || attachment.Name == "" || len(attachment.Name) > 150 || strings.ContainsAny(attachment.Name, "\r\n/\\") {
			return "", ErrInvalidMessage
		}
		h := textproto.MIMEHeader{}
		h.Set("Content-Type", kind)
		h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Name}))
		h.Set("Content-Transfer-Encoding", "base64")
		part, err := writer.CreatePart(h)
		if err != nil {
			return "", err
		}
		encoded := base64.StdEncoding.EncodeToString(attachment.Data)
		for len(encoded) > 0 {
			n := min(76, len(encoded))
			if _, err = part.Write([]byte(encoded[:n] + "\r\n")); err != nil {
				return "", err
			}
			encoded = encoded[n:]
		}
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	out.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n\r\n")
	out.Write(body.Bytes())
	return out.String(), nil
}
