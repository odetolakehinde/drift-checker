APP_NAME=drift-checker
SRC=./
PKG=github.com/odetolakehinde/drift-checker

# Run the CLI with args
run:
	go run $(SRC)

run-json:
	go run $(SRC) -json

# Build the binary
build:
	go build -o $(APP_NAME) $(SRC)

# Run all unit tests with coverage
test:
	go test ./... -v -cover

# Generate coverage profile + report
coverage:
	go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out

# HTML coverage report
coverage-html:
	go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Format code
fmt:
	go fmt ./...

# Clean build and test artifacts
clean:
	rm -f $(APP_NAME) coverage.out

.PHONY: run build test coverage coverage-html fmt clean
