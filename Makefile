lint:
	@staticcheck ./...

build:
	@go build ./cmd/bot/

check: lint build
