#!/bin/bash
# install-cli.sh

echo "Building ipset CLI..."

# Собираем CLI для разных платформ
mkdir -p bin

# Linux
GOOS=linux GOARCH=amd64 go build -o bin/ipset-cli-linux-amd64 ./cmd/cli
GOOS=linux GOARCH=arm64 go build -o bin/ipset-cli-linux-arm64 ./cmd/cli

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/ipset-cli-darwin-amd64 ./cmd/cli
GOOS=darwin GOARCH=arm64 go build -o bin/ipset-cli-darwin-arm64 ./cmd/cli

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/ipset-cli-windows-amd64.exe ./cmd/cli

echo "CLI binaries built in ./bin/"
