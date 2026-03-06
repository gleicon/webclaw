//go:build js && wasm

package notion

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gleicon/webclaw/internal/tools"
)

// NotionToolSet holds all Notion tools for registration.
type NotionToolSet struct {
	client    *Client
	discovery *DatabaseDiscovery
}

// NewNotionToolSet creates a new tool set with the given client.
func NewNotionToolSet(client *Client) *NotionToolSet {
	return &NotionToolSet{
		client:    client,
		discovery: NewDatabaseDiscovery(client),
	}
}

// RegisterAll registers all Notion tools with the registry.
func (nts *NotionToolSet) RegisterAll(registry *tools.Registry) {
	registry.Register(nts.NewListDatabasesTool())
	registry.Register(nts.NewQueryTool())
	registry.Register(nts.NewReadTool())
	registry.Register(nts.NewUpdateTool())
	registry.Register(nts.NewSearchTool())
}

// NewListDatabasesTool creates the notion_list_databases tool.
func (nts *NotionToolSet) NewListDatabasesTool() *tools.Tool {
	return &tools.Tool{
		Name:        "notion_list_databases",
		Description: "List all available Notion databases you have access to. Returns database titles and IDs.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: nts.executeListDatabases,
	}
}

func (nts *NotionToolSet) executeListDatabases(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	databases, err := nts.client.ListDatabases()
	if err != nil {
		if IsNotConnectedError(err) {
			return &tools.ToolResult{
				Content:        "Please connect Notion in Settings first.",
				DisplayContent: "❌ Notion not connected",
				IsError:        true,
				ToolName:       "notion_list_databases",
				Status:         "error",
			}, nil
		}
		return nil, err
	}

	content := formatDatabaseList(databases)
	return &tools.ToolResult{
		Content:        content,
		DisplayContent: fmt.Sprintf("📚 Found %d databases", len(databases)),
		IsError:        false,
		ToolName:       "notion_list_databases",
		Status:         "done",
	}, nil
}

// NewQueryTool creates the notion_query tool.
func (nts *NotionToolSet) NewQueryTool() *tools.Tool {
	return &tools.Tool{
		Name:        "notion_query",
		Description: "Query a Notion database to find pages matching criteria. Can filter by properties and sort results.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"database_id": map[string]interface{}{
					"type":        "string",
					"description": "Database ID or name (e.g., 'Tasks', 'Notes')",
				},
				"filter_property": map[string]interface{}{
					"type":        "string",
					"description": "Property name to filter on (optional, e.g., 'Status', 'Priority')",
				},
				"filter_value": map[string]interface{}{
					"type":        "string",
					"description": "Value to filter for (optional, e.g., 'Done', 'High')",
				},
				"filter_condition": map[string]interface{}{
					"type":        "string",
					"description": "Filter condition: equals, contains, is_empty (default: equals for select, contains for text)",
				},
				"sort_by": map[string]interface{}{
					"type":        "string",
					"description": "Property to sort by (optional, e.g., 'Created', 'Name')",
				},
				"sort_direction": map[string]interface{}{
					"type":        "string",
					"description": "Sort direction: ascending or descending (default: descending)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 20, max: 100)",
				},
			},
			"required": []string{"database_id"},
		},
		Execute: nts.executeQuery,
	}
}

func (nts *NotionToolSet) executeQuery(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	databaseID, _ := params["database_id"].(string)
	if databaseID == "" {
		return &tools.ToolResult{
			Content:        "database_id is required",
			DisplayContent: "❌ Missing database_id",
			IsError:        true,
			ToolName:       "notion_query",
			Status:         "error",
		}, nil
	}

	// Resolve database name to ID if needed
	db, err := nts.discovery.FindByName(databaseID)
	if err != nil {
		// Try as direct ID
		db, err = nts.client.GetDatabase(databaseID)
	}
	if err != nil {
		if IsNotConnectedError(err) {
			return &tools.ToolResult{
				Content:        "Please connect Notion in Settings first.",
				DisplayContent: "❌ Notion not connected",
				IsError:        true,
				ToolName:       "notion_query",
				Status:         "error",
			}, nil
		}
		if IsNotFoundError(err) {
			return &tools.ToolResult{
				Content:        fmt.Sprintf("Database '%s' not found", databaseID),
				DisplayContent: fmt.Sprintf("❌ Database '%s' not found", databaseID),
				IsError:        true,
				ToolName:       "notion_query",
				Status:         "error",
			}, nil
		}
		return nil, err
	}

	// Build query
	builder := NewQuery()

	// Add filter if specified
	filterProp, hasFilterProp := params["filter_property"].(string)
	filterValue, hasFilterValue := params["filter_value"].(string)

	if hasFilterProp && filterProp != "" && hasFilterValue && filterValue != "" {
		// Get schema for type inference
		schema := db.Properties
		propSchema, exists := schema[filterProp]
		if !exists {
			// Try case-insensitive
			for name, ps := range schema {
				if strings.EqualFold(name, filterProp) {
					propSchema = ps
					filterProp = name
					exists = true
					break
				}
			}
		}

		if exists {
			condition, _ := params["filter_condition"].(string)
			builder = nts.addFilterByType(builder, filterProp, filterValue, propSchema.Type, condition)
		}
	}

	// Add sort if specified
	sortBy, hasSortBy := params["sort_by"].(string)
	if hasSortBy && sortBy != "" {
		sortDirection, _ := params["sort_direction"].(string)
		if sortDirection == "" {
			sortDirection = "descending"
		}
		builder.OrderBy(sortBy, sortDirection)
	}

	// Set limit
	limit := 20
	if limitVal, ok := params["limit"].(float64); ok {
		limit = int(limitVal)
	} else if limitStr, ok := params["limit"].(string); ok {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	builder.Limit(limit)

	// Execute query
	query := builder.Build()
	resp, err := nts.client.QueryDatabase(db.ID, query)
	if err != nil {
		return nil, err
	}

	// Format results
	content := formatQueryResults(resp.Results, db)
	display := fmt.Sprintf("📄 Found %d pages in '%s'", len(resp.Results), getDatabaseTitle(db))
	if resp.HasMore {
		display += " (more available)"
	}

	return &tools.ToolResult{
		Content:        content,
		DisplayContent: display,
		IsError:        false,
		ToolName:       "notion_query",
		Status:         "done",
	}, nil
}

func (nts *NotionToolSet) addFilterByType(builder *QueryBuilder, propName, value, propType, condition string) *QueryBuilder {
	switch propType {
	case "title":
		if condition == "" || condition == "equals" {
			return builder.WhereTitle(propName, "equals", value)
		}
		return builder.WhereTitle(propName, condition, value)
	case "rich_text":
		if condition == "" {
			condition = "contains"
		}
		return builder.WhereRichText(propName, condition, value)
	case "select":
		if condition == "" || condition == "equals" {
			return builder.WhereSelect(propName, value)
		}
		if condition == "does_not_equal" {
			return builder.WhereSelectNotEquals(propName, value)
		}
		return builder.WhereSelect(propName, value)
	case "multi_select":
		return builder.WhereMultiSelectContains(propName, value)
	case "status":
		return builder.WhereStatus(propName, value)
	case "checkbox":
		checked := strings.ToLower(value) == "true" ||
			strings.ToLower(value) == "yes" ||
			strings.ToLower(value) == "checked" ||
			strings.ToLower(value) == "done"
		return builder.WhereCheckbox(propName, checked)
	case "date":
		if condition == "" || condition == "equals" {
			return builder.WhereDateEquals(propName, value)
		}
		if condition == "after" {
			return builder.WhereDateAfter(propName, value)
		}
		if condition == "before" {
			return builder.WhereDateBefore(propName, value)
		}
		return builder.WhereDateEquals(propName, value)
	case "number":
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			return builder.WhereNumberEquals(propName, num)
		}
		return builder.WhereRichText(propName, "contains", value)
	default:
		// Fall back to rich_text
		return builder.WhereRichText(propName, "contains", value)
	}
}

// NewReadTool creates the notion_read tool.
func (nts *NotionToolSet) NewReadTool() *tools.Tool {
	return &tools.Tool{
		Name:        "notion_read",
		Description: "Read the content of a Notion page. Returns the page properties and optionally the full block content.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"page_id": map[string]interface{}{
					"type":        "string",
					"description": "Page ID or full Notion URL",
				},
				"include_content": map[string]interface{}{
					"type":        "boolean",
					"description": "Include page content blocks (default: true)",
				},
			},
			"required": []string{"page_id"},
		},
		Execute: nts.executeRead,
	}
}

func (nts *NotionToolSet) executeRead(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	pageID, _ := params["page_id"].(string)
	if pageID == "" {
		return &tools.ToolResult{
			Content:        "page_id is required",
			DisplayContent: "❌ Missing page_id",
			IsError:        true,
			ToolName:       "notion_read",
			Status:         "error",
		}, nil
	}

	includeContent := true
	if val, ok := params["include_content"].(bool); ok {
		includeContent = val
	}

	// Get page metadata
	page, err := nts.client.GetPage(pageID)
	if err != nil {
		if IsNotConnectedError(err) {
			return &tools.ToolResult{
				Content:        "Please connect Notion in Settings first.",
				DisplayContent: "❌ Notion not connected",
				IsError:        true,
				ToolName:       "notion_read",
				Status:         "error",
			}, nil
		}
		if IsNotFoundError(err) {
			return &tools.ToolResult{
				Content:        fmt.Sprintf("Page '%s' not found", pageID),
				DisplayContent: fmt.Sprintf("❌ Page '%s' not found", pageID),
				IsError:        true,
				ToolName:       "notion_read",
				Status:         "error",
			}, nil
		}
		return nil, err
	}

	// Get page content if requested
	var blocks []Block
	if includeContent {
		blocks, err = nts.client.GetPageContentAll(pageID)
		if err != nil {
			// Continue without content
			blocks = nil
		}
	}

	content := formatPage(page, includeContent, blocks)
	title := getPageTitle(page)
	display := fmt.Sprintf("📄 Reading: %s", title)

	return &tools.ToolResult{
		Content:        content,
		DisplayContent: display,
		IsError:        false,
		ToolName:       "notion_read",
		Status:         "done",
	}, nil
}

// NewUpdateTool creates the notion_update tool.
func (nts *NotionToolSet) NewUpdateTool() *tools.Tool {
	return &tools.Tool{
		Name:        "notion_update",
		Description: "Update properties of a Notion page (not block content). Can update status, dates, text fields, etc.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"page_id": map[string]interface{}{
					"type":        "string",
					"description": "Page ID or full Notion URL",
				},
				"properties": map[string]interface{}{
					"type":        "object",
					"description": "Properties to update as key-value pairs (e.g., {'Status': 'Done', 'Notes': 'Completed review'})",
				},
			},
			"required": []string{"page_id", "properties"},
		},
		Execute: nts.executeUpdate,
	}
}

func (nts *NotionToolSet) executeUpdate(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	pageID, _ := params["page_id"].(string)
	if pageID == "" {
		return &tools.ToolResult{
			Content:        "page_id is required",
			DisplayContent: "❌ Missing page_id",
			IsError:        true,
			ToolName:       "notion_update",
			Status:         "error",
		}, nil
	}

	propsParam, ok := params["properties"].(map[string]interface{})
	if !ok || len(propsParam) == 0 {
		return &tools.ToolResult{
			Content:        "properties object is required",
			DisplayContent: "❌ Missing properties to update",
			IsError:        true,
			ToolName:       "notion_update",
			Status:         "error",
		}, nil
	}

	// Get page to determine its database (for schema)
	page, err := nts.client.GetPage(pageID)
	if err != nil {
		if IsNotConnectedError(err) {
			return &tools.ToolResult{
				Content:        "Please connect Notion in Settings first.",
				DisplayContent: "❌ Notion not connected",
				IsError:        true,
				ToolName:       "notion_update",
				Status:         "error",
			}, nil
		}
		if IsNotFoundError(err) {
			return &tools.ToolResult{
				Content:        fmt.Sprintf("Page '%s' not found", pageID),
				DisplayContent: fmt.Sprintf("❌ Page '%s' not found", pageID),
				IsError:        true,
				ToolName:       "notion_update",
				Status:         "error",
			}, nil
		}
		return nil, err
	}

	// Get database schema for type inference
	var db *Database
	if page.Parent != nil && page.Parent.Type == "database_id" && page.Parent.DatabaseID != "" {
		db, _ = nts.client.GetDatabase(page.Parent.DatabaseID)
	}

	// Convert properties to PropertyValues
	propertyValues := make(map[string]PropertyValue)
	for propName, value := range propsParam {
		strValue := fmt.Sprintf("%v", value)

		if db != nil {
			if propSchema, exists := db.Properties[propName]; exists {
				pv, err := nts.discovery.ConvertStringToPropertyValue(propSchema.Type, strValue)
				if err == nil {
					propertyValues[propName] = pv
					continue
				}
			}
		}

		// Fallback: try to infer type
		pv := inferPropertyValue(strValue)
		propertyValues[propName] = pv
	}

	// Update page
	updatedPage, err := nts.client.UpdatePage(pageID, propertyValues)
	if err != nil {
		if IsValidationError(err) {
			return &tools.ToolResult{
				Content:        fmt.Sprintf("Validation error: %v", err),
				DisplayContent: "❌ Invalid property value",
				IsError:        true,
				ToolName:       "notion_update",
				Status:         "error",
			}, nil
		}
		return nil, err
	}

	content := fmt.Sprintf("Updated page: %s\nURL: %s\n\nUpdated properties:\n", getPageTitle(updatedPage), updatedPage.URL)
	for name := range propertyValues {
		content += fmt.Sprintf("  - %s\n", name)
	}

	return &tools.ToolResult{
		Content:        content,
		DisplayContent: fmt.Sprintf("✅ Updated: %s", getPageTitle(updatedPage)),
		IsError:        false,
		ToolName:       "notion_update",
		Status:         "done",
	}, nil
}

// NewSearchTool creates the notion_search tool.
func (nts *NotionToolSet) NewSearchTool() *tools.Tool {
	return &tools.Tool{
		Name:        "notion_search",
		Description: "Search for pages and databases in Notion by title or content.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query text",
				},
				"filter_type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"page", "database", ""},
					"description": "Filter by type: 'page', 'database', or leave empty for both (optional)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 10, max: 100)",
				},
			},
			"required": []string{"query"},
		},
		Execute: nts.executeSearch,
	}
}

func (nts *NotionToolSet) executeSearch(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return &tools.ToolResult{
			Content:        "query is required",
			DisplayContent: "❌ Missing search query",
			IsError:        true,
			ToolName:       "notion_search",
			Status:         "error",
		}, nil
	}

	filterType, _ := params["filter_type"].(string)

	limit := 10
	if limitVal, ok := params["limit"].(float64); ok {
		limit = int(limitVal)
	} else if limitStr, ok := params["limit"].(string); ok {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	resp, err := nts.client.Search(query, filterType, limit)
	if err != nil {
		if IsNotConnectedError(err) {
			return &tools.ToolResult{
				Content:        "Please connect Notion in Settings first.",
				DisplayContent: "❌ Notion not connected",
				IsError:        true,
				ToolName:       "notion_search",
				Status:         "error",
			}, nil
		}
		return nil, err
	}

	content := formatSearchResults(resp.Results)
	display := fmt.Sprintf("🔍 Found %d results for '%s'", len(resp.Results), query)

	return &tools.ToolResult{
		Content:        content,
		DisplayContent: display,
		IsError:        false,
		ToolName:       "notion_search",
		Status:         "done",
	}, nil
}

// Helper functions

func getPageTitle(page *Page) string {
	// Look for title property
	for _, prop := range page.Properties {
		if prop.Type == "title" && prop.Title != nil {
			return richTextToPlainText(prop.Title.Title)
		}
	}
	return "Untitled"
}

func getDatabaseTitle(db *Database) string {
	return richTextToPlainText(db.Title)
}

func richTextToPlainText(richTexts []RichText) string {
	var parts []string
	for _, rt := range richTexts {
		parts = append(parts, rt.PlainText)
	}
	return strings.Join(parts, "")
}

func formatDatabaseList(databases []*Database) string {
	if len(databases) == 0 {
		return "No databases found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d database(s):\n\n", len(databases)))

	for i, db := range databases {
		title := getDatabaseTitle(db)
		if title == "" {
			title = "Untitled"
		}

		// Count properties
		propCount := len(db.Properties)

		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, title))
		sb.WriteString(fmt.Sprintf("   ID: %s\n", db.ID))
		sb.WriteString(fmt.Sprintf("   Properties: %d\n", propCount))

		// List some key properties
		var propNames []string
		for name := range db.Properties {
			propNames = append(propNames, name)
			if len(propNames) >= 5 {
				propNames = append(propNames, "...")
				break
			}
		}
		if len(propNames) > 0 {
			sb.WriteString(fmt.Sprintf("   Fields: %s\n", strings.Join(propNames, ", ")))
		}

		sb.WriteString(fmt.Sprintf("   URL: %s\n", db.URL))
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatQueryResults(pages []Page, db *Database) string {
	if len(pages) == 0 {
		return fmt.Sprintf("No pages found in '%s'.", getDatabaseTitle(db))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d page(s) in '%s':\n\n", len(pages), getDatabaseTitle(db)))

	for i, page := range pages {
		title := getPageTitle(&page)
		if title == "" {
			title = "Untitled"
		}

		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, title))
		sb.WriteString(fmt.Sprintf("   ID: %s\n", page.ID))

		// Show key properties
		for propName, propValue := range page.Properties {
			if propName == "Name" || propName == "Title" {
				continue // Already shown as title
			}
			value := formatPropertyValue(&propValue)
			if value != "" {
				sb.WriteString(fmt.Sprintf("   %s: %s\n", propName, value))
			}
		}

		sb.WriteString(fmt.Sprintf("   URL: %s\n", page.URL))
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatPropertyValue(pv *PropertyValue) string {
	switch pv.Type {
	case "title":
		if pv.Title != nil {
			return richTextToPlainText(pv.Title.Title)
		}
	case "rich_text":
		if pv.RichText != nil {
			return richTextToPlainText(pv.RichText.RichText)
		}
	case "select":
		if pv.Select != nil && pv.Select.Select != nil {
			return pv.Select.Select.Name
		}
	case "multi_select":
		if pv.MultiSelect != nil {
			var names []string
			for _, opt := range pv.MultiSelect.MultiSelect {
				names = append(names, opt.Name)
			}
			return strings.Join(names, ", ")
		}
	case "status":
		if pv.Status != nil && pv.Status.Status != nil {
			return pv.Status.Status.Name
		}
	case "checkbox":
		if pv.Checkbox != nil {
			if pv.Checkbox.Checkbox {
				return "✓"
			}
			return "✗"
		}
	case "date":
		if pv.Date != nil && pv.Date.Date != nil {
			return pv.Date.Date.Start
		}
	case "number":
		if pv.Number != nil {
			return fmt.Sprintf("%v", pv.Number.Number)
		}
	case "url":
		if pv.URLProp != nil {
			return pv.URLProp.URL
		}
	case "email":
		if pv.Email != nil {
			return pv.Email.Email
		}
	case "phone_number":
		if pv.Phone != nil {
			return pv.Phone.PhoneNumber
		}
	}
	return ""
}

func formatPage(page *Page, includeContent bool, blocks []Block) string {
	var sb strings.Builder

	title := getPageTitle(page)
	if title == "" {
		title = "Untitled"
	}

	// Icon
	if page.Icon != nil && page.Icon.Emoji != "" {
		sb.WriteString(fmt.Sprintf("%s ", page.Icon.Emoji))
	}
	sb.WriteString(fmt.Sprintf("%s\n", title))
	sb.WriteString(strings.Repeat("=", len(title)+3) + "\n\n")

	// Properties
	if len(page.Properties) > 0 {
		sb.WriteString("Properties:\n")
		for name, propValue := range page.Properties {
			value := formatPropertyValue(&propValue)
			if value == "" {
				value = "(empty)"
			}
			sb.WriteString(fmt.Sprintf("  %s: %s\n", name, value))
		}
		sb.WriteString("\n")
	}

	// Content
	if includeContent && len(blocks) > 0 {
		sb.WriteString("Content:\n")
		for _, block := range blocks {
			formatted := formatBlock(&block, 0)
			if formatted != "" {
				sb.WriteString(formatted)
			}
		}
	}

	// Footer
	sb.WriteString(fmt.Sprintf("\n---\n"))
	sb.WriteString(fmt.Sprintf("URL: %s\n", page.URL))
	sb.WriteString(fmt.Sprintf("Last edited: %s\n", page.LastEditedTime))

	return sb.String()
}

func formatBlock(block *Block, depth int) string {
	indent := strings.Repeat("  ", depth)

	switch block.Type {
	case "paragraph":
		if block.Paragraph != nil {
			text := richTextToPlainText(block.Paragraph.RichText)
			if text == "" {
				return indent + "\n"
			}
			return indent + text + "\n"
		}
	case "heading_1":
		if block.Heading1 != nil {
			return "\n" + indent + "# " + richTextToPlainText(block.Heading1.RichText) + "\n"
		}
	case "heading_2":
		if block.Heading2 != nil {
			return "\n" + indent + "## " + richTextToPlainText(block.Heading2.RichText) + "\n"
		}
	case "heading_3":
		if block.Heading3 != nil {
			return "\n" + indent + "### " + richTextToPlainText(block.Heading3.RichText) + "\n"
		}
	case "bulleted_list_item":
		if block.BulletedListItem != nil {
			return indent + "• " + richTextToPlainText(block.BulletedListItem.RichText) + "\n"
		}
	case "numbered_list_item":
		if block.NumberedListItem != nil {
			return indent + "1. " + richTextToPlainText(block.NumberedListItem.RichText) + "\n"
		}
	case "to_do":
		if block.ToDo != nil {
			checkbox := "☐"
			if block.ToDo.Checked {
				checkbox = "☑"
			}
			return indent + checkbox + " " + richTextToPlainText(block.ToDo.RichText) + "\n"
		}
	case "code":
		if block.Code != nil {
			lang := block.Code.Language
			if lang == "" {
				lang = "text"
			}
			return "\n" + indent + "```" + lang + "\n" + richTextToPlainText(block.Code.RichText) + "\n" + indent + "```\n"
		}
	case "quote":
		if block.Quote != nil {
			return indent + "> " + richTextToPlainText(block.Quote.RichText) + "\n"
		}
	case "callout":
		if block.Callout != nil {
			icon := "ℹ"
			if block.Callout.Icon != nil && block.Callout.Icon.Emoji != "" {
				icon = block.Callout.Icon.Emoji
			}
			return indent + icon + " " + richTextToPlainText(block.Callout.RichText) + "\n"
		}
	case "divider":
		return indent + "---\n"
	case "image":
		if block.Image != nil {
			url := ""
			if block.Image.External != nil {
				url = block.Image.External.URL
			} else if block.Image.File != nil {
				url = block.Image.File.URL
			}
			caption := richTextToPlainText(block.Image.Caption)
			if caption != "" {
				return indent + fmt.Sprintf("[Image: %s](%s)\n", caption, url)
			}
			return indent + fmt.Sprintf("[Image](%s)\n", url)
		}
	case "bookmark":
		if block.Bookmark != nil {
			return indent + fmt.Sprintf("[Bookmark: %s](%s)\n", block.Bookmark.URL, block.Bookmark.URL)
		}
	case "toggle":
		if block.Toggle != nil {
			return indent + "▶ " + richTextToPlainText(block.Toggle.RichText) + "\n"
		}
	}

	return ""
}

func formatSearchResults(results []SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d result(s):\n\n", len(results)))

	for i, result := range results {
		if result.IsPage() {
			page := result.Page
			title := getPageTitle(page)
			if title == "" {
				title = "Untitled Page"
			}
			sb.WriteString(fmt.Sprintf("%d. 📄 %s\n", i+1, title))
			sb.WriteString(fmt.Sprintf("   Page ID: %s\n", page.ID))
			sb.WriteString(fmt.Sprintf("   URL: %s\n", page.URL))
		} else if result.IsDatabase() {
			db := result.Database
			title := getDatabaseTitle(db)
			if title == "" {
				title = "Untitled Database"
			}
			sb.WriteString(fmt.Sprintf("%d. 📚 %s\n", i+1, title))
			sb.WriteString(fmt.Sprintf("   Database ID: %s\n", db.ID))
			sb.WriteString(fmt.Sprintf("   URL: %s\n", db.URL))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func inferPropertyValue(value string) PropertyValue {
	// Try checkbox first
	lower := strings.ToLower(value)
	if lower == "true" || lower == "yes" || lower == "checked" || lower == "done" {
		return PropertyValue{
			Type:     "checkbox",
			Checkbox: &CheckboxProperty{Checkbox: true},
		}
	}
	if lower == "false" || lower == "no" || lower == "unchecked" {
		return PropertyValue{
			Type:     "checkbox",
			Checkbox: &CheckboxProperty{Checkbox: false},
		}
	}

	// Try number
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		return PropertyValue{
			Type:   "number",
			Number: &NumberProperty{Number: num},
		}
	}

	// Default to rich text
	return PropertyValue{
		Type: "rich_text",
		RichText: &RichTextProperty{
			RichText: []RichText{
				{Type: "text", Text: &Text{Content: value}, PlainText: value},
			},
		},
	}
}
