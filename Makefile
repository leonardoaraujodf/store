.PHONY: test vet fmt-check check
test:
	@go test -v ./...
vet:
	@go vet ./...
fmt-check:
	@test -z "$$(gofmt -l $$(git ls-files '*.go'))" || \
	(echo "Run gofmt on the files listed above."; exit 1)
check: fmt-check vet test