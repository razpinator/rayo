# Rayo

Rayo (read as rah-YOH) is a readable, Python-inspired programming language that transpiles to Golang. It emphasizes null safety, error handling, and clean syntax for building reliable applications.

## Design Goals

- **Readability-first**: Python-like syntax with curly braces for blocks.
- **Null Safety**: Single `None` value, optionals, safe navigation.
- **Error Handling**: `try/except/finally` blocks.
- **Dicts and Objects**: Flexible data structures with attribute access.
- **Transpilation**: Generates buildable Go 1.22+ code.
- **Precise Spec**: Well-defined grammar and semantics.

## Why Named Rayo?

Raza's Python-inspired Language that transpiles to Golang :P

## Quickstart

### Install (Linux & macOS)

One command:

```sh
curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/razpinator/rayo/main/install.sh | sh
```

This installs the latest release to `~/.local/bin` (or `/usr/local/bin` if needed). Ensure that directory is in your `PATH`; the script will remind you if not.

Verify:

```sh
rayo version
```

### Run a Rayo file

```sh
rayo run examples/web/api.ryo
```

(Optional) Transpile to Go manually:

```sh
rayo transpile examples/web/api.ryo -o output.go
go run output.go
```

### Other ways to install

- **Manual download**: Get the right archive from [Releases](https://github.com/razpinator/rayo/releases) (e.g. `rayo_0.2.0_Linux_x86_64.tar.gz`), extract, and move `rayo` and `rayoc` to a directory in your `PATH`.
- **From source**: Clone the repo, then run `./install-from-source.sh` (requires Go).

### Manual Build (Development)

If you prefer to build manually:

1. Install dependencies:
   ```sh
   go mod tidy
   ```

2. Build the CLI:
   ```sh
   make build
   ```

3. Use the local binary:
   ```sh
   ./build/rayo run examples/web/api.ryo
   ```

## CLI Usage

```sh
rayo [command]

Available Commands:
  lex         Lex source file
  parse       Parse source file
  check       Check semantics
  run         Transpile and run
  transpile   Transpile to Go
  fmt         Format .ryo source files
  lint        Lint .ryo source files
  repl        Start the interactive Rayo REPL
  lsp         Run the Language Server Protocol server
  version     Print version information

Flags:
  -I, --include stringSlice   Include paths
  -o, --output string         Output directory
  -v, --verbose               Verbose output
      --emit-go               Emit Go code
```

## Editor support (LSP)

Rayo ships a Language Server (`tools/lsp`) exposed through `rayo lsp`:

```sh
rayo lsp            # listen on TCP :2087 (used by the bundled VS Code client)
rayo lsp :9000      # listen on a custom TCP address
rayo lsp --stdio    # communicate over stdin/stdout (Neovim, Emacs, Helix, ...)
```

Features: real-time diagnostics, hover, go-to-definition, completion (keywords,
names in scope, and `def`/`if`/`try` snippets), find references, document
symbols, and workspace symbols. The server negotiates client capabilities on
`initialize` (for example, emitting snippet syntax only when the client
advertises snippet support) and reports errors with structured JSON-RPC codes.

## Formatter

`rayo fmt` (implemented in `tools/fmt`) rewrites `.ryo` sources with a stable,
idempotent style: `fmt(fmt(x)) == fmt(x)`. It works on the lexer token stream so
it round-trips arbitrary source and preserves comments, deriving indentation
(four spaces per level) from curly-brace nesting while keeping the author's
statement layout.

```sh
rayo fmt                 # format every .ryo file under the current directory (in place)
rayo fmt examples        # format a directory tree
rayo fmt --check path    # report unformatted files without rewriting (exit 1 if any)
rayo fmt --stdout file   # print the formatted result instead of writing
```

Formatting rules at a glance: binary operators get surrounding spaces; commas
and colons are followed by a single space; call/index brackets and `.` hug their
operands; dict literals stay inline; `} else {` / `} except … {` / `} finally {`
clauses stay attached to the preceding brace.

## Linter

`rayo lint` (implemented in `tools/lint`) reports readability and correctness
issues with **stable rule IDs** so findings can be referenced and documented
independently of their message text:

| Rule  | Name                   | Severity | What it catches |
|-------|------------------------|----------|-----------------|
| RY001 | `unused-binding`       | warning  | A variable is declared/assigned but never read. |
| RY002 | `unused-import`        | warning  | An imported module is never referenced. |
| RY003 | `suspicious-attr`      | info     | `obj.attr` on a dict-like value; you likely meant `obj["attr"]`. |
| RY004 | `unsafe-optional-deref`| warning  | A possibly-`None` value is dereferenced (`.attr` or `[i]`) without a check. |
| RY005 | `high-complexity`      | warning  | A function's cyclomatic complexity exceeds the threshold (default 10). |

```sh
rayo lint                # lint every .ryo file under the current directory
rayo lint examples       # lint a directory tree
```

Each finding prints as `path:line: RYNNN [severity] message`. Findings are sorted
by source position for deterministic output, and `rayo lint` exits non-zero only
when an error-severity finding is present.

## Examples

See `/examples/` for a cookbook of 10+ examples covering CLI tools, data processing, web APIs, error handling, and null safety.

## Documentation

- [Language Spec](/docs/spec.md)
- [Tutorial](/docs/tutorial.md)
- [Core Features](/docs/core.md)
- [Data Structures](/docs/data.md)
- [I/O Operations](/docs/io.md)

## Semantic analysis

The checker in `internal/sem` runs after parsing and before code generation. It
performs:

- **Type inference**: literals, dict/list literals, unary/binary results, and
  name lookups resolve through the lexical symbol table. A bare `None` and a
  `.get(key)` call infer as optional types.
- **Null safety**: dereferencing (`x.attr`) or indexing (`x[i]`) a value whose
  inferred type is optional is reported as `unsafe dereference of optional value`
  / `unsafe index of optional value`.
- **Definite assignment**: using a declared name before it is assigned reports
  `use of unassigned variable: <name>`.
- **Must-return analysis**: if any path in a function returns a value, every
  path must terminate with a `return` or `raise`. Otherwise the checker reports
  `missing return: not all paths in '<fn>' return a value`. `if/elif/else`,
  `try/except/finally`, and `while true` are all accounted for.
- **Unused variables**: emitted as warnings (severity `warning`), so they show
  up in diagnostics without failing compilation.

Diagnostics carry a severity (`error`/`warning`/`info`); only errors abort a
build.

## Code generation

Generated Go is run through `go/format`, so the output is gofmt-clean and passes
`go vet`. The generator lowers:

- **Control flow**: `if/elif/else` → `if/else if/else`, `while` → `for`,
  `for x in xs` → `for _, x := range xs`, and `try/except/finally` → a
  `recover`-based closure with `defer`.
- **Data literals**: dicts → `map[string]any{...}`, lists → `[]any{...}`.
- **Operators**: `and`/`or`/`not` → `&&`/`||`/`!`.
- **Functions**: parameters and a synthetic `return nil` guard so
  expression-oriented bodies always type-check as `func(...) any`.

## Testing

Run the full suite:

```sh
go test ./...
```

Golden fixtures live in `testdata/golden/`. Each `<name>.ryo` can carry sidecar
expectations: `.tokens`, `.ast`, `.go`, `.out`, and `.diags`. A `.diags` sidecar
snapshots the diagnostics of a "bad" program as `line:col: severity: message`
(one per line, sorted by position), covering both parse errors and semantic
warnings such as unused variables. `rayo test [filter]` runs the same harness.

Lexer and parser fuzz targets seed from the golden sources:

```sh
go test ./internal/lex   -run x -fuzz FuzzLexerRoundTrip  -fuzztime 15s
go test ./internal/parse -run x -fuzz FuzzParserRoundTrip -fuzztime 15s
```

See [ARCHITECTURE.md](/ARCHITECTURE.md) for the shared diagnostics/source-span
model used across the lexer, parser, semantic analysis, and LSP.

## Continuous integration

Two GitHub Actions workflows live in `.github/workflows/`:

- **`ci.yml`** runs on every push to `main`/`master` and on pull requests. The
  `build-test` job runs `go build ./...`, `go vet ./...`, `go test ./...`, and
  the golden suite (`go test -run TestGolden ./golden`) on Linux and macOS. A
  separate `fmt-lint` job runs `rayo fmt --check examples` and
  `rayo lint examples` as informational (non-blocking) steps, since the bundled
  examples are not yet normalized to the formatter's output.
- **`release.yml`** runs GoReleaser on tag pushes to publish binaries.

The same checks are available locally through the Makefile: `make test`,
`make vet`, `make fmt-check`, and `make lint`.

## Contributing

Rayo is in active development. See the spec for implementation details.
