package main

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "seconds", in: 45 * time.Second, want: "45s ago"},
		{name: "minutes", in: 2*time.Minute + 5*time.Second, want: "2m ago"},
		{name: "hours", in: 3*time.Hour + 11*time.Minute, want: "3h ago"},
		{name: "days", in: 48 * time.Hour, want: "2d ago"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := formatDuration(tc.in); got != tc.want {
				t.Fatalf("formatDuration(%v): want %q, got %q", tc.in, tc.want, got)
			}
		})
	}
}

func TestAbbreviateAndTruncation(t *testing.T) {
	t.Parallel()

	if got, want := abbreviate("", 8), "-"; got != want {
		t.Fatalf("abbreviate(empty): want %q, got %q", want, got)
	}
	if got, want := abbreviate("abc", 8), "abc"; got != want {
		t.Fatalf("abbreviate(short): want %q, got %q", want, got)
	}
	if got, want := abbreviate("abcdefghijk", 8), "abcdefgh..."; got != want {
		t.Fatalf("abbreviate(long): want %q, got %q", want, got)
	}
	if got, want := abbreviate("abcdefghijk", 0), "..."; got != want {
		t.Fatalf("abbreviate(prefix=0): want %q, got %q", want, got)
	}
	if got, want := truncateWithEllipsis("abcdefghijk", 8), "abcde..."; got != want {
		t.Fatalf("truncateWithEllipsis(max=8): want %q, got %q", want, got)
	}
}
