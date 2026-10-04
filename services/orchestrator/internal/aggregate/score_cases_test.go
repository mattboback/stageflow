package aggregate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

// The web report recomputes the site score in clients/web/app/lib/report/score.ts
// so it can say what lowers it. Both sides run these cases; a formula change on
// either side fails until the fixture and the other side follow.
func TestCalculateAccessibilityScoreMatchesSharedGoldenCases(t *testing.T) {
	data, readErr := os.ReadFile(
		filepath.Join(reportContractRoot, "libs/contracts/report/fixtures/score-cases.json"),
	)
	if readErr != nil {
		t.Fatal(readErr)
	}

	var fixture struct {
		Cases []struct {
			Name   string `json:"name"`
			Counts struct {
				Critical int  `json:"critical"`
				Serious  int  `json:"serious"`
				Moderate int  `json:"moderate"`
				Minor    int  `json:"minor"`
				Info     *int `json:"info"`
			} `json:"counts"`
			PagesScanned int `json:"pagesScanned"`
			Score        int `json:"score"`
		} `json:"cases"`
	}

	unmarshalErr := json.Unmarshal(data, &fixture)
	if unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}

	if len(fixture.Cases) == 0 {
		t.Fatal("score-cases.json has no cases")
	}

	for _, golden := range fixture.Cases {
		t.Run(golden.Name, func(t *testing.T) {
			counts := report.SeverityCounts{
				Critical: golden.Counts.Critical,
				Serious:  golden.Counts.Serious,
				Moderate: golden.Counts.Moderate,
				Minor:    golden.Counts.Minor,
				Info:     golden.Counts.Info,
			}

			score, _ := calculateAccessibilityScore(counts, golden.PagesScanned)
			if score != golden.Score {
				t.Fatalf("score = %d, want %d", score, golden.Score)
			}
		})
	}
}
