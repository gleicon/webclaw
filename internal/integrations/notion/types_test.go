//go:build js && wasm

package notion

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestTypesMarshalUnmarshal(t *testing.T) {
	// Test Page marshaling
	page := &Page{
		ID:             "test-page-id",
		CreatedTime:    "2024-01-01T00:00:00.000Z",
		LastEditedTime: "2024-01-02T00:00:00.000Z",
		URL:            "https://notion.so/test-page-id",
		Properties: map[string]PropertyValue{
			"Name": {
				Type: "title",
				Title: &TitleProperty{
					Title: []RichText{
						{Type: "text", Text: &Text{Content: "Test Page"}, PlainText: "Test Page"},
					},
				},
			},
			"Status": {
				Type: "select",
				Select: &SelectProperty{
					Select: &SelectOption{Name: "In Progress"},
				},
			},
		},
	}

	data, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("Failed to marshal page: %v", err)
	}

	var unmarshaled Page
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal page: %v", err)
	}

	if unmarshaled.ID != page.ID {
		t.Errorf("ID mismatch: got %s, want %s", unmarshaled.ID, page.ID)
	}

	titleProp := unmarshaled.Properties["Name"]
	if titleProp.Type != "title" {
		t.Errorf("Title type mismatch: got %s, want title", titleProp.Type)
	}
}

func TestPropertyValueTypes(t *testing.T) {
	tests := []struct {
		name  string
		value PropertyValue
	}{
		{
			name: "title",
			value: PropertyValue{
				Type: "title",
				Title: &TitleProperty{
					Title: []RichText{{PlainText: "Title"}},
				},
			},
		},
		{
			name: "rich_text",
			value: PropertyValue{
				Type: "rich_text",
				RichText: &RichTextProperty{
					RichText: []RichText{{PlainText: "Rich text"}},
				},
			},
		},
		{
			name: "select",
			value: PropertyValue{
				Type: "select",
				Select: &SelectProperty{
					Select: &SelectOption{Name: "Option 1"},
				},
			},
		},
		{
			name: "multi_select",
			value: PropertyValue{
				Type: "multi_select",
				MultiSelect: &MultiSelectProperty{
					MultiSelect: []SelectOption{
						{Name: "Tag 1"},
						{Name: "Tag 2"},
					},
				},
			},
		},
		{
			name: "checkbox",
			value: PropertyValue{
				Type:     "checkbox",
				Checkbox: &CheckboxProperty{Checkbox: true},
			},
		},
		{
			name: "number",
			value: PropertyValue{
				Type:   "number",
				Number: &NumberProperty{Number: 42.5},
			},
		},
		{
			name: "date",
			value: PropertyValue{
				Type: "date",
				Date: &DateProperty{
					Date: &DateRange{Start: "2024-01-01"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Failed to marshal %s: %v", tt.name, err)
			}

			var unmarshaled PropertyValue
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal %s: %v", tt.name, err)
			}

			if unmarshaled.Type != tt.value.Type {
				t.Errorf("Type mismatch: got %s, want %s", unmarshaled.Type, tt.value.Type)
			}
		})
	}
}

func TestRichTextParsing(t *testing.T) {
	richTexts := []RichText{
		{
			Type:      "text",
			Text:      &Text{Content: "Hello ", Link: nil},
			PlainText: "Hello ",
			Annotations: &Annotations{
				Bold: true,
			},
		},
		{
			Type:      "text",
			Text:      &Text{Content: "World", Link: &Link{URL: "https://example.com"}},
			PlainText: "World",
		},
	}

	result := richTextToPlainText(richTexts)
	expected := "Hello World"
	if result != expected {
		t.Errorf("Plain text mismatch: got '%s', want '%s'", result, expected)
	}
}

func TestBlockStructure(t *testing.T) {
	block := Block{
		ID:   "block-id",
		Type: "paragraph",
		Paragraph: &Paragraph{
			RichText: []RichText{
				{PlainText: "Test paragraph"},
			},
		},
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Failed to marshal block: %v", err)
	}

	var unmarshaled Block
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal block: %v", err)
	}

	if unmarshaled.Type != "paragraph" {
		t.Errorf("Type mismatch: got %s, want paragraph", unmarshaled.Type)
	}
}

func TestNotionError(t *testing.T) {
	err := &NotionError{
		Object:  "error",
		Status:  400,
		Code:    "validation_error",
		Message: "Invalid property value",
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "validation_error") {
		t.Errorf("Error message should contain code: %s", errStr)
	}
	if !strings.Contains(errStr, "Invalid property value") {
		t.Errorf("Error message should contain message: %s", errStr)
	}
}

func TestIDCleaning(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"12345678-1234-1234-1234-123456789abc", "12345678123412341234123456789abc"},
		{"12345678123412341234123456789abc", "12345678123412341234123456789abc"},
		{"https://www.notion.so/workspace/My-Page-12345678123412341234123456789abc", "12345678123412341234123456789abc"},
		{"https://notion.so/page/12345678123412341234123456789abc", "12345678123412341234123456789abc"},
	}

	for _, tt := range tests {
		result := cleanID(tt.input)
		if result != tt.expected {
			t.Errorf("cleanID(%s): got %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestQueryBuilder(t *testing.T) {
	// Test simple filter
	query := NewQuery().
		WhereSelect("Status", "Done").
		OrderByCreated("descending").
		Limit(50).
		Build()

	if query.Filter == nil {
		t.Fatal("Filter should not be nil")
	}
	if query.Filter.Property != "Status" {
		t.Errorf("Filter property: got %s, want Status", query.Filter.Property)
	}
	if query.PageSize != 50 {
		t.Errorf("Page size: got %d, want 50", query.PageSize)
	}

	// Test compound filter
	query2 := NewQuery().
		WhereSelect("Status", "Done").
		WhereCheckbox("Archived", false).
		Build()

	if query2.Filter.And == nil {
		t.Fatal("Compound filter should use And")
	}
	if len(query2.Filter.And) != 2 {
		t.Errorf("And filter should have 2 conditions, got %d", len(query2.Filter.And))
	}
}

func TestQueryValidation(t *testing.T) {
	schema := map[string]PropertySchema{
		"Name":   {Name: "Name", Type: "title"},
		"Status": {Name: "Status", Type: "select"},
		"Done":   {Name: "Done", Type: "checkbox"},
	}

	query := NewQuery().WhereSelect("Status", "Done").Build()
	if err := ValidateQuery(query, schema); err != nil {
		t.Errorf("Valid query should not error: %v", err)
	}

	query2 := NewQuery().WhereSelect("UnknownProp", "Value").Build()
	if err := ValidateQuery(query2, schema); err == nil {
		t.Error("Invalid query should error")
	}
}

func TestDiscoveryConvertPropertyValue(t *testing.T) {
	discovery := NewDatabaseDiscovery(nil)

	tests := []struct {
		propType string
		value    string
		expected string
	}{
		{"checkbox", "true", "checkbox"},
		{"checkbox", "false", "checkbox"},
		{"checkbox", "yes", "checkbox"},
		{"number", "42.5", "number"},
		{"number", "100", "number"},
		{"select", "Option 1", "select"},
		{"title", "Title Text", "title"},
		{"rich_text", "Some text", "rich_text"},
		{"date", "2024-01-01", "date"},
		{"url", "https://example.com", "url"},
		{"email", "test@example.com", "email"},
		{"unknown_type", "value", "rich_text"}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.propType+"_"+tt.value, func(t *testing.T) {
			pv, err := discovery.ConvertStringToPropertyValue(tt.propType, tt.value)
			if err != nil {
				t.Fatalf("Failed to convert %s: %v", tt.propType, err)
			}
			if pv.Type != tt.expected {
				t.Errorf("Type mismatch: got %s, want %s", pv.Type, tt.expected)
			}
		})
	}
}

func TestDiscoveryCheckboxValues(t *testing.T) {
	discovery := NewDatabaseDiscovery(nil)

	trueValues := []string{"true", "True", "TRUE", "yes", "YES", "checked", "done", "Done"}
	falseValues := []string{"false", "False", "FALSE", "no", "unchecked"}

	for _, v := range trueValues {
		pv, err := discovery.ConvertStringToPropertyValue("checkbox", v)
		if err != nil {
			t.Errorf("Failed to convert %s: %v", v, err)
			continue
		}
		if !pv.Checkbox.Checkbox {
			t.Errorf("%s should be true", v)
		}
	}

	for _, v := range falseValues {
		pv, err := discovery.ConvertStringToPropertyValue("checkbox", v)
		if err != nil {
			t.Errorf("Failed to convert %s: %v", v, err)
			continue
		}
		if pv.Checkbox.Checkbox {
			t.Errorf("%s should be false", v)
		}
	}
}

func TestDiscoveryMultiSelect(t *testing.T) {
	discovery := NewDatabaseDiscovery(nil)

	pv, err := discovery.ConvertStringToPropertyValue("multi_select", "Tag 1, Tag 2, Tag 3")
	if err != nil {
		t.Fatalf("Failed to convert multi_select: %v", err)
	}

	if pv.Type != "multi_select" {
		t.Errorf("Type mismatch: got %s, want multi_select", pv.Type)
	}

	if len(pv.MultiSelect.MultiSelect) != 3 {
		t.Errorf("Expected 3 options, got %d", len(pv.MultiSelect.MultiSelect))
	}
}

func TestDiscoveryNumberParsing(t *testing.T) {
	discovery := NewDatabaseDiscovery(nil)

	tests := []struct {
		value    string
		expected float64
	}{
		{"42", 42},
		{"42.5", 42.5},
		{"-10", -10},
		{"0.001", 0.001},
	}

	for _, tt := range tests {
		pv, err := discovery.ConvertStringToPropertyValue("number", tt.value)
		if err != nil {
			t.Errorf("Failed to convert %s: %v", tt.value, err)
			continue
		}
		if pv.Number.Number != tt.expected {
			t.Errorf("Number mismatch for %s: got %f, want %f", tt.value, pv.Number.Number, tt.expected)
		}
	}
}

func TestInferPropertyValue(t *testing.T) {
	// Test checkbox inference
	pv := inferPropertyValue("true")
	if pv.Type != "checkbox" || !pv.Checkbox.Checkbox {
		t.Error("Should infer true as checkbox")
	}

	pv = inferPropertyValue("false")
	if pv.Type != "checkbox" || pv.Checkbox.Checkbox {
		t.Error("Should infer false as checkbox")
	}

	// Test number inference
	pv = inferPropertyValue("42")
	if pv.Type != "number" || pv.Number.Number != 42 {
		t.Error("Should infer 42 as number")
	}

	// Test text fallback
	pv = inferPropertyValue("some text")
	if pv.Type != "rich_text" {
		t.Errorf("Should infer text as rich_text, got %s", pv.Type)
	}
}

func TestSearchResultUnmarshal(t *testing.T) {
	// Test page result
	pageJSON := `{
		"object": "page",
		"id": "page-id",
		"properties": {},
		"url": "https://notion.so/page-id"
	}`

	var pageResult SearchResult
	if err := json.Unmarshal([]byte(pageJSON), &pageResult); err != nil {
		t.Fatalf("Failed to unmarshal page: %v", err)
	}

	if !pageResult.IsPage() {
		t.Error("Should be a page")
	}
	if pageResult.IsDatabase() {
		t.Error("Should not be a database")
	}

	// Test database result
	dbJSON := `{
		"object": "database",
		"id": "db-id",
		"title": [],
		"properties": {},
		"url": "https://notion.so/db-id"
	}`

	var dbResult SearchResult
	if err := json.Unmarshal([]byte(dbJSON), &dbResult); err != nil {
		t.Fatalf("Failed to unmarshal database: %v", err)
	}

	if !dbResult.IsDatabase() {
		t.Error("Should be a database")
	}
	if dbResult.IsPage() {
		t.Error("Should not be a page")
	}
}

func TestQueryJSON(t *testing.T) {
	builder := NewQuery().
		WhereSelect("Status", "Done").
		OrderBy("Created", "descending").
		Limit(10)

	jsonStr, err := builder.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	// Verify it contains expected fields
	if !strings.Contains(jsonStr, "Status") {
		t.Error("JSON should contain property name")
	}
	if !strings.Contains(jsonStr, "Done") {
		t.Error("JSON should contain filter value")
	}
}

func TestDateFilters(t *testing.T) {
	// Test various date filter builders
	builder := NewQuery().WhereDateEquals("Due Date", "2024-01-01")
	query := builder.Build()
	if query.Filter == nil || query.Filter.Date == nil || query.Filter.Date.Equals != "2024-01-01" {
		t.Error("Date equals filter not working")
	}

	builder2 := NewQuery().WhereDateAfter("Due Date", "2024-01-01")
	query2 := builder2.Build()
	if query2.Filter.Date.After != "2024-01-01" {
		t.Error("Date after filter not working")
	}

	builder3 := NewQuery().WhereDateBefore("Due Date", "2024-12-31")
	query3 := builder3.Build()
	if query3.Filter.Date.Before != "2024-12-31" {
		t.Error("Date before filter not working")
	}
}

func TestNumberFilters(t *testing.T) {
	builder := NewQuery().WhereNumberEquals("Priority", 5)
	query := builder.Build()
	if query.Filter.Number.Equals != 5 {
		t.Errorf("Number equals: got %f, want 5", query.Filter.Number.Equals)
	}

	builder2 := NewQuery().WhereNumberGreaterThan("Score", 100)
	query2 := builder2.Build()
	if query2.Filter.Number.GreaterThan != 100 {
		t.Errorf("Number greater than: got %f, want 100", query2.Filter.Number.GreaterThan)
	}

	builder3 := NewQuery().WhereNumberLessThan("Score", 50)
	query3 := builder3.Build()
	if query3.Filter.Number.LessThan != 50 {
		t.Errorf("Number less than: got %f, want 50", query3.Filter.Number.LessThan)
	}
}

func TestFormatBlock(t *testing.T) {
	tests := []struct {
		block    Block
		expected string
	}{
		{
			block: Block{
				Type: "paragraph",
				Paragraph: &Paragraph{
					RichText: []RichText{{PlainText: "Hello"}},
				},
			},
			expected: "Hello",
		},
		{
			block: Block{
				Type:     "heading_1",
				Heading1: &Heading{RichText: []RichText{{PlainText: "Title"}}},
			},
			expected: "# Title",
		},
		{
			block: Block{
				Type: "to_do",
				ToDo: &ToDo{RichText: []RichText{{PlainText: "Task"}}, Checked: true},
			},
			expected: "☑",
		},
		{
			block: Block{
				Type:    "divider",
				Divider: &Divider{},
			},
			expected: "---",
		},
	}

	for _, tt := range tests {
		result := formatBlock(&tt.block, 0)
		if !strings.Contains(result, tt.expected) {
			t.Errorf("formatBlock: expected to contain '%s', got '%s'", tt.expected, result)
		}
	}
}

func TestFormatPropertyValue(t *testing.T) {
	tests := []struct {
		name     string
		pv       PropertyValue
		expected string
	}{
		{
			name: "title",
			pv: PropertyValue{
				Type:  "title",
				Title: &TitleProperty{Title: []RichText{{PlainText: "Title"}}},
			},
			expected: "Title",
		},
		{
			name: "select",
			pv: PropertyValue{
				Type:   "select",
				Select: &SelectProperty{Select: &SelectOption{Name: "Done"}},
			},
			expected: "Done",
		},
		{
			name: "multi_select",
			pv: PropertyValue{
				Type:        "multi_select",
				MultiSelect: &MultiSelectProperty{MultiSelect: []SelectOption{{Name: "A"}, {Name: "B"}}},
			},
			expected: "A, B",
		},
		{
			name: "checkbox_true",
			pv: PropertyValue{
				Type:     "checkbox",
				Checkbox: &CheckboxProperty{Checkbox: true},
			},
			expected: "✓",
		},
		{
			name: "checkbox_false",
			pv: PropertyValue{
				Type:     "checkbox",
				Checkbox: &CheckboxProperty{Checkbox: false},
			},
			expected: "✗",
		},
		{
			name: "number",
			pv: PropertyValue{
				Type:   "number",
				Number: &NumberProperty{Number: 42},
			},
			expected: "42",
		},
		{
			name: "date",
			pv: PropertyValue{
				Type: "date",
				Date: &DateProperty{Date: &DateRange{Start: "2024-01-01"}},
			},
			expected: "2024-01-01",
		},
		{
			name:     "empty",
			pv:       PropertyValue{Type: "url"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPropertyValue(&tt.pv)
			if result != tt.expected {
				t.Errorf("formatPropertyValue: got '%s', want '%s'", result, tt.expected)
			}
		})
	}
}

func TestIsNotConnectedError(t *testing.T) {
	if IsNotConnectedError(nil) {
		t.Error("nil error should not be 'not connected'")
	}

	if !IsNotConnectedError(fmt.Errorf("please connect Notion in Settings")) {
		t.Error("Should detect 'please connect' message")
	}

	if !IsNotConnectedError(fmt.Errorf("token not found")) {
		t.Error("Should detect 'not found' + 'token' message")
	}

	if IsNotConnectedError(fmt.Errorf("some other error")) {
		t.Error("Should not detect unrelated error")
	}
}

func TestQueryFromMap(t *testing.T) {
	schema := map[string]PropertySchema{
		"Title":  {Name: "Title", Type: "title"},
		"Status": {Name: "Status", Type: "select"},
		"Due":    {Name: "Due", Type: "date"},
		"Done":   {Name: "Done", Type: "checkbox"},
		"Count":  {Name: "Count", Type: "number"},
	}

	filters := map[string]string{
		"Title":  "project",
		"Status": "active",
		"Due":    "2024-01-01",
		"Done":   "true",
		"Count":  "5",
	}

	query, err := QueryFromMap(filters, schema)
	if err != nil {
		t.Fatalf("QueryFromMap failed: %v", err)
	}

	if query.Filter == nil {
		t.Fatal("Filter should not be nil")
	}

	// Should create compound filter
	if query.Filter.And == nil {
		t.Error("Should create AND filter for multiple conditions")
	}
}
