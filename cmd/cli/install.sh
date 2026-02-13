#!/bin/bash

echo "Building ipset-cli..."
go build -o ipset-cli cmd/cli/main.go

echo "Installing to /usr/local/bin..."
sudo mv ipset-cli /usr/local/bin/

echo "Creating example config..."
if [ ! -f "$HOME/.ipset-cli.yaml" ]; then
    cp ipset-cli.yaml.example "$HOME/.ipset-cli.yaml"
    echo "Example config created at ~/.ipset-cli.yaml"
fi

echo "Installation complete! Run 'ipset-cli --help' to get started"