.PHONY: build test vet lint clean install

# Build variables
BINARY   := kcli
MAIN     := ./cmd/kcli
LDFLAGS  := -s -w

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) $(MAIN)

test:
	go test -v -count=1 -timeout=120s ./...

vet:
	go vet ./...

lint: vet
	@echo "Lint passed (go vet)"

clean:
	rm -rf bin/ dist/
	go clean -testcache

install: build
	cp bin/$(BINARY) $(GOPATH)/bin/$(BINARY) 2>/dev/null || cp bin/$(BINARY) /usr/local/bin/$(BINARY)

# Cross-compilation targets
.PHONY: build-all
build-all:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64   $(MAIN)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64   $(MAIN)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64  $(MAIN)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64  $(MAIN)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe $(MAIN)
	chmod +x dist/$(BINARY)-linux-* dist/$(BINARY)-darwin-*

# E2E tests (requires live cluster)
.PHONY: e2e
e2e: build
	bash scripts/e2e-test.sh
