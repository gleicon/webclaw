# Stage 1: build Go WASM + Vite bundle
# go.mod requires Go 1.25+. Update this tag if golang:1.26-alpine is unavailable.
FROM golang:1.26-alpine AS builder

# Install Node.js, npm, brotli
RUN apk add --no-cache nodejs npm brotli

WORKDIR /app

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Cache npm packages
COPY package*.json ./
RUN npm ci

# Copy everything else
COPY . .

# 1. Compile Go → WASM
RUN mkdir -p dist static && \
    GOOS=js GOARCH=wasm go build -o dist/webclaw.wasm ./cmd/webclaw/

# 2. Brotli-compress WASM (viteStaticCopy expects both files)
RUN brotli --best -f dist/webclaw.wasm -o dist/webclaw.wasm.br

# 3. Copy wasm_exec.js from Go toolchain
RUN cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" static/wasm_exec.js

# 4. Compile Tailwind CSS
RUN npx tailwindcss -i src/styles/main.css -o static/main.css \
      --content "index.html,src/**/*.{js,ts,jsx,tsx}"

# 5. Vite bundle → dist-bundle/
RUN npm run build

# Stage 2: serve with nginx
FROM nginx:alpine

COPY --from=builder /app/dist-bundle /usr/share/nginx/html
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
