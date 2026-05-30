package diffrender

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func intPtr(i int) *int { return &i }

func TestIsRegressed(t *testing.T) {
	drop := intPtr(-5)
	up := intPtr(2)

	cases := []struct {
		name string
		env  Envelope
		want bool
	}{
		{"score drop", Envelope{Delta: Delta{ScoreDelta: drop}}, true},
		{"new issues", Envelope{Delta: Delta{NewIssues: 1}}, true},
		{"score up no new", Envelope{Delta: Delta{ScoreDelta: up}}, false},
		{"no change", Envelope{Delta: Delta{}}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsRegressed(tc.env); got != tc.want {
				t.Fatalf("IsRegressed = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsRemoteTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{name: "http url", target: "http://example.com", want: true},
		{name: "https url", target: "https://x/y", want: true},
		{name: "bare hostname", target: "example.com", want: true},
		{name: "bare hostname with path", target: "example.com/docs", want: true},
		{name: "localhost port", target: "localhost:3000", want: true},
		{name: "ipv4 port", target: "127.0.0.1:5173", want: true},
		{name: "absolute path", target: "/path/to/file.json", want: false},
		{name: "json file", target: "file.json", want: false},
		{name: "relative path", target: "reports/current", want: false},
		{name: "unsupported scheme", target: "ftp://example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRemoteTarget(tt.target); got != tt.want {
				t.Fatalf("IsRemoteTarget(%q) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}

func TestEvaluateRegression(t *testing.T) {
	oneNew := Envelope{New: []report.IssueDetail{{Severity: "critical"}}, Delta: Delta{NewIssues: 1}}

	t.Run("fail-on-new any with new issues", func(t *testing.T) {
		got, err := EvaluateRegression(oneNew, false, "any", nil)
		if err != nil || !got {
			t.Fatalf("got %v, err %v", got, err)
		}
	})

	t.Run("no flags, no fail", func(t *testing.T) {
		got, err := EvaluateRegression(oneNew, false, "", nil)
		if err != nil || got {
			t.Fatalf("got %v, err %v", got, err)
		}
	})

	t.Run("fail-on-regression with regression", func(t *testing.T) {
		got, err := EvaluateRegression(Envelope{Delta: Delta{NewIssues: 1}}, true, "", nil)
		if err != nil || !got {
			t.Fatalf("got %v err %v", got, err)
		}
	})

	t.Run("fail-on-new severity via checker", func(t *testing.T) {
		called := false
		checker := func(_ []report.IssueDetail, minSeverity string) (bool, error) {
			called = true

			if minSeverity != "serious" {
				t.Fatalf("min = %q", minSeverity)
			}

			return true, nil
		}

		got, err := EvaluateRegression(oneNew, false, "serious", checker)
		if err != nil || !got || !called {
			t.Fatalf("got %v err %v called=%v", got, err, called)
		}
	})

	t.Run("fail-on-new severity without checker errors", func(t *testing.T) {
		_, err := EvaluateRegression(oneNew, false, "serious", nil)
		if err == nil {
			t.Fatal("expected error when checker is nil")
		}
	})

	t.Run("fail-on-new severity checker error propagates", func(t *testing.T) {
		checker := func(_ []report.IssueDetail, _ string) (bool, error) {
			return false, errors.New("bad severity")
		}

		if _, err := EvaluateRegression(oneNew, false, "bogus", checker); err == nil {
			t.Fatal("expected propagated error")
		}
	})
}

func TestRender_JSONAndText(t *testing.T) {
	env := Envelope{
		Schema: "x",
		Delta:  Delta{NewIssues: 1, FixedIssues: 0, UnchangedIssues: 2},
		New: []report.IssueDetail{
			{Severity: "critical", Title: "T", Scanner: "axe", RuleId: "r1", PageUrl: "http://a"},
		},
	}

	var buf bytes.Buffer
	if err := Render(&buf, env, FormatJSON); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), `"schema": "x"`) {
		t.Fatalf("json output missing schema: %s", buf.String())
	}

	buf.Reset()

	if err := Render(&buf, env, FormatText); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "New issues: 1") {
		t.Fatalf("text missing summary: %s", buf.String())
	}

	if !strings.Contains(buf.String(), "New Issues (Regressions):") {
		t.Fatalf("text missing section header: %s", buf.String())
	}

	buf.Reset()

	if err := Render(&buf, env, FormatMarkdown); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "## Regression Diff") {
		t.Fatalf("markdown missing heading: %s", buf.String())
	}
}

func TestRender_ScoreFormat(t *testing.T) {
	env := Envelope{
		Baseline: BaselineMeta{Score: intPtr(90)},
		Current:  CurrentMeta{Score: intPtr(85)},
	}

	var buf bytes.Buffer
	if err := Render(&buf, env, FormatText); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "90 → 85 (-5)") {
		t.Fatalf("missing score format: %s", buf.String())
	}
}
