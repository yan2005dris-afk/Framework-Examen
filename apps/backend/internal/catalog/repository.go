package catalog

import "context"

/** Repository consulta listas ordenadas y su relación geográfica en PostgreSQL. */
type Repository interface {
	List(context.Context, string, string) ([]string, error)
}

/** Cache conserva listas JSON; un error de caché no sustituye la fuente persistente. */
type Cache interface {
	GetOrSet(context.Context, string, func() (string, error)) (string, error)
}
