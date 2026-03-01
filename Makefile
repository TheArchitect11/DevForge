VERSION ?= 1.0.0
GOFLAGS := -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 windows/amd64
DIST_DIR  := dist

.PHONY: build build-cli build-agent build-server build-all release test clean

# ─── Individual Binaries ─────────────────────────────────────
build-cli:
	@echo "Building devforge v$(VERSION)..."
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o devforge .

build-agent:
	@echo "Building devforge-agent v$(VERSION)..."
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o devforge-agent ./cmd/agent

build-server:
	@echo "Building devforge-server v$(VERSION)..."
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o devforge-server ./cmd/server

# ─── Local Build (all 3) ────────────────────────────────────
build: build-cli build-agent build-server
	@echo "All binaries built successfully."

# ─── Cross-compile ───────────────────────────────────────────
build-all: clean
	@echo "Cross-compiling v$(VERSION) for all platforms..."
	@mkdir -p $(DIST_DIR)
	@$(foreach platform,$(PLATFORMS), \
		$(eval OS := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT := $(if $(filter windows,$(OS)),.exe,)) \
		echo "  → devforge $(OS)/$(ARCH)"; \
		GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS) \
			-ldflags '$(LDFLAGS)' \
			-o $(DIST_DIR)/devforge-$(OS)-$(ARCH)$(EXT) . ; \
		echo "  → devforge-agent $(OS)/$(ARCH)"; \
		GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS) \
			-ldflags '$(LDFLAGS)' \
			-o $(DIST_DIR)/devforge-agent-$(OS)-$(ARCH)$(EXT) ./cmd/agent ; \
		echo "  → devforge-server $(OS)/$(ARCH)"; \
		GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS) \
			-ldflags '$(LDFLAGS)' \
			-o $(DIST_DIR)/devforge-server-$(OS)-$(ARCH)$(EXT) ./cmd/server ; \
	)
	@echo "All binaries written to $(DIST_DIR)/"

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
	@rm -f devforge devforge-agent devforge-server
