GOROOT := $(shell go env GOROOT)
WASM_EXEC_JS := $(GOROOT)/lib/wasm/wasm_exec.js
WASM_EXEC_BIN := $(GOROOT)/lib/wasm

.PHONY: build css serve clean test test-wasm test-all docker-build docker-run

build: css
	mkdir -p dist static
	GOOS=js GOARCH=wasm go build -o dist/webclaw.wasm ./cmd/webclaw/
	brotli --best -f dist/webclaw.wasm -o dist/webclaw.wasm.br
	cp $(WASM_EXEC_JS) static/wasm_exec.js

css:
	npx tailwindcss -i src/styles/main.css -o static/main.css --content "index.html,src/**/*.{js,ts,jsx,tsx}"

# Run non-WASM Go unit tests (fast, no browser needed)
test:
	go test ./internal/agent/... ./internal/tools/...

# Run WASM-tagged provider tests via Node (requires node in PATH)
test-wasm:
	GOOS=js GOARCH=wasm PATH="$(WASM_EXEC_BIN):$(PATH)" go test ./internal/provider/...

# Run all tests (non-WASM + WASM)
test-all: test test-wasm

serve:
	go run ./cmd/devserver/

# Build the Docker image (full Go+Node build inside container)
docker-build:
	docker build -t webclaw:latest .

# Run the Docker image on http://localhost:8080
docker-run:
	docker run --rm -p 8080:80 webclaw:latest

clean:
	rm -f dist/webclaw.wasm dist/webclaw.wasm.br static/wasm_exec.js static/main.css
