#!/bin/bash
# Deploy WebClaw to a static hosting directory

set -e

BUILD_DIR="${1:-./dist-static}"

echo "Building WebClaw..."
make clean
make build

echo "Creating deployment directory: $BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Copy static files
cp index.html "$BUILD_DIR/"
cp -r static "$BUILD_DIR/"
cp dist/webclaw.wasm.br "$BUILD_DIR/"

echo ""
echo "✅ Deployment files ready in: $BUILD_DIR"
echo ""
echo "Files to deploy:"
ls -lh "$BUILD_DIR/"
echo ""
echo "Deploy to:"
echo "  - GitHub Pages"
echo "  - Netlify"
echo "  - Vercel"
echo "  - AWS S3"
echo "  - Any static hosting"
