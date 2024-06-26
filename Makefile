GO ?= go
GOFMT ?= gofmt "-s"
GOFILES := $(shell find . -name "*.go")
VETPACKAGES ?= $(shell $(GO) list ./... | grep -v /example/)

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GOFILES)

.PHONY: vet
vet:
	$(GO) vet $(VETPACKAGES)

.PHONY: lint
lint:
	#revive -exclude chirp/*_test.go -exclude cmd/prscd/epoll.go -formatter friendly ./...
	revive -formatter friendly -exclude TEST ./...

.PHONY: build
build:
	$(GO) build -o bin/prscd ./cmd/prscd

.PHONY: dist
dist: clean
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "-s -w" -o dist/prscd-x86_64-linux ./cmd/prscd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -ldflags "-s -w" -o dist/prscd-arm64-linux ./cmd/prscd
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "-s -w" -o dist/prscd-x86_64-darwin ./cmd/prscd
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "-s -w" -o dist/prscd-arm64-darwin ./cmd/prscd
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "-s -w" -o dist/prscd-x86_64-windows.exe ./cmd/prscd
	GOOS=windows GOARCH=arm64 $(GO) build -ldflags "-s -w" -o dist/prscd-arm64-windows.exe ./cmd/prscd

.PHONY: dev
dev:
	YOMO_LOG_LEVEL=warn $(GO) run -race ./cmd/prscd

.PHONY: test
test:
	# MESH_ID=test go test ./...
	go test -race github.com/pilarjs/prscd/psig

.PHONY: coverage
coverage:
	go test -race -coverprofile=cover.out github.com/pilarjs/prscd/psig
 

.PHONY: bench
bench:
	MESH_ID=bench LOG_LEVEL=2 go test -bench=. -benchmem github.com/pilarjs/prscd/chirp

.PHONY: testpage
testpage:
	@mkdir -p ./test_pages
	@cp msgpack.js ./test_pages
	@cp websocket.html test_pages/.
	@sed -i '' 's/URL_DEBG/URL_PROD/g' test_pages/websocket.html
	@cp webtrans.html test_pages/.
	@sed -i '' 's/URL_DEBG/URL_PROD/g' test_pages/webtrans.html

.PHONY: clean
clean:
	@rm -rf dist
	@rm -rf bin
