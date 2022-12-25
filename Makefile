ifeq ($(OS),Windows_NT)
	RM = rmdir /s /q
else
	RM = rm -f
endif

.PHONY: build snapshot lint test benchmark clean act
.NOTPARALLEL: benchmark

build:
	go generate ./...
	go build .

# Requires https://github.com/goreleaser/goreleaser
snapshot: test
	goreleaser release --rm-dist --snapshot

# Requires https://github.com/golangci/golangci-lint
lint:
	golangci-lint run

test:
	go test -v ./...

test-all: lint test benchmark

benchmark:
	go test -v ./... -run ^$$ -bench=. -test.benchmem

clean:
	$(RM) dist

# Requires https://github.com/nektos/act
act:
	act -W .github/workflows/test.yml
