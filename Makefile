SHELL = /bin/bash
.SHELLFLAGS = -

.ONESHELL:
.PHONY: run help generate test

all: help

## test: run test
test:
	@go test ./...
	@govulncheck ./...

## generate: generate files
generate:
	@go generate ./...

## run: Run skynet(for develop use)
run: generate
	@go run . run -v --debug --persist-session --disable-csrf

## help: Show this help.
help: Makefile
	@echo Usage: make [command]
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
