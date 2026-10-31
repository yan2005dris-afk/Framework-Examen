package submission

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestDurationMatchesSharedFrontendCases(t *testing.T) {
	raw, err := os.ReadFile("testdata/duration_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Duration
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.Start+"/"+test.End, func(t *testing.T) {
			start, _ := time.Parse(time.DateOnly, test.Start)
			end, _ := time.Parse(time.DateOnly, test.End)
			if got := CalculateDuration(start, end); got != test.Duration {
				t.Fatalf("obtenido %+v, esperado %+v", got, test.Duration)
			}
		})
	}
}

func TestCurrentExperienceUsesEcuadorDateAndIgnoresClientDuration(t *testing.T) {
	f := completeForm()
	e := &f.Experience.Experiences[0]
	e.StartDate = "2026-10-30"
	e.Current = true
	e.EndDate = ""
	e.Duration = &Duration{Years: 999, Text: "alterado"}
	normalizeDurations(f, time.Date(2026, 10, 31, 2, 0, 0, 0, time.UTC))
	if *e.Duration != (Duration{Text: "0 días"}) {
		t.Fatalf("fecha de Ecuador esperada, obtenido %+v", e.Duration)
	}
}
