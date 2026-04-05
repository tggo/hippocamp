BIN := hippocamp
CMD := ./cmd/hippocamp

.PHONY: build test lint tidy clean run

build:
	go build -o $(BIN) $(CMD)

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
