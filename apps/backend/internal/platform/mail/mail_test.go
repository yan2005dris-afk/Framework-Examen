package mail

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

/**
 * fakeSMTP es un servidor SMTP mínimo para verificar el remitente.
 */
type fakeSMTP struct {
	listener net.Listener
	messages chan string
}

func startFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo abrir el servidor SMTP de prueba: %v", err)
	}

	server := &fakeSMTP{listener: listener, messages: make(chan string, 1)}

	go server.accept()

	t.Cleanup(func() {
		_ = listener.Close()
	})

	return server
}

func (s *fakeSMTP) accept() {
	connection, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer connection.Close()

	reader := bufio.NewReader(connection)
	_, _ = connection.Write([]byte("220 mailpit ESMTP\r\n"))

	var body strings.Builder
	inData := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		if inData {
			if line == ".\r\n" {
				inData = false
				s.messages <- body.String()
				_, _ = connection.Write([]byte("250 OK\r\n"))
				continue
			}
			body.WriteString(line)
			continue
		}

		switch {
		case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
			_, _ = connection.Write([]byte("250-mailpit\r\n250 8BITMIME\r\n"))
		case strings.HasPrefix(line, "DATA"):
			inData = true
			_, _ = connection.Write([]byte("354 Fin del dato\r\n"))
		case strings.HasPrefix(line, "QUIT"):
			_, _ = connection.Write([]byte("221 Bye\r\n"))
			return
		default:
			_, _ = connection.Write([]byte("250 OK\r\n"))
		}
	}
}

/**
 * TestSMTPSenderSend verifica que el correo llega al servidor con los
 * encabezados y el cuerpo correctos.
 */
func TestSMTPSenderSend(t *testing.T) {
	server := startFakeSMTP(t)

	port := server.listener.Addr().(*net.TCPAddr).Port
	sender := NewSMTPSender(Config{
		Host:    "127.0.0.1",
		Port:    port,
		From:    "no-reply@biess.local",
		Timeout: 5 * time.Second,
	})

	message := Message{
		To:      "postulante@example.com",
		Subject: "Recuperación de contraseña",
		Body:    "Tu código es 123456.",
	}

	if err := sender.Send(context.Background(), message); err != nil {
		t.Fatalf("no se esperaba error y se obtuvo %v", err)
	}

	select {
	case received := <-server.messages:
		if !strings.Contains(received, "To: postulante@example.com") {
			t.Fatalf("falta el destinatario en el correo: %s", received)
		}
		if !strings.Contains(received, "Subject: Recuperación de contraseña") {
			t.Fatalf("falta el asunto en el correo: %s", received)
		}
		if !strings.Contains(received, "Tu código es 123456.") {
			t.Fatalf("falta el cuerpo en el correo: %s", received)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("el servidor SMTP no recibió ningún correo")
	}
}

/**
 * TestSMTPSenderRejectsEmptyRecipient verifica la validación del destinatario.
 */
func TestSMTPSenderRejectsEmptyRecipient(t *testing.T) {
	sender := NewSMTPSender(Config{Host: "127.0.0.1", Port: 2525, From: "no-reply@biess.local"})

	if err := sender.Send(context.Background(), Message{Subject: "Prueba"}); err != ErrEmptyRecipient {
		t.Fatalf("se esperaba ErrEmptyRecipient y se obtuvo %v", err)
	}
}

/**
 * TestSMTPSenderWithoutHost verifica que sin servidor configurado no se intenta
 * enviar nada.
 */
func TestSMTPSenderWithoutHost(t *testing.T) {
	sender := NewSMTPSender(Config{From: "no-reply@biess.local"})

	err := sender.Send(context.Background(), Message{To: "postulante@example.com"})
	if err != ErrSMTPDisabled {
		t.Fatalf("se esperaba ErrSMTPDisabled y se obtuvo %v", err)
	}
}
