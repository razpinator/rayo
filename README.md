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

One command, same idea as [Rust](https://rustup.rs):

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

Flags:
  -I, --include stringSlice   Include paths
  -o, --output string         Output directory
  -v, --verbose               Verbose output
      --emit-go               Emit Go code
```

## Examples

See `/examples/` for a cookbook of 10+ examples covering CLI tools, data processing, web APIs, error handling, and null safety.

## Documentation

- [Language Spec](/docs/spec.md)
- [Tutorial](/docs/tutorial.md)
- [Core Features](/docs/core.md)
- [Data Structures](/docs/data.md)
- [I/O Operations](/docs/io.md)

## Contributing

Rayo is in active development. See the spec for implementation details.
