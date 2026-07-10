package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SchemaDoc is the minimal JSON-Schema-shaped descriptor used by the exercise
// per ADR-014. Real Data Products on Hermes use full JSON Schema (Draft 2020-12);
// here we ship a deliberately tiny validator so the teaching point — schema
// versioning + lifecycle — stays visible.
type SchemaDoc struct {
	ID         string         `json:"$id"`
	Version    string         `json:"version"`
	Lifecycle  string         `json:"lifecycle"`
	Required   []string       `json:"required"`
	Properties map[string]any `json:"properties"`
}

// SchemaRegistry holds loaded SchemaDocs by id.
type SchemaRegistry struct {
	docs map[string]SchemaDoc
}

// LoadSchemaRegistry reads every *.json file under dir as a SchemaDoc.
func LoadSchemaRegistry(dir string) (*SchemaRegistry, error) {
	r := &SchemaRegistry{docs: map[string]SchemaDoc{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("schema registry: read dir %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("schema registry: read %s: %w", e.Name(), err)
		}
		var doc SchemaDoc
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("schema registry: parse %s: %w", e.Name(), err)
		}
		if doc.ID == "" {
			return nil, fmt.Errorf("schema registry: %s missing $id", e.Name())
		}
		if doc.Lifecycle != "stable" && doc.Lifecycle != "deprecated" && doc.Lifecycle != "sunset" {
			return nil, fmt.Errorf("schema registry: %s invalid lifecycle %q", e.Name(), doc.Lifecycle)
		}
		r.docs[doc.ID] = doc
	}
	return r, nil
}

// Validate checks that every required field of the schema is present in payload.
// Type checks are intentionally skipped — this validator is a teaching stand-in.
func (r *SchemaRegistry) Validate(schemaID string, payload map[string]any) error {
	doc, ok := r.docs[schemaID]
	if !ok {
		return fmt.Errorf("schema registry: unknown id %q", schemaID)
	}
	for _, k := range doc.Required {
		v, ok := payload[k]
		if !ok || v == nil {
			return fmt.Errorf("schema %s: missing required field %q", schemaID, k)
		}
		if s, isStr := v.(string); isStr && s == "" {
			return fmt.Errorf("schema %s: required field %q is empty", schemaID, k)
		}
	}
	return nil
}

// Lifecycle returns the lifecycle stage of a schema.
func (r *SchemaRegistry) Lifecycle(schemaID string) (string, error) {
	doc, ok := r.docs[schemaID]
	if !ok {
		return "", errors.New("schema registry: unknown id " + schemaID)
	}
	return doc.Lifecycle, nil
}
