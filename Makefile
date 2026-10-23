BINARY_NAME = sqldash
BUILD_PATH = bin/$(BINARY_NAME)
MAIN_PATH = ./$(BINARY_NAME)

ifeq ($(OS),Windows_NT)
BUILD_PATH = bin/$(BINARY_NAME).exe
endif

.PHONY: hooks setup clean tidy build run proxy web dev dev-stop format format-check all

hooks:
	@git config --local core.hooksPath .githooks

setup: hooks
	@go mod download
	@go mod tidy

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
	@docker compose -f compose.dev.yml up --build

dev-stop:
	@docker compose -f compose.dev.yml down

format:
	@go run ./tools/format --write templates static/css

format-check:
	@go run ./tools/format --check templates static/css

all: setup clean build run

.SILENT:
