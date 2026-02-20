package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/jordanhubbard/loom/internal/memory"
)

// migrateProjectMemory creates the project_memory table if it doesn't exist.
func (d *Database) migrateProjectMemory() error {
	schema := `
	CREATE TABLE IF NOT EXISTS project_memory (
		project_id  TEXT NOT NULL,
		category    TEXT NOT NULL,
		key         TEXT NOT NULL,
		value       TEXT NOT NULL,
		confidence  REAL NOT NULL DEFAULT 1.0,
		updated_at  TIMESTAMP NOT NULL,
		source_bead TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (project_id, category, key)
	);
	CREATE INDEX IF NOT EXISTS idx_project_memory_project ON project_memory(project_id);
	CREATE INDEX IF NOT EXISTS idx_project_memory_category ON project_memory(project_id, category);
	`
	_, err := d.db.Exec(schema)
	return err
}

// UpsertMemory inserts or updates a project memory entry.
func (d *Database) UpsertMemory(ctx context.Context, m *memory.ProjectMemory) error {
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = time.Now().UTC()
	}
	_, err := d.db.ExecContext(ctx, rebind(`
		INSERT INTO project_memory (project_id, category, key, value, confidence, updated_at, source_bead)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (project_id, category, key)
		DO UPDATE SET value = EXCLUDED.value,
		              confidence = EXCLUDED.confidence,
		              updated_at = EXCLUDED.updated_at,
		              source_bead = EXCLUDED.source_bead
	`),
		m.ProjectID, string(m.Category), m.Key, m.Value,
		m.Confidence, m.UpdatedAt, m.SourceBead,
	)
	return err
}

// GetMemory retrieves a single memory entry. Returns nil, nil if not found.
func (d *Database) GetMemory(ctx context.Context, projectID string, category memory.MemoryCategory, key string) (*memory.ProjectMemory, error) {
	row := d.db.QueryRowContext(ctx, rebind(`
		SELECT project_id, category, key, value, confidence, updated_at, source_bead
		FROM project_memory
		WHERE project_id = ? AND category = ? AND key = ?
	`), projectID, string(category), key)

	return scanProjectMemory(row)
}

// ListMemory returns all memory entries for a project.
func (d *Database) ListMemory(ctx context.Context, projectID string) ([]*memory.ProjectMemory, error) {
	rows, err := d.db.QueryContext(ctx, rebind(`
		SELECT project_id, category, key, value, confidence, updated_at, source_bead
		FROM project_memory
		WHERE project_id = ?
		ORDER BY category, key
	`), projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjectMemoryRows(rows)
}

// ListMemoryByCategory returns memory entries for a project filtered by category.
func (d *Database) ListMemoryByCategory(ctx context.Context, projectID string, category memory.MemoryCategory) ([]*memory.ProjectMemory, error) {
	rows, err := d.db.QueryContext(ctx, rebind(`
		SELECT project_id, category, key, value, confidence, updated_at, source_bead
		FROM project_memory
		WHERE project_id = ? AND category = ?
		ORDER BY key
	`), projectID, string(category))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjectMemoryRows(rows)
}

// DeleteMemory removes a memory entry.
func (d *Database) DeleteMemory(ctx context.Context, projectID string, category memory.MemoryCategory, key string) error {
	_, err := d.db.ExecContext(ctx, rebind(`
		DELETE FROM project_memory WHERE project_id = ? AND category = ? AND key = ?
	`), projectID, string(category), key)
	return err
}

func scanProjectMemory(row *sql.Row) (*memory.ProjectMemory, error) {
	var m memory.ProjectMemory
	var cat string
	err := row.Scan(&m.ProjectID, &cat, &m.Key, &m.Value, &m.Confidence, &m.UpdatedAt, &m.SourceBead)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Category = memory.MemoryCategory(cat)
	return &m, nil
}

func scanProjectMemoryRows(rows *sql.Rows) ([]*memory.ProjectMemory, error) {
	var result []*memory.ProjectMemory
	for rows.Next() {
		var m memory.ProjectMemory
		var cat string
		if err := rows.Scan(&m.ProjectID, &cat, &m.Key, &m.Value, &m.Confidence, &m.UpdatedAt, &m.SourceBead); err != nil {
			return nil, err
		}
		m.Category = memory.MemoryCategory(cat)
		result = append(result, &m)
	}
	return result, rows.Err()
}
