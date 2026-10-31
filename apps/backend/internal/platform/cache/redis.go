package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

/**
 * RedisClient es el adaptador de caché que usa Redis como backend.
 * Implementa la interfaz Cache.
 */
type RedisClient struct {
	client *redis.Client
}

/**
 * NewRedis abre la conexión a Redis y verifica que esté disponible con un ping.
 *
 * @param {string} url - URL de conexión en formato redis://[:password@]host[:port][/db].
 * @returns {*RedisClient} Cliente listo para usar.
 * @throws {error} Si no se puede parsear la URL o Redis no responde.
 */
func NewRedis(ctx context.Context, url string) (*RedisClient, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("cache: URL de Redis inválida: %w", err)
	}

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("cache: no se pudo conectar a Redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

/**
 * Get devuelve el valor de la clave o ErrNotFound si no existe.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} key - Clave a consultar.
 * @returns {string} Valor almacenado.
 * @throws {error} ErrNotFound o error de Redis.
 */
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("cache: Get(%q): %w", key, err)
	}
	return val, nil
}

/**
 * Set guarda el valor con el TTL indicado.
 * Si ttl es cero, la clave no expira.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} key - Clave a guardar.
 * @param {string} value - Valor a almacenar.
 * @param {Duration} ttl - Tiempo de vida de la clave.
 * @throws {error} Error de Redis.
 */
func (r *RedisClient) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache: Set(%q): %w", key, err)
	}
	return nil
}

/**
 * Incr incrementa el contador de la clave de forma atómica.
 * Si la clave es nueva y se pasa un TTL positivo, lo asigna en el mismo pipeline.
 * Esto garantiza que no haya ventana de tiempo en que la clave exista sin expiración.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} key - Clave del contador.
 * @param {Duration} ttl - TTL a asignar cuando la clave se crea por primera vez.
 * @returns {int64} Valor del contador después del incremento.
 * @throws {error} Error de Redis.
 */
func (r *RedisClient) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	// Usamos un pipeline para hacer INCR + EXPIRE atómicamente cuando aplica.
	// INCR crea la clave con valor 1 si no existe; EXPIRE solo se aplica
	// cuando el contador acaba de crearse (valor == 1) para no reiniciar el TTL
	// en incrementos posteriores dentro de la misma ventana.
	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("cache: Incr(%q): %w", key, err)
	}

	return incrCmd.Val(), nil
}

/**
 * Delete elimina la clave. No devuelve error si la clave no existía.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} key - Clave a eliminar.
 * @throws {error} Error de Redis.
 */
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache: Delete(%q): %w", key, err)
	}
	return nil
}

/**
 * Close cierra la conexión con Redis.
 *
 * @throws {error} Error al cerrar la conexión.
 */
func (r *RedisClient) Close() error {
	return r.client.Close()
}
