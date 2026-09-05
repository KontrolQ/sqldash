BINARY_NAME = sqldash
BUILD_PATH = bin/$(BINARY_NAME)
MAIN_PATH = ./$(BINARY_NAME)

.PHONY: setup clean tidy build run proxy web dev all

setup:
	@go mod download
	@go mod tidy
	@which air > /dev/null 2>&1 || go install github.com/air-verse/air@latest

clean:
	@rm -rf bin

tidy:
	@go mod tidy

build:
	@go build -o $(BUILD_PATH) $(MAIN_PATH)

run:
	@if [ ! -f $(BUILD_PATH) ]; then $(MAKE) -s build; fi
	@$(BUILD_PATH)

proxy:
	@if [ ! -f $(BUILD_PATH) ]; then $(MAKE) -s build; fi
	@$(BUILD_PATH) proxy

web:
	@if [ ! -f $(BUILD_PATH) ]; then $(MAKE) -s build; fi
	@$(BUILD_PATH) web

dev:
	@air

all: setup clean build run

.SILENT:
