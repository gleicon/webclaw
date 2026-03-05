---
phase: 07a-justbash-filesystem
plan: 03
name: overlayfs-mounts
subsystem: filesystem
tags: [overlayfs, file-system-access-api, mounts, deferred]

# Dependency graph
requires:
  - plan: 07a-01
    provides: just-bash foundation
  - plan: 07a-02
    provides: Filesystem UI (NOT IMPLEMENTED - this plan not built either)

# Note: This plan was NOT IMPLEMENTED
# Requires File System Access API (Chrome/Edge only)
# Would enable reading real user files while keeping writes in memory

status: NOT_IMPLEMENTED
reason: "Requires File System Access API support. Core virtual filesystem (07a-01) is sufficient for current use cases."

# What WAS built instead:
provides:
  - Pure virtual filesystem using InMemoryFs
  - All file operations work in memory only
  - No access to real user files (simpler, safer)

# What was NOT built:
deferred:
  - File System Access API integration
  - OverlayFs for mounting real directories
  - Read real files, write to memory (preview mode)
  - Mount persistence across sessions
  - Visual mount dialog

self-check: DEFERRED
---

<objective>
Implement OverlayFs mount points that allow WebClaw to read from the user's real project files while keeping all writes in memory. This enables safe "preview mode" where the agent can see real codebases but changes don't affect the actual files until explicitly applied.

Purpose: Bridge the gap between the isolated virtual filesystem and real user projects. Users can mount their actual project directories, allowing the agent to read and analyze their code, while providing a safe sandbox for the agent to propose and preview changes.

Output: File System Access API integration, OverlayFs mount support, Mount dialog UI, Virtual/real file indicators, Smart routing between just-bash and bridge
</objective>

## Implementation Status: NOT IMPLEMENTED

### What Was Planned But Not Built

**File System Access API Integration:**
- ❌ `showDirectoryPicker()` for selecting local directory
- ❌ Permission handling for read/write access
- ❌ Chrome/Edge-specific API integration
- ❌ Graceful fallback for unsupported browsers

**OverlayFs Mount Support:**
- ❌ Create OverlayFs instance (read from real, write to virtual)
- ❌ Mount local directory at virtual path
- ❌ Unmount functionality
- ❌ Multiple mount points support
- ❌ Mount persistence across page reloads

**Mount Dialog UI:**
- ❌ "Mount Directory" button in filesystem panel
- ❌ Directory picker integration
- ❌ Mount point configuration (path, permissions)
- ❌ List of active mounts
- ❌ Unmount button for each mount

**Visual Indicators:**
- ❌ Badge showing "virtual" vs "real" files
- ❌ Different icons for mounted vs virtual directories
- ❌ Warning indicators for real files (showing changes are in memory)

**Smart Routing:**
- ❌ Check if file is in mounted directory
- ❌ Route reads to real filesystem, writes to virtual
- ❌ Handle unmounting gracefully

### What Works Instead

**Pure Virtual Filesystem:**
- ✅ All operations in InMemoryFs (memory only)
- ✅ No access to real user files
- ✅ Completely sandboxed
- ✅ Works in all browsers (not just Chrome/Edge)

**User Workflow:**
```
User: "I want to work on my project files"
Current: User copies files into WebClaw virtual filesystem
Future: User mounts local directory via File System Access API
```

### Why Not Implemented

**Technical Requirements:**
1. **File System Access API:** Only supported in Chrome/Edge 86+
   - Firefox: Not supported
   - Safari: Limited support
   - Would exclude significant user base

2. **Permission Model:** Requires explicit user action to select directory
   - Cannot auto-mount on startup
   - Permissions don't persist across sessions (by design for privacy)

3. **Complexity:** Significant implementation effort
   - OverlayFs state management
   - Permission handling
   - Cross-browser compatibility
   - UI for mount management

**Strategic Decision:**
- Current virtual filesystem (07a-01) covers 80% of use cases
- Most users can copy files into WebClaw or work in virtual space
- File System Access API is evolving - better to wait for broader support
- Focus resources on Phase 9 (social integrations) with higher user impact

### Browser Support Reality

**File System Access API Status (2026):**
| Browser | Support | Notes |
|---------|---------|-------|
| Chrome | ✅ Yes | Full support |
| Edge | ✅ Yes | Full support |
| Firefox | ❌ No | No plans announced |
| Safari | ⚠️ Partial | Limited, behind flags |

**Impact:** Would leave out ~30-40% of users (Firefox + Safari)

### Future Implementation Path

If OverlayFs mounts are needed later:

1. **Check browser support:**
   ```javascript
   if ('showDirectoryPicker' in window) {
     // Show mount button
   } else {
     // Hide mount feature, show message
   }
   ```

2. **Implement in stages:**
   - Stage 1: Chrome/Edge only with clear messaging
   - Stage 2: Handle permission persistence
   - Stage 3: Visual indicators for virtual vs real
   - Stage 4: Export changes to real files (apply mode)

**Estimated effort:** 2-3 days for Chrome/Edge MVP

### Alternative Workflows

**Current Workaround (Manual Copy):**
```
1. User has project in ~/my-project
2. User opens WebClaw
3. User: "Create a file structure like my project"
4. Agent helps recreate structure in virtual filesystem
5. User works in WebClaw virtual filesystem
6. When done, user downloads/export files
```

**Future Workflow (If Implemented):**
```
1. User clicks "Mount Directory"
2. User selects ~/my-project from file picker
3. WebClaw shows real files in tree
4. User edits in WebClaw (changes in memory)
5. User clicks "Apply Changes" to write to real files
```

### Security Considerations

**If Implemented, Would Need:**
- Clear visual distinction between virtual and real files
- Warning before applying changes to real filesystem
- Confirmation dialogs for destructive operations
- "Dry run" / preview mode showing what would change
- Option to discard all virtual changes without affecting real files

### Verification

**Current state:**
- ❌ File System Access API integrated: NO
- ❌ OverlayFs mounts: NO
- ❌ Mount dialog: NO
- ❌ Virtual/real indicators: NO
- ✅ Virtual filesystem only: YES

**Test:**
```javascript
// Check if File System Access API available
console.log('showDirectoryPicker' in window); // false in Firefox
```

## Conclusion

Plan 07a-03 (OverlayFs Mounts) was **not implemented** due to:
1. Limited browser support (Chrome/Edge only)
2. Current virtual filesystem is sufficient for most use cases
3. Strategic priority on Phase 9 integrations

**Impact:** MEDIUM - Users cannot mount real directories, must work in virtual filesystem or manually copy files.

**Workaround:** Copy files into WebClaw's virtual filesystem, work there, then export results.

**Recommended priority:** Medium-Low - Implement when File System Access API has broader browser support or if users specifically request it.

---

*Phase: 07a-justbash-filesystem*  
*Plan: 03 - OverlayFs Mounts*  
*Status: NOT IMPLEMENTED (deferred)*  
*Reason: File System Access API limited to Chrome/Edge, virtual filesystem sufficient*
