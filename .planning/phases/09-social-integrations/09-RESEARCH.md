# Phase 9: Social & Productivity Integrations - Technical Research

**Research Date:** 2026-03-05  
**Phase:** 09-social-integrations  
**Focus:** OAuth 2.0 PKCE flow and API integration patterns for browser-based AI assistant

---

## 1. OAuth 2.0 with PKCE for Browser Applications

### 1.1 Why PKCE?

Standard OAuth 2.0 authorization code flow is vulnerable to authorization code interception attacks in mobile and browser apps where the client secret cannot be kept confidential. PKCE (Proof Key for Code Exchange, pronounced "pixy") solves this by introducing a secret created by the calling application that can be verified by the authorization server.

**Flow:**
1. App generates PKCE parameters: code_verifier (random string) and code_challenge (hash of verifier)
2. App opens authorization URL with code_challenge
3. User authenticates, redirects back with authorization code
4. App exchanges code for token using code_verifier
5. Authorization server verifies code_challenge matches code_verifier

### 1.2 Browser-Specific Considerations

**Redirect URIs:**
- Must be exact match with registered URI
- For SPAs: Use `https://domain.com/oauth/callback` or custom scheme
- **Challenge:** WebClaw is a static file - no server to handle redirects

**Solutions:**
1. **Popup Window Flow:**
   - Open OAuth in popup window
   - Listen for window.postMessage from popup
   - Extract token from URL hash/params when popup redirects
   - Close popup after token extraction

2. **Service Worker Interception:**
   - Service Worker intercepts fetch to `/oauth/callback`
   - Extracts token from request
   - Posts message to main window
   - Returns 200 OK to prevent navigation

3. **Hash Fragment + postMessage:**
   - Redirect to parent window with token in hash
   - Parent reads hash and clears it
   - Communicate success to opener

**Recommended:** Popup window flow - simplest, most compatible

### 1.3 Token Storage

**Requirements:**
- Secure storage (never plaintext)
- Survives page reloads
- Scoped per integration
- Auto-refresh handling

**Implementation:**
- Store in IndexedDB with same encryption as API keys
- Use Web Crypto API for encryption
- Refresh tokens stored separately from access tokens
- Expiration timestamps stored for proactive refresh

### 1.4 Platform-Specific OAuth URLs

**Twitter/X:**
- Authorize: `https://twitter.com/i/oauth2/authorize`
- Token: `https://api.twitter.com/2/oauth2/token`
- Scopes: `tweet.read`, `tweet.write`, `users.read`, `offline.access`
- PKCE: Required for confidential clients
- Docs: https://developer.twitter.com/en/docs/authentication/oauth-2-0

**Google (Gmail/Calendar):**
- Authorize: `https://accounts.google.com/o/oauth2/v2/auth`
- Token: `https://oauth2.googleapis.com/token`
- Scopes: 
  - Gmail: `https://www.googleapis.com/auth/gmail.modify`
  - Calendar: `https://www.googleapis.com/auth/calendar.events`
- PKCE: Supported, recommended for SPAs
- Docs: https://developers.google.com/identity/protocols/oauth2/native-app

**GitHub:**
- Authorize: `https://github.com/login/oauth/authorize`
- Token: `https://github.com/login/oauth/access_token`
- Scopes: `repo`, `issues`, `pull_requests`, `read:user`
- PKCE: Required for OAuth apps
- Docs: https://docs.github.com/en/developers/apps/building-oauth-apps

**Notion:**
- Authorize: `https://api.notion.com/v1/oauth/authorize`
- Token: `https://api.notion.com/v1/oauth/token`
- Scopes: Determined by integration capabilities
- PKCE: Required
- Docs: https://developers.notion.com/docs/authorization

---

## 2. API Integration Patterns

### 2.1 Common API Patterns

**REST APIs:**
- Twitter/X: REST API v2 (JSON)
- GitHub: REST API v3 (JSON) + GraphQL v4
- Notion: REST API (JSON)

**Authentication:**
- Bearer token in `Authorization: Bearer {token}` header
- Rate limiting: 429 responses with retry-after header

**Pagination:**
- Twitter/X: Cursor-based (`next_token`)
- GitHub: Link headers + page-based
- Notion: Cursor-based (`start_cursor`)

### 2.2 CORS Considerations

All major APIs support CORS for browser requests:
- Twitter/X: ✅ CORS enabled for api.twitter.com
- Google APIs: ✅ CORS enabled
- GitHub: ✅ CORS enabled for api.github.com
- Notion: ✅ CORS enabled

**Handling:**
- Standard fetch() with credentials: 'omit' (don't send cookies)
- Authorization header with bearer token
- Handle 401/403 for token expiration

### 2.3 Rate Limiting Strategies

**Detection:**
- HTTP 429 Too Many Requests
- Response headers: `x-rate-limit-remaining`, `x-rate-limit-reset`

**Handling:**
1. Exponential backoff: 1s, 2s, 4s, 8s...
2. Queue requests when rate limited
3. Notify user when hitting limits
4. Cache responses to reduce API calls

---

## 3. Implementation Architecture

### 3.1 OAuth Manager (Go WASM)

```go
type OAuthManager struct {
    providers map[string]OAuthProvider
    tokenStore *TokenStore // Encrypted IndexedDB
}

type OAuthProvider struct {
    Name string
    AuthURL string
    TokenURL string
    Scopes []string
    ClientID string // Public client ID for SPAs
}

type TokenStore struct {
    // Encrypted storage in IndexedDB
    // Methods: SaveToken, LoadToken, DeleteToken, RefreshToken
}
```

**Responsibilities:**
- Initiate OAuth flow (generate PKCE, open popup)
- Handle callback (extract token from popup)
- Store tokens securely
- Refresh expired tokens
- Provide tokens to API clients

### 3.2 Integration Tools Pattern

Each integration follows the same pattern as existing WebClaw tools:

```go
func NewTwitterTool(oauthMgr *OAuthManager) *Tool {
    return &Tool{
        Name: "twitter_post",
        Description: "Post a tweet to Twitter/X",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "text": map[string]interface{}{
                    "type": "string",
                    "description": "Tweet text (max 280 chars)",
                },
                "reply_to": map[string]interface{}{
                    "type": "string",
                    "description": "Tweet ID to reply to (optional)",
                },
            },
            "required": []string{"text"},
        },
        Execute: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
            // Get OAuth token
            token, err := oauthMgr.GetToken("twitter")
            if err != nil {
                return &ToolResult{IsError: true, ...}, nil
            }
            
            // Call Twitter API
            // Return result
        },
    }
}
```

### 3.3 JavaScript Bridge Functions

**OAuth Flow:**
```javascript
// Called from Go WASM
window.webclaw.oauth = {
    // Open popup and initiate OAuth
    initiate: async (provider, codeChallenge) => {
        const popup = window.open(authUrl, 'oauth', 'width=500,height=600');
        // Listen for postMessage from popup
        return new Promise((resolve) => {
            window.addEventListener('message', (e) => {
                if (e.data.type === 'oauth-callback') {
                    resolve(e.data.code);
                }
            });
        });
    },
    
    // Exchange code for token (called from popup callback page)
    exchangeCode: async (provider, code, codeVerifier) => {
        const response = await fetch(tokenUrl, {
            method: 'POST',
            body: JSON.stringify({
                grant_type: 'authorization_code',
                code,
                code_verifier: codeVerifier,
                client_id: clientId,
                redirect_uri: redirectUri,
            }),
        });
        return response.json();
    }
};
```

---

## 4. Security Considerations

### 4.1 Token Security

**Threats:**
- XSS attacks stealing tokens
- Token leakage in browser history
- Man-in-the-middle attacks

**Mitigations:**
1. Store tokens encrypted in IndexedDB (same as API keys)
2. Use `Authorization: Bearer` header (not URL params)
3. Implement CSP (Content Security Policy)
4. Use PKCE (prevents authorization code interception)
5. Short-lived access tokens + refresh tokens
6. Clear tokens on logout

### 4.2 Scope Management

**Principle of Least Privilege:**
- Request minimum scopes needed
- Document why each scope is needed
- Allow user to review scopes before auth
- Store granted scopes, check before API calls

**Example Twitter Scopes:**
- `tweet.read` - Read tweets (for timeline)
- `tweet.write` - Post tweets
- `users.read` - Read user profiles
- `offline.access` - Refresh token support

---

## 5. UI/UX Considerations

### 5.1 OAuth Connection Flow

1. **Settings Panel:**
   - List available integrations (Twitter, GitHub, Google, Notion)
   - Show connection status (connected/disconnected)
   - "Connect" button for each

2. **Connection Process:**
   - Click "Connect to Twitter"
   - Popup opens to Twitter OAuth
   - User authenticates, grants permission
   - Popup closes, WebClaw stores token
   - Settings shows "Connected as @username"
   - "Disconnect" button available

3. **Error Handling:**
   - If popup blocked: Show error, ask user to allow popups
   - If user denies: Show "Access denied" message
   - If token expired: Auto-refresh or prompt re-auth

### 5.2 Tool Usage Flow

Example: User says "Post to Twitter: Hello world!"

1. LLM recognizes intent, selects `twitter_post` tool
2. Tool executes:
   - Check if Twitter connected (OAuth token exists)
   - If not: Return error "Please connect Twitter in Settings"
   - If yes: Get token, call API
3. API response: Success or error
4. Tool returns result to LLM
5. LLM responds to user: "Posted! View at https://twitter.com/..."

---

## 6. Testing Strategy

### 6.1 OAuth Testing

**Mock Mode:**
- Create mock OAuth server for testing
- Bypass real authentication
- Return fake tokens
- Test flow without API keys

**Integration Testing:**
- Use real OAuth with test accounts
- Never use production credentials
- Revoke test tokens after tests

### 6.2 API Testing

**VCR/Recording:**
- Record API responses
- Replay for consistent tests
- Redact sensitive data (tokens, PII)

**Rate Limit Testing:**
- Mock 429 responses
- Verify exponential backoff
- Test queue behavior

---

## 7. Implementation Order

Based on complexity and value:

1. **OAuth Infrastructure (09-01)** - Foundation for all integrations
2. **Twitter/X (09-02)** - High value, clear API, OAuth 2.0
3. **Google Gmail (09-03)** - High value, complex scopes
4. **Google Calendar (09-03)** - Same OAuth as Gmail
5. **GitHub (09-04)** - Developer-focused, GraphQL option
6. **Notion (09-05)** - Database integration, complex queries

---

## 8. References

- OAuth 2.0 for Browser-Based Apps: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-browser-based-apps
- PKCE RFC 7636: https://datatracker.ietf.org/doc/html/rfc7636
- Twitter API v2: https://developer.twitter.com/en/docs/twitter-api
- Google OAuth 2.0: https://developers.google.com/identity/protocols/oauth2
- GitHub OAuth: https://docs.github.com/en/developers/apps/building-oauth-apps
- Notion API: https://developers.notion.com/
