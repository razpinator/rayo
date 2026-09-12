package parse

import (
	"strings"

	"rayo/internal/diag"
	"rayo/internal/lex"
)

// TokenSpan returns a source span for the given token using 1-based line/col.
//
// The lexer records the START line/col of every token. The end position is
// derived from the token's value, correctly accounting for multi-line tokens
// (for example multi-line string literals).
func TokenSpan(t lex.Token) diag.Span {
	start := diag.SourcePos{Offset: t.Offset, Line: t.Line, Col: t.Col}

	endLine := t.Line
	endCol := t.Col + len(t.Value)
	if nl := strings.Count(t.Value, "\n"); nl > 0 {
		endLine += nl
		// Column resets on the final line to the length after the last newline.
		last := t.Value[strings.LastIndex(t.Value, "\n")+1:]
		endCol = 1 + len(last)
	}

	return diag.Span{
		Start: start,
		End:   diag.SourcePos{Offset: t.Offset + len(t.Value), Line: endLine, Col: endCol},
	}
}

// MergeSpans returns a span covering both a and b. Zero spans are ignored so a
// node can merge a real start span with a not-yet-populated end span.
func MergeSpans(a, b diag.Span) diag.Span {
	if a.IsZero() {
		return b
	}
	if b.IsZero() {
		return a
	}
	out := a
	if b.End.Offset > out.End.Offset {
		out.End = b.End
	}
	if b.Start.Offset < out.Start.Offset {
		out.Start = b.Start
	}
	return out
}
