---
phase: 07a-justbash-filesystem
plan: 04
type: execute
wave: 3
depends_on:
  - 07a-02
files_modified:
  - internal/tools/file_tools.go
  - internal/jsbridge/justbash.go
  - static/justbash-bridge.js
autonomous: true
requirements:
  - TOOL-03
  - TOOL-04
  - TOOL-05
  - TOOL-06
must_haves:
  truths:
    - "file_edit tool supports multiple edit operations (replace, insert, delete)"
    - "file_search tool supports regex and recursive search"
    - "dir_list tool shows detailed file information"
    - "file_stat tool provides comprehensive metadata"
    - "All file tools handle large files efficiently"
    - "Edit operations are atomic and reversible in overlay"
  artifacts:
    - path: "internal/tools/file_tools.go"
      provides: "Advanced file edit and search tools"
      exports: ["NewFileEditTool", "NewFileSearchTool"]
      contains: "replace, insert, delete operations"
    - path: "internal/jsbridge/justbash.go"
      provides: "Advanced command execution"
      exports: ["ExecuteSed", "ExecuteGrep", "ExecuteAwk"]
    - path: "static/justbash-bridge.js"
      provides: "Complex bash command support"
      exports: ["sed", "grep", "awk", "find", "xargs"]
  key_links:
    - from: "file_edit tool"
      to: "sed/awk commands"
      via: "ExecuteSed/ExecuteAwk"
      pattern: "sed.*-i\|awk.*replace"
    - from: "file_search tool"
      to: "grep/find commands"
      via: "ExecuteGrep"
      pattern: "grep.*-r\|find.*-name"
user_setup: []
---

<objective>
Implement advanced file manipulation tools including sophisticated edit operations (sed/awk-based), powerful search capabilities (grep with regex), and comprehensive file analysis. These tools provide the agent with professional-grade file editing capabilities within the safe virtual filesystem.

Purpose: Enable the agent to perform complex file operations like find-and-replace, line insertion/deletion, multi-file search, and detailed file analysis. These capabilities are essential for code refactoring, content analysis, and project-wide modifications.

Output:
- file_edit tool with sed/awk operations (replace, insert, delete, append)
- file_search tool with regex, recursive search, and result ranking
- Enhanced file_stat with file type detection and encoding
- Batch operations support for multi-file edits
- Large file handling with streaming/chunked operations
- Preview mode for edits before application
</objective>

<execution_context>
@/Users/gleicon/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/gleicon/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/07a-justbash-filesystem/07a-02-SUMMARY.md
@internal/tools/file_tools.go
@internal/jsbridge/justbash.go
@static/justbash-bridge.js

## just-bash Available Commands

Based on just-bash documentation, these commands should be available:
- File manipulation: cat, head, tail, wc, sort, uniq, cut, paste
- Text processing: sed, awk, grep, egrep, fgrep, tr, rev
- File operations: find, ls, stat, touch, mkdir, rm, cp, mv
- Stream processing: xargs, tee, diff, comm

## Sed Command Patterns

Replace: sed -i 's/old/new/g' file
Insert at line: sed -i '3i\new line' file
Delete line: sed -i '5d' file
Append after line: sed -i '5a\new line' file
Replace line: sed -i '5c\new content' file

## Grep Command Patterns

Basic search: grep pattern file
Recursive: grep -r pattern directory
With line numbers: grep -n pattern file
Case insensitive: grep -i pattern file
Regex: grep -E 'regex' file
Invert match: grep -v pattern file
Context lines: grep -C 3 pattern file
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Implement advanced file_edit tool with sed/awk</name>
  <files>internal/tools/file_tools.go</files>
  <behavior>
    file_edit tool operations:
    
    Test 1: Replace operation
    - Input: {path: "/test.txt", operation: "replace", target: "old", replacement: "new"}
    - File content: "old text old"
    - Expected result: "new text new"
    
    Test 2: Replace with regex
    - Input: {path: "/test.txt", operation: "replace", target: "[aeiou]", replacement: "X", use_regex: true}
    - File content: "hello world"
    - Expected result: "hXllX wXrld"
    
    Test 3: Insert operation
    - Input: {path: "/test.txt", operation: "insert", line: 2, content: "inserted line"}
    - File content: "line1\nline2\nline3"
    - Expected result: "line1\ninserted line\nline2\nline3"
    
    Test 4: Delete operation
    - Input: {path: "/test.txt", operation: "delete", line: 2}
    - File content: "line1\nline2\nline3"
    - Expected result: "line1\nline3"
    
    Test 5: Append operation
    - Input: {path: "/test.txt", operation: "append", content: "appended"}
    - File content: "original"
    - Expected result: "original\nappended"
    
    Test 6: Line range replacement
    - Input: {path: "/test.txt", operation: "replace_lines", start_line: 2, end_line: 3, content: "new lines"}
    - File content: "line1\nline2\nline3\nline4"
    - Expected result: "line1\nnew lines\nline4"
    
    Test 7: Preview mode (dry run)
    - Input: {path: "/test.txt", operation: "replace", target: "old", replacement: "new", preview: true}
    - Expected: Returns diff showing changes without applying
    
    Error cases:
    - File doesn't exist: returns error
    - Line number out of range: returns error
    - Invalid regex: returns error with helpful message
  </behavior>
  <action>
    Extend the file_edit tool to support multiple edit operations using sed and awk commands via just-bash.
    
    Enhanced file_edit tool schema:
    ```go
    InputSchema: map[string]interface{}{
      "type": "object",
      "properties": map[string]interface{}{
        "path": map[string]interface{}{
          "type": "string",
          "description": "Path to the file to edit",
        },
        "operation": map[string]interface{}{
          "type": "string",
          "enum": []string{"replace", "insert", "delete", "append", "prepend", "replace_lines"},
          "description": "Type of edit operation",
        },
        // For replace operation
        "target": map[string]interface{}{
          "type": "string",
          "description": "Text or pattern to find (for replace)",
        },
        "replacement": map[string]interface{}{
          "type": "string",
          "description": "Replacement text (for replace)",
        },
        "use_regex": map[string]interface{}{
          "type": "boolean",
          "description": "Whether target is a regex pattern",
        },
        "replace_all": map[string]interface{}{
          "type": "boolean",
          "description": "Replace all occurrences (default true)",
        },
        // For insert/delete operations
        "line": map[string]interface{}{
          "type": "number",
          "description": "Line number for insert/delete (1-based)",
        },
        "start_line": map[string]interface{}{
          "type": "number",
          "description": "Start line for range operations",
        },
        "end_line": map[string]interface{}{
          "type": "number",
          "description": "End line for range operations",
        },
        // Content for insert/append/replace_lines
        "content": map[string]interface{}{
          "type": "string",
          "description": "Content to insert or replace with",
        },
        // Options
        "preview": map[string]interface{}{
          "type": "boolean",
          "description": "Show diff without applying changes",
        },
      },
      "required": []string{"path", "operation"},
    }
    ```
    
    Implementation:
    1. Create command builder for sed operations:
       ```go
       func buildSedCommand(operation string, params map[string]interface{}) (string, error) {
         switch operation {
         case "replace":
           target := params["target"].(string)
           replacement := params["replacement"].(string)
           useRegex := params["use_regex"].(bool)
           replaceAll := params["replace_all"].(bool)
           
           // Escape special sed characters
           target = escapeForSed(target)
           replacement = escapeForSed(replacement)
           
           flags := ""
           if !useRegex {
             // Use fixed string matching
           }
           if replaceAll {
             flags = "g"
           }
           
           return fmt.Sprintf("sed -i 's/%s/%s/%s'", target, replacement, flags), nil
           
         case "insert":
           line := int(params["line"].(float64))
           content := params["content"].(string)
           content = escapeForSed(content)
           return fmt.Sprintf("sed -i '%di\\%s'", line, content), nil
           
         case "delete":
           line := int(params["line"].(float64))
           return fmt.Sprintf("sed -i '%dd'", line), nil
           
         case "append":
           content := params["content"].(string)
           return fmt.Sprintf("echo '%s' >> ", content), nil // Use append, not sed
           
         case "prepend":
           content := params["content"].(string)
           return fmt.Sprintf("echo '%s' | cat - ", content), nil // Prepend pattern
           
         case "replace_lines":
           startLine := int(params["start_line"].(float64))
           endLine := int(params["end_line"].(float64))
           content := params["content"].(string)
           // Complex: delete range, then insert at start line
           return fmt.Sprintf("sed -i '%d,%dc\\%s'", startLine, endLine, content), nil
         }
       }
       ```
    
    2. Preview mode implementation:
       - Read original file
       - Generate diff showing what would change
       - Return diff in ToolResult without modifying file
       - Use diff command: diff -u original modified
    
    3. Error handling:
       - Validate line numbers before executing
       - Handle sed errors (invalid regex, file not found)
       - Provide helpful error messages with context
       - Rollback on error (if possible)
    
    4. Safety features:
       - Always create backup in overlay layer
       - Validate path is within virtual filesystem
       - Limit operations on very large files (>1MB)
       - Rate limiting for batch operations
    
    5. DisplayContent formatting:
       - For replace: "Replaced 'old' with 'new' (5 occurrences)"
       - For insert: "Inserted line at position 3"
       - For delete: "Deleted line 5"
       - For preview: "Would make 3 changes (see diff)"
  </action>
  <verify>
    <automated>grep -q "operation.*replace\|operation.*insert\|operation.*delete\|sed.*-i" internal/tools/file_tools.go && echo "Advanced edit operations implemented"</automated>
  </verify>
  <done>file_edit tool supports replace (with regex), insert, delete, append, prepend, and replace_lines operations with preview mode</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement advanced file_search with grep/find</name>
  <files>internal/tools/file_tools.go</files>
  <behavior>
    file_search tool capabilities:
    
    Test 1: Basic search
    - Input: {pattern: "function", path: "/project", recursive: true}
    - Expected: Returns list of files containing "function"
    
    Test 2: Regex search
    - Input: {pattern: "func\\s+\\w+\\(", path: "/project", use_regex: true}
    - Expected: Returns Go function definitions
    
    Test 3: Case insensitive
    - Input: {pattern: "TODO", path: "/project", case_insensitive: true}
    - Expected: Finds "todo", "TODO", "Todo"
    
    Test 4: With context lines
    - Input: {pattern: "import", path: "/file.go", context_lines: 2}
    - Expected: Returns matches with 2 lines before and after
    
    Test 5: Invert match
    - Input: {pattern: "test", path: "/project", invert_match: true}
    - Expected: Returns files NOT containing "test"
    
    Test 6: File type filter
    - Input: {pattern: "package", path: "/project", include_pattern: "*.go"}
    - Expected: Only searches .go files
    
    Test 7: Max results limit
    - Input: {pattern: "a", path: "/project", max_results: 10}
    - Expected: Returns at most 10 results
    
    Error cases:
    - Pattern not found: returns empty results with message
    - Invalid regex: returns error with explanation
    - Path not accessible: returns error
  </behavior>
  <action>
    Create an advanced file_search tool using grep and find commands with support for regex, context lines, and result ranking.
    
    Enhanced file_search tool schema:
    ```go
    InputSchema: map[string]interface{}{
      "type": "object",
      "properties": map[string]interface{}{
        "pattern": map[string]interface{}{
          "type": "string",
          "description": "Search pattern or regex",
        },
        "path": map[string]interface{}{
          "type": "string",
          "description": "Directory or file to search in",
        },
        "recursive": map[string]interface{}{
          "type": "boolean",
          "description": "Search recursively in subdirectories",
        },
        "use_regex": map[string]interface{}{
          "type": "boolean",
          "description": "Treat pattern as regex",
        },
        "case_insensitive": map[string]interface{}{
          "type": "boolean",
          "description": "Case-insensitive search",
        },
        "invert_match": map[string]interface{}{
          "type": "boolean",
          "description": "Return lines NOT matching pattern",
        },
        "context_lines": map[string]interface{}{
          "type": "number",
          "description": "Number of context lines before/after match",
        },
        "include_pattern": map[string]interface{}{
          "type": "string",
          "description": "Only search files matching glob (e.g., '*.go')",
        },
        "exclude_pattern": map[string]interface{}{
          "type": "string",
          "description": "Exclude files matching glob (e.g., '*.test')",
        },
        "max_results": map[string]interface{}{
          "type": "number",
          "description": "Maximum number of results to return",
        },
        "max_file_size": map[string]interface{}{
          "type": "number",
          "description": "Skip files larger than this (bytes)",
        },
      },
      "required": []string{"pattern", "path"},
    }
    ```
    
    Implementation:
    1. Build grep command with options:
       ```go
       func buildGrepCommand(pattern string, params map[string]interface{}) []string {
         args := []string{"grep"}
         
         if params["recursive"].(bool) {
           args = append(args, "-r")
         }
         
         if params["use_regex"].(bool) {
           args = append(args, "-E") // Extended regex
         } else {
           args = append(args, "-F") // Fixed strings
         }
         
         if params["case_insensitive"].(bool) {
           args = append(args, "-i")
         }
         
         if params["invert_match"].(bool) {
           args = append(args, "-v")
         }
         
         if lines, ok := params["context_lines"].(float64); ok && lines > 0 {
           args = append(args, "-C", fmt.Sprintf("%d", int(lines)))
         } else {
           args = append(args, "-n") // Line numbers always
         }
         
         // Add pattern and path
         args = append(args, pattern, params["path"].(string))
         
         return args
       }
       ```
    
    2. Handle include/exclude patterns with find:
       ```go
       func searchWithFilters(pattern string, path string, include string, exclude string) ([]SearchResult, error) {
         // Use find to get file list, then grep each
         // Or use find ... -exec grep ...
         findCmd := fmt.Sprintf("find %s -type f", path)
         if include != "" {
           findCmd += fmt.Sprintf(" -name '%s'", include)
         }
         if exclude != "" {
           findCmd += fmt.Sprintf(" ! -name '%s'", exclude)
         }
         // ... execute and process results
       }
       ```
    
    3. Result parsing and formatting:
       ```go
       type SearchResult struct {
         Path    string `json:"path"`
         Line    int    `json:"line"`
         Column  int    `json:"column"`
         Content string `json:"content"`
         Context struct {
           Before []string `json:"before"`
           After  []string `json:"after"`
         } `json:"context,omitempty"`
       }
       ```
    
    4. Result ranking (simple heuristic):
       - Exact matches score higher than partial
       - Matches in filename score higher
       - Shorter files score higher (more relevant)
       - Recent modifications score higher
    
    5. Large file handling:
       - Check file size before searching
       - Skip files > max_file_size (default 10MB)
       - For very large files, use head/tail first
    
    6. DisplayContent formatting:
       - Summary: "Found 15 matches in 7 files"
       - Top matches shown in display
       - Full results in Content (structured JSON)
    
    7. Integration with file_read:
       - When agent sees search results, can call file_read on specific lines
       - Return clickable paths that agent can reference
  </action>
  <verify>
    <automated>grep -q "use_regex\|context_lines\|include_pattern\|grep.*-E\|grep.*-r" internal/tools/file_tools.go && echo "Advanced search features implemented"</automated>
  </verify>
  <done>file_search tool supports regex, context lines, include/exclude patterns, case sensitivity, and result limiting</done>
</task>

<task type="auto">
  <name>Task 3: Extend file_stat with detailed metadata</name>
  <files>internal/tools/file_tools.go</files>
  <action>
    Enhance the file_stat tool to provide comprehensive file analysis including type detection, encoding, line counts, and content hashing.
    
    Extended file_stat schema:
    ```go
    InputSchema: map[string]interface{}{
      "type": "object",
      "properties": map[string]interface{}{
        "path": map[string]interface{}{
          "type": "string",
          "description": "Path to the file or directory",
        },
        "compute_hash": map[string]interface{}{
          "type": "boolean",
          "description": "Compute MD5/SHA256 hash of content",
        },
        "detect_encoding": map[string]interface{}{
          "type": "boolean",
          "description": "Detect file encoding (UTF-8, ASCII, etc.)",
        },
        "count_lines": map[string]interface{}{
          "type": "boolean",
          "description": "Count lines, words, characters",
        },
        "detect_language": map[string]interface{}{
          "type": "boolean",
          "description": "Detect programming language from extension and content",
        },
      },
      "required": []string{"path"},
    }
    ```
    
    Implementation:
    1. Use stat command for basic info:
       ```go
       func getBasicStats(path string) (*BasicStats, error) {
         cmd := fmt.Sprintf("stat -c '%%s|%%Y|%%F|%%a|%%U|%%G' %s", path)
         // Parse: size|mtime|type|permissions|owner|group
       }
       ```
    
    2. Line/word/character counting with wc:
       ```go
       func getWordCount(path string) (*WordCount, error) {
         cmd := fmt.Sprintf("wc -lwc %s", path)
         // Parse: lines words bytes filename
       }
       ```
    
    3. File type detection:
       - Extension-based detection (.go, .js, .py, etc.)
       - Content-based detection (shebang, magic numbers)
       - Use file command if available: file -b --mime-type
    
    4. Encoding detection:
       - Check for BOM (UTF-8, UTF-16)
       - Try UTF-8 decoding
       - Check for null bytes (binary)
       - Return "utf-8", "ascii", "binary", "unknown"
    
    5. Hash computation:
       ```go
       func computeHash(path string) (*FileHash, error) {
         md5Cmd := fmt.Sprintf("md5sum %s | cut -d' ' -f1", path)
         shaCmd := fmt.Sprintf("sha256sum %s | cut -d' ' -f1", path)
         // Execute both, return results
       }
       ```
    
    6. Language detection (simple):
       ```go
       func detectLanguage(path string, content []byte) string {
         ext := filepath.Ext(path)
         langMap := map[string]string{
           ".go": "Go",
           ".js": "JavaScript",
           ".ts": "TypeScript",
           ".py": "Python",
           // ... more mappings
         }
         if lang, ok := langMap[ext]; ok {
           return lang
         }
         // Check shebang
         if bytes.HasPrefix(content, []byte("#!/usr/bin/env python")) {
           return "Python"
         }
         return "Unknown"
       }
       ```
    
    7. Directory statistics:
       - Count files, subdirectories
       - Total size
       - File type breakdown
       - Most recently modified
    
    8. Result structure:
       ```go
       type FileStatResult struct {
         Path         string       `json:"path"`
         Type         string       `json:"type"` // file, directory, symlink
         Size         int64        `json:"size"`
         Permissions  string       `json:"permissions"`
         Owner        string       `json:"owner"`
         Group        string       `json:"group"`
         Modified     time.Time    `json:"modified"`
         Language     string       `json:"language,omitempty"`
         Encoding     string       `json:"encoding,omitempty"`
         LineCount    int          `json:"line_count,omitempty"`
         WordCount    int          `json:"word_count,omitempty"`
         ByteCount    int64        `json:"byte_count"`
         MD5          string       `json:"md5,omitempty"`
         SHA256       string       `json:"sha256,omitempty"`
         IsText       bool         `json:"is_text"`
         IsBinary     bool         `json:"is_binary"`
       }
       ```
    
    9. DisplayContent formatting:
       - File: "README.md — 2.4KB, 45 lines, Markdown"
       - Directory: "src/ — 15 files, 3 dirs, 125KB total"
       - With hash: "main.go — 1.2KB, SHA256: a1b2c3..."
  </action>
  <verify>
    <automated>grep -q "detect_language\|compute_hash\|detect_encoding\|wc.*-lwc" internal/tools/file_tools.go && echo "Enhanced file stats implemented"</automated>
  </verify>
  <done>file_stat tool provides comprehensive metadata including language detection, encoding, word counts, and content hashing</done>
</task>

<task type="auto">
  <name>Task 4: Add batch operations support</name>
  <files>internal/tools/file_tools.go</files>
  <action>
    Implement batch operation capabilities that allow the agent to perform multiple file edits or searches in a single tool call.
    
    New tool: batch_edit
    ```go
    InputSchema: map[string]interface{}{
      "type": "object",
      "properties": map[string]interface{}{
        "operations": map[string]interface{}{
          "type": "array",
          "items": map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
              "path": {"type": "string"},
              "operation": {"type": "string", "enum": []string{"replace", "insert", "delete"}},
              // ... operation-specific params
            },
          },
          "description": "List of edit operations to perform",
        },
        "atomic": map[string]interface{}{
          "type": "boolean",
          "description": "All operations succeed or none (transaction)",
        },
        "continue_on_error": map[string]interface{}{
          "type": "boolean",
          "description": "Continue with remaining operations if one fails",
        },
      },
      "required": []string{"operations"},
    }
    ```
    
    Implementation:
    1. Transaction support (atomic mode):
       - Create backup of all affected files
       - Execute all operations
       - If any fail, restore all backups
       - If all succeed, remove backups
    
    2. Batch result structure:
       ```go
       type BatchResult struct {
         TotalOperations int                `json:"total_operations"`
         Successful      int                `json:"successful"`
         Failed          int                `json:"failed"`
         Results         []OperationResult  `json:"results"`
         RolledBack      bool               `json:"rolled_back"`
       }
       
       type OperationResult struct {
         Path    string `json:"path"`
         Success bool   `json:"success"`
         Error   string `json:"error,omitempty"`
         Changes int    `json:"changes,omitempty"` // For replace: number of replacements
       }
       ```
    
    3. Progress tracking:
       - Emit progress events if operation takes >1 second
       - Show "Processing 5/10 files..." in display
    
    4. Safety limits:
       - Max 100 operations per batch
       - Max total size of affected files: 10MB
       - Timeout: 30 seconds
    
    5. DisplayContent formatting:
       - Success: "Batch complete: 8/10 operations successful"
       - With failures: "Batch complete with errors: 2 failed (see details)"
       - Atomic rollback: "Batch failed, all changes rolled back"
    
    6. Integration with existing tools:
       - batch_edit uses file_edit internally
       - batch_search uses file_search internally
       - Maintains same safety and preview features
  </action>
  <verify>
    <automated>grep -q "batch_edit\|operations.*array\|atomic.*bool\|OperationResult" internal/tools/file_tools.go && echo "Batch operations implemented"</automated>
  </verify>
  <done>batch_edit tool supports multiple file operations with transaction support and error handling</done>
</task>

</tasks>

<verification>
After completing all tasks:

1. Build and run WebClaw
2. Test file_edit operations:
   - Replace: "Replace all 'foo' with 'bar' in /test.txt"
   - Insert: "Insert 'header' at line 1 in /test.txt"
   - Delete: "Delete line 5 in /test.txt"
   - Preview: "Preview replacing 'old' with 'new' in /test.txt"
3. Test file_search:
   - "Search for 'func' in /project with regex"
   - "Find all TODO comments in /project"
   - "Search for 'error' in Go files only"
4. Test file_stat:
   - "Get detailed stats for /project/main.go"
   - Verify language detection works
   - Verify word/line counts are accurate
5. Test batch operations:
   - "Replace 'foo' with 'bar' in /a.txt, /b.txt, /c.txt"
6. Verify all operations work via agent loop
</verification>

<success_criteria>
Phase 7a-04 is successful when:

1. **file_edit advanced**: Supports regex replace, insert, delete, append, prepend, replace_lines
2. **Preview mode**: Can show diff without applying changes
3. **file_search advanced**: Supports regex, context lines, include/exclude patterns, case sensitivity
4. **Result ranking**: Most relevant results shown first
5. **file_stat detailed**: Provides language, encoding, hashes, word counts
6. **Batch operations**: Can perform multiple edits atomically
7. **Error handling**: Clear error messages with context
8. **Safety limits**: Large files handled gracefully, operations have timeouts
9. **Agent integration**: All tools work end-to-end with agent loop
10. **Performance**: Operations complete in reasonable time (<5s for typical files)
</success_criteria>

<output>
After completion, create `.planning/phases/07a-justbash-filesystem/07a-04-SUMMARY.md`

Summary should document:
- All file tools and their capabilities
- Sed/awk command patterns used
- Regex support and limitations
- Performance characteristics and safety limits
- Batch operation transaction model
- Error handling approach
- Common use cases and examples
</output>
