package render

import (
	"errors"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
)

func TestWrapRenderError_NilReturnsNil(t *testing.T) {
	if got := WrapError(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWrapRenderError_ExitCode1IsPreserved(t *testing.T) {
	inner := errors.New("threshold exceeded")
	err := WrapError(exitcode.Error{Code: 1, Err: inner})

	var ece exitcode.Error
	if !errors.As(err, &ece) {
		t.Fatalf("expected exitcode.Error, got %T: %v", err, err)
	}

	if ece.Code != 1 {
		t.Errorf("expected exit code 1, got %d", ece.Code)
	}
}

func TestWrapRenderError_PlainErrorBecomesExitCode2(t *testing.T) {
	err := WrapError(errors.New("render failed"))

	var ece exitcode.Error
	if !errors.As(err, &ece) {
		t.Fatalf("expected exitcode.Error, got %T: %v", err, err)
	}

	if ece.Code != 2 {
		t.Errorf("expected exit code 2, got %d", ece.Code)
	}
}

func TestWrapRenderError_ExitCode2IsWrappedAsCode2(t *testing.T) {
	original := exitcode.Error{Code: 2, Err: errors.New("original")}
	err := WrapError(original)

	var ece exitcode.Error
	if !errors.As(err, &ece) {
		t.Fatalf("expected exitcode.Error, got %T: %v", err, err)
	}

	if ece.Code != 2 {
		t.Errorf("expected exit code 2, got %d", ece.Code)
	}
}
