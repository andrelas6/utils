BIN := bin

.PHONY: build build-linux install test clean

# Build every program in cmd/ into bin/
build:
	go build -o $(BIN)/ ./cmd/...

# Cross-compile for Linux (amd64) into bin/linux/
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BIN)/linux/ ./cmd/...

# Install every program into $(go env GOPATH)/bin
install:
	go install ./cmd/...

test:
	go test ./...

clean:
	rm -rf $(BIN)
