package database

func (d *Database) migrateProviderScoring() error {
	if d.dbType == "postgres" {
		stmts := []string{
			"ALTER TABLE providers ADD COLUMN IF NOT EXISTS model_params_b REAL",
			"ALTER TABLE providers ADD COLUMN IF NOT EXISTS capability_score REAL",
			"ALTER TABLE providers ADD COLUMN IF NOT EXISTS avg_latency_ms REAL",
		}
		for _, stmt := range stmts {
			if _, err := d.db.Exec(stmt); err != nil {
				return err
			}
		}
		return nil
	}

	var hasModelParams, hasCapabilityScore, hasAvgLatency bool

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
		case "model_params_b":
			hasModelParams = true
		case "capability_score":
			hasCapabilityScore = true
		case "avg_latency_ms":
			hasAvgLatency = true
		}
	}

	if !hasModelParams {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN model_params_b REAL"); err != nil {
			return err
		}
	}

	if !hasCapabilityScore {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN capability_score REAL"); err != nil {
			return err
		}
	}

	if !hasAvgLatency {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN avg_latency_ms REAL"); err != nil {
			return err
		}
	}

	return nil
}
