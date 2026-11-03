package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/telemetry"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

/**
 * DBTX es el contrato mínimo que deben cumplir el pool y una transacción.
 * Permite que los repositorios funcionen igual dentro o fuera de una
 * transacción.
 */
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type transactionKey struct{}

/** WithTransaction comparte la transacción entre adaptadores sin exponerla al dominio.
 * @returns {context.Context} Contexto limitado al callback transaccional.
 */
func WithTransaction(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, transactionKey{}, tx)
}

/** Executor reutiliza la conexión transaccional o el pool fuera de una transacción.
 * @returns {DBTX} Ejecutor para las queries generadas.
 */
func Executor(ctx context.Context, fallback DBTX) DBTX {
	if tx, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		return tx
	}
	return fallback
}

/** Transaction permite a adaptadores con queries preconstruidas compartir la conexión.
 * @returns {pgx.Tx, bool} Transacción vigente solo dentro del callback.
 */
func Transaction(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(transactionKey{}).(pgx.Tx)
	return tx, ok
}

/**
 * Connect crea el pool de conexiones de PostgreSQL y verifica la
 * conectividad mediante un ping con timeout corto.
 *
 * @param {Context} ctx - Contexto de la aplicación.
 * @param {string} url - Cadena de conexión a PostgreSQL.
 * @returns {pgxpool.Pool} Pool listo para usar.
 * @throws {Error} Si la URL es inválida o PostgreSQL no responde.
 */
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("configuración de base de datos inválida")
	}

	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("no se pudo crear el pool de conexiones")
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, errors.New("no hay conexión con PostgreSQL")
	}

	return pool, nil
}

/**
 * TxManager ejecuta bloques de trabajo dentro de una transacción de PostgreSQL.
 */
type TxManager struct {
	pool *pgxpool.Pool
}

/**
 * NewTxManager crea el gestor de transacciones del pool indicado.
 *
 * @param {pgxpool.Pool} pool - Pool de conexiones de la aplicación.
 * @returns {TxManager} Gestor listo para usar.
 */
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

/**
 * WithinTx ejecuta la función dentro de una transacción y confirma el cambio
 * si no hay error; en caso contrario, revierte todo.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {function} fn - Trabajo a ejecutar con la transacción abierta.
 * @returns {void}
 * @throws {Error} El error original si la transacción falla o se revierte.
 */
func (m *TxManager) WithinTx(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	start := time.Now()
	defer func() { telemetry.Default.Observe("database", "transaction", telemetry.Result(err), time.Since(start)) }()
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar la transacción: %w", err)
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	a := domain.AuditFromContext(ctx)
	actor, request, role := "", "", ""
	if a.ActorID != uuid.Nil {
		actor = a.ActorID.String()
		role = a.Role
	}
	if a.RequestID != uuid.Nil {
		request = a.RequestID.String()
	}
	// SET LOCAL no adquiere snapshot: el callback aún puede pedir REPEATABLE READ.
	// Los UUID serializados y el rol cerrado no admiten comillas ni SQL arbitrario.
	if role != "" && role != "postulante" && role != "analista" && role != "administrador" {
		return errors.New("rol de auditoría inválido")
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL audit.actor_id TO '%s'; SET LOCAL audit.actor_role TO '%s'; SET LOCAL audit.request_id TO '%s'", actor, role, request)); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("no se pudo confirmar la transacción: %w", err)
	}

	return nil
}
