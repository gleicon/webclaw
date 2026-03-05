# WebClaw Agent Instructions

## Overview

WebClaw is a browser-native AI assistant that runs entirely in your browser using WebAssembly (WASM). It implements the OpenClaw specification and provides a comprehensive set of tools for web operations, file management, and information retrieval - all without requiring any server or local installation.

## Architecture

WebClaw consists of three main components:

1. **Go Core (WASM)**: Compiled from Go to WebAssembly, handles the agent loop, provider routing, memory system, and tool execution
2. **JavaScript Host**: Thin layer that provides browser APIs (fetch, IndexedDB, Web Crypto) to the WASM core
3. **Web Worker**: Handles streaming LLM responses without blocking the UI thread

## Available Tools

### Web & Search Tools

#### `web_fetch`
Fetch and extract text content from any URL.

**Input Schema:**
```json
{
  "url": "string (required) - URL to fetch",
  "selector": "string (optional) - CSS selector to extract specific content",
  "max_length": "number (optional) - Maximum characters to return (default: 10000)"
}
```

**Example:**
```
web_fetch url="https://example.com" max_length=5000
```

#### `web_search`
Search the web using multiple search engines with automatic fallback.

**Input Schema:**
```json
{
  "query": "string (required) - Search query",
  "num_results": "number (optional) - Number of results (default: 5, max: 10)"
}
```

**Example:**
```
web_search query="latest AI developments" num_results=8
```

### File Operations (Phase 7a)

WebClaw includes a complete virtual filesystem powered by [just-bash](https://github.com/vercel-labs/just-bash), providing 79+ bash commands in the browser without requiring a local bridge binary.

#### `file_read`
Read the contents of a file from the workspace.

**Input Schema:**
```json
{
  "path": "string (required) - Path to the file (relative to workspace root)",
  "offset": "number (optional) - Line number to start reading from (0-indexed)",
  "limit": "number (optional) - Maximum number of lines to read"
}
```

**Examples:**
```
file_read path="main.go"
file_read path="README.md" limit=50
file_read path="large-file.txt" offset=100 limit=20
```

#### `file_write`
Write content to a file. Creates parent directories automatically.

**Input Schema:**
```json
{
  "path": "string (required) - Path to the file",
  "content": "string (required) - Content to write",
  "append": "boolean (optional) - Append to existing content instead of overwriting"
}
```

**Examples:**
```
file_write path="hello.txt" content="Hello, World!"
file_write path="notes.md" content="# Project Notes" append=true
```

#### `dir_list`
List files and directories in a given path.

**Input Schema:**
```json
{
  "path": "string (optional) - Directory path (default: current directory)",
  "recursive": "boolean (optional) - Include subdirectories"
}
```

**Examples:**
```
dir_list path="src"
dir_list path="." recursive=true
```

#### `file_search`
Search for text patterns in files using grep syntax.

**Input Schema:**
```json
{
  "pattern": "string (required) - Search pattern (regex supported)",
  "path": "string (optional) - Path to search (default: current directory)",
  "recursive": "boolean (optional) - Search subdirectories",
  "ignore_case": "boolean (optional) - Case-insensitive search"
}
```

**Examples:**
```
file_search pattern="TODO" recursive=true
file_search pattern="func.*main" path="src" ignore_case=true
```

**Available bash commands:** cat, cp, file, ln, ls, mkdir, mv, readlink, rm, rmdir, split, stat, touch, tree, awk, base64, column, comm, cut, diff, expand, fold, grep, head, join, md5sum, nl, od, paste, printf, rev, rg, sed, sha1sum, sha256sum, sort, strings, tac, tail, tr, unexpand, uniq, wc, xargs, jq, sqlite3, gzip, gunzip, zcat, tar, basename, cd, dirname, du, echo, env, export, find, hostname, printenv, pwd, tee, alias, bash, chmod, clear, date, expr, false, help, history, seq, sh, sleep, time, timeout, true, unalias, which, whoami, curl

### Memory Tools

#### `memory_store`
Store a memory document for later retrieval.

**Input Schema:**
```json
{
  "content": "string (required) - Content to store",
  "tags": "array of strings (optional) - Tags for categorization",
  "source": "string (optional) - Source of the information"
}
```

**Example:**
```
memory_store content="User prefers TypeScript over JavaScript" tags=["preferences", "tech"]
```

#### `memory_search`
Search stored memories using hybrid BM25 + vector search.

**Input Schema:**
```json
{
  "query": "string (required) - Search query",
  "limit": "number (optional) - Maximum results (default: 5)"
}
```

**Example:**
```
memory_search query="user preferences" limit=3
```

### System Tools

#### `help`
Get documentation and help for available tools.

**Input Schema:**
```json
{
  "tool": "string (optional) - Name of tool to get help for (lists all if omitted)",
  "verbose": "boolean (optional) - Show full input schema details"
}
```

**Examples:**
```
help                    # List all tools
help tool="file_read"   # Get help for specific tool
help tool="file_write" verbose=true  # Detailed help with schema
```

## How to Use Tools

Tools are automatically available to the LLM. When you ask the agent to perform an action, it will:

1. Select the appropriate tool(s)
2. Call them with the correct parameters
3. Process the results
4. Provide you with a response

You can also explicitly request tool usage:
- "Search for information about Go error handling"
- "Read the main.go file in my workspace"
- "List all files in the src directory"

## Getting Help

Use the `/help` command or ask:
- "What tools are available?"
- "How do I use the file_write tool?"
- "Show me help for memory_search"

## Security Model

- **Sandboxed**: File operations use a virtual filesystem (InMemoryFs) - all writes stay in memory
- **No Server**: Everything runs in your browser, no data leaves your machine (except LLM API calls)
- **Encrypted Keys**: API keys are encrypted with Web Crypto API (AES-256-GCM)
- **CORS-Aware**: Web requests respect browser CORS policies

## Browser Compatibility

WebClaw requires a modern browser with WebAssembly support:
- Chrome/Edge 80+
- Firefox 78+
- Safari 14+

File System Access API (for OverlayFs mode) requires Chrome/Edge 86+.

## OpenClaw Compatibility

WebClaw implements the OpenClaw specification:
- Identity files (IDENTITY.md, SOUL.md, USER.md, etc.)
- Config format (JSON5 with camelCase and snake_case support)
- Tool schemas
- Memory system (hybrid search)

Differences from TypeScript OpenClaw:
- Browser-native (no Node.js required)
- WASM-based runtime
- JS/TS plugin system instead of Node.js plugins
- Optional local bridge for file system access

## Development

WebClaw is built with:
- **Go**: Core agent logic, compiled to WASM via TinyGo
- **JavaScript**: Host layer, bridge to browser APIs
- **Tailwind CSS**: Dark-mode UI

Repository: https://github.com/gleicon/webclaw
