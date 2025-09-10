# Rayo Lexer

- Deterministic, robust lexer for Rayo language.
- Python keywords only.
- Produces tokens with offset, line, col.
- Ignores indentation.
- Preserves comments/trivia for formatter.
- Error recovery: unknown char -> error token + continue.
- See `lexer.go`, `tokens.go`, and `lexer_test.go` for details.
