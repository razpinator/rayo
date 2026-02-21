# VS Code workspace setup for Rayo

This folder contains shared VS Code configuration to improve developer experience.

## Contents

- **extensions.json** – Recommends the Go extension for the codebase. Install when prompted.
- **settings.json** – Go format/lint on save, file association for `.ryo` → `rayo`, trim trailing whitespace.
- **launch.json** – Debug configurations:
  - Run Rayo CLI (default: `examples/web/api.ryo`)
  - Run Rayo CLI with the currently open file
  - Transpile current `.ryo` file to Go
  - Debug tests (current package or all)
- **tasks.json** – Tasks: `make build`, `go test ./...`, `make lint`, `make fmt`, run examples.
- **rayo.code-snippets** – Snippets for `def`, `if`/`for`/`while`, `try/except`, `import`, `print` in `.ryo` files.

## Quick actions

- **Build**: `Cmd+Shift+B` (default build task) or Run Task → `build`.
- **Test**: Run Task → `test`, or use the Go extension’s test codelens.
- **Run a .ryo file**: Open a `.ryo` file, then Run → Start Debugging and choose “Run Rayo CLI (current file)”, or Run Task → `run current .ryo file` (after building once).

## Rayo LSP

If you use the Rayo LSP (see `tools/lsp/`), configure your client to connect to the language server for `.ryo` files. The workspace treats `*.ryo` as language `rayo` for syntax and snippets.
