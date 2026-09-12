# Rayo I/O Helpers

## Files
- `ReadText(path)` — Read file as string
- `WriteText(path, data)` — Write string to file
- `ReadBytes(path)` — Read file as bytes
- `WriteBytes(path, data)` — Write bytes to file
- `ReadFile(path)` / `WriteFile(path, data)` — Aliases for `ReadText`/`WriteText`
- `Exists(path) bool` — Report whether a path exists

> These helpers use the `os` package (`os.ReadFile`/`os.WriteFile`); the
> deprecated `io/ioutil` calls have been removed.

## JSON
- `LoadJSON(path, v)` — Load JSON file into value
- `DumpJSON(path, v)` — Dump value to JSON file

## CSV
- `LoadCSV(path)` — Load CSV file into list of dicts
- `DumpCSV(path, rows)` — Dump list of dicts to CSV file

---

See unit tests for round-trip usage examples.
