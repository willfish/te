package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS elements (
    hjid       TEXT    PRIMARY KEY,
    type       TEXT    NOT NULL,
    data       TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_elements_type ON elements(type);
CREATE VIEW IF NOT EXISTS type_counts AS
    SELECT type, COUNT(*) AS count FROM elements GROUP BY type ORDER BY count DESC;
`

const batchSize = 10000

type TypeCount struct {
	Type  string
	Count int
}

type Element struct {
	Hjid string
	Type string
	Data string
}

type Store struct {
	db    *sql.DB
	tx    *sql.Tx
	stmt  *sql.Stmt
	count int
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "te", "tariff.db")
}

func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}

	if _, err := db.Exec("DELETE FROM elements"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("clearing elements: %w", err)
	}

	s := &Store{db: db}
	if err := s.beginBatch(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func OpenReadOnly(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) beginBatch() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO elements (hjid, type, data) VALUES (?, ?, ?)")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("preparing insert: %w", err)
	}

	s.tx = tx
	s.stmt = stmt
	s.count = 0
	return nil
}

func (s *Store) InsertElement(hjid, elementType, jsonData string) error {
	if _, err := s.stmt.Exec(hjid, elementType, jsonData); err != nil {
		return fmt.Errorf("inserting element: %w", err)
	}

	s.count++
	if s.count >= batchSize {
		if err := s.Flush(); err != nil {
			return err
		}
		if err := s.beginBatch(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) Flush() error {
	if s.stmt != nil {
		_ = s.stmt.Close()
		s.stmt = nil
	}
	if s.tx != nil {
		if err := s.tx.Commit(); err != nil {
			return fmt.Errorf("committing batch: %w", err)
		}
		s.tx = nil
	}
	s.count = 0
	return nil
}

func (s *Store) TypeCounts() ([]TypeCount, error) {
	rows, err := s.db.Query("SELECT type, count FROM type_counts")
	if err != nil {
		return nil, fmt.Errorf("querying type counts: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var counts []TypeCount
	for rows.Next() {
		var tc TypeCount
		if err := rows.Scan(&tc.Type, &tc.Count); err != nil {
			return nil, fmt.Errorf("scanning type count: %w", err)
		}
		counts = append(counts, tc)
	}
	return counts, rows.Err()
}

func (s *Store) Elements(elementType string, limit, offset int) ([]Element, error) {
	rows, err := s.db.Query(
		"SELECT hjid, type, data FROM elements WHERE type = ? LIMIT ? OFFSET ?",
		elementType, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("querying elements: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var elements []Element
	for rows.Next() {
		var e Element
		if err := rows.Scan(&e.Hjid, &e.Type, &e.Data); err != nil {
			return nil, fmt.Errorf("scanning element: %w", err)
		}
		elements = append(elements, e)
	}
	return elements, rows.Err()
}

func (s *Store) ElementCount(elementType string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM elements WHERE type = ?", elementType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting elements: %w", err)
	}
	return count, nil
}

func (s *Store) Element(hjid string) (*Element, error) {
	var e Element
	err := s.db.QueryRow(
		"SELECT hjid, type, data FROM elements WHERE hjid = ?", hjid,
	).Scan(&e.Hjid, &e.Type, &e.Data)
	if err != nil {
		return nil, fmt.Errorf("querying element: %w", err)
	}
	return &e, nil
}

func (s *Store) Close() error {
	if s.stmt != nil {
		_ = s.stmt.Close()
	}
	if s.tx != nil {
		_ = s.tx.Rollback()
	}
	return s.db.Close()
}
