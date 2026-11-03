// Command backup-crypt mantiene la clave fuera de argumentos, archivos cifrados y logs.
package main

import (
	"encoding/base64"
	"fmt"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/backup"
	"io"
	"os"
	"path/filepath"
)

func run() error {
	if len(os.Args) != 4 || (os.Args[1] != "encrypt" && os.Args[1] != "decrypt") {
		return fmt.Errorf("uso: backup-crypt encrypt|decrypt origen destino")
	}
	key, e := base64.StdEncoding.DecodeString(os.Getenv("BACKUP_ENCRYPTION_KEY"))
	if e != nil || len(key) != 32 {
		return fmt.Errorf("se requiere BACKUP_ENCRYPTION_KEY base64 de 32 bytes")
	}
	in, e := os.Open(os.Args[2])
	if e != nil {
		return fmt.Errorf("no se pudo abrir el respaldo")
	}
	defer in.Close()
	dest, e := os.OpenFile(os.Args[3], os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return fmt.Errorf("el destino debe ser nuevo y accesible")
	}
	defer dest.Close()
	tmp, e := os.CreateTemp(filepath.Dir(os.Args[3]), ".backup-private-*")
	if e != nil {
		_ = dest.Close()
		_ = os.Remove(os.Args[3])
		return fmt.Errorf("no se pudo preparar un destino privado")
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	action := backup.Seal
	if os.Args[1] == "decrypt" {
		action = backup.Open
	}
	e = action(tmp, in, key)
	if e == nil {
		_, e = tmp.Seek(0, 0)
	}
	if e == nil {
		_, e = io.Copy(dest, tmp)
	}
	if e == nil {
		e = dest.Sync()
	}
	if e != nil {
		dest.Close()
		_ = os.Remove(os.Args[3])
		return fmt.Errorf("no se verificó el respaldo; destino descartado")
	}
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println("Respaldo procesado y verificado sin revelar su clave")
}
