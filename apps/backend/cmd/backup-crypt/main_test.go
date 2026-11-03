package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestCommandNeverPublishesUnauthenticatedOrOverwritesOutput(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain")
	encrypted := filepath.Join(dir, "encrypted")
	restored := filepath.Join(dir, "restored")
	if e := os.WriteFile(plain, []byte("DATOS SINTETICOS - NO VALIDO"), 0600); e != nil {
		t.Fatal(e)
	}
	old := os.Args
	defer func() { os.Args = old }()
	t.Setenv("BACKUP_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	os.Args = []string{"backup-crypt", "encrypt", plain, encrypted}
	if e := run(); e != nil {
		t.Fatal(e)
	}
	t.Setenv("BACKUP_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32)))
	os.Args = []string{"backup-crypt", "decrypt", encrypted, restored}
	if run() == nil {
		t.Fatal("clave incorrecta aceptada")
	}
	if _, e := os.Stat(restored); !os.IsNotExist(e) {
		t.Fatal("publicó salida sin autenticar")
	}
	os.Args = []string{"backup-crypt", "encrypt", plain, plain}
	if run() == nil {
		t.Fatal("sobrescribió original")
	}
	value, e := os.ReadFile(plain)
	if e != nil || string(value) != "DATOS SINTETICOS - NO VALIDO" {
		t.Fatal("perdió origen")
	}
}
