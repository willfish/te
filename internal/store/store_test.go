package store

import (
	"os"
	"path/filepath"
	"testing"
)

func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "test.db")
}

func TestOpenAndClose(t *testing.T) {
	s, err := Open(tempDB(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestInsertAndQuery(t *testing.T) {
	path := tempDB(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := s.InsertElement("1", "Measure", `{"hjid":"1","sid":"100"}`); err != nil {
		t.Fatalf("InsertElement: %v", err)
	}
	if err := s.InsertElement("2", "Measure", `{"hjid":"2","sid":"200"}`); err != nil {
		t.Fatalf("InsertElement: %v", err)
	}
	if err := s.InsertElement("3", "GoodsNomenclature", `{"hjid":"3","code":"010101"}`); err != nil {
		t.Fatalf("InsertElement: %v", err)
	}

	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	counts, err := s.TypeCounts()
	if err != nil {
		t.Fatalf("TypeCounts: %v", err)
	}

	if len(counts) != 2 {
		t.Fatalf("expected 2 type counts, got %d", len(counts))
	}

	if counts[0].Type != "Measure" || counts[0].Count != 2 {
		t.Errorf("expected Measure:2, got %s:%d", counts[0].Type, counts[0].Count)
	}
	if counts[1].Type != "GoodsNomenclature" || counts[1].Count != 1 {
		t.Errorf("expected GoodsNomenclature:1, got %s:%d", counts[1].Type, counts[1].Count)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestElements(t *testing.T) {
	path := tempDB(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	for i := 0; i < 5; i++ {
		hjid := string(rune('a' + i))
		if err := s.InsertElement(hjid, "Foo", `{"hjid":"`+hjid+`"}`); err != nil {
			t.Fatalf("InsertElement: %v", err)
		}
	}
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	elems, err := s.Elements("Foo", 3, 0)
	if err != nil {
		t.Fatalf("Elements: %v", err)
	}
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}

	elems, err = s.Elements("Foo", 3, 3)
	if err != nil {
		t.Fatalf("Elements offset: %v", err)
	}
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}
}

func TestElementCount(t *testing.T) {
	path := tempDB(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	if err := s.InsertElement("1", "A", `{}`); err != nil {
		t.Fatalf("InsertElement: %v", err)
	}
	if err := s.InsertElement("2", "A", `{}`); err != nil {
		t.Fatalf("InsertElement: %v", err)
	}
	if err := s.InsertElement("3", "B", `{}`); err != nil {
		t.Fatalf("InsertElement: %v", err)
	}
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	count, err := s.ElementCount("A")
	if err != nil {
		t.Fatalf("ElementCount: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestElementLookup(t *testing.T) {
	path := tempDB(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	if err := s.InsertElement("42", "Widget", `{"hjid":"42","name":"test"}`); err != nil {
		t.Fatalf("InsertElement: %v", err)
	}
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	e, err := s.Element("42")
	if err != nil {
		t.Fatalf("Element: %v", err)
	}
	if e.Hjid != "42" || e.Type != "Widget" {
		t.Errorf("unexpected element: %+v", e)
	}
}

func TestOpenReadOnly(t *testing.T) {
	path := tempDB(t)

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.InsertElement("1", "X", `{}`); err != nil {
		t.Fatalf("InsertElement: %v", err)
	}
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	ro, err := OpenReadOnly(path)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer func() { _ = ro.Close() }()

	counts, err := ro.TypeCounts()
	if err != nil {
		t.Fatalf("TypeCounts: %v", err)
	}
	if len(counts) != 1 || counts[0].Type != "X" {
		t.Errorf("unexpected counts: %+v", counts)
	}
}

func TestDefaultPath(t *testing.T) {
	p := DefaultPath()
	if p == "" {
		t.Fatal("DefaultPath returned empty string")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cache", "te", "tariff.db")
	if p != expected {
		t.Errorf("expected %s, got %s", expected, p)
	}
}

func TestBatchCommit(t *testing.T) {
	path := tempDB(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	for i := 0; i < batchSize+100; i++ {
		hjid := string(rune(i%26+'a')) + string(rune(i/26+'a'))
		if err := s.InsertElement(hjid, "Bulk", `{}`); err != nil {
			t.Fatalf("InsertElement %d: %v", i, err)
		}
	}
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	count, err := s.ElementCount("Bulk")
	if err != nil {
		t.Fatalf("ElementCount: %v", err)
	}
	if count == 0 {
		t.Error("expected non-zero count")
	}
}
