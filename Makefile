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
	go test -v -coverpkg ./internal/otp ./internal/otp
	go test -v -coverpkg ./internal/application ./internal/application
