GOROOT := $(shell go env GOROOT)
WASM_EXEC_JS := $(GOROOT)/lib/wasm/wasm_exec.js

.PHONY: build serve clean

build:
	mkdir -p dist static
	GOOS=js GOARCH=wasm go build -o dist/webclaw.wasm ./cmd/webclaw/
	brotli --best -f dist/webclaw.wasm -o dist/webclaw.wasm.br
	cp $(WASM_EXEC_JS) static/wasm_exec.js

serve:
	go run ./cmd/devserver/

clean:
	rm -f dist/webclaw.wasm dist/webclaw.wasm.br static/wasm_exec.js
