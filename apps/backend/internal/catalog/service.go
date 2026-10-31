package catalog

import (
	"context"
	"encoding/json"
	"strings"
)

/** Service publica las listas del RF con PostgreSQL como fuente persistente. */
type Service struct {
	repo  Repository
	cache Cache
}

/** NewService recibe puertos independientes de Redis, SQLC y HTTP. */
func NewService(repo Repository, cache Cache) *Service { return &Service{repo: repo, cache: cache} }

/** Countries devuelve los nombres que guarda el EF-12, sin reducir el catálogo a una muestra. */
func (s *Service) Countries(ctx context.Context) ([]string, error) {
	return s.list(ctx, "countries", "", "catalog:countries")
}

/** Provinces devuelve la lista ecuatoriana o una lista vacía para un país extranjero. */
func (s *Service) Provinces(ctx context.Context, country string) ([]string, error) {
	country, err := s.country(ctx, country)
	if err != nil {
		return nil, err
	}
	if country != "EC" {
		return []string{}, nil
	}
	return s.list(ctx, "provinces", "EC", "catalog:provinces:EC")
}

/** Cities valida la relación país/provincia antes de consultar o cachear sus cantones. */
func (s *Service) Cities(ctx context.Context, country, province string) ([]string, error) {
	country, err := s.country(ctx, country)
	if err != nil {
		return nil, err
	}
	province = strings.TrimSpace(province)
	if province == "" {
		return nil, ErrInvalidFilter
	}
	if country != "EC" {
		return []string{}, nil
	}
	provinces, err := s.Provinces(ctx, "EC")
	if err != nil {
		return nil, err
	}
	province, ok := match(provinces, province)
	if !ok {
		return nil, ErrInvalidFilter
	}
	return s.list(ctx, "cities", "EC/"+province, "catalog:cities:EC:"+province)
}

/** Options publica los catálogos cerrados de dominio con sus nombres del contrato EF-12. */
func (s *Service) Options(ctx context.Context, name string) ([]string, error) {
	if !validOption(name) {
		return nil, ErrUnknownCatalog
	}
	return s.list(ctx, name, "", "catalog:options:"+name)
}

func validOption(name string) bool {
	switch name {
	case "generos", "estadosCiviles", "respuestas", "parentescos", "instituciones", "sectores", "niveles", "tiposInstitucion", "areas", "tiposExperiencia", "tiposPublicacion", "tiposEvento":
		return true
	default:
		return false
	}
}

func (s *Service) country(ctx context.Context, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrInvalidFilter
	}
	// EC es el alias del contrato HTTP; el formulario sigue guardando "Ecuador".
	if strings.EqualFold(value, "EC") || strings.EqualFold(value, "Ecuador") {
		return "EC", nil
	}
	countries, err := s.Countries(ctx)
	if err != nil {
		return "", err
	}
	if name, ok := match(countries, value); ok {
		return name, nil
	}
	return "", ErrInvalidFilter
}

func match(values []string, value string) (string, bool) {
	for _, candidate := range values {
		if strings.EqualFold(candidate, value) {
			return candidate, true
		}
	}
	return "", false
}

func (s *Service) list(ctx context.Context, kind, parent, key string) ([]string, error) {
	fetch := func() (string, error) {
		items, err := s.repo.List(ctx, kind, parent)
		if err != nil {
			return "", err
		}
		if items == nil {
			items = []string{}
		}
		data, err := json.Marshal(items)
		return string(data), err
	}
	var raw string
	var err error
	if s.cache == nil {
		raw, err = fetch()
	} else {
		raw, err = s.cache.GetOrSet(ctx, key, fetch)
	}
	if err != nil {
		return nil, err
	}
	var items []string
	if json.Unmarshal([]byte(raw), &items) != nil || items == nil {
		raw, err = fetch()
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &items); err != nil {
			return nil, err
		}
	}
	return items, nil
}
