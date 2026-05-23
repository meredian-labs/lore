package store

import (
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (s *Store) migrate() error {
	current, err := s.schemaVersion()
	if err != nil {
		return err
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		// Parse the leading integer index from the filename (e.g., "001_initial.sql" → 1).
		indexStr := strings.SplitN(name, "_", 2)[0]
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			continue
		}
		if index <= current {
			continue
		}

		sql, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}

		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("beginning transaction for %s: %w", name, err)
		}
		if _, err := tx.Exec(string(sql)); err != nil {
			tx.Rollback()
			return fmt.Errorf("applying migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %s: %w", name, err)
		}
	}
	return nil
}

func (s *Store) schemaVersion() (int, error) {
	// Check if meta table exists first.
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='meta'`).Scan(&count)
	if err != nil || count == 0 {
		return 0, nil
	}

	var val string
	err = s.db.QueryRow(`SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&val)
	if err != nil {
		return 0, nil
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		return 0, nil
	}
	return v, nil
}
