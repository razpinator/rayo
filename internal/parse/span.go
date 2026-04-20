package parse

import (
	"rayo/internal/diag"
	"rayo/internal/lex"
)

// TokenSpan returns a source span for the given token (1-based line/col).
// For single-line tokens only; multi-line tokens get an approximate end.
func TokenSpan(t lex.Token) diag.Span {
	endOffset := t.Offset + len(t.Value)
	endLine := t.Line
	endCol := t.Col + len(t.Value)
	return diag.Span{
		Start: diag.SourcePos{Offset: t.Offset, Line: t.Line, Col: t.Col},
		End:   diag.SourcePos{Offset: endOffset, Line: endLine, Col: endCol},
	}
}
