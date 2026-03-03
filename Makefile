# VibeCat Makefile
# Build + codesign for stable Keychain access across rebuilds.

SWIFT_DIR     = VibeCat
BIN_PATH      = $(shell cd $(SWIFT_DIR) && swift build --show-bin-path 2>/dev/null || echo "$(SWIFT_DIR)/.build/arm64-apple-macosx/debug")
BINARY        = $(BIN_PATH)/VibeCat
SIGN_IDENTITY ?= Apple Development: sangguen@2weeks.co (5ZMDJBXU63)
ENTITLEMENTS  = $(SWIFT_DIR)/VibeCat.entitlements
LOG_DIR       = $(SWIFT_DIR)/.build

.PHONY: build sign run run-log test clean

## Build the Swift package
build:
	@cd $(SWIFT_DIR) && swift build

## Codesign for dev (no sandbox entitlements)
sign: build
	@if security find-identity -v -p codesigning 2>/dev/null | grep -q "$(SIGN_IDENTITY)"; then \
		codesign --force --sign "$(SIGN_IDENTITY)" "$(BINARY)" && \
		echo "[sign] Signed with $(SIGN_IDENTITY) (dev, no sandbox)"; \
	else \
		echo "[sign] Signing identity not found — skipping (CI mode)"; \
	fi

## Codesign for release (with sandbox entitlements)
sign-release: build
	@if security find-identity -v -p codesigning 2>/dev/null | grep -q "$(SIGN_IDENTITY)"; then \
		codesign --force --sign "$(SIGN_IDENTITY)" --entitlements "$(ENTITLEMENTS)" "$(BINARY)" && \
		echo "[sign-release] Signed with $(SIGN_IDENTITY) + entitlements"; \
	else \
		echo "[sign-release] Signing identity not found — skipping (CI mode)"; \
	fi

## Build + sign + run
run: sign
	@cd $(SWIFT_DIR) && "$(BINARY)"

## Build + sign + run with full logging
run-log: sign
	@cd $(SWIFT_DIR) && NSUnbufferedIO=YES "$(BINARY)" 2>&1 | tee .build/vibecat.log

## Run tests
test:
	@cd $(SWIFT_DIR) && swift test

## Clean build artifacts
clean:
	@cd $(SWIFT_DIR) && swift package clean
	@rm -f "$(LOG_DIR)/vibecat.log"

## Build backend services
backend-build:
	@cd backend/realtime-gateway && go build ./...
	@cd backend/adk-orchestrator && go build ./...

## Test backend services
backend-test:
	@cd backend/realtime-gateway && go test ./...
	@cd backend/adk-orchestrator && go test ./...

## Build Docker images
docker-build:
	@docker build -t vibecat-gateway backend/realtime-gateway/
	@docker build -t vibecat-orchestrator backend/adk-orchestrator/
