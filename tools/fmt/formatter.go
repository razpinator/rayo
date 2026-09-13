// Package fmt implements Rayo's stable, idempotent source code formatter.
//
// The formatter works on the lexer token stream rather than the AST. The
// parser is intentionally lenient (it drops trivia and skips some forms), so a
// token-stream formatter is the only representation that can round-trip
// arbitrary source without losing user code. Working from tokens also lets the
// formatter preserve comments, which the AST discards.
//
// Design: the formatter keeps the *line structure* of the input as the signal
// for statement boundaries (a token that starts on a new source line, while not
// inside parentheses or brackets, begins a new output line) and derives
// *indentation* purely from curly-brace nesting. This avoids fragile heuristics
// for reconstructing statement ends when many words (as, in, func, raise) are
// not reserved keywords in the lexer.
//
// Guarantees:
//   - Idempotent: Format(Format(src)) == Format(src) for any input that lexes
//     without error tokens.
//   - Structural: each "{ ... }" block increases indentation by one unit (four
//     spaces); leading whitespace in the input is ignored.
//   - Stable spacing: binary operators are surrounded by single spaces; commas
//     and colons are followed by a single space; call/index brackets and dots
//     hug their operands.
package fmt

import (
	"strings"

	"rayo/internal/lex"
)

// indentUnit is the string emitted per level of block nesting.
const indentUnit = "    "

// Format formats Rayo source text and returns the formatted result.
func Format(src string) string {
	return FormatTokens(lexAll(src))
}

// lexAll runs the lexer over src and collects every token (excluding EOF).
func lexAll(src string) []lex.Token {
	lx := lex.NewLexer(src)
	var toks []lex.Token
	for {
		t := lx.Next()
		if t.Kind == lex.TokenEOF {
			break
		}
		toks = append(toks, t)
	}
	return toks
}

// FormatTokens formats a token stream with stable, idempotent rules.
func FormatTokens(tokens []lex.Token) string {
	toks := stripTrivia(tokens)

	var sb strings.Builder
	indent := 0
	// bracketDepth tracks (), [], and dict-literal {} nesting. Inside brackets
	// we never break lines on our own, so multi-line call args and collection
	// literals stay compact and idempotent.
	bracketDepth := 0
	// dictStack records, for each currently-open "{", whether it is a dict
	// literal (true) rather than a block (false). Dict literals are treated
	// like bracketed expressions: no forced newline or indentation.
	var dictStack []bool
	atLineStart := true
	// prevLine is the source line of the previously emitted structural token,
	// used to detect author line breaks between statements.
	prevLine := -1

	writeIndent := func() {
		for i := 0; i < indent; i++ {
			sb.WriteString(indentUnit)
		}
	}
	newline := func() {
		sb.WriteString("\n")
		atLineStart = true
	}

	for i := 0; i < len(toks); i++ {
		t := toks[i]

		// Decide whether this token begins a new output line. Braces manage
		// their own layout below, so skip the generic line-break logic for them.
		if t.Kind != lex.TokenLBrace && t.Kind != lex.TokenRBrace {
			if !atLineStart && bracketDepth == 0 && prevLine >= 0 && t.Line > prevLine {
				newline()
			}
		}

		switch t.Kind {
		case lex.TokenLBrace:
			if isDictBrace(toks, i) {
				// Dict literal: format inline like a collection.
				if !atLineStart && needsSpaceBeforeDict(toks, i) {
					sb.WriteString(" ")
				} else if atLineStart {
					writeIndent()
				}
				sb.WriteString("{")
				dictStack = append(dictStack, true)
				bracketDepth++
				atLineStart = false
				prevLine = t.Line
				break
			}
			// Block brace.
			if !atLineStart {
				sb.WriteString(" ")
			} else {
				writeIndent()
			}
			sb.WriteString("{")
			dictStack = append(dictStack, false)
			indent++
			newline()
			prevLine = t.Line

		case lex.TokenRBrace:
			isDict := len(dictStack) > 0 && dictStack[len(dictStack)-1]
			if len(dictStack) > 0 {
				dictStack = dictStack[:len(dictStack)-1]
			}
			if isDict {
				sb.WriteString("}")
				if bracketDepth > 0 {
					bracketDepth--
				}
				atLineStart = false
				prevLine = t.Line
				break
			}
			if indent > 0 {
				indent--
			}
			if !atLineStart {
				newline()
			}
			writeIndent()
			sb.WriteString("}")
			atLineStart = false
			prevLine = t.Line
			// If this block is a lambda body nested inside a call/collection
			// (bracketDepth > 0), keep the closing bracket on the same line and
			// do not force a statement break.
			if bracketDepth > 0 {
				break
			}
			// A "}" not followed by a continuation clause (else/elif/except/
			// finally) on the same construct ends the statement.
			if !nextIsContinuation(toks, i+1) {
				newline()
			}

		case lex.TokenComment:
			if !atLineStart {
				sb.WriteString(" ")
			} else {
				writeIndent()
			}
			sb.WriteString(strings.TrimRight(t.Value, " \t"))
			newline()
			prevLine = t.Line

		default:
			if atLineStart {
				writeIndent()
			} else if needsSpaceBefore(toks, i) {
				sb.WriteString(" ")
			}
			sb.WriteString(t.Value)
			atLineStart = false
			prevLine = t.Line

			switch t.Kind {
			case lex.TokenLParen, lex.TokenLBracket:
				bracketDepth++
			case lex.TokenRParen, lex.TokenRBracket:
				if bracketDepth > 0 {
					bracketDepth--
				}
			}
		}
	}

	return normalizeBlankLines(sb.String())
}

// stripTrivia removes whitespace and EOF tokens, which the formatter
// regenerates, while preserving all other tokens (including comments).
func stripTrivia(tokens []lex.Token) []lex.Token {
	out := make([]lex.Token, 0, len(tokens))
	for _, t := range tokens {
		if t.Kind == lex.TokenWhitespace || t.Kind == lex.TokenEOF {
			continue
		}
		out = append(out, t)
	}
	return out
}

// isDictBrace reports whether the "{" at index i opens a dict literal rather
// than a block. A dict literal appears in expression position: right after an
// assignment/operator, an opening bracket, a comma, a colon, a "return", or
// another dict opener. A block follows a ")", an identifier statement header,
// or block keywords (try/else/finally). At the start of input a bare "{" is
// treated as a block (matching the existing formatter tests).
func isDictBrace(toks []lex.Token, i int) bool {
	if i == 0 {
		return false
	}
	prev := toks[i-1]
	switch prev.Kind {
	case lex.TokenOp:
		return true
	case lex.TokenLParen, lex.TokenLBracket, lex.TokenComma, lex.TokenColon:
		return true
	case lex.TokenLBrace:
		return true
	}
	// "return {" and Rayo has no other value-producing keyword before "{".
	if prev.Kind == lex.TokenKeyword && prev.Value == "return" {
		return true
	}
	// A "{" immediately following "=" is an assignment; the lexer emits "=" as
	// an op, already handled above.
	return false
}

// needsSpaceBeforeDict reports whether a dict-literal "{" needs a leading space
// given the previous token (e.g. after ":" in a nested dict value it does; hugging
// "(" it does not).
func needsSpaceBeforeDict(toks []lex.Token, i int) bool {
	if i == 0 {
		return false
	}
	prev := toks[i-1]
	switch prev.Kind {
	case lex.TokenLParen, lex.TokenLBracket, lex.TokenDot:
		return false
	case lex.TokenColon, lex.TokenComma, lex.TokenOp, lex.TokenKeyword:
		return true
	}
	return true
}

// nextIsContinuation reports whether the token at index j begins a clause that
// must stay attached to the preceding "}" (rendering "} else {" on one line).
func nextIsContinuation(toks []lex.Token, j int) bool {
	if j >= len(toks) {
		return false
	}
	t := toks[j]
	// These may be lexed as keywords or (for the lexer's small keyword set) as
	// identifiers, so match on value regardless of kind.
	switch t.Value {
	case "else", "elif", "except", "finally":
		return t.Kind == lex.TokenKeyword || t.Kind == lex.TokenIdent
	}
	return false
}

// needsSpaceBefore decides whether a single space should precede toks[i] given
// the previous token. It is only consulted mid-line.
func needsSpaceBefore(toks []lex.Token, i int) bool {
	if i == 0 {
		return false
	}
	prev := toks[i-1]
	cur := toks[i]

	switch cur.Kind {
	case lex.TokenComma:
		return false
	case lex.TokenColon:
		return false
	case lex.TokenDot:
		return false
	case lex.TokenRParen, lex.TokenRBracket:
		return false
	case lex.TokenLParen, lex.TokenLBracket:
		// Call/index brackets hug the preceding value token ("foo(", "arr[").
		// After an operator, comma, colon, or keyword the bracket opens a
		// grouped expression or literal and takes a leading space.
		switch prev.Kind {
		case lex.TokenIdent, lex.TokenRParen, lex.TokenRBracket, lex.TokenString, lex.TokenNumber:
			return false
		}
		return true
	}

	switch prev.Kind {
	case lex.TokenDot:
		return false
	case lex.TokenLParen, lex.TokenLBracket:
		return false
	case lex.TokenComma, lex.TokenColon:
		return true
	case lex.TokenKeyword:
		return true
	case lex.TokenOp:
		return true
	}

	switch cur.Kind {
	case lex.TokenOp:
		return true
	case lex.TokenKeyword:
		return true
	}

	// Two adjacent word/value tokens (e.g. "as e", "in items", "raise Err")
	// need a separating space.
	switch prev.Kind {
	case lex.TokenIdent, lex.TokenNumber, lex.TokenString, lex.TokenRParen, lex.TokenRBracket:
		switch cur.Kind {
		case lex.TokenIdent, lex.TokenNumber, lex.TokenString:
			return true
		}
	}
	return false
}

// normalizeBlankLines collapses runs of blank lines to at most one, trims
// leading/trailing blanks, and guarantees a single trailing newline. This is
// what makes the formatter idempotent regardless of incoming blank-line noise.
func normalizeBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blank := 0
	for _, ln := range lines {
		ln = strings.TrimRight(ln, " \t")
		if ln == "" {
			blank++
			if blank > 1 {
				continue
			}
			out = append(out, "")
			continue
		}
		blank = 0
		out = append(out, ln)
	}
	for len(out) > 0 && out[0] == "" {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}
