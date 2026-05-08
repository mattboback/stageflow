package main

import (
	"errors"
	"testing"
)

func TestWrapRenderError_NilReturnsNil(t *testing.T) {
	if got := wrapRenderError(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWrapRenderError_ExitCode1IsPreserved(t *testing.T) {
	inner := errors.New("threshold exceeded")
	err := wrapRenderError(exitCodeError{Code: 1, Err: inner})

	var ece exitCodeError
	if !errors.As(err, &ece) {
		t.Fatalf("expected exitCodeError, got %T: %v", err, err)
	}

	if ece.Code != 1 {
		t.Errorf("expected exit code 1, got %d", ece.Code)
	}
}

func TestWrapRenderError_PlainErrorBecomesExitCode2(t *testing.T) {
	err := wrapRenderError(errors.New("render failed"))

	var ece exitCodeError
	if !errors.As(err, &ece) {
		t.Fatalf("expected exitCodeError, got %T: %v", err, err)
	}

	if ece.Code != 2 {
		t.Errorf("expected exit code 2, got %d", ece.Code)
	}
}

func TestWrapRenderError_ExitCode2IsWrappedAsCode2(t *testing.T) {
	original := exitCodeError{Code: 2, Err: errors.New("original")}
	err := wrapRenderError(original)

	var ece exitCodeError
	if !errors.As(err, &ece) {
		t.Fatalf("expected exitCodeError, got %T: %v", err, err)
	}

	if ece.Code != 2 {
		t.Errorf("expected exit code 2, got %d", ece.Code)
	}
}
