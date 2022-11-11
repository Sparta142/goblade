ifeq ($(OS),Windows_NT)
	RM = rmdir /s /q
else
	RM = rm -f
endif

.PHONY: snapshot test benchmark clean
.NOTPARALLEL: benchmark

build:
	go build .

snapshot: test
	goreleaser release --rm-dist --snapshot

test:
	go vet
	go test -v ./...

benchmark:
	go test -v ./... -run ^$$ -bench=. -test.benchmem

clean:
	$(RM) dist
