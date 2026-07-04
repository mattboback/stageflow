// Package exitcode carries the CLI's exit-code contract through error
// returns: 0 pass, 1 policy failure (severity gate or regression), 2 error.
package exitcode

// Error wraps an error with the process exit code it should produce.
type Error struct {
	Code int
	Err  error
}

func (e Error) Error() string {
	if e.Err == nil {
		return ""
	}

	return e.Err.Error()
}

func (e Error) Unwrap() error {
	return e.Err
}
