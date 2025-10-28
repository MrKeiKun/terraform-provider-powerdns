TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)

.EXPORT_ALL_VARIABLES:
GOFLAGS=-mod=vendor

default: build

build: fmtcheck
	go install

test: fmtcheck
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=5

testacc: fmtcheck
	TF_ACC=1 go test $(TEST) -v -parallel 5 $(TESTARGS) -timeout 120m

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@if [ -n "$$(gofmt -l $(GOFMT_FILES))" ]; then \
		echo "Go code is not formatted. Please run 'make fmt'"; \
		exit 1; \
	fi

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./internal/provider"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

.PHONY: build test testacc vet lint fmt fmtcheck test-compile
