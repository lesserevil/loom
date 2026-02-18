package database

// Migration to add routing metadata to providers table
func (d *Database) migrateProviderRouting() error {
	// Skip migrations for PostgreSQL (schema is complete in initSchemaPostgres)
	if d.dbType == "postgres" {
		return nil
	}

	// Check if columns already exist
	var hasCost, hasContext, hasFunction, hasVision, hasStreaming, hasTags bool

	rows, err := d.db.Query("PRAGMA table_info(providers)")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue interface{}

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			continue
		}

		switch name {
		case "cost_per_mtoken":
			hasCost = true
		case "context_window":
			hasContext = true
		case "supports_function":
			hasFunction = true
		case "supports_vision":
			hasVision = true
		case "supports_streaming":
			hasStreaming = true
		case "tags_json":
			hasTags = true
		}
	}

	// Add columns if they don't exist
	if !hasCost {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN cost_per_mtoken REAL NOT NULL DEFAULT 0"); err != nil {
			return err
		}
	}

	if !hasContext {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN context_window INTEGER NOT NULL DEFAULT 4096"); err != nil {
			return err
		}
	}

	if !hasFunction {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN supports_function BOOLEAN NOT NULL DEFAULT 0"); err != nil {
			return err
		}
	}

	if !hasVision {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN supports_vision BOOLEAN NOT NULL DEFAULT 0"); err != nil {
			return err
		}
	}

	if !hasStreaming {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN supports_streaming BOOLEAN NOT NULL DEFAULT 1"); err != nil {
			return err
		}
	}

	if !hasTags {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN tags_json TEXT"); err != nil {
			return err
		}
	}

	return nil
}
