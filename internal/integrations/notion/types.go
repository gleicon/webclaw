//go:build js && wasm

// Package notion provides Notion API integration for WebClaw.
// It implements the OpenClaw tool specification for knowledge management operations.
package notion

import (
	"encoding/json"
	"fmt"
)

// Notion API version constant
const (
	APIVersion = "2022-06-28"
	BaseURL    = "https://api.notion.com/v1"
)

// Page represents a Notion page with metadata and properties.
type Page struct {
	ID             string                   `json:"id"`
	CreatedTime    string                   `json:"created_time"`
	LastEditedTime string                   `json:"last_edited_time"`
	CreatedBy      *User                    `json:"created_by,omitempty"`
	LastEditedBy   *User                    `json:"last_edited_by,omitempty"`
	Cover          *File                    `json:"cover,omitempty"`
	Icon           *Icon                    `json:"icon,omitempty"`
	Parent         *Parent                  `json:"parent"`
	Archived       bool                     `json:"archived"`
	Properties     map[string]PropertyValue `json:"properties"`
	URL            string                   `json:"url"`
}

// Database represents a Notion database with schema information.
type Database struct {
	ID             string                    `json:"id"`
	CreatedTime    string                    `json:"created_time"`
	LastEditedTime string                    `json:"last_edited_time"`
	Title          []RichText                `json:"title"`
	Description    []RichText                `json:"description,omitempty"`
	Properties     map[string]PropertySchema `json:"properties"`
	Parent         *Parent                   `json:"parent"`
	URL            string                    `json:"url"`
	IsInline       bool                      `json:"is_inline,omitempty"`
}

// PropertySchema defines the schema for a database property.
type PropertySchema struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"` // title, rich_text, select, multi_select, date, etc.
	Select      *SelectOptions  `json:"select,omitempty"`
	MultiSelect *SelectOptions  `json:"multi_select,omitempty"`
	Number      *NumberConfig   `json:"number,omitempty"`
	Formula     *FormulaConfig  `json:"formula,omitempty"`
	Relation    *RelationConfig `json:"relation,omitempty"`
	Rollup      *RollupConfig   `json:"rollup,omitempty"`
}

// PropertyValue represents the value of a property in a page.
type PropertyValue struct {
	ID             string                  `json:"id"`
	Type           string                  `json:"type"`
	Title          *TitleProperty          `json:"title,omitempty"`
	RichText       *RichTextProperty       `json:"rich_text,omitempty"`
	Select         *SelectProperty         `json:"select,omitempty"`
	MultiSelect    *MultiSelectProperty    `json:"multi_select,omitempty"`
	Date           *DateProperty           `json:"date,omitempty"`
	Checkbox       *CheckboxProperty       `json:"checkbox,omitempty"`
	Number         *NumberProperty         `json:"number,omitempty"`
	URLProp        *URLProperty            `json:"url,omitempty"`
	Email          *EmailProperty          `json:"email,omitempty"`
	Phone          *PhoneProperty          `json:"phone_number,omitempty"`
	Status         *StatusProperty         `json:"status,omitempty"`
	Relation       *RelationProperty       `json:"relation,omitempty"`
	Formula        *FormulaProperty        `json:"formula,omitempty"`
	Rollup         *RollupProperty         `json:"rollup,omitempty"`
	CreatedTime    *CreatedTimeProperty    `json:"created_time,omitempty"`
	LastEditedTime *LastEditedTimeProperty `json:"last_edited_time,omitempty"`
}

// TitleProperty represents a title property value.
type TitleProperty struct {
	Title []RichText `json:"title"`
}

// RichTextProperty represents a rich text property value.
type RichTextProperty struct {
	RichText []RichText `json:"rich_text"`
}

// SelectProperty represents a single-select property value.
type SelectProperty struct {
	Select *SelectOption `json:"select"`
}

// MultiSelectProperty represents a multi-select property value.
type MultiSelectProperty struct {
	MultiSelect []SelectOption `json:"multi_select"`
}

// DateProperty represents a date property value.
type DateProperty struct {
	Date *DateRange `json:"date"`
}

// CheckboxProperty represents a checkbox property value.
type CheckboxProperty struct {
	Checkbox bool `json:"checkbox"`
}

// NumberProperty represents a number property value.
type NumberProperty struct {
	Number float64 `json:"number"`
}

// URLProperty represents a URL property value.
type URLProperty struct {
	URL string `json:"url"`
}

// EmailProperty represents an email property value.
type EmailProperty struct {
	Email string `json:"email"`
}

// PhoneProperty represents a phone number property value.
type PhoneProperty struct {
	PhoneNumber string `json:"phone_number"`
}

// StatusProperty represents a status property value.
type StatusProperty struct {
	Status *SelectOption `json:"status"`
}

// RelationProperty represents a relation property value.
type RelationProperty struct {
	Relation []RelationEntry `json:"relation"`
}

// RelationEntry represents a single relation entry.
type RelationEntry struct {
	ID string `json:"id"`
}

// FormulaProperty represents a formula property value.
type FormulaProperty struct {
	Formula struct {
		Type    string     `json:"type"`
		String  string     `json:"string,omitempty"`
		Number  float64    `json:"number,omitempty"`
		Boolean bool       `json:"boolean,omitempty"`
		Date    *DateRange `json:"date,omitempty"`
	} `json:"formula"`
}

// RollupProperty represents a rollup property value.
type RollupProperty struct {
	Rollup struct {
		Type     string          `json:"type"`
		Function string          `json:"function"`
		Number   float64         `json:"number,omitempty"`
		Date     *DateRange      `json:"date,omitempty"`
		Array    []PropertyValue `json:"array,omitempty"`
	} `json:"rollup"`
}

// CreatedTimeProperty represents the created time property.
type CreatedTimeProperty struct {
	CreatedTime string `json:"created_time"`
}

// LastEditedTimeProperty represents the last edited time property.
type LastEditedTimeProperty struct {
	LastEditedTime string `json:"last_edited_time"`
}

// RichText represents a rich text element.
type RichText struct {
	Type        string       `json:"type"` // text, mention, equation
	Text        *Text        `json:"text,omitempty"`
	Mention     *Mention     `json:"mention,omitempty"`
	Equation    *Equation    `json:"equation,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	PlainText   string       `json:"plain_text"`
	Href        string       `json:"href,omitempty"`
}

// Text represents a plain text element.
type Text struct {
	Content string `json:"content"`
	Link    *Link  `json:"link,omitempty"`
}

// Link represents a hyperlink.
type Link struct {
	URL string `json:"url"`
}

// Mention represents a mention of a page, database, or user.
type Mention struct {
	Type string `json:"type"` // page, database, user, date, link_preview
	Page *struct {
		ID string `json:"id"`
	} `json:"page,omitempty"`
	Database *struct {
		ID string `json:"id"`
	} `json:"database,omitempty"`
	User *User      `json:"user,omitempty"`
	Date *DateRange `json:"date,omitempty"`
}

// Equation represents a mathematical equation.
type Equation struct {
	Expression string `json:"expression"`
}

// Annotations represents text formatting.
type Annotations struct {
	Bold          bool   `json:"bold"`
	Italic        bool   `json:"italic"`
	Strikethrough bool   `json:"strikethrough"`
	Underline     bool   `json:"underline"`
	Code          bool   `json:"code"`
	Color         string `json:"color,omitempty"`
}

// Block represents a content block within a page.
type Block struct {
	ID               string         `json:"id"`
	Type             string         `json:"type"`
	CreatedTime      string         `json:"created_time"`
	LastEditedTime   string         `json:"last_edited_time"`
	HasChildren      bool           `json:"has_children"`
	Paragraph        *Paragraph     `json:"paragraph,omitempty"`
	Heading1         *Heading       `json:"heading_1,omitempty"`
	Heading2         *Heading       `json:"heading_2,omitempty"`
	Heading3         *Heading       `json:"heading_3,omitempty"`
	Callout          *Callout       `json:"callout,omitempty"`
	Quote            *Quote         `json:"quote,omitempty"`
	BulletedListItem *ListItem      `json:"bulleted_list_item,omitempty"`
	NumberedListItem *ListItem      `json:"numbered_list_item,omitempty"`
	ToDo             *ToDo          `json:"to_do,omitempty"`
	Toggle           *Toggle        `json:"toggle,omitempty"`
	Code             *CodeBlock     `json:"code,omitempty"`
	Divider          *Divider       `json:"divider,omitempty"`
	Image            *Image         `json:"image,omitempty"`
	Bookmark         *Bookmark      `json:"bookmark,omitempty"`
	Embed            *Embed         `json:"embed,omitempty"`
	Table            *Table         `json:"table,omitempty"`
	TableRow         *TableRow      `json:"table_row,omitempty"`
	ColumnList       *ColumnList    `json:"column_list,omitempty"`
	Column           *Column        `json:"column,omitempty"`
	LinkPreview      *LinkPreview   `json:"link_preview,omitempty"`
	EquationBlock    *EquationBlock `json:"equation,omitempty"`
	LinkToPage       *LinkToPage    `json:"link_to_page,omitempty"`
	SyncedBlock      *SyncedBlock   `json:"synced_block,omitempty"`
	Template         *Template      `json:"template,omitempty"`
	Unsupported      *Unsupported   `json:"unsupported,omitempty"`
}

// Paragraph represents a paragraph block.
type Paragraph struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// Heading represents a heading block (h1, h2, h3).
type Heading struct {
	RichText     []RichText `json:"rich_text"`
	Color        string     `json:"color,omitempty"`
	IsToggleable bool       `json:"is_toggleable,omitempty"`
}

// Callout represents a callout block.
type Callout struct {
	RichText []RichText `json:"rich_text"`
	Icon     *Icon      `json:"icon,omitempty"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// Quote represents a quote block.
type Quote struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// ListItem represents a list item (bulleted or numbered).
type ListItem struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// ToDo represents a todo item block.
type ToDo struct {
	RichText []RichText `json:"rich_text"`
	Checked  bool       `json:"checked"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// Toggle represents a toggle block.
type Toggle struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// CodeBlock represents a code block.
type CodeBlock struct {
	RichText []RichText `json:"rich_text"`
	Language string     `json:"language"`
}

// Divider represents a divider block.
type Divider struct{}

// Image represents an image block.
type Image struct {
	Type     string `json:"type"` // external, file
	External *struct {
		URL string `json:"url"`
	} `json:"external,omitempty"`
	File *struct {
		URL        string `json:"url"`
		ExpiryTime string `json:"expiry_time,omitempty"`
	} `json:"file,omitempty"`
	Caption []RichText `json:"caption,omitempty"`
}

// Bookmark represents a bookmark block.
type Bookmark struct {
	URL     string     `json:"url"`
	Caption []RichText `json:"caption,omitempty"`
}

// Embed represents an embed block.
type Embed struct {
	URL     string     `json:"url"`
	Caption []RichText `json:"caption,omitempty"`
}

// Table represents a table block.
type Table struct {
	TableWidth      int  `json:"table_width"`
	HasColumnHeader bool `json:"has_column_header"`
	HasRowHeader    bool `json:"has_row_header"`
}

// TableRow represents a table row block.
type TableRow struct {
	Cells [][]RichText `json:"cells"`
}

// ColumnList represents a column list block.
type ColumnList struct {
	Children []Block `json:"children,omitempty"`
}

// Column represents a column block.
type Column struct {
	Children []Block `json:"children,omitempty"`
}

// LinkPreview represents a link preview block.
type LinkPreview struct {
	URL string `json:"url"`
}

// EquationBlock represents an equation block.
type EquationBlock struct {
	Expression string `json:"expression"`
}

// LinkToPage represents a link to page block.
type LinkToPage struct {
	Type       string `json:"type"` // page_id, database_id, comment_id
	PageID     string `json:"page_id,omitempty"`
	DatabaseID string `json:"database_id,omitempty"`
}

// SyncedBlock represents a synced block.
type SyncedBlock struct {
	SyncedFrom *struct {
		BlockID string `json:"block_id,omitempty"`
	} `json:"synced_from,omitempty"`
	Children []Block `json:"children,omitempty"`
}

// Template represents a template block.
type Template struct {
	RichText []RichText `json:"rich_text"`
	Children []Block    `json:"children,omitempty"`
}

// Unsupported represents an unsupported block type.
type Unsupported struct{}

// SelectOptions holds options for select/multi-select properties.
type SelectOptions struct {
	Options []SelectOption `json:"options"`
}

// SelectOption represents a single select option.
type SelectOption struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// NumberConfig holds number format configuration.
type NumberConfig struct {
	Format string `json:"format"`
}

// FormulaConfig holds formula expression.
type FormulaConfig struct {
	Expression string `json:"expression"`
}

// RelationConfig holds relation configuration.
type RelationConfig struct {
	DatabaseID     string    `json:"database_id"`
	Type           string    `json:"type,omitempty"`
	SingleProperty *struct{} `json:"single_property,omitempty"`
	DualProperty   *struct {
		SyncedPropertyName string `json:"synced_property_name"`
	} `json:"dual_property,omitempty"`
}

// RollupConfig holds rollup configuration.
type RollupConfig struct {
	RelationPropertyName string `json:"relation_property_name"`
	RollupPropertyName   string `json:"rollup_property_name"`
	Function             string `json:"function"`
}

// DateRange represents a date or date range.
type DateRange struct {
	Start    string `json:"start"` // ISO 8601
	End      string `json:"end,omitempty"`
	TimeZone string `json:"time_zone,omitempty"`
}

// User represents a Notion user.
type User struct {
	ID        string `json:"id"`
	Type      string `json:"type,omitempty"` // person, bot
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Person    *struct {
		Email string `json:"email"`
	} `json:"person,omitempty"`
	Bot *struct {
		Owner *User `json:"owner,omitempty"`
	} `json:"bot,omitempty"`
}

// Parent represents the parent of a page or database.
type Parent struct {
	Type       string `json:"type"` // database_id, page_id, workspace, block_id
	DatabaseID string `json:"database_id,omitempty"`
	PageID     string `json:"page_id,omitempty"`
	BlockID    string `json:"block_id,omitempty"`
	Workspace  bool   `json:"workspace,omitempty"`
}

// Icon represents an icon (emoji or file).
type Icon struct {
	Type     string `json:"type"` // emoji, external, file
	Emoji    string `json:"emoji,omitempty"`
	External *struct {
		URL string `json:"url"`
	} `json:"external,omitempty"`
	File *struct {
		URL        string `json:"url"`
		ExpiryTime string `json:"expiry_time,omitempty"`
	} `json:"file,omitempty"`
}

// File represents a file (external or uploaded).
type File struct {
	Type     string `json:"type"` // external, file
	External *struct {
		URL string `json:"url"`
	} `json:"external,omitempty"`
	File *struct {
		URL        string `json:"url"`
		ExpiryTime string `json:"expiry_time,omitempty"`
	} `json:"file,omitempty"`
}

// QueryResponse represents the response from a database query.
type QueryResponse struct {
	Object     string `json:"object"` // "list"
	Results    []Page `json:"results"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

// SearchResponse represents the response from a search query.
type SearchResponse struct {
	Object     string         `json:"object"`
	Results    []SearchResult `json:"results"`
	NextCursor string         `json:"next_cursor"`
	HasMore    bool           `json:"has_more"`
}

// SearchResult is a wrapper for search results that can be Page or Database.
type SearchResult struct {
	Object   string    `json:"object"`
	Page     *Page     `json:"-"`
	Database *Database `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling for SearchResult.
func (sr *SearchResult) UnmarshalJSON(data []byte) error {
	// First, unmarshal to get the object type
	var typeCheck struct {
		Object string `json:"object"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return err
	}
	sr.Object = typeCheck.Object

	switch typeCheck.Object {
	case "page":
		var page Page
		if err := json.Unmarshal(data, &page); err != nil {
			return err
		}
		sr.Page = &page
	case "database":
		var db Database
		if err := json.Unmarshal(data, &db); err != nil {
			return err
		}
		sr.Database = &db
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for SearchResult.
func (sr SearchResult) MarshalJSON() ([]byte, error) {
	switch sr.Object {
	case "page":
		if sr.Page != nil {
			return json.Marshal(sr.Page)
		}
	case "database":
		if sr.Database != nil {
			return json.Marshal(sr.Database)
		}
	}
	return json.Marshal(map[string]interface{}{"object": sr.Object})
}

// IsPage returns true if this result is a page.
func (sr *SearchResult) IsPage() bool {
	return sr.Object == "page" && sr.Page != nil
}

// IsDatabase returns true if this result is a database.
func (sr *SearchResult) IsDatabase() bool {
	return sr.Object == "database" && sr.Database != nil
}

// BlockChildrenResponse represents the response when fetching block children.
type BlockChildrenResponse struct {
	Object     string  `json:"object"`
	Results    []Block `json:"results"`
	NextCursor string  `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// NotionError represents an error response from the Notion API.
type NotionError struct {
	Object  string `json:"object"`
	Status  int    `json:"status"`
	Code    string `json:"code"` // validation_error, object_not_found, etc.
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *NotionError) Error() string {
	return fmt.Sprintf("Notion API error [%d]: %s - %s", e.Status, e.Code, e.Message)
}
