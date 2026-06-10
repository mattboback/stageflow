package main

import "testing"

func TestNormalizeOutputFormat_DefaultsToText(t *testing.T) {
	format, err := normalizeOutputFormat("")
	requireNoErr(t, err)
	requireEqual(t, format, outputFormatText, "format")
}

func TestNormalizeOutputFormat_RejectsUnknownValue(t *testing.T) {
	_, err := normalizeOutputFormat("yaml")
	if err == nil {
		t.Fatal("expected invalid format error")
	}
}
