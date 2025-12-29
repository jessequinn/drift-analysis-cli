package sql

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// SchemaCache manages local caching of inspected database schemas
type SchemaCache struct {
	cacheDir string
}

// CachedSchema represents a cached database schema with metadata
type CachedSchema struct {
	ConnectionName string          `json:"connection_name" yaml:"connection_name"`
	Database       string          `json:"database" yaml:"database"`
	Timestamp      time.Time       `json:"timestamp" yaml:"timestamp"`
	Schema         *DatabaseSchema `json:"schema" yaml:"schema"`
}

// NewSchemaCache creates a new schema cache manager
func NewSchemaCache(cacheDir string) (*SchemaCache, error) {
	if cacheDir == "" {
		// Default to .drift-cache in current directory
		cacheDir = ".drift-cache/database-schemas"
	}
	
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	return &SchemaCache{
		cacheDir: cacheDir,
	}, nil
}

// Save stores a database schema to local cache
func (sc *SchemaCache) Save(connectionName string, database string, schema *DatabaseSchema) error {
	cached := &CachedSchema{
		ConnectionName: connectionName,
		Database:       database,
		Timestamp:      time.Now(),
		Schema:         schema,
	}
	
	filename := sc.getCacheFilename(connectionName, database)
	filepath := filepath.Join(sc.cacheDir, filename)
	
	// Save as JSON for better performance
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}
	
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}
	
	fmt.Printf("Cached schema to: %s\n", filepath)
	return nil
}

// Load retrieves a cached database schema
func (sc *SchemaCache) Load(connectionName string, database string) (*CachedSchema, error) {
	filename := sc.getCacheFilename(connectionName, database)
	filepath := filepath.Join(sc.cacheDir, filename)
	
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache not found for %s/%s", connectionName, database)
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}
	
	var cached CachedSchema
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}
	
	return &cached, nil
}

// Exists checks if a cache file exists for the given connection
func (sc *SchemaCache) Exists(connectionName string, database string) bool {
	filename := sc.getCacheFilename(connectionName, database)
	filepath := filepath.Join(sc.cacheDir, filename)
	
	_, err := os.Stat(filepath)
	return err == nil
}

// GetAge returns how old the cached schema is
func (sc *SchemaCache) GetAge(connectionName string, database string) (time.Duration, error) {
	cached, err := sc.Load(connectionName, database)
	if err != nil {
		return 0, err
	}
	
	return time.Since(cached.Timestamp), nil
}

// List returns all cached schemas
func (sc *SchemaCache) List() ([]CachedSchema, error) {
	files, err := os.ReadDir(sc.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}
	
	var schemas []CachedSchema
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		
		data, err := os.ReadFile(filepath.Join(sc.cacheDir, file.Name()))
		if err != nil {
			continue
		}
		
		var cached CachedSchema
		if err := json.Unmarshal(data, &cached); err != nil {
			continue
		}
		
		schemas = append(schemas, cached)
	}
	
	return schemas, nil
}

// Delete removes a cached schema
func (sc *SchemaCache) Delete(connectionName string, database string) error {
	filename := sc.getCacheFilename(connectionName, database)
	filepath := filepath.Join(sc.cacheDir, filename)
	
	if err := os.Remove(filepath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete cache file: %w", err)
	}
	
	return nil
}

// Clear removes all cached schemas
func (sc *SchemaCache) Clear() error {
	files, err := os.ReadDir(sc.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}
	
	for _, file := range files {
		if !file.IsDir() {
			filepath := filepath.Join(sc.cacheDir, file.Name())
			if err := os.Remove(filepath); err != nil {
				return fmt.Errorf("failed to delete %s: %w", file.Name(), err)
			}
		}
	}
	
	return nil
}

// ExportYAML exports a cached schema to YAML format
func (sc *SchemaCache) ExportYAML(connectionName string, database string, outputPath string) error {
	cached, err := sc.Load(connectionName, database)
	if err != nil {
		return err
	}
	
	data, err := yaml.Marshal(cached)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}
	
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}
	
	return nil
}

// getCacheFilename generates a safe filename for the cache
func (sc *SchemaCache) getCacheFilename(connectionName string, database string) string {
	// Replace special characters to create a safe filename
	safe := filepath.Base(connectionName + "_" + database)
	safe = filepath.Clean(safe)
	return safe + ".json"
}

// GetCacheDir returns the cache directory path
func (sc *SchemaCache) GetCacheDir() string {
	return sc.cacheDir
}

// CompareSchemas compares two schemas and returns differences
func CompareSchemas(old *DatabaseSchema, new *DatabaseSchema) *SchemaDiff {
	diff := &SchemaDiff{
		OldTimestamp: old.DatabaseName,
		NewTimestamp: new.DatabaseName,
	}
	
	// Compare tables
	oldTables := make(map[string]TableInfo)
	for _, t := range old.Tables {
		key := fmt.Sprintf("%s.%s", t.Schema, t.Name)
		oldTables[key] = t
	}
	
	newTables := make(map[string]TableInfo)
	for _, t := range new.Tables {
		key := fmt.Sprintf("%s.%s", t.Schema, t.Name)
		newTables[key] = t
	}
	
	// Find added and modified tables
	for key, newTable := range newTables {
		if oldTable, exists := oldTables[key]; !exists {
			diff.AddedTables = append(diff.AddedTables, newTable)
		} else {
			// Compare columns count as a simple diff indicator
			if len(oldTable.Columns) != len(newTable.Columns) {
				diff.ModifiedTables = append(diff.ModifiedTables, newTable)
			}
		}
	}
	
	// Find deleted tables
	for key, oldTable := range oldTables {
		if _, exists := newTables[key]; !exists {
			diff.DeletedTables = append(diff.DeletedTables, oldTable)
		}
	}
	
	// Similar logic for views, roles, extensions
	diff.compareViews(old.Views, new.Views)
	diff.compareRoles(old.Roles, new.Roles)
	diff.compareExtensions(old.Extensions, new.Extensions)
	
	return diff
}

// SchemaDiff represents differences between two database schemas
type SchemaDiff struct {
	OldTimestamp string `json:"old_timestamp" yaml:"old_timestamp"`
	NewTimestamp string `json:"new_timestamp" yaml:"new_timestamp"`
	
	AddedTables    []TableInfo `json:"added_tables,omitempty" yaml:"added_tables,omitempty"`
	DeletedTables  []TableInfo `json:"deleted_tables,omitempty" yaml:"deleted_tables,omitempty"`
	ModifiedTables []TableInfo `json:"modified_tables,omitempty" yaml:"modified_tables,omitempty"`
	
	AddedViews    []ViewInfo `json:"added_views,omitempty" yaml:"added_views,omitempty"`
	DeletedViews  []ViewInfo `json:"deleted_views,omitempty" yaml:"deleted_views,omitempty"`
	
	AddedRoles   []string `json:"added_roles,omitempty" yaml:"added_roles,omitempty"`
	DeletedRoles []string `json:"deleted_roles,omitempty" yaml:"deleted_roles,omitempty"`
	
	AddedExtensions   []Extension `json:"added_extensions,omitempty" yaml:"added_extensions,omitempty"`
	DeletedExtensions []Extension `json:"deleted_extensions,omitempty" yaml:"deleted_extensions,omitempty"`
}

func (sd *SchemaDiff) compareViews(old []ViewInfo, new []ViewInfo) {
	oldViews := make(map[string]ViewInfo)
	for _, v := range old {
		key := fmt.Sprintf("%s.%s", v.Schema, v.Name)
		oldViews[key] = v
	}
	
	newViews := make(map[string]ViewInfo)
	for _, v := range new {
		key := fmt.Sprintf("%s.%s", v.Schema, v.Name)
		newViews[key] = v
	}
	
	for key, newView := range newViews {
		if _, exists := oldViews[key]; !exists {
			sd.AddedViews = append(sd.AddedViews, newView)
		}
	}
	
	for key, oldView := range oldViews {
		if _, exists := newViews[key]; !exists {
			sd.DeletedViews = append(sd.DeletedViews, oldView)
		}
	}
}

func (sd *SchemaDiff) compareRoles(old []Role, new []Role) {
	oldRoles := make(map[string]bool)
	for _, r := range old {
		oldRoles[r.Name] = true
	}
	
	newRoles := make(map[string]bool)
	for _, r := range new {
		newRoles[r.Name] = true
	}
	
	for role := range newRoles {
		if !oldRoles[role] {
			sd.AddedRoles = append(sd.AddedRoles, role)
		}
	}
	
	for role := range oldRoles {
		if !newRoles[role] {
			sd.DeletedRoles = append(sd.DeletedRoles, role)
		}
	}
}

func (sd *SchemaDiff) compareExtensions(old []Extension, new []Extension) {
	oldExts := make(map[string]Extension)
	for _, e := range old {
		oldExts[e.Name] = e
	}
	
	newExts := make(map[string]Extension)
	for _, e := range new {
		newExts[e.Name] = e
	}
	
	for name, newExt := range newExts {
		if _, exists := oldExts[name]; !exists {
			sd.AddedExtensions = append(sd.AddedExtensions, newExt)
		}
	}
	
	for name, oldExt := range oldExts {
		if _, exists := newExts[name]; !exists {
			sd.DeletedExtensions = append(sd.DeletedExtensions, oldExt)
		}
	}
}

// HasChanges returns true if there are any differences
func (sd *SchemaDiff) HasChanges() bool {
	return len(sd.AddedTables) > 0 || len(sd.DeletedTables) > 0 || len(sd.ModifiedTables) > 0 ||
		len(sd.AddedViews) > 0 || len(sd.DeletedViews) > 0 ||
		len(sd.AddedRoles) > 0 || len(sd.DeletedRoles) > 0 ||
		len(sd.AddedExtensions) > 0 || len(sd.DeletedExtensions) > 0
}
