BIN := hippocamp
CMD := ./cmd/hippocamp

.PHONY: build test lint tidy clean run

build:
	go build -ldflags "-X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o $(BIN) $(CMD)

test:
	go test ./...

test-v:
	go test -v ./...

lint:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f $(BIN)

run: build
	./$(BIN) --config config.yaml
