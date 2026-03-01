VERSION ?= 1.0.0
BINARY  := devforge
GOFLAGS := -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 windows/amd64
DIST_DIR  := dist

.PHONY: build build-all release test clean

# ─── Build ───────────────────────────────────────────────────
build:
	@echo "Building $(BINARY) v$(VERSION)..."
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINARY) .

# ─── Cross-compile ───────────────────────────────────────────
build-all: clean
	@echo "Cross-compiling $(BINARY) v$(VERSION) for all platforms..."
	@mkdir -p $(DIST_DIR)
	@$(foreach platform,$(PLATFORMS), \
		$(eval OS := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT := $(if $(filter windows,$(OS)),.exe,)) \
		echo "  → $(OS)/$(ARCH)"; \
		GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS) \
			-ldflags '$(LDFLAGS)' \
			-o $(DIST_DIR)/$(BINARY)-$(OS)-$(ARCH)$(EXT) . ; \
	)
	@echo "Binaries written to $(DIST_DIR)/"

# ─── Release ─────────────────────────────────────────────────
release: build-all
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && shasum -a 256 * > checksums.txt
	@echo "Release artifacts ready in $(DIST_DIR)/"
	@cat $(DIST_DIR)/checksums.txt

# ─── Test ────────────────────────────────────────────────────
test:
	go test ./... -v -race -count=1

# ─── Clean ───────────────────────────────────────────────────
clean:
	@rm -rf $(DIST_DIR)
	@rm -f $(BINARY)
