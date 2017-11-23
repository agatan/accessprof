SRCS := $(shell find . -type f -name "*.go")

.PHONY: test
test: $(SRCS)
	@echo "Testing..."
	@go test
	@echo "Run Example..."
	@go run _example/asstring.go

