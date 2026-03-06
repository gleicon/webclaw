//go:build js && wasm

package notion

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Query represents a database query with filters and sorts.
type Query struct {
	Filter      *Filter `json:"filter,omitempty"`
	Sorts       []Sort  `json:"sorts,omitempty"`
	StartCursor string  `json:"start_cursor,omitempty"`
	PageSize    int     `json:"page_size,omitempty"`
}

// Filter represents a Notion database query filter.
type Filter struct {
	// Single property filters (mutually exclusive)
	Property       string             `json:"property,omitempty"`
	Title          *TitleFilter       `json:"title,omitempty"`
	RichText       *RichTextFilter    `json:"rich_text,omitempty"`
	Select         *SelectFilter      `json:"select,omitempty"`
	MultiSelect    *MultiSelectFilter `json:"multi_select,omitempty"`
	Date           *DateFilter        `json:"date,omitempty"`
	Checkbox       *CheckboxFilter    `json:"checkbox,omitempty"`
	Number         *NumberFilter      `json:"number,omitempty"`
	Status         *StatusFilter      `json:"status,omitempty"`
	Email          *EmailFilter       `json:"email,omitempty"`
	URL            *URLFilter         `json:"url,omitempty"`
	Phone          *PhoneFilter       `json:"phone,omitempty"`
	Relation       *RelationFilter    `json:"relation,omitempty"`
	Formula        *FormulaFilter     `json:"formula,omitempty"`
	CreatedTime    *TimestampFilter   `json:"created_time,omitempty"`
	LastEditedTime *TimestampFilter   `json:"last_edited_time,omitempty"`

	// Compound filters
	And []*Filter `json:"and,omitempty"`
	Or  []*Filter `json:"or,omitempty"`
}

// TitleFilter filters by title property.
type TitleFilter struct {
	Equals         string `json:"equals,omitempty"`
	DoesNotEqual   string `json:"does_not_equal,omitempty"`
	Contains       string `json:"contains,omitempty"`
	DoesNotContain string `json:"does_not_contain,omitempty"`
	StartsWith     string `json:"starts_with,omitempty"`
	EndsWith       string `json:"ends_with,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// RichTextFilter filters by rich text property.
type RichTextFilter struct {
	Equals         string `json:"equals,omitempty"`
	DoesNotEqual   string `json:"does_not_equal,omitempty"`
	Contains       string `json:"contains,omitempty"`
	DoesNotContain string `json:"does_not_contain,omitempty"`
	StartsWith     string `json:"starts_with,omitempty"`
	EndsWith       string `json:"ends_with,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// SelectFilter filters by select property.
type SelectFilter struct {
	Equals       string `json:"equals,omitempty"`
	DoesNotEqual string `json:"does_not_equal,omitempty"`
	IsEmpty      bool   `json:"is_empty,omitempty"`
	IsNotEmpty   bool   `json:"is_not_empty,omitempty"`
}

// MultiSelectFilter filters by multi-select property.
type MultiSelectFilter struct {
	Contains       string `json:"contains,omitempty"`
	DoesNotContain string `json:"does_not_contain,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// StatusFilter filters by status property.
type StatusFilter struct {
	Equals       string `json:"equals,omitempty"`
	DoesNotEqual string `json:"does_not_equal,omitempty"`
	IsEmpty      bool   `json:"is_empty,omitempty"`
	IsNotEmpty   bool   `json:"is_not_empty,omitempty"`
}

// DateFilter filters by date property.
type DateFilter struct {
	Equals     string    `json:"equals,omitempty"`       // ISO 8601
	Before     string    `json:"before,omitempty"`       // ISO 8601
	After      string    `json:"after,omitempty"`        // ISO 8601
	OnOrBefore string    `json:"on_or_before,omitempty"` // ISO 8601
	OnOrAfter  string    `json:"on_or_after,omitempty"`  // ISO 8601
	PastWeek   *struct{} `json:"past_week,omitempty"`
	PastMonth  *struct{} `json:"past_month,omitempty"`
	PastYear   *struct{} `json:"past_year,omitempty"`
	ThisWeek   *struct{} `json:"this_week,omitempty"`
	IsEmpty    bool      `json:"is_empty,omitempty"`
	IsNotEmpty bool      `json:"is_not_empty,omitempty"`
}

// CheckboxFilter filters by checkbox property.
type CheckboxFilter struct {
	Equals       bool `json:"equals,omitempty"`
	DoesNotEqual bool `json:"does_not_equal,omitempty"`
}

// NumberFilter filters by number property.
type NumberFilter struct {
	Equals               float64 `json:"equals,omitempty"`
	DoesNotEqual         float64 `json:"does_not_equal,omitempty"`
	GreaterThan          float64 `json:"greater_than,omitempty"`
	LessThan             float64 `json:"less_than,omitempty"`
	GreaterThanOrEqualTo float64 `json:"greater_than_or_equal_to,omitempty"`
	LessThanOrEqualTo    float64 `json:"less_than_or_equal_to,omitempty"`
	IsEmpty              bool    `json:"is_empty,omitempty"`
	IsNotEmpty           bool    `json:"is_not_empty,omitempty"`
}

// EmailFilter filters by email property.
type EmailFilter struct {
	Equals         string `json:"equals,omitempty"`
	DoesNotEqual   string `json:"does_not_equal,omitempty"`
	Contains       string `json:"contains,omitempty"`
	DoesNotContain string `json:"does_not_contain,omitempty"`
	StartsWith     string `json:"starts_with,omitempty"`
	EndsWith       string `json:"ends_with,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// URLFilter filters by URL property.
type URLFilter struct {
	Equals         string `json:"equals,omitempty"`
	DoesNotEqual   string `json:"does_not_equal,omitempty"`
	Contains       string `json:"contains,omitempty"`
	DoesNotContain string `json:"does_not_contain,omitempty"`
	StartsWith     string `json:"starts_with,omitempty"`
	EndsWith       string `json:"ends_with,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// PhoneFilter filters by phone property.
type PhoneFilter struct {
	Equals         string `json:"equals,omitempty"`
	DoesNotEqual   string `json:"does_not_equal,omitempty"`
	Contains       string `json:"contains,omitempty"`
	DoesNotContain string `json:"does_not_contain,omitempty"`
	StartsWith     string `json:"starts_with,omitempty"`
	EndsWith       string `json:"ends_with,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// RelationFilter filters by relation property.
type RelationFilter struct {
	Contains       string `json:"contains,omitempty"` // Page ID
	DoesNotContain string `json:"does_not_contain,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// FormulaFilter filters by formula result.
type FormulaFilter struct {
	Text     *RichTextFilter `json:"text,omitempty"`
	Checkbox *CheckboxFilter `json:"checkbox,omitempty"`
	Number   *NumberFilter   `json:"number,omitempty"`
	Date     *DateFilter     `json:"date,omitempty"`
}

// TimestampFilter filters by created_time or last_edited_time.
type TimestampFilter struct {
	Equals     string    `json:"equals,omitempty"`
	Before     string    `json:"before,omitempty"`
	After      string    `json:"after,omitempty"`
	OnOrBefore string    `json:"on_or_before,omitempty"`
	OnOrAfter  string    `json:"on_or_after,omitempty"`
	PastWeek   *struct{} `json:"past_week,omitempty"`
	PastMonth  *struct{} `json:"past_month,omitempty"`
	PastYear   *struct{} `json:"past_year,omitempty"`
	ThisWeek   *struct{} `json:"this_week,omitempty"`
}

// Sort represents a sort order for query results.
type Sort struct {
	Property  string `json:"property,omitempty"`
	Timestamp string `json:"timestamp,omitempty"` // created_time, last_edited_time
	Direction string `json:"direction"`           // ascending, descending
}

// QueryBuilder provides a fluent API for building Notion queries.
type QueryBuilder struct {
	query Query
}

// NewQuery creates a new QueryBuilder.
func NewQuery() *QueryBuilder {
	return &QueryBuilder{
		query: Query{
			PageSize: 100,
		},
	}
}

// WhereTitle adds a title filter.
func (qb *QueryBuilder) WhereTitle(property string, condition string, value string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Title:    &TitleFilter{},
	}

	switch strings.ToLower(condition) {
	case "equals", "=":
		filter.Title.Equals = value
	case "does_not_equal", "!=":
		filter.Title.DoesNotEqual = value
	case "contains":
		filter.Title.Contains = value
	case "does_not_contain":
		filter.Title.DoesNotContain = value
	case "starts_with":
		filter.Title.StartsWith = value
	case "ends_with":
		filter.Title.EndsWith = value
	case "is_empty", "empty":
		filter.Title.IsEmpty = true
	case "is_not_empty", "not_empty":
		filter.Title.IsNotEmpty = true
	}

	qb.addFilter(filter)
	return qb
}

// WhereRichText adds a rich text filter.
func (qb *QueryBuilder) WhereRichText(property string, condition string, value string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		RichText: &RichTextFilter{},
	}

	switch strings.ToLower(condition) {
	case "equals", "=":
		filter.RichText.Equals = value
	case "does_not_equal", "!=":
		filter.RichText.DoesNotEqual = value
	case "contains":
		filter.RichText.Contains = value
	case "does_not_contain":
		filter.RichText.DoesNotContain = value
	case "starts_with":
		filter.RichText.StartsWith = value
	case "ends_with":
		filter.RichText.EndsWith = value
	case "is_empty", "empty":
		filter.RichText.IsEmpty = true
	case "is_not_empty", "not_empty":
		filter.RichText.IsNotEmpty = true
	}

	qb.addFilter(filter)
	return qb
}

// WhereSelect adds a select filter.
func (qb *QueryBuilder) WhereSelect(property string, value string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Select:   &SelectFilter{Equals: value},
	}
	qb.addFilter(filter)
	return qb
}

// WhereSelectNotEquals adds a select does-not-equal filter.
func (qb *QueryBuilder) WhereSelectNotEquals(property string, value string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Select:   &SelectFilter{DoesNotEqual: value},
	}
	qb.addFilter(filter)
	return qb
}

// WhereMultiSelectContains adds a multi-select contains filter.
func (qb *QueryBuilder) WhereMultiSelectContains(property string, value string) *QueryBuilder {
	filter := &Filter{
		Property:    property,
		MultiSelect: &MultiSelectFilter{Contains: value},
	}
	qb.addFilter(filter)
	return qb
}

// WhereStatus adds a status filter.
func (qb *QueryBuilder) WhereStatus(property string, value string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Status:   &StatusFilter{Equals: value},
	}
	qb.addFilter(filter)
	return qb
}

// WhereDateEquals adds a date equals filter.
func (qb *QueryBuilder) WhereDateEquals(property string, date string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Date:     &DateFilter{Equals: date},
	}
	qb.addFilter(filter)
	return qb
}

// WhereDateAfter adds a date after filter.
func (qb *QueryBuilder) WhereDateAfter(property string, date string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Date:     &DateFilter{After: date},
	}
	qb.addFilter(filter)
	return qb
}

// WhereDateBefore adds a date before filter.
func (qb *QueryBuilder) WhereDateBefore(property string, date string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Date:     &DateFilter{Before: date},
	}
	qb.addFilter(filter)
	return qb
}

// WhereDateOnOrAfter adds a date on-or-after filter.
func (qb *QueryBuilder) WhereDateOnOrAfter(property string, date string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Date:     &DateFilter{OnOrAfter: date},
	}
	qb.addFilter(filter)
	return qb
}

// WhereDateOnOrBefore adds a date on-or-before filter.
func (qb *QueryBuilder) WhereDateOnOrBefore(property string, date string) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Date:     &DateFilter{OnOrBefore: date},
	}
	qb.addFilter(filter)
	return qb
}

// WhereCheckbox adds a checkbox filter.
func (qb *QueryBuilder) WhereCheckbox(property string, checked bool) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Checkbox: &CheckboxFilter{Equals: checked},
	}
	qb.addFilter(filter)
	return qb
}

// WhereNumberEquals adds a number equals filter.
func (qb *QueryBuilder) WhereNumberEquals(property string, value float64) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Number:   &NumberFilter{Equals: value},
	}
	qb.addFilter(filter)
	return qb
}

// WhereNumberGreaterThan adds a number greater-than filter.
func (qb *QueryBuilder) WhereNumberGreaterThan(property string, value float64) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Number:   &NumberFilter{GreaterThan: value},
	}
	qb.addFilter(filter)
	return qb
}

// WhereNumberLessThan adds a number less-than filter.
func (qb *QueryBuilder) WhereNumberLessThan(property string, value float64) *QueryBuilder {
	filter := &Filter{
		Property: property,
		Number:   &NumberFilter{LessThan: value},
	}
	qb.addFilter(filter)
	return qb
}

// OrderBy adds a property sort.
func (qb *QueryBuilder) OrderBy(property string, direction string) *QueryBuilder {
	// Normalize direction
	dir := strings.ToLower(direction)
	if dir != "ascending" && dir != "descending" {
		dir = "ascending"
	}

	qb.query.Sorts = append(qb.query.Sorts, Sort{
		Property:  property,
		Direction: dir,
	})
	return qb
}

// OrderByCreated adds a created_time sort.
func (qb *QueryBuilder) OrderByCreated(direction string) *QueryBuilder {
	dir := strings.ToLower(direction)
	if dir != "ascending" && dir != "descending" {
		dir = "descending"
	}

	qb.query.Sorts = append(qb.query.Sorts, Sort{
		Timestamp: "created_time",
		Direction: dir,
	})
	return qb
}

// OrderByLastEdited adds a last_edited_time sort.
func (qb *QueryBuilder) OrderByLastEdited(direction string) *QueryBuilder {
	dir := strings.ToLower(direction)
	if dir != "ascending" && dir != "descending" {
		dir = "descending"
	}

	qb.query.Sorts = append(qb.query.Sorts, Sort{
		Timestamp: "last_edited_time",
		Direction: dir,
	})
	return qb
}

// Limit sets the page size.
func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	if n > 0 && n <= 100 {
		qb.query.PageSize = n
	}
	return qb
}

// And combines multiple filters with AND logic.
func (qb *QueryBuilder) And(filters ...*Filter) *QueryBuilder {
	if len(filters) == 0 {
		return qb
	}

	compound := &Filter{
		And: filters,
	}

	qb.addFilter(compound)
	return qb
}

// Or combines multiple filters with OR logic.
func (qb *QueryBuilder) Or(filters ...*Filter) *QueryBuilder {
	if len(filters) == 0 {
		return qb
	}

	compound := &Filter{
		Or: filters,
	}

	qb.addFilter(compound)
	return qb
}

// addFilter adds a filter to the query.
func (qb *QueryBuilder) addFilter(filter *Filter) {
	if qb.query.Filter == nil {
		qb.query.Filter = filter
		return
	}

	// If we already have a compound AND filter, add to it
	if qb.query.Filter.And != nil {
		qb.query.Filter.And = append(qb.query.Filter.And, filter)
		return
	}

	// Wrap existing and new filter in an AND compound
	qb.query.Filter = &Filter{
		And: []*Filter{qb.query.Filter, filter},
	}
}

// Build returns the constructed Query.
func (qb *QueryBuilder) Build() *Query {
	return &qb.query
}

// ToJSON returns the query as JSON string (for debugging).
func (qb *QueryBuilder) ToJSON() (string, error) {
	data, err := json.Marshal(qb.query)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ValidateQuery validates that property names exist in the database schema.
func ValidateQuery(query *Query, schema map[string]PropertySchema) error {
	if query == nil || query.Filter == nil {
		return nil
	}

	return validateFilter(query.Filter, schema)
}

func validateFilter(filter *Filter, schema map[string]PropertySchema) error {
	// Check compound filters
	if filter.And != nil {
		for _, f := range filter.And {
			if err := validateFilter(f, schema); err != nil {
				return err
			}
		}
		return nil
	}
	if filter.Or != nil {
		for _, f := range filter.Or {
			if err := validateFilter(f, schema); err != nil {
				return err
			}
		}
		return nil
	}

	// Check single property filter
	if filter.Property == "" {
		return nil // No property to validate
	}

	propSchema, exists := schema[filter.Property]
	if !exists {
		// Try case-insensitive match
		for name, ps := range schema {
			if strings.EqualFold(name, filter.Property) {
				propSchema = ps
				exists = true
				break
			}
		}
	}

	if !exists {
		return fmt.Errorf("property '%s' not found in database schema", filter.Property)
	}

	// Validate filter type matches property type
	filterType := getFilterType(filter)
	if filterType != "" && filterType != propSchema.Type {
		return fmt.Errorf("filter type '%s' doesn't match property type '%s' for '%s'",
			filterType, propSchema.Type, filter.Property)
	}

	return nil
}

// getFilterType returns the type of filter based on which field is set.
func getFilterType(filter *Filter) string {
	if filter.Title != nil {
		return "title"
	}
	if filter.RichText != nil {
		return "rich_text"
	}
	if filter.Select != nil {
		return "select"
	}
	if filter.MultiSelect != nil {
		return "multi_select"
	}
	if filter.Status != nil {
		return "status"
	}
	if filter.Date != nil {
		return "date"
	}
	if filter.Checkbox != nil {
		return "checkbox"
	}
	if filter.Number != nil {
		return "number"
	}
	if filter.Email != nil {
		return "email"
	}
	if filter.URL != nil {
		return "url"
	}
	if filter.Phone != nil {
		return "phone_number"
	}
	if filter.Relation != nil {
		return "relation"
	}
	if filter.Formula != nil {
		return "formula"
	}
	if filter.CreatedTime != nil {
		return "created_time"
	}
	if filter.LastEditedTime != nil {
		return "last_edited_time"
	}
	return ""
}

// QueryFromMap creates a query from a simple map of property filters.
// This is useful for building queries from natural language.
func QueryFromMap(filters map[string]string, schema map[string]PropertySchema) (*Query, error) {
	builder := NewQuery()

	for propName, value := range filters {
		// Find the property in schema
		var propSchema PropertySchema
		found := false

		for name, ps := range schema {
			if strings.EqualFold(name, propName) {
				propSchema = ps
				found = true
				break
			}
		}

		if !found {
			// Try as title property
			for name, ps := range schema {
				if ps.Type == "title" {
					builder.WhereTitle(name, "contains", value)
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("property '%s' not found in database", propName)
			}
			continue
		}

		// Build appropriate filter based on property type
		switch propSchema.Type {
		case "title":
			builder.WhereTitle(propSchema.Name, "contains", value)
		case "rich_text":
			builder.WhereRichText(propSchema.Name, "contains", value)
		case "select":
			builder.WhereSelect(propSchema.Name, value)
		case "multi_select":
			builder.WhereMultiSelectContains(propSchema.Name, value)
		case "status":
			builder.WhereStatus(propSchema.Name, value)
		case "checkbox":
			checked := strings.ToLower(value) == "true" ||
				strings.ToLower(value) == "yes" ||
				strings.ToLower(value) == "checked"
			builder.WhereCheckbox(propSchema.Name, checked)
		case "number":
			var num float64
			if _, err := fmt.Sscanf(value, "%f", &num); err == nil {
				builder.WhereNumberEquals(propSchema.Name, num)
			}
		case "date":
			builder.WhereDateEquals(propSchema.Name, value)
		default:
			// Default to rich_text filter
			builder.WhereRichText(propSchema.Name, "contains", value)
		}
	}

	return builder.Build(), nil
}
