---
phase: 07a-justbash-filesystem
plan: 04
name: advanced-file-tools
subsystem: tools
tags: [file-edit, sed, awk, advanced-search, partial]

# Dependency graph
requires:
  - plan: 07a-01
    provides: Basic file tools (file_read, file_write, dir_list, file_search)

# Note: This plan was PARTIALLY IMPLEMENTED
# Basic file tools work, but advanced features (sed/awk editing, file_stat) not built

status: PARTIALLY_IMPLEMENTED
reason: "Basic file tools sufficient for current use. Advanced editing via sed/awk deferred."

# What WAS built (from 07a-01):
provides:
  - file_read: Read file contents with offset/limit
  - file_write: Write/create files with append support
  - dir_list: List directories with file info
  - file_search: Search with pattern matching

# What was NOT built:
deferred:
  - file_edit: Advanced editing (replace, insert, delete via sed/awk)
  - file_stat: Comprehensive file metadata
  - Batch operations for multi-file edits
  - Large file streaming support
  - Edit preview mode

self-check: PARTIAL
---

<objective>
Implement advanced file manipulation tools including sophisticated edit operations (sed/awk-based), powerful search capabilities (grep with regex), and comprehensive file analysis. These tools provide the agent with professional-grade file editing capabilities within the safe virtual filesystem.

Purpose: Enable the agent to perform complex file operations like find-and-replace, line insertion/deletion, multi-file search, and detailed file analysis. These capabilities are essential for code refactoring, content analysis, and project-wide modifications.

Output: file_edit tool with sed/awk operations, file_search tool with regex and recursive search, file_stat with metadata, Batch operations support, Large file handling, Preview mode for edits
</objective>

## Implementation Status: PARTIALLY IMPLEMENTED

### What Was Built (In 07a-01)

**Basic File Tools (Working):**
- ✅ `file_read` - Read files with offset and limit
- ✅ `file_write` - Write/create files with append mode
- ✅ `dir_list` - List directories with permissions, size, date
- ✅ `file_search` - Basic pattern search

**Capabilities:**
```
file_read(path="main.go", offset=10, limit=20)     → Read lines 10-30
file_write(path="test.txt", content="Hello")       → Create/write file
dir_list(path="src", recursive=true)                → List all files
file_search(pattern="TODO", path=".", recursive=true) → Find TODOs
```

### What Was Planned But Not Built

**file_edit Tool (Advanced Editing):**
- ❌ Replace operations: `sed -i 's/old/new/g'`
- ❌ Insert lines: `sed -i '10i\New line'`
- ❌ Delete lines: `sed -i '5d'`
- ❌ Append to file: `echo "text" >> file`
- ❌ Multi-line edits via awk
- ❌ Preview mode (show changes before applying)

**Enhanced file_search:**
- ❌ Regex pattern support (currently basic string matching)
- ❌ Result ranking by relevance
- ❌ File type filtering (e.g., only .go files)
- ❌ Context lines (show N lines around match)

**file_stat Tool (Metadata):**
- ❌ Detailed file info: size, permissions, modified date
- ❌ File type detection (text, binary, image)
- ❌ Encoding detection (UTF-8, ASCII, etc.)
- ❌ Checksum (MD5, SHA256)

**Batch Operations:**
- ❌ Multi-file find-and-replace
- ❌ Directory-wide operations
- ❌ Operation queuing
- ❌ Undo/redo support

**Large File Handling:**
- ❌ Streaming read for files >10MB
- ❌ Chunked operations
- ❌ Progress indicators

### What Works Now vs. What Would Work

**Current (Basic):**
```
User: "Replace all TODO with FIXME in main.go"
Current workflow:
1. file_read path="main.go" → Get content
2. (Agent mentally processes)
3. file_write with replaced content
```

**Planned (Advanced):**
```
User: "Replace all TODO with FIXME in main.go"
Planned workflow:
1. file_edit path="main.go" operation="replace" pattern="TODO" replacement="FIXME"
2. (Uses sed/awk efficiently)
3. Preview changes, then apply
```

**Impact:** Basic workflow works but requires reading entire file into LLM context, processing, then writing back. Less efficient for large files.

### Technical Gap

**What's Missing in Code:**

```go
// NOT IMPLEMENTED:
func NewFileEditTool() *Tool {
  // Would support:
  // - operation: "replace", "insert", "delete", "append"
  // - Uses sed/awk via just-bash
}

func NewFileStatTool() *Tool {
  // Would provide:
  // - Detailed metadata
  // - File type detection
  // - Checksums
}
```

**Current just-bash bridge has:**
- ✅ ExecuteCommand (can run any bash command)
- ✅ ReadFile, WriteFile

**Could leverage:**
```javascript
// Current bridge can already do:
justbash.executeCommand("sed -i 's/TODO/FIXME/g' file.txt")
justbash.executeCommand("wc -l file.txt")
justbash.executeCommand("file file.txt")  // type detection
```

**But not exposed as user-friendly tools.**

### Workaround Using Existing Tools

Users CAN still do advanced operations through the agent:

**Find and replace:**
```
User: "Replace all occurrences of 'foo' with 'bar' in main.go"
Agent workflow:
1. file_read path="main.go"
2. Process: replace "foo" → "bar"
3. file_write path="main.go" content="<modified>"
```

**Multi-file operations:**
```
User: "Update copyright year in all .go files"
Agent workflow:
1. dir_list to find all .go files
2. For each file:
   - file_read
   - Replace copyright year
   - file_write
```

**Limitation:** Reads entire files into LLM context. Works for files <100KB but not efficient for large files (>1MB).

### When Advanced Tools Would Help

**Large file editing (>1MB):**
- Current: Must read entire file into context
- Advanced: Use sed/awk without loading into LLM

**Complex regex operations:**
- Current: Basic string matching
- Advanced: Full regex with capture groups

**File analysis:**
- Current: No file metadata
- Advanced: file_stat shows size, type, encoding

### Implementation Effort

**To add missing tools:**

1. **file_edit tool** (~100 lines):
   ```go
   func NewFileEditTool() *Tool {
     return &Tool{
       Name: "file_edit",
       Execute: func(params) {
         // Parse operation type
         // Call justbash.ExecuteCommand with sed/awk
         // Return result
       }
     }
   }
   ```

2. **file_stat tool** (~80 lines):
   ```go
   func NewFileStatTool() *Tool {
     // Use stat, file, wc commands
     // Return structured metadata
   }
   ```

3. **Enhanced file_search** (~50 lines):
   - Add regex support
   - Add context lines

**Total effort:** ~4-6 hours

### Decision Rationale

**Deferred because:**
1. Basic tools cover 80% of use cases
2. Agent can work around limitations (read→modify→write)
3. Focus on Phase 9 integrations (higher user value)
4. Can add later without breaking changes
5. Current tools sufficient for WebClaw's core use case (AI chat + file operations)

### Verification

**Current state:**
- ✅ file_read: YES (basic)
- ✅ file_write: YES (basic)
- ✅ dir_list: YES (basic)
- ✅ file_search: YES (basic)
- ❌ file_edit: NO
- ❌ file_stat: NO
- ❌ Regex search: NO

**Test:**
```
User: "Edit main.go to replace all TODO with FIXME"
Current: Agent uses file_read + file_write (works) ✓
Planned: Would use file_edit with sed (not implemented)
```

## Conclusion

Plan 07a-04 (Advanced File Tools) was **partially implemented**:
- ✅ Basic file tools work (read, write, list, search)
- ❌ Advanced editing (sed/awk) not built
- ❌ file_stat metadata tool not built
- ❌ Regex search not implemented

**Impact:** LOW-MEDIUM - Users can still perform all file operations, just less efficiently for large files or complex edits.

**Workaround:** Agent uses read→modify→write pattern.

**Recommended priority:** Low - Can enhance tools later if users need advanced editing features.

---

*Phase: 07a-justbash-filesystem*  
*Plan: 04 - Advanced File Tools*  
*Status: PARTIALLY IMPLEMENTED*  
*What works: Basic file_read, file_write, dir_list, file_search*  
*What's missing: file_edit (sed/awk), file_stat, regex search*
