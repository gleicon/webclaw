---
phase: 07a-justbash-filesystem
plan: 03
type: execute
wave: 3
depends_on:
  - 07a-02
files_modified:
  - static/justbash-bridge.js
  - internal/jsbridge/justbash.go
  - internal/tools/file_tools.go
  - static/filesystem-ui.js
  - index.html
autonomous: false
requirements:
  - BRIDGE-03
  - BRIDGE-04
  - TOOL-01
must_haves:
  truths:
    - "User can mount local project directory via File System Access API"
    - "OverlayFs allows reading real files while writing to memory"
    - "Virtual and real files are visually distinguished in UI"
    - "Smart routing selects just-bash when bridge unavailable"
    - "Mount configuration persists across sessions"
    - "Preview mode shows what changes would be made to real files"
  artifacts:
    - path: "static/justbash-bridge.js"
      provides: "OverlayFs mount support"
      exports: ["mountDirectory", "unmountDirectory", "getMountPoints"]
    - path: "internal/jsbridge/justbash.go"
      provides: "Mount management bindings"
      exports: ["MountOverlay", "Unmount", "GetMounts", "IsMounted"]
    - path: "static/filesystem-ui.js"
      provides: "Mount dialog and indicators"
      exports: ["showMountDialog", "updateMountIndicators"]
    - path: "index.html"
      provides: "Mount dialog UI"
      contains: "mount-dialog, mount-button"
  key_links:
    - from: "filesystem-ui.js"
      to: "File System Access API"
      via: "showDirectoryPicker"
      pattern: "showDirectoryPicker"
    - from: "justbash-bridge.js"
      to: "OverlayFs"
      via: "createOverlayFs"
      pattern: "OverlayFs"
    - from: "file_tools.go"
      to: "justbash mounts"
      via: "CheckMountStatus"
      pattern: "IsMounted\|GetMounts"
user_setup: []
---

<objective>
Implement OverlayFs mount points that allow WebClaw to read from the user's real project files while keeping all writes in memory. This enables safe "preview mode" where the agent can see real codebases but changes don't affect the actual files until explicitly applied.

Purpose: Bridge the gap between the isolated virtual filesystem and real user projects. Users can mount their actual project directories, allowing the agent to read and analyze their code, while providing a safe sandbox for the agent to propose and preview changes.

Output:
- File System Access API integration for directory mounting
- OverlayFs implementation in just-bash (read real, write virtual)
- Mount management UI with mount/unmount controls
- Visual indicators for virtual vs real files
- Smart routing that prefers just-bash for reads when bridge unavailable
- Mount persistence across sessions (optional, via IndexedDB)
</objective>

<execution_context>
@/Users/gleicon/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/gleicon/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/07a-justbash-filesystem/07a-01-SUMMARY.md
@.planning/phases/07a-justbash-filesystem/07a-02-SUMMARY.md
@static/justbash-bridge.js
@static/filesystem-ui.js
@index.html

## File System Access API Reference

Chrome/Edge support: showDirectoryPicker(), showOpenFilePicker(), showSaveFilePicker()

```javascript
// Request directory access
const dirHandle = await window.showDirectoryPicker();

// Read file
const fileHandle = await dirHandle.getFileHandle('filename.txt');
const file = await fileHandle.getFile();
const content = await file.text();

// Write file (requires user gesture)
const newHandle = await dirHandle.getFileHandle('newfile.txt', { create: true });
const writable = await newHandle.createWritable();
await writable.write(content);
await writable.close();

// Check permission
const permission = await dirHandle.queryPermission({ mode: 'read' });
// 'granted', 'prompt', or 'denied'

// Request permission
await dirHandle.requestPermission({ mode: 'readwrite' });
```

## just-bash OverlayFs API (expected)

```typescript
interface OverlayFs {
  // Mount a real directory handle
  mount(mountPoint: string, dirHandle: FileSystemDirectoryHandle): void;
  
  // Unmount
  unmount(mountPoint: string): void;
  
  // Read file (checks overlay first, then real)
  readFile(path: string): Promise<string>;
  
  // Write file (always to overlay layer)
  writeFile(path: string, content: string): Promise<void>;
  
  // List directory (merges overlay and real)
  readdir(path: string): Promise<DirEntry[]>;
}
```
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add OverlayFs support to just-bash bridge</name>
  <files>static/justbash-bridge.js</files>
  <action>
    Extend the just-bash JavaScript bridge to support OverlayFs mounting using the File System Access API. This allows users to mount their local project directories for agent access.
    
    Implementation:
    1. Add mount state management:
       ```javascript
       const mountState = {
         mounts: new Map(), // path -> {handle, mode, name}
         overlayFs: null    // just-bash OverlayFs instance
       };
       ```
    
    2. Implement mountDirectory(virtualPath, dirHandle, mode):
       - mode: 'read' (read real, write overlay) or 'readonly' (read only)
       - Store the FileSystemDirectoryHandle
       - Create just-bash OverlayFs instance
       - Set up path resolution: virtual paths map to real paths
       - Return mount info object
    
    3. Implement unmountDirectory(virtualPath):
       - Remove mount from state
       - Clean up just-bash overlay
       - Return success/failure
    
    4. Implement getMountPoints():
       - Return array of {path, name, mode, active}
    
    5. Extend existing operations to use OverlayFs:
       - readFile: Check mounts first, use handle.getFile() for real files
       - writeFile: Always write to overlay layer (in-memory)
       - listDir: Merge real directory contents with overlay changes
       - fileExists: Check both overlay and real
    
    6. Path resolution:
       ```javascript
       function resolvePath(virtualPath) {
         // Find mount point that matches path prefix
         for (const [mountPath, mount] of mountState.mounts) {
           if (virtualPath.startsWith(mountPath)) {
             const relativePath = virtualPath.slice(mountPath.length);
             return { mount, relativePath };
           }
         }
         return null; // Not in any mount
       }
       ```
    
    7. File System Access API integration:
       ```javascript
       async function readRealFile(mount, relativePath) {
         const parts = relativePath.split('/').filter(p => p);
         let currentHandle = mount.handle;
         
         for (const part of parts) {
           currentHandle = await currentHandle.getDirectoryHandle(part);
         }
         
         const fileHandle = await currentHandle.getFileHandle(parts[parts.length - 1]);
         const file = await fileHandle.getFile();
         return await file.text();
       }
       ```
    
    8. Permission handling:
       - Check permissions on mount
       - Request permissions if needed
       - Handle permission denied gracefully
    
    9. Update window.justbash API:
       - mountDirectory(path, handle, mode) -> Promise<MountInfo>
       - unmountDirectory(path) -> Promise<void>
       - getMountPoints() -> Promise<MountInfo[]>
       - isPathVirtual(path) -> boolean
       - getFileSource(path) -> 'virtual'|'real'|'overlay'
    
    Security notes:
    - File System Access API requires secure context (HTTPS or localhost)
    - User must explicitly select directory via picker
    - Permissions can be revoked by user at any time
    - Store handles in memory only (not persisted)
  </action>
  <verify>
    <automated>grep -q "mountDirectory\|unmountDirectory\|getMountPoints\|OverlayFs" static/justbash-bridge.js | wc -l | xargs test {} -ge 3 && echo "OverlayFs mount support added"</automated>
  </verify>
  <done>static/justbash-bridge.js includes mountDirectory, unmountDirectory, getMountPoints functions with File System Access API integration</done>
</task>

<task type="auto">
  <name>Task 2: Add mount management UI</name>
  <files>index.html, static/filesystem-ui.js</files>
  <action>
    Create a mount management dialog and UI controls for mounting/unmounting directories.
    
    Changes to index.html:
    1. Add mount button to filesystem toolbar:
       ```html
       <button id="fs-mount-btn" class="p-2 rounded hover:bg-gray-700 text-green-400" title="Mount Directory">🔗</button>
       ```
    
    2. Add mount dialog (hidden by default):
       ```html
       <div id="mount-dialog" class="hidden fixed inset-0 bg-black/50 z-50 flex items-center justify-center">
         <div class="bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4">
           <div class="px-4 py-3 border-b border-gray-700 flex justify-between items-center">
             <h3 class="text-sm font-semibold text-gray-200">Mount Directory</h3>
             <button id="mount-close" class="text-gray-400 hover:text-gray-200">✕</button>
           </div>
           <div class="p-4 space-y-4">
             <div>
               <label class="block text-xs text-gray-400 mb-1">Virtual Mount Point</label>
               <input id="mount-path" type="text" value="/project" 
                 class="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100"
                 placeholder="/project">
               <p class="text-xs text-gray-500 mt-1">Where to mount in virtual filesystem</p>
             </div>
             <div>
               <label class="block text-xs text-gray-400 mb-1">Access Mode</label>
               <select id="mount-mode" class="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100">
                 <option value="read">Read + Write Overlay (Safe Preview)</option>
                 <option value="readonly">Read Only</option>
               </select>
               <p class="text-xs text-gray-500 mt-1">
                 <span id="mode-read-help">Changes are stored in memory and don't affect real files</span>
                 <span id="mode-readonly-help" class="hidden">Can only read files, no modifications allowed</span>
               </p>
             </div>
             <div class="flex gap-2">
               <button id="mount-select-btn" class="flex-1 bg-indigo-600 hover:bg-indigo-500 text-white rounded px-4 py-2 text-sm">
                 Select Directory...
               </button>
             </div>
             <div id="mount-error" class="hidden p-3 bg-red-900/50 border border-red-700 rounded text-xs text-red-200"></div>
           </div>
           <div class="px-4 py-3 border-t border-gray-700 flex justify-end gap-2">
             <button id="mount-cancel" class="px-4 py-2 text-sm text-gray-400 hover:text-gray-200">Cancel</button>
             <button id="mount-confirm" class="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-sm" disabled>
               Mount
             </button>
           </div>
         </div>
       </div>
       ```
    
    3. Add mount indicators to file tree header:
       ```html
       <div id="fs-mount-list" class="mt-2 space-y-1">
         <!-- Dynamic mount point badges -->
       </div>
       ```
    
    Changes to filesystem-ui.js:
    1. Add mount dialog functions:
       - showMountDialog() - Opens dialog, resets state
       - hideMountDialog() - Closes dialog
       - handleSelectDirectory() - Calls showDirectoryPicker()
       - handleMount() - Validates and creates mount
       - handleUnmount(path) - Removes mount
    
    2. Directory picker integration:
       ```javascript
       async function handleSelectDirectory() {
         try {
           const dirHandle = await window.showDirectoryPicker();
           mountState.selectedHandle = dirHandle;
           mountState.selectedName = dirHandle.name;
           updateMountDialogUI();
         } catch (err) {
           if (err.name === 'AbortError') {
             // User cancelled, no action needed
             return;
           }
           showMountError('Failed to select directory: ' + err.message);
         }
       }
       ```
    
    3. Mount list rendering:
       - Show active mounts in file tree header
       - Each mount shows: path, directory name, mode badge
       - Unmount button for each mount
       - Click to jump to mount point in tree
    
    4. Visual indicators:
       - Files from real filesystem: different icon or badge
       - Virtual-only files: standard icon
       - Modified files (overlay): highlight or indicator
       - Read-only mounts: lock icon
    
    5. Error handling:
       - Permission denied: show helpful message
       - Directory not accessible: explain security restrictions
       - Already mounted: prevent duplicate mounts
       - Invalid path: validate mount point format
  </action>
  <verify>
    <automated>grep -q "mount-dialog\|mountDirectory\|showDirectoryPicker" index.html static/filesystem-ui.js 2>/dev/null | wc -l | xargs test {} -ge 2 && echo "Mount dialog UI added"</automated>
  </verify>
  <done>index.html includes mount dialog and mount button; filesystem-ui.js has mount dialog handling and directory picker integration</done>
</task>

<task type="auto">
  <name>Task 3: Implement smart routing for file operations</name>
  <files>internal/jsbridge/justbash.go, internal/tools/file_tools.go</files>
  <action>
    Implement smart routing that automatically selects the appropriate filesystem backend (just-bash vs bridge) based on availability and mount status.
    
    Implementation:
    1. Create filesystem router in justbash.go:
       ```go
       type FilesystemRouter struct {
         justBashAvailable bool
         bridgeAvailable    bool
         preferredBackend   string // "justbash" or "bridge"
       }
       
       func NewFilesystemRouter() *FilesystemRouter {
         return &FilesystemRouter{
           justBashAvailable: JustBashAvailable(),
           bridgeAvailable:   BridgeAvailable(), // Check if bridge is connected
           preferredBackend:  "justbash", // Default to just-bash
         }
       }
       ```
    
    2. Backend detection:
       ```go
       func BridgeAvailable() bool {
         // Check if bridge WebSocket is connected
         bridge := js.Global().Get("webclaw").Get("bridge")
         return !bridge.IsUndefined() && bridge.Get("connected").Bool()
       }
       ```
    
    3. Routing logic for each operation:
       ```go
       func (fr *FilesystemRouter) ReadFile(path string) (string, error) {
         // Check if path is in a just-bash mount
         if fr.isPathInJustBashMount(path) {
           return justBashReadFile(path)
         }
         
         // Check if bridge is available and preferred
         if fr.bridgeAvailable && fr.preferredBackend == "bridge" {
           return bridgeReadFile(path)
         }
         
         // Fall back to just-bash if available
         if fr.justBashAvailable {
           return justBashReadFile(path)
         }
         
         return "", fmt.Errorf("no filesystem backend available")
       }
       ```
    
    4. Path classification:
       ```go
       func (fr *FilesystemRouter) ClassifyPath(path string) PathInfo {
         return PathInfo{
           InJustBashMount: fr.isPathInJustBashMount(path),
           InBridgeMount:   fr.isPathInBridgeMount(path),
           IsVirtualOnly:   fr.isVirtualPath(path),
           RecommendedBackend: fr.recommendBackend(path),
         }
       }
       ```
    
    5. Update file tools to use router:
       ```go
       func NewFileReadTool(router *FilesystemRouter) *Tool {
         return &Tool{
           Name: "file_read",
           // ...
           Execute: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
             path := params["path"].(string)
             content, err := router.ReadFile(path)
             // ...
           },
         }
       }
       ```
    
    6. Backend preference configuration:
       - Allow setting preferred backend via config
       - Store preference in IndexedDB
       - UI toggle: "Prefer bridge for file operations" (when bridge available)
    
    7. Mount point coordination:
       - When bridge is connected, it may have its own mounts
       - just-bash mounts are separate from bridge mounts
       - Show both sets of mounts in UI
       - Allow user to choose which mount to use for operations
    
    8. Transparent failover:
       - If just-bash fails, try bridge (if available)
       - If bridge fails, try just-bash (if available)
       - Log which backend was used for debugging
    
    Note: For Phase 7a, bridge availability is always false (not implemented yet). The router should gracefully handle this and use just-bash exclusively.
  </action>
  <verify>
    <automated>grep -q "FilesystemRouter\|preferredBackend\|BridgeAvailable" internal/jsbridge/justbash.go | wc -l | xargs test {} -ge 2 && echo "Smart routing implemented"</automated>
  </verify>
  <done>internal/jsbridge/justbash.go includes FilesystemRouter with backend detection, routing logic, and transparent failover between just-bash and bridge</done>
</task>

<task type="auto">
  <name>Task 4: Add virtual/real file indicators to UI</name>
  <files>static/filesystem-ui.js, static/justbash-bridge.js</files>
  <action>
    Implement visual indicators that distinguish between virtual (in-memory), real (mounted), and modified (overlay) files in the filesystem tree.
    
    Implementation:
    1. Extend justbash-bridge.js to track file sources:
       ```javascript
       async function getFileMetadata(path) {
         const info = await this.getFileInfo(path);
         const source = await this.getFileSource(path);
         // source: 'virtual' (in-memory only), 'real' (from mount), 'overlay' (modified real file)
         return { ...info, source };
       }
       ```
    
    2. Update file tree rendering in filesystem-ui.js:
       ```javascript
       function renderFileItem(entry, level) {
         const indent = level * 16;
         const isReal = entry.source === 'real';
         const isOverlay = entry.source === 'overlay';
         const isVirtual = entry.source === 'virtual';
         
         let icon, badge, tooltip;
         if (entry.type === 'directory') {
           icon = '📁';
         } else if (isReal) {
           icon = '📄';
           badge = '<span class="ml-1 text-xs text-green-400" title="Real file (from mount)">●</span>';
         } else if (isOverlay) {
           icon = '📝';
           badge = '<span class="ml-1 text-xs text-yellow-400" title="Modified (unsaved to real file)">●</span>';
         } else {
           icon = '📄';
           badge = '<span class="ml-1 text-xs text-gray-500" title="Virtual file (in-memory only)">○</span>';
         }
         
         return `
           <div class="flex items-center py-1 hover:bg-gray-800 cursor-pointer" 
                style="padding-left: ${indent}px"
                data-path="${entry.path}">
             <span class="mr-1">${icon}</span>
             <span class="text-sm ${isOverlay ? 'text-yellow-300' : 'text-gray-300'}">${entry.name}</span>
             ${badge}
           </div>
         `;
       }
       ```
    
    3. Add legend/explanation:
       - Small legend in file tree header explaining indicators
       - Tooltip on hover showing file source
       - Status bar shows source of selected file
    
    4. Overlay change tracking:
       - Track which files have been modified in overlay
       - Show "modified count" badge on mount point
       - Highlight directories containing modified files
    
    5. "Apply Changes" feature (when bridge available):
       - Button to sync overlay changes to real files
       - Confirmation dialog showing list of changes
       - Progress indicator during sync
       - Success/error feedback
       
       Note: For Phase 7a, this is a stub that shows "Bridge not connected" message
    
    6. Filter/sort options:
       - Filter to show only modified files
       - Filter to show only real files
       - Sort by source (real first, then virtual)
    
    Styling:
    - Real files: green dot, normal text
    - Overlay/modified: yellow dot, yellow text
    - Virtual: gray circle, normal text
    - Hover effects for all items
    - Selected file: highlighted background
  </action>
  <verify>
    <automated>grep -q "source.*real\|source.*virtual\|source.*overlay\|getFileSource\|isReal\|isOverlay" static/filesystem-ui.js static/justbash-bridge.js 2>/dev/null | wc -l | xargs test {} -ge 3 && echo "File source indicators added"</automated>
  </verify>
  <done>filesystem-ui.js shows visual indicators for real (green), overlay/modified (yellow), and virtual (gray) files with tooltips and legend</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <what-built>
    OverlayFs mount system with:
    - File System Access API integration for directory mounting
    - Mount dialog with path configuration and mode selection
    - Visual indicators distinguishing real vs virtual vs modified files
    - Smart routing that prefers available backends
    - Mount list in file tree header with unmount controls
  </what-built>
  <how-to-verify>
    Prerequisites: Use Chrome/Edge with File System Access API support (localhost or HTTPS)
    
    1. Test mount dialog:
       - Open Filesystem tab
       - Click 🔗 (mount) button in toolbar
       - Verify mount dialog opens with:
         - Virtual Mount Point field (default: /project)
         - Access Mode dropdown (Read + Write Overlay / Read Only)
         - Select Directory button
         - Cancel and Mount buttons
    
    2. Test directory mounting:
       - Click "Select Directory..."
       - Choose a local project directory
       - Verify directory name appears in dialog
       - Click Mount
       - Verify dialog closes
       - Verify mount appears in file tree header
    
    3. Test mounted directory browsing:
       - Verify mounted files appear in tree
       - Real files should show green ● indicator
       - Double-click directories to navigate
       - Click files to open in editor
       - Verify file content loads correctly
    
    4. Test safe preview mode:
       - Open a real file from mount
       - Modify content in editor
       - Save file
       - Verify file shows yellow ● (modified/overlay)
       - Verify real file on disk is NOT changed
       - Verify virtual filesystem has the changes
    
    5. Test unmounting:
       - Click unmount button (✕) next to mount in header
       - Verify mount disappears from tree
       - Verify files are no longer accessible
    
    6. Test read-only mount:
       - Mount directory with "Read Only" mode
       - Try to modify and save a file
       - Verify error message appears
       - Verify file remains unchanged
    
    7. Test agent integration:
       - With mount active, go to Chat tab
       - Ask: "Read the file /project/README.md"
       - Verify agent successfully reads the file
       - Verify file content is correct
    
    8. Expected behavior:
       - Mount dialog works smoothly
       - File System Access API permission prompt appears
       - Real files load quickly
       - Modified files clearly marked
       - Unmounting removes all mount content from view
  </how-to-verify>
  <resume-signal>
    Type "approved" if all tests pass, or describe any issues found.
  </resume-signal>
</task>

</tasks>

<verification>
After completing all tasks:

1. Build and run WebClaw
2. Open browser with Chrome/Edge (File System Access API required)
3. Open Filesystem tab
4. Click mount button
5. Select a local directory
6. Verify files appear in tree with green indicators
7. Open and modify a file
8. Save and verify yellow indicator appears
9. Unmount and verify files disappear
10. Test with agent: "List files in /project"
11. Verify smart routing works (uses just-bash when bridge unavailable)
</verification>

<success_criteria>
Phase 7a-03 is successful when:

1. **Mount dialog works**: Can open dialog, select directory, configure options
2. **File System Access API**: Permission prompt appears, directory handle obtained
3. **Directory mounting**: Mounted files appear in file tree
4. **OverlayFs functional**: Can read real files, writes go to overlay
5. **Visual indicators**: Real (green), modified (yellow), virtual (gray) clearly shown
6. **Safe preview**: Changes don't affect real files until explicitly applied
7. **Unmount works**: Can remove mount, files disappear from tree
8. **Read-only mode**: Can mount read-only, writes are blocked
9. **Agent integration**: Agent can read files from mounted directories
10. **Smart routing**: Automatically uses available backend (just-bash in Phase 7a)
</success_criteria>

<output>
After completion, create `.planning/phases/07a-justbash-filesystem/07a-03-SUMMARY.md`

Summary should document:
- OverlayFs architecture: real layer + overlay layer
- File System Access API requirements and browser support
- Mount configuration options and their meanings
- Smart routing algorithm and backend preference
- Security model: user-initiated permission, no automatic access
- Performance characteristics: lazy loading, caching strategy
- How to use preview mode effectively
- Differences between virtual, real, and overlay files
</output>
