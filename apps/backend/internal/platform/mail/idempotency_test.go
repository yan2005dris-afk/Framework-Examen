package mail

import (
	"bufio"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSMTPAcceptedDataSurvivesBrokenQuit(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		reader := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 qa SMTP\r\n")
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				fmt.Fprint(conn, "250 qa\r\n")
			case strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
				fmt.Fprint(conn, "250 OK\r\n")
			case strings.HasPrefix(line, "DATA"):
				fmt.Fprint(conn, "354 Datos\r\n")
				for {
					line, err = reader.ReadString('\n')
					if err != nil {
						return
					}
					if line == ".\r\n" {
						break
					}
				}
				fmt.Fprint(conn, "250 Aceptado\r\n")
				return
			default:
				return
			}
		}
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	number, _ := strconv.Atoi(port)
	sender := NewSMTPSender(Config{Host: host, Port: number, From: "noreply@example.test", Timeout: 3 * time.Second})
	if err = sender.Send(context.Background(), Message{To: "qa@example.test", Subject: "Prueba", Body: "Mensaje sintético"}); err != nil {
		t.Fatal("un QUIT roto no revierte DATA aceptado", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("servidor de prueba no terminó")
	}
}

func TestOutboxDeduplicatesLogicalMessageDespiteNewExpiry(t *testing.T) {
	capture := &captureOutbox{}
	queue, err := NewOutbox(sqlcgen.New(capture), testKey())
	if err != nil {
		t.Fatal(err)
	}
	msg := Message{Kind: "submission.confirmation", ResourceID: uuid.NewString(), To: "qa@example.test", Body: "Snapshot de prueba", ExpiresAt: time.Now().Add(time.Hour)}
	if err = queue.Send(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	key := capture.args[1]
	msg.ExpiresAt = msg.ExpiresAt.Add(time.Hour)
	if err = queue.Send(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if key != capture.args[1] {
		t.Fatal("cambió la clave idempotente por un vencimiento nuevo")
	}
	msg.ResourceID = uuid.NewString()
	if err = queue.Send(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if key == capture.args[1] {
		t.Fatal("se mezclaron recursos")
	}
}
func TestSMTPMessageIDIsStableAndCannotInjectHeaders(t *testing.T) {
	id := uuid.NewString()
	msg := Message{ID: id, To: "qa@example.test", Subject: "Prueba", Body: "Prueba"}
	a := buildMessage("noreply@example.test", msg)
	if a != buildMessage("noreply@example.test", msg) || !strings.Contains(a, "Message-ID: <"+id+"@") {
		t.Fatal("sin identidad estable")
	}
	msg.ID = "x\r\nBcc: indebido@example.test"
	if strings.Contains(buildMessage("noreply@example.test", msg), "Bcc:") {
		t.Fatal("inyección de cabecera")
	}
}
