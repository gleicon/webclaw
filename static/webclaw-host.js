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
