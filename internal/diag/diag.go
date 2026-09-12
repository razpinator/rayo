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

// IsZero reports whether the span carries no position information.
func (s Span) IsZero() bool {
	return s == Span{}
}

// Severity classifies the importance of a diagnostic.
type Severity int

const (
	// SeverityError marks a diagnostic that prevents successful compilation.
	SeverityError Severity = iota
	// SeverityWarning marks a diagnostic that is advisory (e.g. unused binding).
	SeverityWarning
	// SeverityInfo marks an informational hint.
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return "error"
	}
}

// Reporter handles reporting diagnostics.
//
// Report is the minimal, backward-compatible entry point (defaults to
// SeverityError). Reporters that care about severity may also implement
// SeverityReporter; callers should use ReportAt when available.
type Reporter interface {
	Report(span Span, msg string)
}

// SeverityReporter is an optional interface a Reporter may implement to receive
// a severity classification alongside the span and message.
type SeverityReporter interface {
	Reporter
	ReportAt(span Span, sev Severity, msg string)
}

// ReportAt reports a diagnostic with an explicit severity. If rep implements
// SeverityReporter the severity is preserved; otherwise it falls back to the
// plain Report method so existing reporters keep working.
func ReportAt(rep Reporter, span Span, sev Severity, msg string) {
	if sr, ok := rep.(SeverityReporter); ok {
		sr.ReportAt(span, sev, msg)
		return
	}
	rep.Report(span, msg)
}
