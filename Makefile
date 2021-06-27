SHELL = /bin/bash
OUTPUTDIR = ./bin

.ONESHELL:
.PHONY: plugin run help

all: help

## plugin: Build all plugin.
plugin:
	@for d in ./plugin/*/;do	\
		pushd . > /dev/null;	\
		cd $$d;					\
		echo Building $$d;		\
		go build -buildmode=plugin -ldflags "-s -w" .;	\
		popd > /dev/null;		\
	done
	@echo Success

## run: Run skynet
run: plugin
	@go run . run

## help: Show this help.
help: Makefile
	@echo Usage: make [command]
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'