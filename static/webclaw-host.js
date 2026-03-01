// webclaw-host.js — WASM loader and bridge harness
(async function() {
    const go = new Go();  // Go class provided by wasm_exec.js

    // Polyfill for browsers without instantiateStreaming
    if (!WebAssembly.instantiateStreaming) {
        WebAssembly.instantiateStreaming = async (resp, importObject) => {
            const source = await (await resp).arrayBuffer();
            return await WebAssembly.instantiate(source, importObject);
        };
    }

    // Wait for bridges before exposing smoke test
    window.addEventListener('webclaw:ready', () => {
        console.log('webclaw: bridges available', window.webclaw);
        // Smoke test instructions logged to console:
        console.log('Smoke test:\n  window.webclaw.jsFetch("https://example.com").then(r => r.text()).then(console.log)\n  window.webclaw.jsIndexedDB.open("test-db", 1)');
    }, { once: true });

    try {
        const result = await WebAssembly.instantiateStreaming(
            fetch("dist/webclaw.wasm"),
            go.importObject
        );
        go.run(result.instance);  // starts Go runtime; main() runs asynchronously
    } catch (err) {
        console.error("webclaw: failed to load WASM:", err);
    }
})();

// File input handling for import/export
(function() {
    // Listen for custom events from WASM
    window.addEventListener('webclaw:request-export', (e) => {
        const { filename, content } = e.detail;
        downloadConfig(filename || 'webclaw-config.json', content);
    });
    
    window.addEventListener('webclaw:request-import', () => {
        triggerFileImport();
    });
    
    // Download helper
    function downloadConfig(filename, content) {
        const blob = new Blob([content], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }
    
    // Import helper
    function triggerFileImport() {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = '.json,application/json';
        input.style.display = 'none';
        
        input.onchange = async (e) => {
            const file = e.target.files[0];
            if (!file) return;
            
            try {
                const content = await file.text();
                // Dispatch event with content for WASM to handle
                window.dispatchEvent(new CustomEvent('webclaw:import-data', {
                    detail: { content, filename: file.name }
                }));
            } catch (err) {
                console.error('Failed to read file:', err);
            }
            
            document.body.removeChild(input);
        };
        
        document.body.appendChild(input);
        input.click();
    }
    
    // Expose helpers globally for WASM
    window.webclawHelpers = {
        downloadConfig,
        triggerFileImport
    };
})();
