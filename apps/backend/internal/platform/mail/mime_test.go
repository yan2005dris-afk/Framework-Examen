package mail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/google/uuid"
	"io"
	"mime"
	"mime/multipart"
	netmail "net/mail"
	"strings"
	"testing"
)

func TestMultipartPDFRoundTrip(t *testing.T) {
	pdf := bytes.Repeat([]byte("%PDF-1.7\nñ\x00"), 30)
	m := Message{ID: uuid.NewString(), To: "destino@example.test", Subject: "Confirmación áéí\r\nBcc: ignorado@example.test", Body: "Contenido sellado áéí", Attachments: []Attachment{{Name: "EF12-prueba.pdf", MIME: "application/pdf", Data: pdf}}}
	raw, err := composeMessage("Portal <portal@example.test>", m)
	if err != nil {
		t.Fatal(err)
	}
	message, err := netmail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if message.Header.Get("Bcc") != "" || !strings.Contains(message.Header.Get("Message-ID"), m.ID) {
		t.Fatal("cabeceras inyectadas o identificador perdido")
	}
	kind, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil || kind != "multipart/mixed" {
		t.Fatal(kind, err)
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	text, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(text)
	if string(body) != m.Body {
		t.Fatal("cuerpo alterado")
	}
	part, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, part))
	if err != nil || !bytes.Equal(data, pdf) || part.FileName() != m.Attachments[0].Name {
		t.Fatal("adjunto alterado", err)
	}
	if _, err = reader.NextPart(); err != io.EOF {
		t.Fatal("partes inesperadas")
	}
	again, _ := composeMessage("Portal <portal@example.test>", m)
	if again != raw {
		t.Fatal("MIME no reproducible")
	}
}
func TestMIMERejectsUnsafeOrUnpreparedContent(t *testing.T) {
	base := Message{To: "destino@example.test", Body: "x"}
	for _, name := range []string{"pending", "address", "path", "type"} {
		t.Run(name, func(t *testing.T) {
			m := base
			switch name {
			case "pending":
				m.RenderPayload = []byte("snapshot")
			case "address":
				m.To += "\r\nBcc: otro@example.test"
			case "path":
				m.Attachments = []Attachment{{Name: "../documento.pdf", MIME: "application/pdf", Data: []byte("pdf")}}
			case "type":
				m.Attachments = []Attachment{{Name: "x.pdf", MIME: "text/html", Data: []byte("html")}}
			}
			_, err := composeMessage("portal@example.test", m)
			if err == nil {
				t.Fatal("aceptó correo inseguro")
			}
			if name == "pending" && !errors.Is(err, ErrUnpreparedMessage) {
				t.Fatal(err)
			}
		})
	}
}
