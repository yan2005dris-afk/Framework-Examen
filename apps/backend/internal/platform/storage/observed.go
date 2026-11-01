package storage

import (
	"context"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/telemetry"
	"io"
	"time"
)

type Observed struct{ Next document.Storage }

func (s *Observed) Save(ctx context.Context, key string, r io.Reader) (err error) {
	t := time.Now()
	defer func() { telemetry.Default.Observe("storage", "save", telemetry.Result(err), time.Since(t)) }()
	return s.Next.Save(ctx, key, r)
}
func (s *Observed) Delete(ctx context.Context, key string) (err error) {
	t := time.Now()
	defer func() { telemetry.Default.Observe("storage", "delete", telemetry.Result(err), time.Since(t)) }()
	return s.Next.Delete(ctx, key)
}
func (s *Observed) PresignGet(ctx context.Context, key, name string, ttl time.Duration) (url string, err error) {
	t := time.Now()
	defer func() { telemetry.Default.Observe("storage", "download", telemetry.Result(err), time.Since(t)) }()
	return s.Next.PresignGet(ctx, key, name, ttl)
}
