//go:build js && wasm

package integrations

import (
	"github.com/gleicon/webclaw/internal/integrations/twitter"
	"github.com/gleicon/webclaw/internal/oauth"
	"github.com/gleicon/webclaw/internal/tools"
)

// RegisterTwitterTools registers all Twitter integration tools with the registry
func RegisterTwitterTools(registry *tools.Registry, oauthMgr *oauth.OAuthManager) {
	toolSet := twitter.NewTwitterToolSet(oauthMgr)
	toolSet.RegisterAll(registry)
}
