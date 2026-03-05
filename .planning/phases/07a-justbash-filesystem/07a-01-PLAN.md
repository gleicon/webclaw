---
phase: 07a-justbash-filesystem
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - package.json
  - static/justbash-bridge.js
  - internal/jsbridge/justbash.go
  - internal/tools/file_tools.go
autonomous: true
requirements:
  - BRIDGE-01
  - BRIDGE-02
  - TOOL-01
  - TOOL-02
  - TOOL-03
  - TOOL-04
  - TOOL-05
  - TOOL-06
must_haves:
  truths:
    - "User can read files from virtual filesystem without bridge binary"
    - "User can write files to virtual filesystem without bridge binary"
    - "User can list directory contents via just-bash commands"
    - "User can search for text patterns in files"
    - "File operations work immediately without waiting for bridge"
    - "just-bash library is loaded and initialized on startup"
  artifacts:
    - path: "package.json"
      provides: "just-bash npm dependency"
      contains: "@jstz-dev/just-bash"
    - path: "static/justbash-bridge.js"
      provides: "just-bash initialization and JS bridge"
      min_lines: 150
    - path: "internal/jsbridge/justbash.go"
      provides: "Go→JS bridge for just-bash commands"
      exports: ["InitJustBash", "ExecuteCommand", "ReadFile", "WriteFile", "ListDir"]
    - path: "internal/tools/file_tools.go"
      provides: "File tool implementations using just-bash"
      exports: ["NewFileReadTool", "NewFileWriteTool", "NewDirListTool", "NewFileSearchTool"]
  key_links:
    - from: "internal/tools/file_tools.go"
      to: "internal/jsbridge/justbash.go"
      via: "function calls"
      pattern: "justbash\\."
    - from: "internal/jsbridge/justbash.go"
      to: "static/justbash-bridge.js"
      via: "syscall/js"
      pattern: "js\\.Global\\(\\)\\.Call.*justbash"
    - from: "index.html"
      to: "static/justbash-bridge.js"
      via: "script tag"
      pattern: "justbash-bridge\\.js"
user_setup: []
---

<objective>
Integrate just-bash into WebClaw to enable browser-only file operations without requiring the local bridge binary. This plan establishes the foundation for virtual filesystem operations using just-bash's OverlayFs and InMemoryFs.

Purpose: Enable immediate file operations in WebClaw by providing a JavaScript-based virtual filesystem that runs entirely in the browser. This eliminates the dependency on the local bridge binary for file I/O, making WebClaw instantly usable.

Output: 
- just-bash npm dependency installed
- JavaScript bridge layer for just-bash integration
- Go bindings to call just-bash from WASM
- File tools (read, write, list, search) implemented via just-bash
- Tools registered in the agent loop
</objective>

<execution_context>
@/Users/gleicon/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/gleicon/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@cmd/webclaw/main.go
@internal/tools/tool.go
@internal/tools/registry.go
@internal/jsbridge/bridge.go
@internal/jsbridge/files.go
@static/webclaw-host.js
@static/worker.js
@index.html

## Key Interfaces (from existing codebase)

From internal/tools/tool.go:
```go
type ToolResult struct {
    Content        string
    DisplayContent string
    IsError        bool
    ToolName       string
    Status         string
}

type Tool struct {
    Name        string
    Description string
    InputSchema map[string]interface{}
    Execute     func(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}
```

From internal/jsbridge/bridge.go:
```go
func RegisterCallback(fn js.Func) {
    liveCallbacks = append(liveCallbacks, fn)
}
```

From internal/tools/web_fetch.go (tool pattern):
```go
func NewWebFetchTool() *Tool {
    return &Tool{
        Name:        "web_fetch",
        Description: "Fetch the content of a URL...",
        InputSchema: map[string]interface{}{...},
        Execute: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
            // Implementation using jsbridge
        },
    }
}
```
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add just-bash npm dependency</name>
  <files>package.json</files>
  <action>
    Create or update package.json to include @jstz-dev/just-bash as a dependency. This is the core library providing the virtual filesystem and bash command simulation.
    
    Steps:
    1. Check if package.json exists in the project root
    2. If not exists, create a new package.json with just-bash as the only dependency
    3. If exists, add @jstz-dev/just-bash to the dependencies section
    4. Use a compatible version (latest stable or ^0.x.x)
    
    Content should include:
    {
      "name": "webclaw",
      "version": "1.0.0",
      "dependencies": {
        "@jstz-dev/just-bash": "^0.1.0"
      }
    }
    
    Note: The actual version should be determined from npm registry. just-bash provides:
    - OverlayFs: Read from real filesystem, write to memory
    - InMemoryFs: Completely sandboxed virtual filesystem
    - 79+ bash commands available in browser
  </action>
  <verify>
    <automated>cat package.json | grep -q "just-bash" && echo "just-bash dependency added"</automated>
  </verify>
  <done>package.json exists with @jstz-dev/just-bash dependency specified</done>
</task>

<task type="auto">
  <name>Task 2: Create just-bash JavaScript bridge layer</name>
  <files>static/justbash-bridge.js</files>
  <action>
    Create a JavaScript module that initializes just-bash and exposes an API for Go WASM to call. This layer runs in the main browser thread (not the worker).
    
    Implementation requirements:
    1. Import just-bash library (from CDN or node_modules)
    2. Create a virtual filesystem instance (start with InMemoryFs for safety)
    3. Expose functions on window.justbash that Go can call via syscall/js:
       - executeCommand(cmd, args[]) -> {stdout, stderr, exitCode}
       - readFile(path) -> content or error
       - writeFile(path, content) -> success or error
       - listDir(path) -> [{name, type, size}]
       - searchFiles(pattern, path) -> [{path, line, content}]
       - mountOverlay(realPath, virtualPath) -> mount info
       - getFilesystemInfo() -> {type, mountPoints, stats}
    
    4. Use just-bash's command implementations:
       - cat, head, tail for reading
       - echo with redirection for writing
       - ls, tree for listing
       - grep, find for searching
       - stat for file info
       - mkdir, touch for creation
       - sed, awk for editing
    
    5. Handle async operations with Promises (Go's syscall/js expects promises for async)
    
    6. Add error handling that returns structured error objects
    
    7. Dispatch a 'justbash:ready' event when initialization completes
    
    Key just-bash APIs to use:
    - createOverlayFs(mountPoint, realDirHandle) - for preview mode
    - createInMemoryFs() - for virtual mode
    - execute(commandString, options) - run bash commands
    
    Note: Since we're in a browser, we can't easily access the real filesystem without user permission (File System Access API). Start with InMemoryFs for full sandboxing, add OverlayFs support in later tasks.
  </action>
  <verify>
    <automated>test -f static/justbash-bridge.js && head -50 static/justbash-bridge.js | grep -q "just-bash\|JustBash\|window.justbash" && echo "Bridge file created with just-bash references"</automated>
  </verify>
  <done>static/justbash-bridge.js exists with exported window.justbash API containing executeCommand, readFile, writeFile, listDir, searchFiles functions</done>
</task>

<task type="auto">
  <name>Task 3: Create Go bindings for just-bash bridge</name>
  <files>internal/jsbridge/justbash.go</files>
  <action>
    Create Go bindings that call the JavaScript just-bash bridge via syscall/js. This file provides the interface between Go WASM code and the just-bash JavaScript library.
    
    Implementation:
    1. Create package jsbridge (already exists)
    2. Define types for file info, search results, command results
    3. Implement functions:
       - InitJustBash() - waits for justbash:ready event, initializes connection
       - ExecuteCommand(cmd string, args []string) (stdout, stderr string, exitCode int, err error)
       - ReadFile(path string) (content string, err error) - uses cat command
       - WriteFile(path string, content string) (err error) - uses echo with redirection
       - ListDir(path string) ([]FileInfo, error) - uses ls -la
       - SearchFiles(pattern string, path string) ([]SearchResult, error) - uses grep -r
       - FileExists(path string) (bool, error) - uses test -f
       - GetFilesystemInfo() (FsInfo, error) - returns type, mount points, stats
    
    4. Each function should:
       - Check if window.justbash is available
       - Call the appropriate JS function via js.Global().Get("justbash").Call(...)
       - Handle Promise resolution (async operations in Go WASM)
       - Parse JSON results back to Go types
       - Provide clear error messages
    
    5. Add helper for converting js.Value to Go types
    6. Register the just-bash ready state for other packages to check
    
    Pattern to follow (from jsbridge/bridge.go):
    ```go
    func ExecuteCommand(cmd string, args []string) (string, string, int, error) {
        justbash := js.Global().Get("justbash")
        if justbash.IsUndefined() {
            return "", "", -1, fmt.Errorf("just-bash not initialized")
        }
        
        // Create args array for JS
        argsJS := js.Global().Get("Array").New(len(args))
        for i, arg := range args {
            argsJS.SetIndex(i, arg)
        }
        
        // Call JS function - returns Promise
        promise := justbash.Call("executeCommand", cmd, argsJS)
        
        // Wait for promise resolution using jsbridge pattern
        resultCh := make(chan js.Value)
        errCh := make(chan error)
        
        // ... promise handling using Then/catch callbacks ...
        
        select {
        case result := <-resultCh:
            // Parse result object
            stdout := result.Get("stdout").String()
            stderr := result.Get("stderr").String()
            exitCode := result.Get("exitCode").Int()
            return stdout, stderr, exitCode, nil
        case err := <-errCh:
            return "", "", -1, err
        }
    }
    ```
    
    7. Add a JustBashAvailable() function for tools to check availability before routing
  </action>
  <verify>
    <automated>test -f internal/jsbridge/justbash.go && grep -q "func.*JustBash" internal/jsbridge/justbash.go | wc -l | xargs test {} -ge 5 && echo "Go bindings created with multiple functions"</automated>
  </verify>
  <done>internal/jsbridge/justbash.go exists with ExecuteCommand, ReadFile, WriteFile, ListDir, SearchFiles, FileExists functions implemented using syscall/js</done>
</task>

<task type="auto">
  <name>Task 4: Implement file tools using just-bash</name>
  <files>internal/tools/file_tools.go</files>
  <action>
    Create file operation tools that use the just-bash Go bindings. These tools will be registered in the tool registry and called by the agent loop.
    
    Tools to implement:
    1. file_read - Read file contents using cat/head/tail
       - Input: {path: string, limit?: number}
       - Output: File content or error
       - Uses: ReadFile binding or ExecuteCommand("cat", [path])
    
    2. file_write - Write content to file using echo/printf
       - Input: {path: string, content: string, append?: boolean}
       - Output: Success confirmation or error
       - Uses: WriteFile binding or ExecuteCommand with redirection
    
    3. dir_list - List directory contents
       - Input: {path: string}
       - Output: Array of {name, type, size, modified}
       - Uses: ListDir binding or ExecuteCommand("ls", ["-la", path])
    
    4. file_search - Search for text patterns
       - Input: {pattern: string, path: string, recursive?: boolean}
       - Output: Array of {path, line, content, match}
       - Uses: SearchFiles binding or ExecuteCommand("grep", ["-r", "-n", pattern, path])
    
    5. file_edit - Edit file using sed/awk
       - Input: {path: string, operation: "replace"|"insert"|"delete", target: string, replacement?: string, line?: number}
       - Output: Success confirmation or error
       - Uses: ExecuteCommand with sed
    
    6. file_stat - Get file metadata
       - Input: {path: string}
       - Output: {size, modified, type, permissions}
       - Uses: ExecuteCommand("stat", [path]) or FileExists + ListDir
    
    Each tool should:
    - Check if just-bash is available (via jsbridge.JustBashAvailable())
    - If not available, return ToolResult with IsError=true and appropriate message
    - Format DisplayContent for UI (summarize results, not full output)
    - Return Content for LLM (may be full output or structured)
    - Set Status to "running" at start, "done" or "error" at end
    - Emit tool events via workerBridge if available
    
    Pattern (from web_fetch.go):
    ```go
    func NewFileReadTool() *Tool {
        return &Tool{
            Name:        "file_read",
            Description: "Read the contents of a file from the virtual filesystem",
            InputSchema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "path": map[string]interface{}{
                        "type":        "string",
                        "description": "Path to the file to read",
                    },
                    "limit": map[string]interface{}{
                        "type":        "number",
                        "description": "Maximum number of lines to read (optional)",
                    },
                },
                "required": []string{"path"},
            },
            Execute: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
                // Implementation
            },
        }
    }
    ```
    
    Security considerations:
    - All paths should be relative to virtual filesystem root
    - No access to real filesystem outside mount points
    - Path traversal attempts (../) should be sanitized
  </action>
  <verify>
    <automated>test -f internal/tools/file_tools.go && grep -q "NewFileReadTool\|NewFileWriteTool\|NewDirListTool\|NewFileSearchTool" internal/tools/file_tools.go | wc -l | xargs test {} -ge 4 && echo "File tools implemented"</automated>
  </verify>
  <done>internal/tools/file_tools.go exists with all 6 file tools (read, write, list, search, edit, stat) implemented using just-bash bindings</done>
</task>

<task type="auto">
  <name>Task 5: Wire file tools into agent loop</name>
  <files>cmd/webclaw/main.go</files>
  <action>
    Update main.go to register the new file tools in the tool registry alongside existing tools (web_fetch, web_search, memory_store, memory_search).
    
    Steps:
    1. Import the tools package (already imported)
    2. After registering existing tools (lines 181-186), add:
       reg.Register(tools.NewFileReadTool())
       reg.Register(tools.NewFileWriteTool())
       reg.Register(tools.NewDirListTool())
       reg.Register(tools.NewFileSearchTool())
       reg.Register(tools.NewFileEditTool())
       reg.Register(tools.NewFileStatTool())
    
    3. Add just-bash initialization before tool registry setup:
       - Call jsbridge.InitJustBash() early in main()
       - Log status: "webclaw: just-bash initialized" or warning if not available
       - just-bash is optional - tools should gracefully degrade
    
    4. Update verifyAgentLoopWiring() to check for file tools:
       - Add check that file tools are registered
       - Log count of file tools
    
    5. Add script tag to index.html to load justbash-bridge.js:
       - Add after webclaw-host.js: <script src="static/justbash-bridge.js"></script>
       - Or add to the import in webclaw-host.js if using ES modules
    
    Implementation details:
    - just-bash initialization should be non-blocking (it loads async)
    - Tools should check availability at execution time, not registration time
    - If just-bash fails to load, file tools return "not available" errors
    - The bridge binary (Phase 7) will provide an alternative implementation
  </action>
  <verify>
    <automated>grep -q "NewFileReadTool\|NewFileWriteTool\|NewDirListTool" cmd/webclaw/main.go && echo "File tools wired into main.go"</automated>
  </verify>
  <done>main.go registers all 6 file tools in the tool registry and includes just-bash initialization</done>
</task>

</tasks>

<verification>
After completing all tasks:

1. Build the WASM module: GOOS=js GOARCH=wasm go build -o dist/webclaw.wasm ./cmd/webclaw
2. Start the dev server: go run cmd/devserver/main.go
3. Open browser to http://localhost:8080
4. Check browser console for "webclaw: just-bash initialized" message
5. Verify file tools are loaded by checking registry: window.webclaw.tools should list file_read, file_write, etc.
6. Test file_read via browser console:
   ```javascript
   // First create a test file
   await window.justbash.writeFile('/test.txt', 'Hello from just-bash!');
   // Then read it back
   const content = await window.justbash.readFile('/test.txt');
   console.log(content); // Should show "Hello from just-bash!"
   ```
7. Verify tool registry integration:
   - Send a message in chat: "Please read the file /test.txt"
   - Check if agent attempts to use file_read tool (will fail gracefully if just-bash not fully ready)
</verification>

<success_criteria>
Phase 7a is successful when:

1. **Dependency present**: package.json includes @jstz-dev/just-bash dependency
2. **Bridge loaded**: static/justbash-bridge.js exists and exports window.justbash API
3. **Go bindings work**: internal/jsbridge/justbash.go compiles and provides functions
4. **Tools implemented**: internal/tools/file_tools.go has 6 working file tools
5. **Wiring complete**: cmd/webclaw/main.go registers all file tools
6. **Manual test passes**: Can create, read, and list files via browser console
7. **Graceful degradation**: When just-bash unavailable, tools return clear error messages
8. **No bridge required**: All operations work without webclaw-bridge binary running
</success_criteria>

<output>
After completion, create `.planning/phases/07a-justbash-filesystem/07a-01-SUMMARY.md`

Summary should document:
- just-bash version used and why
- Architecture: Go WASM → JS bridge → just-bash → Virtual Filesystem
- Filesystem modes: InMemoryFs (default, fully sandboxed)
- Commands available: cat, head, tail, ls, grep, sed, mkdir, touch, etc.
- Security model: Path sanitization, no real filesystem access
- Testing notes: How to verify in browser console
- Next phase preparation: OverlayFs mounting for Phase 7a-02
</output>
