package diag

// SourcePos represents a position in the source file.
type SourcePos struct {
    Offset int // byte offset
    Line   int // 1-based line number
    Col    int // 1-based column number
}

// Span represents a range in the source file.
type Span struct {
    Start SourcePos
    End   SourcePos
}

// Reporter handles reporting diagnostics.
type Reporter interface {
    Report(span Span, msg string)
}
