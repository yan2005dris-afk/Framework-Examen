-- name: ListCatalogValues :many
SELECT value FROM catalogs WHERE kind = $1 AND parent_key = $2
ORDER BY position, value;
