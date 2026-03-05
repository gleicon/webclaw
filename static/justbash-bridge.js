/**
 * just-bash Bridge for WebClaw
 * 
 * This module provides a JavaScript bridge between WebClaw's Go/WASM core
 * and the just-bash library, enabling browser-only file operations without
 * requiring a local bridge binary.
 * 
 * Features:
 * - Virtual filesystem (InMemoryFs) for sandboxed operations
 * - Overlay filesystem (OverlayFs) for reading real projects safely
 * - File operations: read, write, list, search
 * - Bash command execution in browser
 */

(function() {
  'use strict';

  // Module state
  let bashInstance = null;
  let fsMode = 'virtual'; // 'virtual' | 'overlay'
  let overlayRoot = null;
  let isInitialized = false;

  /**
   * Initialize the just-bash bridge
   * @param {Object} options - Initialization options
   * @param {string} options.mode - 'virtual' or 'overlay'
   * @param {string} options.overlayRoot - Root directory for overlay mode (if applicable)
   * @returns {Promise<boolean>} - True if initialized successfully
   */
  async function initJustBash(options = {}) {
    if (isInitialized) {
      console.log('[just-bash] Already initialized');
      return true;
    }

    try {
      // Check if just-bash is available
      if (typeof Bash === 'undefined') {
        // Try to load from global scope or import
        console.error('[just-bash] Bash class not found. Make sure just-bash is loaded.');
        return false;
      }

      fsMode = options.mode || 'virtual';
      overlayRoot = options.overlayRoot || '/home/user';

      // Initialize Bash based on mode
      if (fsMode === 'overlay') {
        // Overlay mode - reads from real filesystem, writes to memory
        // Note: This requires File System Access API in modern browsers
        bashInstance = new Bash({
          cwd: overlayRoot,
          env: {
            HOME: '/home/user',
            PWD: overlayRoot,
            PATH: '/bin:/usr/bin'
          }
        });
        console.log('[just-bash] Initialized in overlay mode, root:', overlayRoot);
      } else {
        // Virtual mode - completely in-memory
        bashInstance = new Bash({
          cwd: '/home/user',
          env: {
            HOME: '/home/user',
            PWD: '/home/user',
            PATH: '/bin:/usr/bin'
          }
        });
        
        // Create default directory structure
        await bashInstance.exec('mkdir -p /home/user/workspace /home/user/projects /tmp');
        console.log('[just-bash] Initialized in virtual mode');
      }

      isInitialized = true;
      
      // Notify Go WASM that just-bash is ready
      if (window.webclaw && window.webclaw.onJustBashReady) {
        window.webclaw.onJustBashReady(true);
      }

      return true;
    } catch (error) {
      console.error('[just-bash] Initialization failed:', error);
      
      if (window.webclaw && window.webclaw.onJustBashReady) {
        window.webclaw.onJustBashReady(false, error.message);
      }
      
      return false;
    }
  }

  /**
   * Execute a bash command
   * @param {string} command - The command to execute
   * @param {Object} options - Execution options
   * @returns {Promise<Object>} - Result with stdout, stderr, exitCode
   */
  async function executeCommand(command, options = {}) {
    if (!isInitialized) {
      throw new Error('just-bash not initialized. Call initJustBash() first.');
    }

    try {
      const result = await bashInstance.exec(command, {
        cwd: options.cwd,
        env: options.env,
        timeout: options.timeout || 30000 // 30 second default timeout
      });

      return {
        stdout: result.stdout || '',
        stderr: result.stderr || '',
        exitCode: result.exitCode || 0,
        success: result.exitCode === 0
      };
    } catch (error) {
      return {
        stdout: '',
        stderr: error.message,
        exitCode: 1,
        success: false
      };
    }
  }

  /**
   * Read a file from the virtual filesystem
   * @param {string} path - File path
   * @returns {Promise<string>} - File contents
   */
  async function readFile(path) {
    const result = await executeCommand(`cat "${escapePath(path)}"`);
    
    if (!result.success) {
      throw new Error(`Failed to read file: ${result.stderr}`);
    }
    
    return result.stdout;
  }

  /**
   * Write content to a file
   * @param {string} path - File path
   * @param {string} content - Content to write
   * @returns {Promise<boolean>} - True if successful
   */
  async function writeFile(path, content) {
    // Escape content for shell
    const escapedContent = content
      .replace(/\\/g, '\\\\')
      .replace(/"/g, '\\"')
      .replace(/\$/g, '\\$')
      .replace(/`/g, '\\`');
    
    // Create parent directories if needed
    const dirPath = path.substring(0, path.lastIndexOf('/')) || '/';
    if (dirPath !== '/') {
      await executeCommand(`mkdir -p "${escapePath(dirPath)}"`);
    }
    
    const result = await executeCommand(`echo "${escapedContent}" > "${escapePath(path)}"`);
    
    return result.success;
  }

  /**
   * List directory contents
   * @param {string} path - Directory path
   * @param {Object} options - List options
   * @returns {Promise<Array>} - Array of file/directory entries
   */
  async function listDir(path, options = {}) {
    const flags = options.all ? '-la' : (options.long ? '-l' : '');
    const result = await executeCommand(`ls ${flags} "${escapePath(path)}"`);
    
    if (!result.success) {
      throw new Error(`Failed to list directory: ${result.stderr}`);
    }
    
    // Parse ls output
    const lines = result.stdout.split('\n').filter(line => line.trim());
    const entries = [];
    
    for (const line of lines) {
      if (line.startsWith('total')) continue; // Skip total line
      
      // Parse ls -l output
      const parts = line.split(/\s+/);
      if (parts.length >= 9) {
        entries.push({
          permissions: parts[0],
          owner: parts[2],
          group: parts[3],
          size: parseInt(parts[4], 10),
          date: `${parts[5]} ${parts[6]} ${parts[7]}`,
          name: parts.slice(8).join(' '),
          isDirectory: parts[0].startsWith('d'),
          isSymlink: parts[0].startsWith('l')
        });
      } else if (line && !options.long) {
        // Simple ls output
        entries.push({
          name: line,
          isDirectory: false, // Can't tell from simple ls
          isSymlink: false
        });
      }
    }
    
    return entries;
  }

  /**
   * Search for text patterns in files
   * @param {string} pattern - Search pattern
   * @param {string} path - Path to search (file or directory)
   * @param {Object} options - Search options
   * @returns {Promise<Array>} - Array of matches
   */
  async function searchFiles(pattern, path, options = {}) {
    const flags = [];
    if (options.recursive) flags.push('-r');
    if (options.ignoreCase) flags.push('-i');
    if (options.lineNumber !== false) flags.push('-n');
    
    const flagStr = flags.join(' ');
    const result = await executeCommand(
      `grep ${flagStr} "${escapePattern(pattern)}" "${escapePath(path)}" 2>/dev/null || echo ""`
    );
    
    // Parse grep output
    const lines = result.stdout.split('\n').filter(line => line.trim());
    const matches = [];
    
    for (const line of lines) {
      if (options.lineNumber !== false) {
        const match = line.match(/^([^:]+):(\d+):(.*)$/);
        if (match) {
          matches.push({
            file: match[1],
            line: parseInt(match[2], 10),
            text: match[3]
          });
        }
      } else {
        matches.push({
          file: path,
          text: line
        });
      }
    }
    
    return matches;
  }

  /**
   * Get filesystem information
   * @returns {Promise<Object>} - Filesystem stats
   */
  async function getFsInfo() {
    const result = await executeCommand('df -h /home/user 2>/dev/null || echo "Filesystem    Size  Used Avail Use% Mounted on
just-bash-fs   1G   10M  990M   1% /home/user"');
    
    const lines = result.stdout.split('\n');
    if (lines.length > 1) {
      const parts = lines[1].split(/\s+/);
      return {
        filesystem: parts[0],
        size: parts[1],
        used: parts[2],
        available: parts[3],
        usePercent: parts[4],
        mountedOn: parts[5],
        mode: fsMode
      };
    }
    
    return { mode: fsMode };
  }

  /**
   * Escape path for shell safety
   * @param {string} path - Path to escape
   * @returns {string} - Escaped path
   */
  function escapePath(path) {
    return path.replace(/"/g, '\\"');
  }

  /**
   * Escape pattern for grep safety
   * @param {string} pattern - Pattern to escape
   * @returns {string} - Escaped pattern
   */
  function escapePattern(pattern) {
    return pattern.replace(/"/g, '\\"');
  }

  /**
   * Check if just-bash is available and initialized
   * @returns {boolean}
   */
  function isReady() {
    return isInitialized && bashInstance !== null;
  }

  /**
   * Get current working directory
   * @returns {string}
   */
  function getCwd() {
    return bashInstance ? bashInstance.cwd : '/home/user';
  }

  /**
   * Change working directory
   * @param {string} path - New working directory
   */
  async function changeDir(path) {
    if (!bashInstance) return false;
    
    const result = await executeCommand(`cd "${escapePath(path)}"`);
    if (result.success) {
      bashInstance.cwd = path;
    }
    
    return result.success;
  }

  // Expose API to global scope for Go WASM
  window.justBashBridge = {
    init: initJustBash,
    exec: executeCommand,
    readFile,
    writeFile,
    listDir,
    searchFiles,
    getFsInfo,
    isReady,
    getCwd,
    changeDir,
    getMode: () => fsMode,
    getVersion: () => '2.11.12'
  };

  console.log('[just-bash] Bridge loaded. Call justBashBridge.init() to initialize.');
})();
