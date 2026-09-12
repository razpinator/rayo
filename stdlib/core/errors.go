package core

import (
	stderrors "errors"
	"fmt"
	"runtime"
)

// Raise creates a new error with a message. It uses errors.New so the message
// is treated as literal text (fmt.Errorf with a dynamic first argument is a
// go vet "non-constant format string" error).
func Raise(msg string) error {
	return stderrors.New(msg)
}

// Raisef creates a new error from a format string and arguments, for callers
// that genuinely need formatting.
func Raisef(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// Wrap wraps an error with a message, preserving the chain for errors.Is/As.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// Cause returns the root cause of an error.
func Cause(err error) error {
	for {
		unwrapped := Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// Unwrap returns the underlying error using standard-library semantics.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// Is reports whether any error in err's chain matches target, using standard
// errors.Is semantics rather than string-based type comparison.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As finds the first error in err's chain assignable to target.
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// WithStack wraps an error with a captured stack trace. The original error is
// preserved in the chain so errors.Is/As still work.
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	return fmt.Errorf("%w\nStack:\n%s", err, string(buf[:n]))
}
