//go:build js && wasm

package notion

import (
	"github.com/gleicon/webclaw/internal/oauth"
	"github.com/gleicon/webclaw/internal/tools"
)

// RegisterTools registers all Notion tools with the given registry.
// This function creates a new Notion client using the provided OAuth manager
// and registers all 5 Notion tools:
//   - notion_list_databases: List available databases
//   - notion_query: Query databases with filters
//   - notion_read: Read page content
//   - notion_update: Update page properties
//   - notion_search: Search pages and databases
func RegisterTools(registry *tools.Registry, oauthMgr *oauth.OAuthManager) {
	client := NewClient(oauthMgr)
	toolSet := NewNotionToolSet(client)
	toolSet.RegisterAll(registry)
}
