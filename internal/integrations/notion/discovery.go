//go:build js && wasm

package notion

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DatabaseDiscovery provides helpers for discovering and working with databases.
type DatabaseDiscovery struct {
	client      *Client
	cache       map[string]*Database
	cacheMu     sync.RWMutex
	cacheExpiry time.Time
}

// NewDatabaseDiscovery creates a new discovery helper.
func NewDatabaseDiscovery(client *Client) *DatabaseDiscovery {
	return &DatabaseDiscovery{
		client:      client,
		cache:       make(map[string]*Database),
		cacheExpiry: time.Now().Add(-1 * time.Hour), // Expired initially
	}
}

// FindByName searches for a database by name (case-insensitive).
func (d *DatabaseDiscovery) FindByName(name string) (*Database, error) {
	// Try cache first
	d.cacheMu.RLock()
	for _, db := range d.cache {
		title := getDatabaseTitle(db)
		if strings.EqualFold(title, name) {
			d.cacheMu.RUnlock()
			return db, nil
		}
	}
	d.cacheMu.RUnlock()

	// Search via API
	databases, err := d.client.ListDatabases()
	if err != nil {
		return nil, err
	}

	// Update cache
	d.cacheMu.Lock()
	for _, db := range databases {
		d.cache[db.ID] = db
	}
	d.cacheExpiry = time.Now().Add(5 * time.Minute)
	d.cacheMu.Unlock()

	// Find best match
	var bestMatch *Database
	for _, db := range databases {
		title := getDatabaseTitle(db)
		if strings.EqualFold(title, name) {
			return db, nil
		}
		// Partial match
		if strings.Contains(strings.ToLower(title), strings.ToLower(name)) {
			bestMatch = db
		}
	}

	if bestMatch != nil {
		return bestMatch, nil
	}

	return nil, fmt.Errorf("database '%s' not found", name)
}

// GetDatabase retrieves a database by ID, using cache when possible.
func (d *DatabaseDiscovery) GetDatabase(databaseID string) (*Database, error) {
	// Check cache
	d.cacheMu.RLock()
	if db, ok := d.cache[databaseID]; ok && time.Now().Before(d.cacheExpiry) {
		d.cacheMu.RUnlock()
		return db, nil
	}
	d.cacheMu.RUnlock()

	// Fetch from API
	db, err := d.client.GetDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	// Update cache
	d.cacheMu.Lock()
	d.cache[databaseID] = db
	d.cacheMu.Unlock()

	return db, nil
}

// GetPropertySchema returns the property schema for a database.
func (d *DatabaseDiscovery) GetPropertySchema(databaseID string) (map[string]PropertySchema, error) {
	db, err := d.GetDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	return db.Properties, nil
}

// BuildQueryFromNatural builds a query from natural language filter strings.
// Example: {"status": "done", "priority": "high"}
func (d *DatabaseDiscovery) BuildQueryFromNatural(databaseID string, filters map[string]string) (*Query, error) {
	schema, err := d.GetPropertySchema(databaseID)
	if err != nil {
		return nil, err
	}

	return QueryFromMap(filters, schema)
}

// ConvertStringToPropertyValue converts a string value to the appropriate PropertyValue type.
func (d *DatabaseDiscovery) ConvertStringToPropertyValue(propType, value string) (PropertyValue, error) {
	switch propType {
	case "title":
		return PropertyValue{
			Type: "title",
			Title: &TitleProperty{
				Title: []RichText{
					{Type: "text", Text: &Text{Content: value}, PlainText: value},
				},
			},
		}, nil

	case "rich_text":
		return PropertyValue{
			Type: "rich_text",
			RichText: &RichTextProperty{
				RichText: []RichText{
					{Type: "text", Text: &Text{Content: value}, PlainText: value},
				},
			},
		}, nil

	case "select":
		return PropertyValue{
			Type: "select",
			Select: &SelectProperty{
				Select: &SelectOption{Name: value},
			},
		}, nil

	case "multi_select":
		// Handle comma-separated values
		options := []SelectOption{}
		for _, v := range strings.Split(value, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				options = append(options, SelectOption{Name: v})
			}
		}
		return PropertyValue{
			Type: "multi_select",
			MultiSelect: &MultiSelectProperty{
				MultiSelect: options,
			},
		}, nil

	case "status":
		return PropertyValue{
			Type: "status",
			Status: &StatusProperty{
				Status: &SelectOption{Name: value},
			},
		}, nil

	case "checkbox":
		checked := strings.ToLower(value) == "true" ||
			strings.ToLower(value) == "yes" ||
			strings.ToLower(value) == "checked" ||
			strings.ToLower(value) == "done"
		return PropertyValue{
			Type:     "checkbox",
			Checkbox: &CheckboxProperty{Checkbox: checked},
		}, nil

	case "date":
		return PropertyValue{
			Type: "date",
			Date: &DateProperty{
				Date: &DateRange{Start: value},
			},
		}, nil

	case "number":
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return PropertyValue{}, fmt.Errorf("cannot parse '%s' as number: %w", value, err)
		}
		return PropertyValue{
			Type:   "number",
			Number: &NumberProperty{Number: num},
		}, nil

	case "url":
		return PropertyValue{
			Type: "url",
			URLProp: &URLProperty{
				URL: value,
			},
		}, nil

	case "email":
		return PropertyValue{
			Type: "email",
			Email: &EmailProperty{
				Email: value,
			},
		}, nil

	case "phone_number":
		return PropertyValue{
			Type: "phone_number",
			Phone: &PhoneProperty{
				PhoneNumber: value,
			},
		}, nil

	case "relation":
		// Relation requires a page ID
		return PropertyValue{
			Type: "relation",
			Relation: &RelationProperty{
				Relation: []RelationEntry{{ID: value}},
			},
		}, nil

	default:
		// Default to rich_text
		return PropertyValue{
			Type: "rich_text",
			RichText: &RichTextProperty{
				RichText: []RichText{
					{Type: "text", Text: &Text{Content: value}, PlainText: value},
				},
			},
		}, nil
	}
}

// ConvertPropertiesMap converts a map of string properties to PropertyValues.
func (d *DatabaseDiscovery) ConvertPropertiesMap(databaseID string, props map[string]string) (map[string]PropertyValue, error) {
	schema, err := d.GetPropertySchema(databaseID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]PropertyValue)
	for propName, value := range props {
		// Find property in schema (case-insensitive)
		var propSchema *PropertySchema
		for name, ps := range schema {
			if strings.EqualFold(name, propName) {
				propSchema = &ps
				break
			}
		}

		if propSchema == nil {
			// Skip unknown properties
			continue
		}

		pv, err := d.ConvertStringToPropertyValue(propSchema.Type, value)
		if err != nil {
			return nil, fmt.Errorf("property '%s': %w", propName, err)
		}

		result[propName] = pv
	}

	return result, nil
}

// ClearCache clears the database cache.
func (d *DatabaseDiscovery) ClearCache() {
	d.cacheMu.Lock()
	d.cache = make(map[string]*Database)
	d.cacheExpiry = time.Now().Add(-1 * time.Hour)
	d.cacheMu.Unlock()
}

// GetCachedDatabases returns all cached databases.
func (d *DatabaseDiscovery) GetCachedDatabases() []*Database {
	d.cacheMu.RLock()
	defer d.cacheMu.RUnlock()

	databases := make([]*Database, 0, len(d.cache))
	for _, db := range d.cache {
		databases = append(databases, db)
	}
	return databases
}

// FindPropertyByName finds a property in a database by name (case-insensitive).
func (d *DatabaseDiscovery) FindPropertyByName(databaseID string, propertyName string) (*PropertySchema, error) {
	schema, err := d.GetPropertySchema(databaseID)
	if err != nil {
		return nil, err
	}

	for name, prop := range schema {
		if strings.EqualFold(name, propertyName) {
			propCopy := prop
			return &propCopy, nil
		}
	}

	return nil, fmt.Errorf("property '%s' not found in database", propertyName)
}

// GetSelectOptions returns the available options for a select or multi_select property.
func (d *DatabaseDiscovery) GetSelectOptions(databaseID string, propertyName string) ([]SelectOption, error) {
	prop, err := d.FindPropertyByName(databaseID, propertyName)
	if err != nil {
		return nil, err
	}

	if prop.Type == "select" && prop.Select != nil {
		return prop.Select.Options, nil
	}

	if prop.Type == "multi_select" && prop.MultiSelect != nil {
		return prop.MultiSelect.Options, nil
	}

	return nil, fmt.Errorf("property '%s' is not a select type", propertyName)
}
