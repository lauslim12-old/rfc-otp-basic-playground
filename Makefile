.PHONY: start
start:
	go run ./cmd/fullstack-otp/main.go

.PHONY: build
build:
	go build -v -o fullstack-otp ./cmd/fullstack-otp/main.go

.PHONY: format
format:
	test -z $(gofmt -l .)

.PHONY: test
test:
	go test -v -cover ./... ./...

.PHONY: e2e
e2e:
	sh ./scripts/e2e-testing.sh
