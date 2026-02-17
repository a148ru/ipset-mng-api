.PHONY: build build-cli run clean test deps install-cli

APP_NAME = ipset-api
CLI_NAME = ipset-cli
BUILD_DIR = bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

build-cli:
	go build -o $(BUILD_DIR)/$(CLI_NAME) ./cmd/cli

build-all: build build-cli

run:
	go run ./cmd/server

run-cli:
	go run ./cmd/cli $(ARGS)

clean:
	rm -rf $(BUILD_DIR)
	rm -rf data/*.json

test:
	go test -v ./...

deps:
	go mod download
	go mod tidy

install-cli:
	./install-cli.sh

docker-build:
	docker build -t $(APP_NAME) .

docker-run:
	docker run -p 8080:8080 $(APP_NAME)

.PHONY: generate-key
generate-key:
	go run cmd/generate_key/main.go

# Примеры использования CLI
cli-examples:
	@echo "=== CLI Examples ==="
	@echo ""
	@echo "# Login"
	@echo "./bin/ipset-cli auth login --key YOUR_API_KEY"
	@echo ""
	@echo "# List records"
	@echo "./bin/ipset-cli records list"
	@echo ""
	@echo "# Create ipset"
	@echo "./bin/ipset-cli ipsets create --name test --type hash:ip --family inet"
	@echo ""
	@echo "# Add entry to ipset"
	@echo "./bin/ipset-cli ipsets entries add --set test --value 192.168.1.1 --comment 'Test entry'"
	@echo ""
	@echo "# Create iptables rule"
	@echo "./bin/ipset-cli iptables create --chain INPUT --src-set test --action ACCEPT"
	@echo ""
	@echo "# Apply configuration"
	@echo "./bin/ipset-cli apply --file config.txt --dry-run"
	@echo ""
	@echo "# Show stats"
	@echo "./bin/ipset-cli stats"