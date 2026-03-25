APP_NAME := lazy-json
CMD_PATH := ./cmd/$(APP_NAME)
DIST_DIR := dist
GO_FILES := $(shell find cmd internal -name '*.go' -type f)
GO_TEST_ENV := GOCACHE=$(CURDIR)/.gocache
PERF_PACKAGES := ./internal/session ./internal/tui
PERF_BENCH_ARGS := -run '^$$' -bench . -benchmem -count=1
PERF_BASELINE ?= .perf/perf.baseline.txt
PERF_CURRENT ?= .perf/perf.current.txt
PERF_CMD = $(GO_TEST_ENV) go test $(PERF_PACKAGES) $(PERF_BENCH_ARGS)

.PHONY: fmt fmt-check vet test perf perf-save perf-compare build build-all clean check

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@test -z "$$(gofmt -l $(GO_FILES))" || (gofmt -l $(GO_FILES); exit 1)

vet:
	go vet ./...

test:
	go test ./...

perf:
	$(PERF_CMD)

perf-save:
	mkdir -p $(dir $(PERF_BASELINE))
	$(PERF_CMD) | tee $(PERF_BASELINE)

perf-compare:
	sh scripts/perf-compare.sh "$(PERF_BASELINE)" "$(PERF_CURRENT)"

build:
	mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/$(APP_NAME) $(CMD_PATH)

build-all:
	mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)
	GOOS=linux GOARCH=arm64 go build -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 $(CMD_PATH)
	GOOS=darwin GOARCH=amd64 go build -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_PATH)
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_PATH)

clean:
	rm -rf $(DIST_DIR)

check: fmt-check vet test
