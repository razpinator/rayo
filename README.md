# Rayo

Rayo (read as rah-YOH) is a readable, Python-inspired programming language that transpiles to Golang. It emphasizes null safety, error handling, and clean syntax for building reliable applications.

# Why Named Rayo?

Raza's Python-inspired Language that transpiles to Golang :P

## Design Goals

- **Readability-first**: Python-like syntax with curly braces for blocks.
- **Null Safety**: Single `None` value, optionals, safe navigation.
- **Error Handling**: `try/except/finally` blocks.
- **Dicts and Objects**: Flexible data structures with attribute access.
- **Transpilation**: Generates buildable Go 1.22+ code.
- **Precise Spec**: Well-defined grammar and semantics.

## Quickstart

1. Install dependencies:
   ```sh
   go mod tidy
   ```

2. Build the CLI:
   ```sh
   go build -o rayoc ./cmd/rayoc
   ```

3. Transpile a Rayo file:
   ```sh
   ./rayoc transpile examples/web/api.ryo -o output.go
   ```

4. Run the generated Go code:
   ```sh
   go run output.go
   ```

## CLI Usage

```sh
rayoc [command]

Available Commands:
  lex         Lex source file
  parse       Parse source file
  check       Check semantics
  transpile   Transpile to Go

Flags:
  -I, --include stringSlice   Include paths
  -o, --output string         Output directory
  -v, --verbose               Verbose output
      --emit-go               Emit Go code
```

## Examples

See `/examples/` for a cookbook of 10+ examples covering CLI tools, data processing, web APIs, error handling, and null safety.

## Documentation

- [Language Spec](/doc/spec.md)
- [Tutorial](/docs/tutorial.md)
- [Core Features](/docs/core.md)
- [Data Structures](/docs/data.md)
- [I/O Operations](/docs/io.md)

## Contributing

Rayo is in active development. See the spec for implementation details.
