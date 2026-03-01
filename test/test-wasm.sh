#!/bin/bash
# Automated WASM verification test for WebClaw Phase 1
# Uses Chrome DevTools Protocol via Chrome's --dump-dom and --virtual-time-budget

set -e

DEV_SERVER="http://localhost:8080"
LOG_FILE="/tmp/chrome-test-$$.log"
USER_DATA_DIR="/tmp/chrome-profile-$$"

echo "🧪 WebClaw WASM Automated Test Suite"
echo ""

# Check if dev server is running
echo "Checking dev server..."
if ! curl -s "$DEV_SERVER" > /dev/null 2>&1; then
    echo "❌ Dev server not running at $DEV_SERVER"
    echo "Start it with: go run ./cmd/devserver/"
    exit 1
fi
echo "✓ Dev server is running"
echo ""

# Check if required files exist
echo "Checking build artifacts..."
if [ ! -f "dist/webclaw.wasm" ]; then
    echo "❌ dist/webclaw.wasm not found. Run: make build"
    exit 1
fi
if [ ! -f "dist/webclaw.wasm.br" ]; then
    echo "❌ dist/webclaw.wasm.br not found. Run: make build"
    exit 1
fi
if [ ! -f "static/wasm_exec.js" ]; then
    echo "❌ static/wasm_exec.js not found. Run: make build"
    exit 1
fi
WASM_SIZE=$(ls -lh dist/webclaw.wasm | awk '{print $5}')
WASM_BR_SIZE=$(ls -lh dist/webclaw.wasm.br | awk '{print $5}')
echo "✓ dist/webclaw.wasm ($WASM_SIZE)"
echo "✓ dist/webclaw.wasm.br ($WASM_BR_SIZE) - compressed"
echo ""

# Function to cleanup
cleanup() {
    rm -rf "$USER_DATA_DIR" "$LOG_FILE"
}
trap cleanup EXIT

# Create a test script to inject
cat > /tmp/test-inject.js << 'INJECT_EOF'
(function() {
    const results = {
        timestamp: new Date().toISOString(),
        userAgent: navigator.userAgent,
        wasmLoaded: false,
        bridgesAvailable: false,
        jsFetchWorks: false,
        jsFetchLength: 0,
        jsIndexedDBWorks: false,
        errors: [],
        console: []
    };

    // Capture console output
    const originalLog = console.log;
    const originalError = console.error;
    
    console.log = function(...args) {
        results.console.push({ type: 'log', message: args.join(' ') });
        originalLog.apply(console, args);
    };
    
    console.error = function(...args) {
        results.console.push({ type: 'error', message: args.join(' ') });
        results.errors.push(args.join(' '));
        originalError.apply(console, args);
    };

    window.addEventListener('error', (e) => {
        results.errors.push('Window error: ' + e.message);
    });

    window.addEventListener('webclaw:ready', () => {
        results.bridgesAvailable = true;
        console.log('webclaw: bridges available - TEST EVENT RECEIVED');
        
        // Test jsFetch
        window.webclaw.jsFetch('https://example.com')
            .then(r => r.text())
            .then(text => {
                results.jsFetchWorks = text.length > 0;
                results.jsFetchLength = text.length;
                console.log('jsFetch test completed: fetched ' + text.length + ' characters');
                
                // Test jsIndexedDB
                try {
                    const request = window.webclaw.jsIndexedDB.open('test-db', 1);
                    results.jsIndexedDBWorks = request !== null && typeof request === 'object';
                    console.log('jsIndexedDB test completed: ' + (results.jsIndexedDBWorks ? 'SUCCESS' : 'FAILED'));
                } catch (e) {
                    results.errors.push('IndexedDB error: ' + e.message);
                    console.error('jsIndexedDB test failed: ' + e.message);
                }
                
                // Write results to DOM for extraction
                const resultsDiv = document.createElement('div');
                resultsDiv.id = 'test-results';
                resultsDiv.style.display = 'none';
                resultsDiv.textContent = JSON.stringify(results);
                document.body.appendChild(resultsDiv);
                console.log('TEST_RESULTS_READY');
            })
            .catch(e => {
                results.errors.push('Fetch error: ' + e.message);
                console.error('jsFetch test failed: ' + e.message);
            });
    });
    
    // Check if wasm_exec.js loaded the Go class
    if (typeof Go === 'undefined') {
        results.errors.push('Go class not available - wasm_exec.js may not have loaded');
    }
})();
INJECT_EOF

# Start Chrome with the test
echo "Starting Chrome headless to run tests..."
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
    --headless=new \
    --disable-gpu \
    --no-sandbox \
    --disable-setuid-sandbox \
    --disable-dev-shm-usage \
    --disable-background-networking \
    --disable-default-apps \
    --disable-background-timer-throttling \
    --disable-backgrounding-occluded-windows \
    --disable-renderer-backgrounding \
    --disable-features=IsolateOrigins,site-per-process,BlockInsecurePrivateNetworkRequests \
    --enable-features=SharedArrayBuffer \
    --disable-web-security \
    --allow-running-insecure-content \
    --user-data-dir="$USER_DATA_DIR" \
    --virtual-time-budget=10000 \
    --run-all-compositor-stages-before-draw \
    --dump-dom \
    --inject-js=/tmp/test-inject.js \
    "$DEV_SERVER" \
    > "$LOG_FILE" 2>&1 &

CHROME_PID=$!

# Wait for Chrome to complete
echo "Waiting for tests to complete (max 15s)..."
MAX_WAIT=15
for i in $(seq 1 $MAX_WAIT); do
    if grep -q "TEST_RESULTS_READY" "$LOG_FILE" 2>/dev/null; then
        echo "✓ Tests completed"
        break
    fi
    sleep 1
    echo -n "."
done
echo ""

# Kill Chrome if still running
kill $CHROME_PID 2>/dev/null || true
wait $CHROME_PID 2>/dev/null || true

# Extract and display results
echo ""
echo "═══════════════════════════════════════════════════"
echo "📊 Test Results"
echo "═══════════════════════════════════════════════════"

# Check for results in the DOM dump
if [ -f "$LOG_FILE" ]; then
    # Try to extract the JSON results
    RESULTS_JSON=$(grep -o '\{[^}]*"timestamp"[^}]*\}' "$LOG_FILE" | grep "wasmLoaded" | tail -1 || echo "")
    
    if [ -n "$RESULTS_JSON" ]; then
        echo "Raw results found in output"
    fi
    
    # Parse console output for test indicators
    WASM_READY=$(grep -c "webclaw: WASM ready" "$LOG_FILE" || echo "0")
    BRIDGES_AVAILABLE=$(grep -c "webclaw: bridges available" "$LOG_FILE" || echo "0")
    FETCH_SUCCESS=$(grep -c "jsFetch test completed" "$LOG_FILE" || echo "0")
    IDB_SUCCESS=$(grep -c "jsIndexedDB test completed: SUCCESS" "$LOG_FILE" || echo "0")
    TEST_RESULTS_READY=$(grep -c "TEST_RESULTS_READY" "$LOG_FILE" || echo "0")
    
    # Extract fetch length
    FETCH_LENGTH=$(grep "jsFetch test completed" "$LOG_FILE" | grep -o "fetched [0-9]* characters" | grep -o "[0-9]*" | head -1 || echo "0")
    
    # Check for errors
    ERROR_COUNT=$(grep -c "error\|Error\|ERROR\|failed\|Failed\|FAILED" "$LOG_FILE" || echo "0")
    
    echo ""
    echo "✓ WASM Ready message seen: $WASM_READY time(s)"
    echo "✓ Bridges Available message seen: $BRIDGES_AVAILABLE time(s)"
    echo "✓ jsFetch test completed: $FETCH_SUCCESS time(s)"
    [ "$FETCH_LENGTH" != "0" ] && echo "  └─ Fetched $FETCH_LENGTH characters from example.com"
    echo "✓ jsIndexedDB test successful: $IDB_SUCCESS time(s)"
    echo ""
    
    # Show any errors
    if [ "$ERROR_COUNT" -gt 0 ]; then
        echo "⚠️  Potential issues found (count: $ERROR_COUNT):"
        grep -i "error\|failed" "$LOG_FILE" | head -5 || true
        echo ""
    fi
    
    # Final verdict
    if [ "$WASM_READY" -gt 0 ] && [ "$BRIDGES_AVAILABLE" -gt 0 ] && [ "$FETCH_SUCCESS" -gt 0 ] && [ "$IDB_SUCCESS" -gt 0 ]; then
        echo "═══════════════════════════════════════════════════"
        echo "🎉 ALL TESTS PASSED - Phase 1 verification complete!"
        echo "═══════════════════════════════════════════════════"
        echo ""
        echo "Requirements satisfied:"
        echo "  ✅ BUILD-01: WASM binary compiles (verified: dist/webclaw.wasm exists)"
        echo "  ✅ BUILD-02: Host page loads WASM in browser (verified via Chrome)"
        echo "  ✅ BUILD-03: jsFetch and jsIndexedDB bridges callable (verified via tests)"
        echo "  ✅ BUILD-04: Brotli-compressed artifact produced (verified: .wasm.br is ${WASM_BR_SIZE})"
        echo ""
        echo "Build artifacts:"
        echo "  - dist/webclaw.wasm: $WASM_SIZE"
        echo "  - dist/webclaw.wasm.br: $WASM_BR_SIZE (compressed)"
        
        rm -f /tmp/test-inject.js
        exit 0
    else
        echo "═══════════════════════════════════════════════════"
        echo "❌ SOME TESTS FAILED"
        echo "═══════════════════════════════════════════════════"
        echo ""
        echo "Checklist:"
        [ "$WASM_READY" -eq 0 ] && echo "  ❌ WASM ready message not seen"
        [ "$BRIDGES_AVAILABLE" -eq 0 ] && echo "  ❌ Bridges available message not seen"
        [ "$FETCH_SUCCESS" -eq 0 ] && echo "  ❌ jsFetch test did not complete"
        [ "$IDB_SUCCESS" -eq 0 ] && echo "  ❌ jsIndexedDB test did not succeed"
        echo ""
        echo "Debug output (last 50 lines):"
        tail -50 "$LOG_FILE" || true
        
        rm -f /tmp/test-inject.js
        exit 1
    fi
else
    echo "❌ No log file generated"
    rm -f /tmp/test-inject.js
    exit 1
fi
