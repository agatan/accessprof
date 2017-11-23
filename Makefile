SRCS := $(shell find . -type f -name "*.go")

.PHONY: test
test: deps $(SRCS)
	@echo "Testing..."
	@go test
	@echo "Run Example..."
	@go run _example/asstring.go

.PHONY: deps
deps:
	@echo "Resolve dependencies..."
	@go get
