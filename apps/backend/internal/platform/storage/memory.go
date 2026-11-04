package storage

import (
	"context"
	"io"
	"sync"
	"time"
)

/** Memory es una implementación en memoria de document.Storage para pruebas unitarias. */
type Memory struct {
	mu      sync.RWMutex
	data    map[string][]byte
	deleted map[string]bool
}

/** NewMemory inicializa el almacenamiento en memoria. */
func NewMemory() *Memory {
	return &Memory{
		data:    make(map[string][]byte),
		deleted: make(map[string]bool),
	}
}

func (m *Memory) Save(ctx context.Context, key string, content io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	bytes, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = bytes
	delete(m.deleted, key)
	return nil
}

func (m *Memory) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	m.deleted[key] = true
	return nil
}

func (m *Memory) PresignGet(ctx context.Context, key, name string, ttl time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "http://memory.storage.test/" + key + "?filename=" + name, nil
}

func (m *Memory) Ready(ctx context.Context) error {
	return nil
}
