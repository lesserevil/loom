package models

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// MigrationFunc is a function that migrates an entity from one version to another.
// It receives the entity and should modify it in place.
// The function should handle the specific field transformations needed.
type MigrationFunc func(entity VersionedEntity) error

// MigrationKey uniquely identifies a migration path
type MigrationKey struct {
	EntityType  EntityType
	FromVersion SchemaVersion
	ToVersion   SchemaVersion
}

// MigrationInfo describes a registered migration
type MigrationInfo struct {
	Key         MigrationKey
	Description string
	Migrate     MigrationFunc
	Breaking    bool // If true, this migration has behavioral changes
}

// MigrationRegistry manages entity migrations
type MigrationRegistry struct {
	mu         sync.RWMutex
	migrations map[MigrationKey]*MigrationInfo
	versions   map[EntityType][]SchemaVersion // Ordered list of versions per entity type
}

// Global registry instance
var globalRegistry = NewMigrationRegistry()

// NewMigrationRegistry creates a new migration registry
func NewMigrationRegistry() *MigrationRegistry {
	return &MigrationRegistry{
		migrations: make(map[MigrationKey]*MigrationInfo),
		versions:   make(map[EntityType][]SchemaVersion),
	}
}

// GetRegistry returns the global migration registry
func GetRegistry() *MigrationRegistry {
	return globalRegistry
}

// Register adds a migration to the registry
func (r *MigrationRegistry) Register(info MigrationInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if info.Migrate == nil {
		return fmt.Errorf("migration function cannot be nil")
	}

	r.migrations[info.Key] = &info

	// Track version ordering
	versions := r.versions[info.Key.EntityType]
	hasFrom, hasTo := false, false
	for _, v := range versions {
		if v == info.Key.FromVersion {
			hasFrom = true
		}
		if v == info.Key.ToVersion {
			hasTo = true
		}
	}
	if !hasFrom {
		versions = append(versions, info.Key.FromVersion)
	}
	if !hasTo {
		versions = append(versions, info.Key.ToVersion)
	}
	// Sort versions (simple string sort works for semver-like versions)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] < versions[j]
	})
	r.versions[info.Key.EntityType] = versions

	return nil
}

// GetMigration retrieves a specific migration
func (r *MigrationRegistry) GetMigration(key MigrationKey) (*MigrationInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.migrations[key]
	return info, ok
}

// GetMigrationPath returns the sequence of migrations needed to go from
// fromVersion to toVersion for a given entity type
func (r *MigrationRegistry) GetMigrationPath(entityType EntityType, fromVersion, toVersion SchemaVersion) ([]*MigrationInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if fromVersion == toVersion {
		return nil, nil // No migration needed
	}

	versions := r.versions[entityType]
	if len(versions) == 0 {
		return nil, fmt.Errorf("no migrations registered for entity type %s", entityType)
	}

	// Find indices
	fromIdx, toIdx := -1, -1
	for i, v := range versions {
		if v == fromVersion {
			fromIdx = i
		}
		if v == toVersion {
			toIdx = i
		}
	}

	// Handle unversioned (empty) as version 0
	if fromVersion == "" {
		fromIdx = -1 // Before first version
	}

	if toIdx == -1 {
		return nil, fmt.Errorf("target version %s not found for entity type %s", toVersion, entityType)
	}

	// Build migration path
	var path []*MigrationInfo
	currentVersion := fromVersion
	if fromIdx == -1 {
		// Migrating from unversioned - start with first registered version
		if len(versions) > 0 {
			key := MigrationKey{EntityType: entityType, FromVersion: "", ToVersion: versions[0]}
			if info, ok := r.migrations[key]; ok {
				path = append(path, info)
				currentVersion = versions[0]
				fromIdx = 0
			}
		}
	}

	// Walk through versions
	for i := fromIdx; i < toIdx; i++ {
		nextVersion := versions[i+1]
		key := MigrationKey{
			EntityType:  entityType,
			FromVersion: currentVersion,
			ToVersion:   nextVersion,
		}
		info, ok := r.migrations[key]
		if !ok {
			return nil, fmt.Errorf("no migration found from %s to %s for entity type %s",
				currentVersion, nextVersion, entityType)
		}
		path = append(path, info)
		currentVersion = nextVersion
	}

	return path, nil
}

// MigrateEntity migrates an entity to the target version
func (r *MigrationRegistry) MigrateEntity(entity VersionedEntity, targetVersion SchemaVersion) error {
	currentVersion := entity.GetSchemaVersion()
	if currentVersion == targetVersion {
		return nil // Already at target version
	}

	path, err := r.GetMigrationPath(entity.GetEntityType(), currentVersion, targetVersion)
	if err != nil {
		return fmt.Errorf("failed to get migration path: %w", err)
	}

	for _, migration := range path {
		if err := migration.Migrate(entity); err != nil {
			return fmt.Errorf("migration from %s to %s failed: %w",
				migration.Key.FromVersion, migration.Key.ToVersion, err)
		}
		entity.SetSchemaVersion(migration.Key.ToVersion)
	}

	// Record migration in metadata
	meta := entity.GetEntityMetadata()
	if meta != nil {
		now := time.Now()
		meta.MigratedAt = &now
		meta.MigratedFrom = currentVersion
	}

	return nil
}

// MigrateToLatest migrates an entity to its latest schema version
func (r *MigrationRegistry) MigrateToLatest(entity VersionedEntity) error {
	targetVersion := GetLatestVersion(entity.GetEntityType())
	return r.MigrateEntity(entity, targetVersion)
}

// HasBreakingChanges checks if the migration path contains breaking changes
func (r *MigrationRegistry) HasBreakingChanges(entityType EntityType, fromVersion, toVersion SchemaVersion) (bool, error) {
	path, err := r.GetMigrationPath(entityType, fromVersion, toVersion)
	if err != nil {
		return false, err
	}
	for _, migration := range path {
		if migration.Breaking {
			return true, nil
		}
	}
	return false, nil
}

// GetLatestVersion returns the latest schema version for an entity type
func GetLatestVersion(entityType EntityType) SchemaVersion {
	switch entityType {
	case EntityTypeAgent:
		return AgentSchemaVersion
	case EntityTypeProject:
		return ProjectSchemaVersion
	case EntityTypeProvider:
		return ProviderSchemaVersion
	case EntityTypeOrgChart:
		return OrgChartSchemaVersion
	case EntityTypePosition:
		return PositionSchemaVersion
	case EntityTypePersona:
		return PersonaSchemaVersion
	case EntityTypeBead:
		return BeadSchemaVersion
	default:
		return "1.0"
	}
}

// RegisterMigration is a convenience function to register with the global registry
func RegisterMigration(entityType EntityType, from, to SchemaVersion, desc string, breaking bool, fn MigrationFunc) error {
	return globalRegistry.Register(MigrationInfo{
		Key: MigrationKey{
			EntityType:  entityType,
			FromVersion: from,
			ToVersion:   to,
		},
		Description: desc,
		Migrate:     fn,
		Breaking:    breaking,
	})
}

// MigrateEntity is a convenience function to migrate using the global registry
func MigrateEntityToLatest(entity VersionedEntity) error {
	return globalRegistry.MigrateToLatest(entity)
}

// EnsureMigrated checks if entity needs migration and applies it
func EnsureMigrated(entity VersionedEntity) error {
	targetVersion := GetLatestVersion(entity.GetEntityType())
	if NeedsMigration(entity, targetVersion) {
		return globalRegistry.MigrateEntity(entity, targetVersion)
	}
	return nil
}
