Using Golang, I want to develop a programming language.

Goal: Write a precise language spec for “rayo”.

Inputs:
- Python’s reserved keywords (use ALL and ONLY these).
- Curly braces {} for blocks (indentation is non-semantic).
- Dicts and attribute access like Python: obj["k"], obj.k (with defined disambiguation).
- Null safety (single null None, optionals, safe navigation).
- Error handling with try/except/finally.
- Transpilation target: Go 1.22+ (buildable code).
- Readability-first design.

Constraints:
- Define lexical grammar (tokens), and syntax (EBNF/PEG).
- Specify precedence, associativity, truthiness, mutability, evaluation order.
- State exact mapping expectations to Go for errors, nulls, dicts, attribute access.
- Document any semantic deviations from Python.

Acceptance Criteria:
- EBNF/PEG grammar compiles in a typical parser generator/hand-written parser.
- Ambiguities are resolved or flagged.
- Includes examples for each feature with expected behavior.

Deliverables:
- /docs/spec.md
- /docs/keywords.md (Python keyword list)

---

Goal: Create minimal “golden” programs that exercise core features.

Coverage:
- Curly-brace blocks, if/elif/else, while/for, def, return.
- Dict literals/updates, obj.k vs obj["k"].
- Null safety: optional binding, unsafe deref errors, safe navigation.
- try/except/finally semantics.

Deliverables:
- /testdata/golden/*.pygb (source)
- /testdata/golden/*.out (expected stdout/behavior)


---

Goal: Scaffold Go modules and folders for a transpiler.

Layout:
- cmd/rayoc/
- internal/{lex,parse,ast,sem,gen,diag}
- runtime/
- stdlib/{core,http,data,io}
- tools/{fmt,lint}
- repl/
- testdata/golden/
- docs/

Deliverables:
- go.mod, minimal main for rayoc that prints help.
- internal/diag with SourcePos, Span, Reporter.
- Makefile or magefile with tasks: test, fmt, lint, build.



---

Goal: Build a golden-test harness.

Behavior:
- For each case: *.pygb source; expected tokens/AST/Go/out files optional.
- Subcommands: lex, parse, check, transpile, run.
- Diff-friendly output.

Deliverables:
- /internal/testutil/golden.go
- make test-golden


---

Goal: Implement a deterministic lexer for rayo.

Inputs:
- Keywords: Python reserved words (only).
- Tokens: identifiers, numbers, strings, braces, parens, brackets, commas, colons, dots, ops, comments, whitespace.

Constraints:
- Curly braces are structural; indentation is ignored (spaces/tabs are non-semantic).
- Produce tokens with file offset, line, col.
- Robust error recovery: unknown char -> error token + continue.
- Preserve comments/trivia as optional channel for formatter.

Acceptance:
- Golden token streams match expectations.
- Fuzz against random inputs without panics.

Deliverables:
- /internal/lex/lexer.go, tokens.go
- Unit tests with table-driven cases.


---
Pending:
Add full Python keyword support or integration with golden tests.

---

Goal: Define Go structs for AST nodes with source spans.

Coverage:
- Module, imports, functions, parameters, blocks.
- Statements: var/assign, if/elif/else, while/for, return, try/except/finally, expr stmt.
- Expressions: literals, names, calls, indexing, attribute, unary/binary ops, dict/list literals, lambda (if in spec).
- Types (lightweight): Optional[T], Any, inferred.

Constraints:
- Every node has Span.
- Constructors for common nodes.

Deliverables:
- /internal/ast/ast.go
- /internal/ast/visit.go (visitor + walker)
- Tests: AST round-trip via pretty-printer (next step).


---

Goal: Implement a recursive-descent or Pratt parser for the EBNF.

Features:
- Curly-brace blocks; colons optional only if defined by spec.
- Good diagnostics: show offending token, expected set, source excerpt.
- Synchronization points (e.g., stmt boundaries) for recovery.

Acceptance:
- Parses all golden examples.
- Operator precedence/associativity match spec.
- Pretty-printer (debug) reconstructs source shape sufficiently.

Deliverables:
- /internal/parse/parser.go
- /internal/parse/errors.go
- /internal/parse/pp.go (debug-only pretty-print)


---

Goal: Implement semantic checks: scopes, bindings, null safety, basic typing/flow.

Scope:
- Module/function/block scopes, shadowing rules, unused var warnings.

Typing/Flow:
- Infer simple types for literals, vars; track Optional[T].
- Null-safety: forbid deref of possibly-None values; provide safe-nav operator if spec'd.
- Disambiguate obj.attr vs obj["attr"] when statically known; otherwise dynamic.

Exceptions/Control:
- Validate try/except/finally structures; ensure finally always reachable.
- Determine must-return paths in functions.

Acceptance:
- Emits precise errors with spans and hints.
- Golden “bad” programs produce expected diagnostics.

Deliverables:
- /internal/sem/check.go
- /internal/sem/types.go
- /internal/sem/flow.go
- Tests: table-driven diagnostic cases.


---

Goal: Transpile AST to buildable, readable Go 1.22+.

Scaffolding:
- Emitter with package/import management, name mangling, temp vars, label gen.
- Map file/module to Go package; public vs private names.

Lowerings:
- try/except -> functions returning (T, error); except-handlers become pattern on error types/tags.
- finally -> defer.
- Dicts -> map[string]any + helpers; known-shape dicts may become struct.
- Attribute vs index: struct field vs map lookup; dynamic objects use helper table.
- Null safety: Option[T] helpers; guard or unwrap with checks.

Acceptance:
- Transpiled Go for golden samples builds and runs matching expected output.
- Generated code passes `go vet`.

Deliverables:
- /internal/gen/gen.go
- /internal/gen/emit.go
- /internal/gen/lower_*.go
- Integration tests executing `go build` and `go run`.


---

Goal: Provide minimal Go runtime for dynamic behaviors.

Packages:
- runtime/core: Any, Option[T], None, truthiness, comparisons.
- runtime/dict: helpers (get/default, set, merge, deep copy).
- runtime/obj: dynamic attribute table (map[string]any) with getattr/setattr.
- runtime/err: error wrapping, tags, stack (best-effort).

Acceptance:
- Unit tests for each helper.
- No external deps beyond stdlib.

Deliverables:
- /runtime/{core,dict,obj,err}/...


---


Goal: Implement core library exposed to user code.

Modules:
- strings, math, list (map/filter/reduce), dict ops, time/datetime.

API style:
- Thin wrappers around Go stdlib with Pythonic naming and ergonomics.

Acceptance:
- Example programs import and use stdlib functions.
- Docs with examples for each function.

Deliverables:
- /stdlib/core/*.go
- /docs/stdlib/core.md


---

Goal: Provide ergonomic error helpers.

Features:
- raise(msg|err), wrap(err, msg), cause(err), is(err, tag), with_stack(err).
- Formatting for trace display in REPL and CLI.

Deliverables:
- /stdlib/core/errors.go
- Tests and usage snippets.



---

Goal: Provide I/O helpers and data formats.

Features:
- Files: read_text, write_text, read_bytes, write_bytes.
- JSON: load_json(path|reader) -> list|dict, dump_json(x, path|writer).
- CSV: load_csv(path) -> list[dict], dump_csv(rows, path).

Acceptance:
- Round-trip tests for JSON/CSV.

Deliverables:
- /stdlib/io/{files.go,json.go,csv.go}
- /docs/stdlib/io.md


---

Goal: Minimal HTTP server with router and JSON helpers.

API:
- app = http.app()
- app.get("/path", fn(req) { ... })
- app.post("/json", fn(req) { return http.json(200, {"ok": true}) })
- Middleware hooks; graceful shutdown.

Implementation:
- Wrap net/http; handlers compiled to Go closures; request/response helpers.

Acceptance:
- Example server with /health and /echo runs; integration test hits endpoints.

Deliverables:
- /stdlib/http/{app.go,router.go,context.go,json.go}
- /examples/web/...


---


Goal: Provide data helpers for ETL-like tasks.

Functions:
- map(list, fn), filter(list, fn), reduce(list, fn, init?)
- group_by(list, key_fn) -> dict[key] -> list
- agg(list, fn) -> any
- select(list, fields[]) -> list[dict]

Acceptance:
- Works with slices of dict and slices of struct via reflection where possible.
- Benchmarks included.

Deliverables:
- /stdlib/data/{funcs.go,reflect.go}
- /docs/stdlib/data.md


---

Goal: Build `rayoc` with subcommands.

Commands:
- lex, parse, check, transpile, run, test
- Common flags: -I include paths, -o output dir, -v verbose, --emit-go

Behavior:
- run: transpile to temp dir, `go run` compiled package.
- test: run golden tests.

Deliverables:
- /cmd/rayoc/main.go
- cobra or flag package; unit/integration tests.


---

Goal: Implement a stable code formatter.

Rules:
- Curly-brace style, spaces around binary ops, newline conventions.
- Keyword spacing and alignment; dict literal line-breaking rules.

Implementation:
- Either print from AST or token stream with minimal loss.
- Idempotent: fmt(fmt(code)) == fmt(code).

Deliverables:
- /tools/fmt/formatter.go
- Tests: snapshot-driven.



---


Goal: Implement a linter focused on readability and correctness.

Rules:
- Unused variables/imports.
- Suspicious attr vs key (e.g., obj.attr when dict likely).
- Null-safety hints (missing checks).
- Cyclomatic complexity threshold warnings.

Deliverables:
- /tools/lint/lint.go
- Tests for each rule with auto-fix suggestions where trivial.


---

Goal: Interactive REPL with persistent scope.

Behavior:
- Read one or more lines; parse/sem-check; transpile snippet into in-memory Go package; run; pretty-print result.
- Supports importing stdlib; keeps variables/functions between entries.
- Multiline editing, history, :help, :type, :vars commands.

Deliverables:
- /repl/repl.go
- Integration tests running scripted sessions.


---

Goal: LSP server with hover, go-to-def, and diagnostics.

Features:
- Parse+semantics on file change; publish diagnostics.
- Hover shows inferred type/nullability and docstrings.
- Definition uses symbol table.

Deliverables:
- /tools/lsp/server.go
- Minimal VSCode client config for testing.


---

Goal: Fuzz lexer and parser; differential tests.

Tasks:
- go test fuzz for lex.Parse round-trip; ensure no panics.
- Corpus from golden tests, plus random seeds.
- Differential: (future) interpreter vs transpiled run for small programs.

Deliverables:
- /internal/lex/lexer_fuzz_test.go
- /internal/parse/parser_fuzz_test.go


---

Goal: Establish baseline and avoid regressions.

Benchmarks:
- Parsing large files; codegen time.
- Dict ops (get/set/merge), data helpers (group_by).
- HTTP server throughput for /echo.

Deliverables:
- *_bench_test.go across runtime/stdlib.
- /docs/perf.md with targets and current numbers.


---

Goal: Ship readable docs and a 10+ example cookbook.

Artifacts:
- README: quickstart, design goals, CLI usage.
- Tutorial: build a web JSON API and a small ETL.
- Spec v1 (from earlier) cross-linked to examples.

Deliverables:
- /README.md
- /docs/spec.md, /docs/tutorial.md
- /examples/{cli,data,web,error,null}/...


---

Goal: CI that builds, tests, lints, runs goldens.

Steps:
- go build ./...
- go test ./... (unit, golden, fuzz-short)
- tools/fmt verifies formatting; tools/lint runs on repo.
- Cache Go build, run on Linux + macOS.

Deliverables:
- .github/workflows/ci.yml (or equivalent)


---

