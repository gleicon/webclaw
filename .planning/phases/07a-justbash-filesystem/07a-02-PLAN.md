---
phase: 07a-justbash-filesystem
plan: 02
type: execute
wave: 2
depends_on:
  - 07a-01
files_modified:
  - index.html
  - static/filesystem-ui.js
  - internal/jsbridge/justbash.go
autonomous: false
requirements:
  - UI-01
  - UI-02
  - UI-03
  - UI-04
  - UI-05
must_haves:
  truths:
    - "User can see virtual filesystem tree in sidebar panel"
    - "User can navigate directories by clicking folders"
    - "User can view file contents in built-in editor"
    - "User can create new files and directories"
    - "User can delete files and directories"
    - "Virtual/real file indicators are clearly visible"
    - "Filesystem operations show loading states and errors"
  artifacts:
    - path: "index.html"
      provides: "Filesystem tab in UI"
      contains: "tab-filesystem, view-filesystem"
    - path: "static/filesystem-ui.js"
      provides: "Filesystem panel UI controller"
      min_lines: 200
      exports: ["FilesystemUI", "buildFilesystemPanel", "renderFileTree"]
    - path: "internal/jsbridge/justbash.go"
      provides: "Extended just-bash operations for UI"
      exports: ["CreateFile", "DeleteFile", "CreateDir", "DeleteDir"]
  key_links:
    - from: "static/filesystem-ui.js"
      to: "window.justbash"
      via: "JavaScript calls"
      pattern: "justbash\\."
    - from: "index.html"
      to: "static/filesystem-ui.js"
      via: "script tag"
      pattern: "filesystem-ui\\.js"
    - from: "filesystem tab"
      to: "just-bash filesystem"
      via: "event handlers"
      pattern: "click.*directory\|select.*file"
user_setup: []
---

<objective>
Build a filesystem management UI that allows users to browse, view, edit, create, and delete files in the just-bash virtual filesystem. This UI provides a VS Code-like file explorer experience for the browser-based virtual filesystem.

Purpose: Give users visual control over the virtual filesystem, making it easier to manage files for agent operations. The UI serves as both a debugging tool and a user-friendly interface for file management.

Output:
- New "Filesystem" tab in the main navigation
- File tree sidebar showing virtual filesystem structure
- File viewer/editor for text files
- Context menu or toolbar for file operations (create, delete, rename)
- Visual indicators for virtual vs real files (when OverlayFs is used)
- Integration with existing identity file editor patterns
</objective>

<execution_context>
@/Users/gleicon/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/gleicon/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/07a-justbash-filesystem/07a-01-SUMMARY.md (after completion)
@index.html
@static/webclaw-host.js

## Current UI Patterns (from index.html)

Tab structure:
```html
<nav class="flex border-b border-gray-700 bg-gray-900 shrink-0">
  <button id="tab-chat" class="...">Chat</button>
  <button id="tab-settings" class="...">Settings</button>
  <button id="tab-identity" class="...">Identity Files</button>
</nav>
```

View structure:
```html
<div id="view-chat" class="flex flex-1 overflow-hidden">...</div>
<div id="view-settings" class="hidden flex-1 overflow-y-auto p-6">...</div>
<div id="view-identity" class="hidden flex-1 overflow-hidden">...</div>
```

Identity editor pattern:
```html
<div class="flex flex-col flex-1 overflow-hidden">
  <div class="px-4 py-2 border-b border-gray-700 flex items-center justify-between">
    <span id="identity-filename">Select a file</span>
    <button id="identity-save-btn" class="hidden ...">Save</button>
  </div>
  <textarea id="identity-editor" class="flex-1 ...">...</textarea>
</div>
```
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add Filesystem tab to main navigation</name>
  <files>index.html</files>
  <action>
    Add a new "Filesystem" tab to the main navigation bar alongside Chat, Settings, and Identity Files.
    
    Steps:
    1. Add new button in tab bar nav section:
       ```html
       <button id="tab-filesystem" class="tab-btn px-6 py-3 text-sm font-medium text-gray-400 hover:text-gray-200">Filesystem</button>
       ```
    
    2. Add new view container after view-identity:
       ```html
       <!-- Filesystem View (hidden) -->
       <div id="view-filesystem" class="hidden flex-1 overflow-hidden">
         <!-- Three-pane layout: toolbar | file tree | editor -->
         <div class="flex h-full">
           <!-- Toolbar -->
           <div class="w-12 border-r border-gray-700 flex flex-col items-center py-2 gap-2 shrink-0 bg-gray-800">
             <button id="fs-new-file-btn" class="p-2 rounded hover:bg-gray-700 text-gray-400" title="New File">📄</button>
             <button id="fs-new-folder-btn" class="p-2 rounded hover:bg-gray-700 text-gray-400" title="New Folder">📁</button>
             <button id="fs-delete-btn" class="p-2 rounded hover:bg-gray-700 text-red-400" title="Delete">🗑️</button>
             <button id="fs-refresh-btn" class="p-2 rounded hover:bg-gray-700 text-gray-400" title="Refresh">🔄</button>
           </div>
           
           <!-- File Tree -->
           <div id="fs-file-tree" class="w-64 border-r border-gray-700 overflow-y-auto bg-gray-900">
             <div class="px-3 py-2 border-b border-gray-700">
               <h2 class="text-xs font-semibold text-gray-400 uppercase">Virtual Filesystem</h2>
               <p class="text-xs text-gray-500 mt-1" id="fs-mount-info">In-Memory Mode</p>
             </div>
             <div id="fs-tree-content" class="p-2">
               <p class="text-xs text-gray-500">Loading...</p>
             </div>
           </div>
           
           <!-- File Editor -->
           <div class="flex flex-col flex-1 overflow-hidden">
             <div class="px-4 py-2 border-b border-gray-700 flex items-center justify-between">
               <div class="flex items-center gap-2">
                 <span id="fs-current-path" class="text-sm text-gray-400">Select a file</span>
                 <span id="fs-file-type" class="hidden text-xs px-2 py-1 rounded bg-gray-700 text-gray-300"></span>
               </div>
               <div class="flex gap-2">
                 <button id="fs-save-btn" class="hidden bg-indigo-600 hover:bg-indigo-500 text-white rounded px-3 py-1 text-xs">Save</button>
                 <button id="fs-close-btn" class="hidden bg-gray-700 hover:bg-gray-600 text-gray-300 rounded px-3 py-1 text-xs">Close</button>
               </div>
             </div>
             <div id="fs-editor-container" class="flex-1 overflow-hidden relative">
               <textarea id="fs-editor" class="flex-1 w-full h-full bg-gray-900 text-gray-100 font-mono text-sm p-4 resize-none focus:outline-none hidden"></textarea>
               <div id="fs-empty-state" class="absolute inset-0 flex items-center justify-center text-gray-500">
                 <p>Select a file to view or edit</p>
               </div>
               <div id="fs-binary-state" class="absolute inset-0 flex items-center justify-center text-gray-500 hidden">
                 <p>Binary file - cannot display</p>
               </div>
             </div>
             <div id="fs-status-bar" class="px-3 py-1 border-t border-gray-700 text-xs text-gray-500 flex justify-between">
               <span id="fs-status-text">Ready</span>
               <span id="fs-file-stats"></span>
             </div>
           </div>
         </div>
       </div>
       ```
    
    3. Add tab switching logic in JavaScript:
       - Add tabFilesystem variable
       - Add viewFilesystem variable
       - Add to switchTab() function:
         ```javascript
         const tabFilesystem = document.getElementById('tab-filesystem');
         const viewFilesystem = document.getElementById('view-filesystem');
         
         // In reset section:
         tabFilesystem.className = 'tab-btn px-6 py-3 text-sm font-medium text-gray-400 hover:text-gray-200';
         viewFilesystem.classList.add('hidden');
         viewFilesystem.classList.remove('flex');
         
         // In activation section:
         } else if (name === 'filesystem') {
           tabFilesystem.className = 'tab-btn px-6 py-3 text-sm font-medium border-b-2 border-indigo-500 text-indigo-400';
           viewFilesystem.classList.remove('hidden');
           viewFilesystem.classList.add('flex');
           initFilesystemUI(); // Initialize when tab is first shown
         }
         ```
    
    4. Add event listener:
       ```javascript
       tabFilesystem.addEventListener('click', () => switchTab('filesystem'));
       ```
    
    Styling notes:
    - Use consistent Tailwind classes with existing tabs
    - Three-pane layout: toolbar (48px) | file tree (256px) | editor (flex-1)
    - Dark theme matching existing UI
    - Status bar at bottom of editor (like VS Code)
  </action>
  <verify>
    <automated>grep -q "tab-filesystem\|view-filesystem" index.html && grep -q "filesystem.*tab" index.html && echo "Filesystem tab added to UI"</automated>
  </verify>
  <done>index.html includes Filesystem tab button and view container with three-pane layout (toolbar, file tree, editor)</done>
</task>

<task type="auto">
  <name>Task 2: Create filesystem UI controller</name>
  <files>static/filesystem-ui.js</files>
  <action>
    Create a JavaScript module that manages the filesystem UI interactions, file tree rendering, and editor functionality.
    
    Implementation structure:
    
    1. State management:
       ```javascript
       const fsState = {
         currentPath: '/',
         selectedFile: null,
         expandedDirs: new Set(['/']),
         fileTree: null,
         isDirty: false,
         currentContent: '',
         originalContent: ''
       };
       ```
    
    2. Core functions:
       - initFilesystemUI() - Called when tab is shown, loads root directory
       - loadDirectory(path) - Fetches directory contents via just-bash
       - renderFileTree() - Renders tree structure with expand/collapse
       - selectFile(path) - Opens file in editor
       - saveFile() - Saves editor content back to filesystem
       - createNewFile() - Prompts for name, creates via just-bash
       - createNewFolder() - Prompts for name, creates directory
       - deleteItem() - Deletes selected file or directory
       - refreshTree() - Reloads current directory
    
    3. Tree rendering:
       - Recursive function to build HTML tree
       - Icons for file types (📄 text, 📁 directory, 📦 binary)
       - Expand/collapse buttons for directories
       - Click handling for file selection
       - Visual highlighting for selected item
       - Different styling for virtual vs real files (when OverlayFs)
    
    4. Editor integration:
       - Show/hide textarea based on selection
       - Track dirty state (content changed)
       - Enable/disable save button based on dirty state
       - Line numbers (optional, can be simple)
       - Status bar: line count, char count, file size
    
    5. Toolbar actions:
       - New file: prompt for filename, create via touch/echo
       - New folder: prompt for name, create via mkdir
       - Delete: confirm dialog, then rm/rmdir
       - Refresh: reload current directory
    
    6. just-bash integration:
       ```javascript
       async function loadDirectory(path) {
         try {
           const result = await window.justbash.listDir(path);
           fsState.fileTree = buildTree(result);
           renderFileTree();
         } catch (err) {
           showError('Failed to load directory: ' + err.message);
         }
       }
       ```
    
    7. Event handlers:
       - Toolbar button clicks
       - Tree item clicks (select vs expand)
       - Editor input (mark dirty)
       - Keyboard shortcuts (Ctrl+S to save)
       - Window beforeunload (warn if dirty)
    
    8. Error handling:
       - Show toast notifications for errors
       - Loading indicators during operations
       - Disable buttons during async operations
    
    File structure pattern:
    ```javascript
    (function() {
      'use strict';
      
      // Private state
      let state = {...};
      
      // DOM refs
      const elements = {...};
      
      // Public API
      window.filesystemUI = {
        init: initFilesystemUI,
        refresh: refreshTree,
        getCurrentPath: () => state.currentPath
      };
      
      // Implementation...
    })();
    ```
  </action>
  <verify>
    <automated>test -f static/filesystem-ui.js && grep -q "initFilesystemUI\|renderFileTree\|loadDirectory" static/filesystem-ui.js | wc -l | xargs test {} -ge 3 && echo "Filesystem UI controller created with core functions"</automated>
  </verify>
  <done>static/filesystem-ui.js exists with initFilesystemUI, loadDirectory, renderFileTree, selectFile, saveFile functions and event handlers</done>
</task>

<task type="auto">
  <name>Task 3: Extend Go bindings for UI operations</name>
  <files>internal/jsbridge/justbash.go</files>
  <action>
    Extend the justbash.go file with additional operations needed by the UI: create file, delete file, create directory, delete directory, rename, and get detailed file info.
    
    Add these functions:
    1. CreateFile(path string, content string) error
       - Uses: echo "content" > path
    
    2. CreateDir(path string) error
       - Uses: mkdir -p path
    
    3. DeleteFile(path string) error
       - Uses: rm path
    
    4. DeleteDir(path string, recursive bool) error
       - Uses: rmdir path (if empty) or rm -r path (recursive)
    
    5. Rename(oldPath, newPath string) error
       - Uses: mv oldPath newPath
    
    6. GetFileInfo(path string) (FileInfo, error)
       - Uses: stat path
       - Returns: size, permissions, modified time, type
    
    7. ReadDir(path string) ([]DirEntry, error)
       - Uses: ls -la path
       - Returns array with name, type, size, modified
    
    Implementation pattern:
    ```go
    func CreateFile(path string, content string) error {
        justbash := js.Global().Get("justbash")
        if justbash.IsUndefined() {
            return fmt.Errorf("just-bash not available")
        }
        
        // Escape content for shell
        escaped := shellescape(content)
        cmd := fmt.Sprintf("echo %s > %s", escaped, path)
        
        promise := justbash.Call("executeCommand", "sh", []string{"-c", cmd})
        
        // Wait for promise and check exit code
        // ... promise handling pattern ...
        
        if exitCode != 0 {
            return fmt.Errorf("failed to create file: %s", stderr)
        }
        return nil
    }
    ```
    
    Notes:
    - Add path sanitization to prevent directory traversal
    - Handle special characters in filenames
    - Use Promise-based async pattern for all operations
    - Return detailed error messages for debugging
    
    Path sanitization:
    ```go
    func sanitizePath(path string) (string, error) {
        // Remove any .. components
        // Ensure path starts with /
        // Reject paths with null bytes
        // Return cleaned path or error
    }
    ```
  </action>
  <verify>
    <automated>grep -q "func CreateFile\|func DeleteFile\|func CreateDir\|func Rename" internal/jsbridge/justbash.go | wc -l | xargs test {} -ge 4 && echo "Extended Go bindings with file management operations"</automated>
  </verify>
  <done>internal/jsbridge/justbash.go extended with CreateFile, DeleteFile, CreateDir, DeleteDir, Rename, and GetFileInfo functions</done>
</task>

<task type="auto">
  <name>Task 4: Implement just-bash JS extensions for UI</name>
  <files>static/justbash-bridge.js</files>
  <action>
    Extend the just-bash JavaScript bridge with additional methods needed by the filesystem UI: detailed directory listing, file operations, and batch operations.
    
    Add to window.justbash API:
    1. createFile(path, content) -> Promise<void>
    2. createDirectory(path) -> Promise<void>
    3. deleteFile(path) -> Promise<void>
    4. deleteDirectory(path, recursive) -> Promise<void>
    5. rename(oldPath, newPath) -> Promise<void>
    6. listDirDetailed(path) -> Promise<[{
         name: string,
         type: 'file'|'directory',
         size: number,
         modified: string,
         permissions: string
       }]>
    7. getFileInfo(path) -> Promise<{
         exists: boolean,
         type: 'file'|'directory',
         size: number,
         modified: string,
         permissions: string
       }>
    8. searchInFiles(pattern, path, options) -> Promise<[{
         path: string,
         line: number,
         content: string
       }]>
    
    Implementation using just-bash commands:
    ```javascript
    async listDirDetailed(path) {
      const result = await this.execute('ls', ['-la', path]);
      if (result.exitCode !== 0) {
        throw new Error(result.stderr);
      }
      
      // Parse ls -la output
      const lines = result.stdout.split('\n').slice(1); // Skip total line
      return lines.filter(line => line.trim()).map(line => {
        const parts = line.split(/\s+/);
        return {
          permissions: parts[0],
          owner: parts[2],
          group: parts[3],
          size: parseInt(parts[4], 10),
          modified: `${parts[5]} ${parts[6]} ${parts[7]}`,
          name: parts.slice(8).join(' ')
        };
      });
    }
    ```
    
    Error handling:
    - Return structured error objects: {code, message, path}
    - Common error codes: ENOENT, EACCES, EEXIST, ENOTDIR
    - Log errors to console for debugging
    
    Performance:
    - Cache directory listings briefly (1 second)
    - Provide refresh option to force reload
    - Lazy load large directories (pagination if needed)
  </action>
  <verify>
    <automated>grep -q "createFile\|createDirectory\|deleteFile\|listDirDetailed\|getFileInfo" static/justbash-bridge.js | wc -l | xargs test {} -ge 5 && echo "Extended JS bridge with UI operations"</automated>
  </verify>
  <done>static/justbash-bridge.js extended with createFile, createDirectory, deleteFile, deleteDirectory, rename, listDirDetailed, and getFileInfo methods</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <what-built>
    Filesystem management UI with:
    - New "Filesystem" tab in main navigation
    - Three-pane layout: toolbar, file tree, editor
    - File operations: create, delete, rename files and directories
    - Text file viewer/editor with save functionality
    - Visual file tree with directory navigation
    - Integration with just-bash virtual filesystem
  </what-built>
  <how-to-verify>
    1. Open WebClaw in browser (http://localhost:8080)
    2. Click on "Filesystem" tab
    3. Verify the three-pane layout appears:
       - Left: Toolbar with icons (new file, new folder, delete, refresh)
       - Middle: File tree showing "Virtual Filesystem" header
       - Right: Editor area with "Select a file" message
    
    4. Test file creation:
       - Click 📄 (new file) button
       - Enter filename: "test.txt"
       - Verify file appears in tree
       - Click on test.txt to open in editor
       - Type some content: "Hello from WebClaw filesystem!"
       - Click Save button
       - Verify success message in status bar
    
    5. Test directory creation:
       - Click 📁 (new folder) button
       - Enter folder name: "projects"
       - Verify folder appears in tree
       - Double-click folder to navigate into it
       - Create a file inside projects folder
    
    6. Test file operations:
       - Create multiple files
       - Delete a file (select and click 🗑️)
       - Verify confirmation dialog appears
       - Confirm deletion
       - Verify file disappears from tree
    
    7. Test read via agent:
       - Go to Chat tab
       - Send: "Read the file /test.txt"
       - Verify agent uses file_read tool
       - Verify file content is shown
    
    8. Expected behavior:
       - All operations complete without errors
       - File tree updates immediately after changes
       - Editor shows unsaved indicator when content changes
       - Status bar shows appropriate messages
  </how-to-verify>
  <resume-signal>
    Type "approved" if all tests pass, or describe any issues found.
  </resume-signal>
</task>

</tasks>

<verification>
After completing all tasks:

1. Load WebClaw in browser
2. Click "Filesystem" tab
3. Verify three-pane layout renders correctly
4. Test creating a file: click 📄, enter name, verify in tree
5. Test editing: click file, type content, save
6. Test directory creation: click 📁, enter name, double-click to enter
7. Test deletion: select file, click 🗑️, confirm, verify removed
8. Test agent integration: "Please list the files in /" should trigger dir_list tool
9. Check console for any JavaScript errors
10. Verify Go→JS bridge calls work without errors
</verification>

<success_criteria>
Phase 7a-02 is successful when:

1. **UI visible**: Filesystem tab appears in navigation and shows three-pane layout
2. **Tree renders**: File tree displays directory structure from just-bash
3. **Navigation works**: Can expand/collapse directories, click to navigate
4. **File creation**: Can create new files via toolbar button
5. **Directory creation**: Can create new directories via toolbar button
6. **File editing**: Can open text files in editor, modify, and save
7. **File deletion**: Can delete files and directories with confirmation
8. **Status feedback**: Status bar shows loading states and operation results
9. **Agent integration**: Agent can use file tools and UI reflects changes
10. **Error handling**: Clear error messages for failed operations (permissions, not found, etc.)
</success_criteria>

<output>
After completion, create `.planning/phases/07a-justbash-filesystem/07a-02-SUMMARY.md`

Summary should document:
- UI components created and their purposes
- User workflows: browsing, editing, creating, deleting
- How virtual filesystem appears in the UI
- Performance characteristics (loading times, large file handling)
- Known limitations and future enhancements
- Connection to agent tool use (UI vs. programmatic access)
</output>
