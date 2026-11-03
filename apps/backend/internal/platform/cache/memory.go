package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
 * MemoryCache implementa Cache y ScriptCache en memoria para pruebas unitarias.
 */
type MemoryCache struct {
	mu   sync.Mutex
	data map[string]string
}

/**
 * NewMemoryCache construye una instancia limpia en memoria.
 */
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data: make(map[string]string),
	}
}

func (m *MemoryCache) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (m *MemoryCache) Set(_ context.Context, key, val string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = val
	return nil
}

func (m *MemoryCache) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var cur int64
	if v, ok := m.data[key]; ok {
		cur, _ = strconv.ParseInt(v, 10, 64)
	}
	cur++
	m.data[key] = strconv.FormatInt(cur, 10)
	return cur, nil
}

func (m *MemoryCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MemoryCache) Close() error {
	return nil
}

/**
 * RunScript simula la ejecución atómica de scripts de Redis en memoria.
 */
func (m *MemoryCache) RunScript(_ context.Context, script string, keys []string, args ...interface{}) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. issueOTP
	if strings.Contains(script, "ARGV[5]") || strings.Contains(script, "sends") {
		if len(keys) < 3 || len(args) < 5 {
			return 0, errors.New("argumentos insuficientes para issueOTP")
		}
		maxSends, _ := strconv.Atoi(fmt.Sprint(args[3]))
		maxAttempts, _ := strconv.Atoi(fmt.Sprint(args[4]))

		if old, ok := m.data[keys[0]]; ok {
			var st otpState
			if err := json.Unmarshal([]byte(old), &st); err == nil {
				if st.Attempts >= maxAttempts {
					return -2, nil
				}
			}
		}
		if _, exists := m.data[keys[1]]; exists {
			return -1, nil
		}
		var sends int
		if sStr, ok := m.data[keys[2]]; ok {
			sends, _ = strconv.Atoi(sStr)
		}
		if sends >= maxSends {
			return -1, nil
		}
		sends++
		m.data[keys[2]] = strconv.Itoa(sends)
		m.data[keys[0]] = fmt.Sprint(args[0])
		m.data[keys[1]] = "1"
		return 1, nil
	}

	// 2. consumeOTP
	if strings.Contains(script, "c.binding==ARGV[2]") {
		if len(keys) < 1 || len(args) < 4 {
			return 0, errors.New("argumentos insuficientes para consumeOTP")
		}
		raw, ok := m.data[keys[0]]
		if !ok {
			return 0, nil
		}
		var st otpState
		if err := json.Unmarshal([]byte(raw), &st); err != nil {
			return 0, nil
		}
		maxAttempts, _ := strconv.Atoi(fmt.Sprint(args[3]))
		if st.Attempts >= maxAttempts {
			return -2, nil
		}
		id := fmt.Sprint(args[0])
		bound := fmt.Sprint(args[1])
		dig := fmt.Sprint(args[2])

		if st.ID == id && st.Binding == bound && st.Digest == dig {
			delete(m.data, keys[0])
			return 1, nil
		}

		st.Attempts++
		updated, _ := json.Marshal(st)
		m.data[keys[0]] = string(updated)
		if st.Attempts >= maxAttempts {
			return -2, nil
		}
		return 0, nil
	}

	// 3. cancelOTP
	if strings.Contains(script, "cancelOTP") || strings.Contains(script, "c.id==ARGV[1]") {
		if len(keys) < 1 || len(args) < 1 {
			return 0, nil
		}
		raw, ok := m.data[keys[0]]
		if !ok {
			return 0, nil
		}
		var st otpState
		if err := json.Unmarshal([]byte(raw), &st); err == nil && st.ID == fmt.Sprint(args[0]) {
			delete(m.data, keys[0])
			return 1, nil
		}
		return 0, nil
	}

	// 4. INCR default
	if strings.Contains(script, "INCR") && len(keys) > 0 {
		var cur int64
		if v, ok := m.data[keys[0]]; ok {
			cur, _ = strconv.ParseInt(v, 10, 64)
		}
		cur++
		m.data[keys[0]] = strconv.FormatInt(cur, 10)
		return cur, nil
	}

	return 0, nil
}
