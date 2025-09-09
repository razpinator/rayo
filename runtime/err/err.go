package err

import "fmt"

// Wrap wraps an error with a message.
func Wrap(e error, msg string) error {
    return fmt.Errorf("%s: %w", msg, e)
}

// Cause returns the root cause of an error.
func Cause(e error) error {
    for {
        unwrapped := Unwrap(e)
        if unwrapped == nil {
            return e
        }
        e = unwrapped
    }
}

// Unwrap returns the underlying error.
func Unwrap(e error) error {
    type unwrapper interface {
        Unwrap() error
    }
    if u, ok := e.(unwrapper); ok {
        return u.Unwrap()
    }
    return nil
}

// Is reports whether err is of the same type as target.
func Is(err, target error) bool {
    return fmt.Sprintf("%T", err) == fmt.Sprintf("%T", target)
}
