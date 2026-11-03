package backup

import (
	"bytes"
	"fmt"
	"testing"
)

func TestEncryptedBackupRoundTripAndUniqueCiphertext(t *testing.T) {
	for _, size := range []int{0, 1, chunkSize, chunkSize + 99} {
		t.Run(fmt.Sprintf("bytes-%d", size), func(t *testing.T) {
			data := bytes.Repeat([]byte("x"), size)
			key := bytes.Repeat([]byte{7}, 32)
			var a, b, plain bytes.Buffer
			if e := Seal(&a, bytes.NewReader(data), key); e != nil {
				t.Fatal(e)
			}
			if e := Seal(&b, bytes.NewReader(data), key); e != nil {
				t.Fatal(e)
			}
			if bytes.Equal(a.Bytes(), b.Bytes()) {
				t.Fatal("nonce reutilizado")
			}
			if e := Open(&plain, bytes.NewReader(a.Bytes()), key); e != nil || !bytes.Equal(data, plain.Bytes()) {
				t.Fatal("no recuperó", e)
			}
		})
	}
}
func TestEncryptedBackupRejectsTamperTruncationAndWrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{3}, 32)
	var b bytes.Buffer
	if e := Seal(&b, bytes.NewReader(bytes.Repeat([]byte("private"), chunkSize/3)), key); e != nil {
		t.Fatal(e)
	}
	source := b.Bytes()
	for _, kind := range []string{"header", "payload", "tail", "truncated", "missing-final", "extra", "reordered", "key", "short-key", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			v := append([]byte(nil), source...)
			k := append([]byte(nil), key...)
			switch kind {
			case "header":
				v[0] ^= 1
			case "payload":
				v[len(magic)+25] ^= 1
			case "tail":
				v[len(v)-1] ^= 1
			case "truncated":
				v = v[:len(v)-1]
			case "missing-final":
				v = v[:len(v)-20]
			case "extra":
				v = append(v, 1)
			case "reordered":
				start := len(magic) + 32
				frame := chunkSize + 20
				first := append([]byte(nil), v[start:start+frame]...)
				copy(v[start:start+frame], v[start+frame:start+2*frame])
				copy(v[start+frame:start+2*frame], first)
			case "key":
				k[0] ^= 1
			case "short-key":
				k = k[:31]
			case "oversize":
				v[len(magic)+32] = 255
			}
			var out bytes.Buffer
			if Open(&out, bytes.NewReader(v), k) == nil {
				t.Fatal("aceptó respaldo inválido")
			}
		})
	}
}
