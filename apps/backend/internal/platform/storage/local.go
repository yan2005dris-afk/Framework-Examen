package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

/**
 * ErrNotFound indica que el objeto no está en el almacenamiento.
 */
var ErrNotFound = errors.New("el objeto no existe en el almacenamiento")

/**
 * Local guarda archivos en el sistema de archivos local. Es el adaptador por
 * defecto en desarrollo; los bounded contexts dependen de la interfaz, no de
 * esta implementación.
 */
type Local struct {
	base string
}

/**
 * NewLocal crea el almacenamiento local y asegura el directorio base.
 *
 * @param {string} base - Directorio raíz donde se guardan los archivos.
 * @returns {Local} Almacenamiento listo para usar.
 * @throws {Error} Si el directorio base está vacío o no se puede crear.
 */
func NewLocal(base string) (*Local, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return nil, errors.New("el directorio base del almacenamiento no puede estar vacío")
	}

	if err := os.MkdirAll(base, 0o750); err != nil {
		return nil, fmt.Errorf("no se pudo preparar el directorio de almacenamiento: %w", err)
	}

	return &Local{base: base}, nil
}

/**
 * Save escribe el contenido en la clave indicada.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {string} key - Clave lógica del archivo dentro del almacenamiento.
 * @param {io.Reader} content - Contenido a persistir.
 * @returns {void}
 * @throws {Error} Si la clave es inválida o no se puede escribir el archivo.
 */
func (l *Local) Save(ctx context.Context, key string, content io.Reader) error {
	relative, err := l.resolvePath(key)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(l.base)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.MkdirAll(filepath.Dir(relative), 0o750); err != nil {
		return err
	}
	// La clave es nueva y privada: solo se firma después de confirmar sus metadatos.
	file, err := root.OpenFile(relative, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, &contextReader{ctx: ctx, reader: content})
	closeErr := file.Close()
	if copyErr != nil {
		_ = root.Remove(relative)
		return copyErr
	}
	if closeErr != nil {
		_ = root.Remove(relative)
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		_ = root.Remove(relative)
		return err
	}
	return nil
}

/**
 * Get abre el archivo almacenado en la clave indicada.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {string} key - Clave lógica del archivo dentro del almacenamiento.
 * @returns {io.ReadCloser} Flujo de lectura del archivo.
 * @throws {Error} ErrNotFound si la clave no existe.
 */
func (l *Local) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	path, err := l.resolvePath(key)
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(l.base)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := root.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("no se pudo abrir el archivo: %w", err)
	}

	return file, nil
}

/**
 * Delete borra el archivo almacenado en la clave indicada.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {string} key - Clave lógica del archivo dentro del almacenamiento.
 * @returns {void}
 * @throws {Error} Si el archivo no puede eliminarse; una clave ausente es idempotente.
 */
func (l *Local) Delete(ctx context.Context, key string) error {
	path, err := l.resolvePath(key)
	if err != nil {
		return err
	}

	root, err := os.OpenRoot(l.base)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("no se pudo eliminar el archivo: %w", err)
	}

	return nil
}

/**
 * resolvePath valida una clave relativa antes de abrirla con os.Root, que
 * impide recorridos fuera del directorio incluso a través de symlinks.
 *
 * @param {string} key - Clave lógica del archivo dentro del almacenamiento.
 * @returns {string} Ruta relativa para las operaciones de os.Root.
 * @throws {Error} Si la clave intenta salir del directorio base.
 */
func (l *Local) resolvePath(key string) (string, error) {
	cleaned := strings.TrimSpace(key)
	if cleaned == "" || cleaned == "." || filepath.IsAbs(cleaned) || strings.ContainsAny(cleaned, "\\:") || strings.HasPrefix(cleaned, "/") {
		return "", errors.New("la clave de almacenamiento no es válida")
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".." || part == "." || part == "" {
			return "", errors.New("la clave de almacenamiento no es válida")
		}
	}
	return filepath.FromSlash(cleaned), nil
}

// contextReader detiene copias de archivos cuando se cancela la solicitud.
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
