package render

import (
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestNormalizeOutputFormat_DefaultsToText(t *testing.T) {
	format, err := NormalizeFormat("")
	testsupport.RequireNoErr(t, err)
	testsupport.RequireEqual(t, format, FormatText, "format")
}

func TestNormalizeOutputFormat_RejectsUnknownValue(t *testing.T) {
	_, err := NormalizeFormat("yaml")
	if err == nil {
		t.Fatal("expected invalid format error")
	}
}
