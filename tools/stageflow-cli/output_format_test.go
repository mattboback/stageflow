package main

import "testing"

func TestNormalizeOutputFormat_DefaultsToText(t *testing.T) {
	format, err := normalizeOutputFormat("", false)
	requireNoErr(t, err)
	requireEqual(t, format, outputFormatText, "format")
}

func TestNormalizeOutputFormat_JsonCompatWins(t *testing.T) {
	format, err := normalizeOutputFormat("markdown", true)
	requireNoErr(t, err)
	requireEqual(t, format, outputFormatJSON, "format")
}

func TestNormalizeOutputFormat_RejectsUnknownValue(t *testing.T) {
	_, err := normalizeOutputFormat("yaml", false)
	if err == nil {
		t.Fatal("expected invalid format error")
	}
}
