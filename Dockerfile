# Multi-stage build for minimal size
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./
RUN npm ci

# Copy source and build
COPY . .
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy built static files
COPY --from=builder /app/dist-bundle /usr/share/nginx/html

# Configure nginx for brotli and SPA routing
RUN echo 'server { \
    listen 80; \
    server_name localhost; \
    root /usr/share/nginx/html; \
    index index.html; \
    \
    # Gzip compression \
    gzip on; \
    gzip_vary on; \
    gzip_min_length 1024; \
    gzip_proxied expired no-cache no-store private auth; \
    gzip_types \
        text/plain \
        text/css \
        text/xml \
        text/javascript \
        application/javascript \
        application/json \
        application/wasm; \
    \
    # Brotli compression support for WASM files \
    location ~ \\.wasm\\.br$ { \
        add_header Content-Encoding br; \
        add_header Content-Type application/wasm; \
        add_header Cache-Control "public, max-age=31536000, immutable"; \
    } \
    \
    # WASM files \
    location ~ \\.wasm$ { \
        add_header Content-Type application/wasm; \
        add_header Cache-Control "public, max-age=31536000, immutable"; \
    } \
    \
    # Static assets with long cache \
    location ~ \\.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|otf|eot)$ { \
        add_header Cache-Control "public, max-age=31536000, immutable"; \
    } \
    \
    # SPA routing - serve index.html for all routes \
    location / { \
        try_files $uri $uri/ /index.html; \
        add_header Cache-Control "public, max-age=0, must-revalidate"; \
    } \
}' > /etc/nginx/conf.d/default.conf

# Remove default nginx config
RUN rm /etc/nginx/conf.d/default.conf 2>/dev/null || true

# Write the config again
RUN echo 'server { \
    listen 80; \
    server_name localhost; \
    root /usr/share/nginx/html; \
    index index.html; \
    \
    # Gzip compression \
    gzip on; \
    gzip_vary on; \
    gzip_min_length 1024; \
    gzip_proxied expired no-cache no-store private auth; \
    gzip_types \
        text/plain \
        text/css \
        text/xml \
        text/javascript \
        application/javascript \
        application/json \
        application/wasm; \
    \
    # Brotli compression support for WASM files \
    location ~ \\.wasm\\.br$ { \
        add_header Content-Encoding br; \
        add_header Content-Type application/wasm; \
        add_header Cache-Control "public, max-age=31536000, immutable"; \
    } \
    \
    # WASM files \
    location ~ \\.wasm$ { \
        add_header Content-Type application/wasm; \
        add_header Cache-Control "public, max-age=31536000, immutable"; \
    } \
    \
    # Static assets with long cache \
    location ~ \\.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|otf|eot)$ { \
        add_header Cache-Control "public, max-age=31536000, immutable"; \
    } \
    \
    # SPA routing - serve index.html for all routes \
    location / { \
        try_files $uri $uri/ /index.html; \
        add_header Cache-Control "public, max-age=0, must-revalidate"; \
    } \
}' > /etc/nginx/conf.d/webclaw.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
