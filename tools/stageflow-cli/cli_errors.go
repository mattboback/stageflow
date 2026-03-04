package main

type exitCodeError struct {
	Code int
	Err  error
}

func (e exitCodeError) Error() string {
	if e.Err == nil {
		return ""
	}

	return e.Err.Error()
}

func (e exitCodeError) Unwrap() error {
	return e.Err
}
