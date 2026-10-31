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
	path, err := l.resolvePath(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("no se pudo crear el directorio del documento: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, content); err != nil {
		return fmt.Errorf("no se pudo escribir el archivo: %w", err)
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

	file, err := os.Open(path)
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
 * @throws {Error} ErrNotFound si la clave no existe.
 */
func (l *Local) Delete(ctx context.Context, key string) error {
	path, err := l.resolvePath(key)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("no se pudo eliminar el archivo: %w", err)
	}

	return nil
}

/**
 * resolvePath resuelve la ruta de un archivo bloqueando cualquier intento de
 * recorrido fuera del directorio base.
 *
 * @param {string} key - Clave lógica del archivo dentro del almacenamiento.
 * @returns {string} Ruta absoluta y segura del archivo.
 * @throws {Error} Si la clave intenta salir del directorio base.
 */
func (l *Local) resolvePath(key string) (string, error) {
	cleaned := strings.TrimSpace(key)
	if cleaned == "" {
		return "", errors.New("la clave de almacenamiento no puede estar vacía")
	}

	resolved := filepath.Clean(filepath.Join(l.base, cleaned))
	relative, err := filepath.Rel(l.base, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", errors.New("la clave de almacenamiento no es válida")
	}

	return resolved, nil
}
