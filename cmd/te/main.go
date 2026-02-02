package main

import (
	"fmt"
	"os"

	"github.com/willfish/te/internal/store"
)

const usage = `te - Tariff Enumerator

Usage:
  te parse <file.xml> [--db path]    Parse XML into SQLite
  te browse [--db path]              Launch TUI browser

Flags:
  --db path    Database path (default: ~/.cache/te/tariff.db)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(0)
	}

	dbPath := store.DefaultPath()

	switch os.Args[1] {
	case "parse":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: te parse <file.xml> [--db path]")
			os.Exit(1)
		}
		filename := os.Args[2]
		for i := 3; i < len(os.Args); i++ {
			if os.Args[i] == "--db" && i+1 < len(os.Args) {
				dbPath = os.Args[i+1]
				i++
			}
		}
		if err := runParse(filename, dbPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "browse":
		for i := 2; i < len(os.Args); i++ {
			if os.Args[i] == "--db" && i+1 < len(os.Args) {
				dbPath = os.Args[i+1]
				i++
			}
		}
		if err := runBrowse(dbPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "--help", "-h", "help":
		fmt.Print(usage)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
}
