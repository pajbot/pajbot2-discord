lint:
	@go tool staticcheck ./...

build:
	@go build ./cmd/bot/

check: lint build
