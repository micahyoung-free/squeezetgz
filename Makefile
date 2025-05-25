.PHONY: build test clean

# Build the binary
build:
	go build -o bin/squeezetgz ./cmd/squeezetgz

# Run the tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Install the binary
install:
	go install ./cmd/squeezetgz