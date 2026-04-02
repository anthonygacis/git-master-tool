BINARY    := gitmt
SRC       := src
DIST      := dist
VERSION   ?= $(shell git -C $(SRC) describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: all release build-amd64 build-arm64 universal build-linux-amd64 build-linux-arm64 build-windows clean install

all: universal

build-amd64:
	@mkdir -p $(DIST)
	cd $(SRC) && GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o ../$(DIST)/$(BINARY)-darwin-amd64 .
	@echo "built $(DIST)/$(BINARY)-darwin-amd64"

build-arm64:
	@mkdir -p $(DIST)
	cd $(SRC) && GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o ../$(DIST)/$(BINARY)-darwin-arm64 .
	@echo "built $(DIST)/$(BINARY)-darwin-arm64"

universal: build-amd64 build-arm64
	lipo -create -output $(DIST)/$(BINARY)-darwin-universal \
		$(DIST)/$(BINARY)-darwin-amd64 \
		$(DIST)/$(BINARY)-darwin-arm64
	@echo "built $(DIST)/$(BINARY)-darwin-universal"

build-linux-amd64:
	@mkdir -p $(DIST)
	cd $(SRC) && GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ../$(DIST)/$(BINARY)-linux-amd64 .
	@echo "built $(DIST)/$(BINARY)-linux-amd64"

build-linux-arm64:
	@mkdir -p $(DIST)
	cd $(SRC) && GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o ../$(DIST)/$(BINARY)-linux-arm64 .
	@echo "built $(DIST)/$(BINARY)-linux-arm64"

build-windows:
	@mkdir -p $(DIST)
	cd $(SRC) && GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o ../$(DIST)/$(BINARY)-windows-amd64.exe .
	@echo "built $(DIST)/$(BINARY)-windows-amd64.exe"

release: universal build-linux-amd64 build-linux-arm64 build-windows
	@echo "all release artifacts in $(DIST)/"
	@ls -lh $(DIST)/

install: universal
	cp $(DIST)/$(BINARY)-darwin-universal /usr/local/bin/$(BINARY)
	@echo "installed /usr/local/bin/$(BINARY)"

clean:
	rm -rf $(DIST)
	@echo "cleaned"
