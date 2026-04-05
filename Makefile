BIN := hippocamp
CMD := ./cmd/hippocamp
SIGN_ID := Developer ID Application: Ruslan Korennoy (4JV3A5SUSZ)

.PHONY: build test lint tidy clean run sign release replace

build:
	go build -ldflags "-X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o $(BIN) $(CMD)

sign: build
	codesign --sign "$(SIGN_ID)" --options runtime --timestamp ./$(BIN)
	@codesign -vvv --deep --strict ./$(BIN)
	@echo "Signed and verified."

release: clean
	@echo "Building darwin/arm64..."
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o dist/$(BIN)_darwin_arm64 $(CMD)
	codesign --sign "$(SIGN_ID)" --options runtime --timestamp dist/$(BIN)_darwin_arm64
	@echo "Building darwin/amd64..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o dist/$(BIN)_darwin_amd64 $(CMD)
	codesign --sign "$(SIGN_ID)" --options runtime --timestamp dist/$(BIN)_darwin_amd64
	@echo "Building linux/amd64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o dist/$(BIN)_linux_amd64 $(CMD)
	@echo "Building linux/arm64..."
	GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o dist/$(BIN)_linux_arm64 $(CMD)
	@codesign -vvv dist/$(BIN)_darwin_arm64 && codesign -vvv dist/$(BIN)_darwin_amd64
	@echo "All binaries built and macOS binaries signed."
	@ls -lh dist/

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
	rm -rf dist/

replace: sign
	@BREW_BIN=$$(ls /opt/homebrew/Caskroom/hippocamp/*/hippocamp 2>/dev/null | head -1); \
	if [ -z "$$BREW_BIN" ]; then echo "Homebrew hippocamp not found"; exit 1; fi; \
	cp ./$(BIN) "$$BREW_BIN"; \
	xattr -cr "$$BREW_BIN"; \
	codesign --force --sign "$(SIGN_ID)" --options runtime --timestamp "$$BREW_BIN"; \
	codesign -vvv "$$BREW_BIN"; \
	echo "Replaced $$BREW_BIN"

run: build
	./$(BIN) --config config.yaml
