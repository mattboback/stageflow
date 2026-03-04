package main

import (
	"reflect"
	"testing"
)

func requireNoErr(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireEqual[T comparable](t *testing.T, got, want T, label string) {
	t.Helper()

	if got != want {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}

func requireDeepEqual(t *testing.T, got, want any, label string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}

func requireBoolPtr(t *testing.T, got *bool, want bool, label string) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s = nil, want %v", label, want)
	}

	if *got != want {
		t.Fatalf("%s = %v, want %v", label, *got, want)
	}
}
