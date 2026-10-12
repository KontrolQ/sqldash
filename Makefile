BINARY_NAME = sqldash
BUILD_PATH = bin/$(BINARY_NAME)
MAIN_PATH = ./$(BINARY_NAME)

AIR_CONFIG = .air.unix.toml

ifeq ($(OS),Windows_NT)
BUILD_PATH = bin/$(BINARY_NAME).exe
AIR_CONFIG = .air.windows.toml
endif

.PHONY: hooks setup clean tidy build run proxy web dev format format-check all

hooks:
	@git config --local core.hooksPath .githooks

setup: hooks
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
	@air -c $(AIR_CONFIG)

format:
	@go run ./tools/format --write templates static/css

format-check:
	@go run ./tools/format --check templates static/css

all: setup clean build run

.SILENT:
