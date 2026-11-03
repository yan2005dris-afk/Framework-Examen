// Package backup cifra respaldos mediante AES-256-GCM por bloques autenticados.
package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
)

const chunkSize = 1 << 20
const magic = "BIESS-BACKUP-v1\x00"

var ErrInvalid = errors.New("el respaldo no es válido o la clave no corresponde")

func aead(key, salt []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, ErrInvalid
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(magic))
	mac.Write(salt)
	b, e := aes.NewCipher(mac.Sum(nil))
	if e != nil {
		return nil, ErrInvalid
	}
	return cipher.NewGCM(b)
}
func nonce(prefix []byte, index uint32) []byte {
	n := make([]byte, 12)
	copy(n[:8], prefix)
	binary.BigEndian.PutUint32(n[8:], index)
	return n
}
func aad(header []byte, index uint32, size uint32) []byte {
	b := make([]byte, len(header)+8)
	copy(b, header)
	binary.BigEndian.PutUint32(b[len(header):], index)
	binary.BigEndian.PutUint32(b[len(header)+4:], size)
	return b
}
func write(w io.Writer, b []byte) error {
	n, e := w.Write(b)
	if e == nil && n != len(b) {
		return io.ErrShortWrite
	}
	return e
}

/** Seal cifra un flujo con clave externa de 32 bytes, prefijo aleatorio por archivo y marcador final autenticado.
 * @returns {error} Fallo de lectura/escritura; no devuelve contenido privado.
 */
func Seal(w io.Writer, r io.Reader, key []byte) error {
	var e error
	header := make([]byte, len(magic)+32)
	copy(header, magic)
	if _, e = rand.Read(header[len(magic):]); e != nil {
		return e
	}
	box, e := aead(key, header[len(magic):])
	if e != nil {
		return e
	}
	if e = write(w, header); e != nil {
		return e
	}
	data := make([]byte, chunkSize)
	index := uint32(0)
	for {
		n, e := io.ReadFull(r, data)
		if e != nil && e != io.EOF && e != io.ErrUnexpectedEOF {
			return e
		}
		if index == ^uint32(0) {
			return ErrInvalid
		}
		size := uint32(n)
		frame := make([]byte, 4)
		binary.BigEndian.PutUint32(frame, size)
		sealed := box.Seal(nil, nonce(header[len(magic):], index), data[:n], aad(header, index, size))
		if e = write(w, frame); e != nil {
			return e
		}
		if e = write(w, sealed); e != nil {
			return e
		}
		index++
		if n == 0 {
			return nil
		}
	}
}

/** Open verifica orden, tamaño, integridad y fin de flujo; el llamador debe usar un destino temporal privado hasta el éxito.
 * @returns {error} ErrInvalid para corrupción, truncamiento, clave incorrecta o datos adicionales.
 */
func Open(w io.Writer, r io.Reader, key []byte) error {
	var e error
	header := make([]byte, len(magic)+32)
	if _, e = io.ReadFull(r, header); e != nil || string(header[:len(magic)]) != magic {
		return ErrInvalid
	}
	box, e := aead(key, header[len(magic):])
	if e != nil {
		return e
	}
	for index := uint32(0); ; index++ {
		if index == ^uint32(0) {
			return ErrInvalid
		}
		frame := make([]byte, 4)
		if _, e = io.ReadFull(r, frame); e != nil {
			return ErrInvalid
		}
		size := binary.BigEndian.Uint32(frame)
		if size > chunkSize {
			return ErrInvalid
		}
		sealed := make([]byte, int(size)+box.Overhead())
		if _, e = io.ReadFull(r, sealed); e != nil {
			return ErrInvalid
		}
		plain, e := box.Open(nil, nonce(header[len(magic):], index), sealed, aad(header, index, size))
		if e != nil {
			return ErrInvalid
		}
		if size == 0 {
			extra := make([]byte, 1)
			n, e := r.Read(extra)
			if n != 0 || e != io.EOF {
				return ErrInvalid
			}
			return nil
		}
		if e = write(w, plain); e != nil {
			return e
		}
	}
}
