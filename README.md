# te — Tariff Enumerator

A CLI tool for parsing EU tariff XML exports into SQLite and browsing the data interactively.

## Usage

```
te parse <file.xml> [--db path]    Parse XML into SQLite
te browse [--db path]              Launch TUI browser
```

The `--db` flag defaults to `~/.cache/te/tariff.db`.

### Parse

```bash
te parse export-20240101.xml
```

Streams the XML through a SAX parser, extracts depth-4 elements, and inserts them into SQLite in batched transactions. Shows a progress bar in interactive terminals or percentage text in non-interactive environments.

### Browse

```bash
te browse
```

Opens a terminal UI with three screens:

- **Types** — element types with counts, sorted by frequency
- **Elements** — paginated table for the selected type (100 per page)
- **Detail** — pretty-printed JSON of the full element

Navigation: `Enter` to drill down, `Esc` to go back, `/` to filter, `q` to quit.

## Build

Requires Go 1.23+. No CGo — uses [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go).

```bash
make build    # produces ./te
make test     # runs all tests
make lint     # runs golangci-lint
```

A `shell.nix` is included for reproducible tooling via `nix-shell` or `direnv`.

## Architecture

```
cmd/te/
  main.go        CLI entry, subcommand dispatch
  parse.go       Parse subcommand with progress UI
  browse.go      Browse subcommand, launches TUI
internal/
  parsing/
    xml.go       SAX parser (gosax), outputs to store
    xml_test.go  Integration test against real XML
  store/
    store.go     SQLite storage layer
    store_test.go
  tui/
    app.go       Root BubbleTea model, screen routing
    types.go     Element types screen (bubble-table)
    elements.go  Elements list screen (bubble-table)
    detail.go    Element detail screen (viewport)
```

### SQLite schema

```sql
CREATE TABLE elements (
    hjid  TEXT PRIMARY KEY,
    type  TEXT NOT NULL,
    data  TEXT NOT NULL        -- full parsed node as JSON
);
CREATE INDEX idx_elements_type ON elements(type);
CREATE VIEW type_counts AS
    SELECT type, COUNT(*) AS count FROM elements GROUP BY type ORDER BY count DESC;
```

## Dependencies

- [bubbletea](https://github.com/charmbracelet/bubbletea) — terminal UI framework
- [bubble-table](https://github.com/evertras/bubble-table) — table component
- [gosax](https://github.com/orisano/gosax) — streaming XML parser
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — pure Go SQLite driver
