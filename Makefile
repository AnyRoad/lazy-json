APP_NAME := lazy-json
CMD_PATH := .
DIST_DIR := dist
GO_FILES := $(shell find . -name '*.go' -type f -not -path './.gocache/*' -not -path './dist/*')
GO_TEST_ENV := GOCACHE=$(CURDIR)/.gocache
GO_BUILD_ENV := GOCACHE=$(CURDIR)/.gocache
GO_BUILD_FLAGS := -buildvcs=false
COVER_DIR := .coverage
COVER_RAW_PROFILE := $(COVER_DIR)/coverage.raw.out
COVER_PROFILE := $(COVER_DIR)/coverage.out
PERF_PACKAGES := ./internal/session ./internal/tui
PERF_BENCH_ARGS := -run '^$$' -bench . -benchmem -count=1
PERF_BASELINE ?= .perf/perf.baseline.txt
PERF_CURRENT ?= .perf/perf.current.txt
PERF_CMD = $(GO_TEST_ENV) go test $(PERF_PACKAGES) $(PERF_BENCH_ARGS)
GORELEASER ?= goreleaser

.PHONY: fmt fmt-check vet test cover badge-cover perf perf-save perf-compare release-check release-snapshot build build-all clean check

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@test -z "$$(gofmt -l $(GO_FILES))" || (gofmt -l $(GO_FILES); exit 1)

vet:
	go vet ./...

test:
	go test ./...

cover:
	sh scripts/coverage-profile.sh "$(COVER_PROFILE)" "$(COVER_RAW_PROFILE)"

badge-cover: cover
	sh scripts/update-coverage-badge.sh "$(COVER_PROFILE)" "docs/badges/coverage.svg"

perf:
	$(PERF_CMD)

perf-save:
	mkdir -p $(dir $(PERF_BASELINE))
	$(PERF_CMD) | tee $(PERF_BASELINE)

perf-compare:
	sh scripts/perf-compare.sh "$(PERF_BASELINE)" "$(PERF_CURRENT)"

release-check:
	$(GO_BUILD_ENV) $(GORELEASER) check

release-snapshot:
	$(GO_BUILD_ENV) $(GORELEASER) release --snapshot --clean

build:
	mkdir -p $(DIST_DIR)
	$(GO_BUILD_ENV) go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME) $(CMD_PATH)

build-all:
	mkdir -p $(DIST_DIR)
	$(GO_BUILD_ENV) GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)
	$(GO_BUILD_ENV) GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 $(CMD_PATH)
	$(GO_BUILD_ENV) GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_PATH)
	$(GO_BUILD_ENV) GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_PATH)
	$(GO_BUILD_ENV) GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_PATH)

clean:
	rm -rf $(DIST_DIR)

check: fmt-check vet test
