package err

import (
	stderrors "errors"
	"fmt"
)

// Wrap wraps an error with a message, preserving the chain so errors.Is and
// errors.As continue to work through the wrap.
func Wrap(e error, msg string) error {
	if e == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, e)
}

// Cause returns the root cause of an error by unwrapping until it can no longer
// be unwrapped.
func Cause(e error) error {
	for {
		unwrapped := Unwrap(e)
		if unwrapped == nil {
			return e
		}
		e = unwrapped
	}
}

// Unwrap returns the underlying error, delegating to the standard library so
// both single- and multi-wrap chains behave consistently.
func Unwrap(e error) error {
	return stderrors.Unwrap(e)
}

// Is reports whether any error in err's chain matches target, using standard
// errors.Is semantics (value identity plus custom Is methods). This replaces
// the previous string-based type comparison, which incorrectly treated any two
// errors of the same concrete type as equal.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As finds the first error in err's chain that matches target's type and, if
// found, sets target and returns true. It mirrors errors.As.
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// New creates a new error with the given text. It is provided so generated code
// and Rayo programs have a single, consistent constructor.
func New(text string) error {
	return stderrors.New(text)
}
