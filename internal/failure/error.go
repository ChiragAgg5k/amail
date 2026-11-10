package failure

import "errors"

// Error is a failure with an exit code and an optional hint on what to do next.
type Error struct {
	Code    Code
	Message string
	Hint    string
	Cause   error
}

// New builds a failure from a message.
func New(code Code, message, hint string) *Error {
	return &Error{Code: code, Message: message, Hint: hint}
}

// Wrap gives an existing error an exit code and a hint.
func Wrap(code Code, cause error, hint string) *Error {
	return &Error{Code: code, Hint: hint, Cause: cause}
}

func (failure *Error) Error() string {
	switch {
	case failure.Cause == nil:
		return failure.Message
	case failure.Message == "":
		return failure.Cause.Error()
	}
	return failure.Message + ": " + failure.Cause.Error()
}

func (failure *Error) Unwrap() error {
	return failure.Cause
}

// CodeOf returns the exit code for an error: Success for nil, General for an error without one.
func CodeOf(err error) Code {
	if err == nil {
		return Success
	}
	var failure *Error
	if errors.As(err, &failure) {
		return failure.Code
	}
	return General
}

// HintOf returns the hint carried by an error, or an empty string.
func HintOf(err error) string {
	var failure *Error
	if errors.As(err, &failure) {
		return failure.Hint
	}
	return ""
}
