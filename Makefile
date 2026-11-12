SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c

.PHONY: check
check: generate generate-check docs docs-check fmt-check vet wasm-vet test modules lint wasm-lint lint-modules grammar-test-setup grammar-test

.PHONY: generate
generate:
	go run ./cmd/xgc .

.PHONY: generate-check
generate-check:
	@out="$$(git status --porcelain -- '*.xg.go')"; \
	if [ -n "$$out" ]; then \
		echo "generated files are stale (run make generate and commit):"; echo "$$out"; exit 1; \
	fi

.PHONY: docs
docs:
	go doc -all . > docs/xango.txt
	GOOS=js GOARCH=wasm go doc -all ./domrt > docs/domrt.txt
	GOOS=js GOARCH=wasm go doc -all ./domrt/browser > docs/domrt-browser.txt
	GOOS=js GOARCH=wasm go doc -all ./domrt/html > docs/domrt-html.txt
	GOOS=js GOARCH=wasm go doc -all ./domrt/svg > docs/domrt-svg.txt
	GOOS=js GOARCH=wasm go doc -all ./domrt/sw > docs/domrt-sw.txt
	GOOS=js GOARCH=wasm go doc -all ./domrt/worker > docs/domrt-worker.txt

.PHONY: docs-check
docs-check:
	@out="$$(git status --porcelain -- 'docs/*.txt')"; \
	if [ -n "$$out" ]; then \
		echo "API dumps are stale (run make docs and commit):"; echo "$$out"; exit 1; \
	fi

.PHONY: fmt-check
fmt-check:
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then \
		echo "gofmt would reformat:"; echo "$$out"; exit 1; \
	fi

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: vet
vet:
	go vet ./...

.PHONY: wasm-vet
wasm-vet:
	GOOS=js GOARCH=wasm go build ./...
	GOOS=js GOARCH=wasm go vet ./...

.PHONY: test
test:
	go test -timeout 30m ./...

# Native Gio builds need system libraries; GIO_NATIVE=1 opts in.
GIO_NATIVE ?=
NESTED_MODULES := $(shell git ls-files '*/go.mod' | xargs -n1 dirname)

.PHONY: modules
modules:
	@for mod in $(NESTED_MODULES); do \
		echo "== $$mod"; \
		(cd $$mod && go mod tidy -diff && \
		case "$$mod" in \
		*gio*) if [ -n "$(GIO_NATIVE)" ]; then go vet ./... && go test ./...; else GOOS=js GOARCH=wasm go vet ./...; fi ;; \
		*) go vet ./... && go test ./... ;; \
		esac); \
	done

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: wasm-lint
wasm-lint:
	GOOS=js GOARCH=wasm golangci-lint run ./...

.PHONY: lint-modules
lint-modules:
	@for mod in $(NESTED_MODULES); do \
		echo "== $$mod"; \
		(cd $$mod && case "$$mod" in \
		*gio*) if [ -n "$(GIO_NATIVE)" ]; then golangci-lint run --config "$(CURDIR)/.golangci.yml" ./...; else GOOS=js GOARCH=wasm golangci-lint run --config "$(CURDIR)/.golangci.yml" ./...; fi ;; \
		*) golangci-lint run --config "$(CURDIR)/.golangci.yml" ./... ;; \
		esac); \
	done

.PHONY: grammar-test-setup
grammar-test-setup:
	cd editors/vscode-xango/test && npm install \
		&& npm run fetch-go-grammar \
		&& npm run fetch-embedded-grammars

.PHONY: grammar-test
grammar-test:
	cd editors/vscode-xango/test && npm test

.PHONY: vscode-package
vscode-package:
	cd editors/vscode-xango && npx --yes @vscode/vsce package
