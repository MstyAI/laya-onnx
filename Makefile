.PHONY: check fmt fmt-check test vet

check: fmt-check vet test

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './build/*')

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './build/*'))"

test:
	go test ./...

vet:
	go vet ./...
