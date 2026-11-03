package results_test

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results/testfixture"
	"strings"
	"testing"
)

func TestBuildRulesAndSources(t *testing.T) {
	for _, name := range []string{"pendiente", "otra_convocatoria", "sello", "reconsideracion", "consolidado_obsoleto", "sin_reglas", "sin_orden", "orden_repetido", "psico_sin_escala"} {
		t.Run(name, func(t *testing.T) {
			period := uuid.New()
			p := testfixture.Policy(period)
			s, err := testfixture.Source(period, p, 80)
			if err != nil {
				t.Fatal(err)
			}
			rules := results.Rules{Order: []string{"resultados"}, Reference: "Acta QA no oficial"}
			switch name {
			case "pendiente":
				s.Theory = nil
			case "otra_convocatoria":
				other := uuid.New()
				s.Snapshot.CompetitionID = &other
			case "sello":
				s.Snapshot.Hash = "cambiado"
			case "reconsideracion":
				s.Appeals = []evaluation.Appeal{{Status: "recibida"}}
			case "consolidado_obsoleto":
				v := s.State.Results["resultados"]
				v.Breakdown[0].ResultID = uuid.New()
				s.State.Results["resultados"] = v
			case "sin_reglas":
				p = nil
			case "sin_orden":
				rules.Order = nil
			case "orden_repetido":
				rules.Order = []string{"resultados", "resultados"}
			case "psico_sin_escala":
				v := s.State.Results["oposicion_psicotecnica"]
				v.Maximum = nil
				s.State.Results[v.Phase] = v
			}
			if _, _, err = results.Build(period, "afiliados", rules, p, []results.Source{s}); err == nil {
				t.Fatal("aceptó fuente/reglas inválidas")
			}
		})
	}
}
func TestListRankingHistoryAndNamePrivacy(t *testing.T) {
	period := uuid.New()
	p := testfixture.Policy(period)
	sources := []results.Source{}
	for _, score := range []int64{80, 90, 80} {
		s, err := testfixture.Source(period, p, score)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, s)
	}
	rules := results.Rules{Order: []string{"resultados", "meritos"}, Reference: "Orden QA"}
	items, hash, err := results.Build(period, "afiliados", rules, p, sources)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].Rank != 1 || items[1].Rank != 2 || items[2].Rank != 2 || !items[1].Tied || !items[2].Tied {
		t.Fatal("desempate inventado", items)
	}
	sources[0], sources[2] = sources[2], sources[0]
	_, hash2, err := results.Build(period, "afiliados", rules, p, sources)
	if err != nil || hash != hash2 {
		t.Fatal("hash depende del orden de lectura", err)
	}
	list := &results.List{ID: uuid.New(), Entries: items}
	raw, err := results.EncodeList(list)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := results.DecodeList(raw)
	if err != nil || len(stored.Entries) != 3 {
		t.Fatal("se perdió snapshot histórico", err)
	}
	names, err := results.PublicNames(items)
	if err != nil || len(names) != 3 {
		t.Fatal(err)
	}
	public, _ := json.Marshal(names)
	for _, private := range []string{"privado@example.test", "NO-PUBLICAR", "0000000000", items[0].SubmissionID.String(), "puntaje"} {
		if strings.Contains(string(public), private) {
			t.Fatal("dato fuera de consentimiento", private)
		}
	}
	items[0].NameConsent = false
	if _, err = results.PublicNames(items); !errors.Is(err, results.ErrPrivacy) {
		t.Fatal("publicó sin autorización", err)
	}
	sources[1].Snapshot.ApplicantID = sources[0].Snapshot.ApplicantID
	if _, _, err = results.Build(period, "afiliados", rules, p, sources); !errors.Is(err, results.ErrConflict) {
		t.Fatal("duplicó persona", err)
	}
}
func TestMeritExclusionDoesNotInventOpposition(t *testing.T) {
	period := uuid.New()
	p := testfixture.Policy(period)
	s, err := testfixture.Source(period, p, 10)
	if err != nil {
		t.Fatal(err)
	}
	s.Merit.Decision = "excluido"
	s.Theory = nil
	s.State = evaluation.State{}
	items, _, err := results.Build(period, "afiliados", results.Rules{Order: []string{"resultados"}, Reference: "QA"}, nil, []results.Source{s})
	if err != nil || items[0].Eligible || len(items[0].Results) != 1 {
		t.Fatal("necesitó una nota ficticia para exclusión", err)
	}
}
