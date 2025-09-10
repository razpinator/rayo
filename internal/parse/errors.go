package parse

import "rayo/internal/diag"

// ParseError represents a parser error with diagnostics.
type ParseError struct {
    Msg  string
    Span diag.Span
    Expected []string
    Excerpt string
}

func (e *ParseError) Error() string {
    return e.Msg
}
