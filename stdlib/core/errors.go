package core

import (
    "fmt"
    "runtime"
)

// Raise creates a new error with a message.
func Raise(msg string) error {
    return fmt.Errorf(msg)
}

// Wrap wraps an error with a message.
func Wrap(err error, msg string) error {
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

// Unwrap returns the underlying error.
func Unwrap(err error) error {
    type unwrapper interface {
        Unwrap() error
    }
    if u, ok := err.(unwrapper); ok {
        return u.Unwrap()
    }
    return nil
}

// Is reports whether err is of the same type as target.
func Is(err, target error) bool {
    return fmt.Sprintf("%T", err) == fmt.Sprintf("%T", target)
}

// WithStack wraps an error with a stack trace.
func WithStack(err error) error {
    buf := make([]byte, 1024)
    n := runtime.Stack(buf, false)
    return fmt.Errorf("%w\nStack:\n%s", err, string(buf[:n]))
}
