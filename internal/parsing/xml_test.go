package parsing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/willfish/te/internal/store"
)

func TestParseXML(t *testing.T) {
	// Use the 11MB test XML file if available
	xmlFile := findTestXML(t)
	if xmlFile == "" {
		t.Skip("no test XML file found")
	}

	f, err := os.Open(xmlFile)
	if err != nil {
		t.Fatalf("opening test XML: %v", err)
	}
	defer f.Close() //nolint:errcheck

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	defer s.Close() //nolint:errcheck

	if err := Parse(f, s); err != nil {
		t.Fatalf("Parse: %v", err)
	}

	counts, err := s.TypeCounts()
	if err != nil {
		t.Fatalf("TypeCounts: %v", err)
	}

	if len(counts) == 0 {
		t.Fatal("expected at least one element type")
	}

	total := 0
	for _, tc := range counts {
		t.Logf("  %s: %d", tc.Type, tc.Count)
		total += tc.Count
	}
	t.Logf("Total elements: %d across %d types", total, len(counts))

	if total < 100 {
		t.Errorf("expected at least 100 elements, got %d", total)
	}
}

func findTestXML(t *testing.T) string {
	t.Helper()

	// Look for XML files in the project root
	root := projectRoot(t)
	patterns := []string{
		"export-*.xml",
		"*.xml",
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(root, pattern))
		for _, m := range matches {
			fi, err := os.Stat(m)
			if err != nil {
				continue
			}
			// Use files larger than 1MB as test files
			if fi.Size() > 1024*1024 {
				return m
			}
		}
	}

	return ""
}

func projectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from the test file to find go.mod
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
