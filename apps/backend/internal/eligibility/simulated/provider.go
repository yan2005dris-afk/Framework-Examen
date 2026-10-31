package simulated

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/eligibility"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"io"
	"os"
)

/** Provider carga una fixture operacional explícita por UUID, sin una aceptación por defecto. */
type Provider struct{ accounts map[string]map[string]bool }

func Load(path string) (*Provider, error) {
	p := &Provider{accounts: map[string]map[string]bool{}}
	if path == "" {
		return p, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return nil, submission.ErrInvalidContent
	}
	var accounts map[string]map[string]*bool
	d := json.NewDecoder(bytes.NewReader(raw))
	if err = d.Decode(&accounts); err != nil {
		return nil, err
	}
	if d.Decode(new(json.RawMessage)) != io.EOF {
		return nil, submission.ErrInvalidContent
	}
	if accounts == nil {
		return nil, submission.ErrInvalidContent
	}
	for account, sectors := range accounts {
		id, err := uuid.Parse(account)
		if err != nil || id == uuid.Nil || id.String() != account {
			return nil, submission.ErrInvalidContent
		}
		if sectors == nil {
			return nil, submission.ErrInvalidContent
		}
		p.accounts[account] = map[string]bool{}
		for sector, value := range sectors {
			if sector != "afiliados" && sector != "jubilados" && sector != "empleadores" {
				return nil, submission.ErrInvalidContent
			}
			if value == nil {
				return nil, submission.ErrInvalidContent
			}
			p.accounts[account][sector] = *value
		}
	}
	return p, nil
}
func (p *Provider) Resolve(ctx context.Context, account uuid.UUID, sector string) (eligibility.Condition, error) {
	if err := ctx.Err(); err != nil {
		return eligibility.Condition{}, err
	}
	if p == nil {
		return eligibility.Condition{}, submission.ErrEligibilityUnavailable
	}
	value, ok := p.accounts[account.String()][sector]
	if !ok {
		return eligibility.Condition{}, submission.ErrEligibilityUnavailable
	}
	return eligibility.Condition{Eligible: value, Source: "simulacion_controlada_servidor"}, nil
}
