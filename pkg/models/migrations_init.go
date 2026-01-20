package models

import "log"

// init registers all known entity migrations
func init() {
	registerInitialMigrations()
}

// registerInitialMigrations sets up the migration registry with all known migrations.
// This function is called during package initialization.
func registerInitialMigrations() {
	// Agent migrations
	// Example: migration from unversioned to 1.0
	if err := RegisterMigration(
		EntityTypeAgent, "", "1.0",
		"Initialize agent schema version",
		false, // Not breaking - just adds version tracking
		func(entity VersionedEntity) error {
			// No field transformations needed for initial version
			meta := entity.GetEntityMetadata()
			if meta.Attributes == nil {
				meta.Attributes = make(map[string]any)
			}
			return nil
		},
	); err != nil {
		log.Printf("Warning: failed to register agent migration: %v", err)
	}

	// Project migrations
	if err := RegisterMigration(
		EntityTypeProject, "", "1.0",
		"Initialize project schema version",
		false,
		func(entity VersionedEntity) error {
			meta := entity.GetEntityMetadata()
			if meta.Attributes == nil {
				meta.Attributes = make(map[string]any)
			}
			return nil
		},
	); err != nil {
		log.Printf("Warning: failed to register project migration: %v", err)
	}

	// Provider migrations
	if err := RegisterMigration(
		EntityTypeProvider, "", "1.0",
		"Initialize provider schema version",
		false,
		func(entity VersionedEntity) error {
			meta := entity.GetEntityMetadata()
			if meta.Attributes == nil {
				meta.Attributes = make(map[string]any)
			}
			return nil
		},
	); err != nil {
		log.Printf("Warning: failed to register provider migration: %v", err)
	}

	// OrgChart migrations
	if err := RegisterMigration(
		EntityTypeOrgChart, "", "1.0",
		"Initialize org chart schema version",
		false,
		func(entity VersionedEntity) error {
			meta := entity.GetEntityMetadata()
			if meta.Attributes == nil {
				meta.Attributes = make(map[string]any)
			}
			return nil
		},
	); err != nil {
		log.Printf("Warning: failed to register org chart migration: %v", err)
	}

	// Position migrations
	if err := RegisterMigration(
		EntityTypePosition, "", "1.0",
		"Initialize position schema version",
		false,
		func(entity VersionedEntity) error {
			meta := entity.GetEntityMetadata()
			if meta.Attributes == nil {
				meta.Attributes = make(map[string]any)
			}
			return nil
		},
	); err != nil {
		log.Printf("Warning: failed to register position migration: %v", err)
	}

	// Persona migrations
	if err := RegisterMigration(
		EntityTypePersona, "", "1.0",
		"Initialize persona schema version",
		false,
		func(entity VersionedEntity) error {
			meta := entity.GetEntityMetadata()
			if meta.Attributes == nil {
				meta.Attributes = make(map[string]any)
			}
			return nil
		},
	); err != nil {
		log.Printf("Warning: failed to register persona migration: %v", err)
	}

	// Bead migrations
	if err := RegisterMigration(
		EntityTypeBead, "", "1.0",
		"Initialize bead schema version",
		false,
		func(entity VersionedEntity) error {
			meta := entity.GetEntityMetadata()
			if meta.Attributes == nil {
				meta.Attributes = make(map[string]any)
			}
			return nil
		},
	); err != nil {
		log.Printf("Warning: failed to register bead migration: %v", err)
	}
}

// Example of a future migration (commented out for documentation):
//
// This shows how to add a migration when bumping from 1.0 to 1.1
// that renames a field or changes behavior:
//
// func init() {
//     RegisterMigration(
//         EntityTypeAgent, "1.0", "1.1",
//         "Rename 'role' to 'job_title' and add default tags",
//         true, // Breaking change - field rename
//         func(entity VersionedEntity) error {
//             agent, ok := entity.(*Agent)
//             if !ok {
//                 return fmt.Errorf("expected *Agent, got %T", entity)
//             }
//             // Move old field to attributes for backward compat
//             if agent.Role != "" {
//                 agent.SetAttribute("legacy.role", agent.Role)
//             }
//             // Initialize new fields
//             if !agent.HasAttribute("tags") {
//                 agent.SetAttribute("tags", []string{})
//             }
//             return nil
//         },
//     )
// }
