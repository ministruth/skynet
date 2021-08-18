SHELL = /bin/bash
OUTPUTDIR = $(CURDIR)/bin

.ONESHELL:
.PHONY: generate plugin run help build binrun

all: help

## binrun: Run binary file
binrun: build
	@cd $(OUTPUTDIR)
	@./skynet run

## build: Build binary file
build:
	@echo Building skynet
	@mkdir -p $(OUTPUTDIR)
	@go build -ldflags "-s -w" -o $(OUTPUTDIR) .
	@cp LICENSE $(OUTPUTDIR)
	@cp conf.yml $(OUTPUTDIR)
	@cp default.webp $(OUTPUTDIR)
	@rm -rf $(OUTPUTDIR)/assets && cp -r assets $(OUTPUTDIR)/assets
	@rm -rf $(OUTPUTDIR)/templates && cp -r templates $(OUTPUTDIR)/templates
	
	@echo Building plugins
	@mkdir -p $(OUTPUTDIR)/plugin
	@for d in ./plugin/*/;do	\
		pushd . > /dev/null;	\
		cd $$d;					\
		if [[ -f "Makefile" ]]; then \
		echo Building $$d;		\
		mkdir -p $(OUTPUTDIR)/$$d;	\
		go build -buildmode=plugin -ldflags "-s -w" -o $(OUTPUTDIR)/$$d .;	\
		make --no-print-directory build;	\
		fi; \
		popd > /dev/null;		\
	done
	@echo Success

## generate: Generate dynamic source code.
generate:
	@echo Generate utils;
	@genny -in=./sn/utils/map_tpl.go -out=./sn/utils/map_int.go gen "MPrefix=Int MTypeA=int MTypeB=interface{}"
	@genny -in=./sn/utils/map_tpl.go -out=./sn/utils/map_string.go gen "MPrefix=String MTypeA=string MTypeB=interface{}"
	@genny -in=./sn/utils/map_tpl.go -out=./sn/utils/map_uuid.go gen "MPrefix=UUID MTypeA=uuid.UUID MTypeB=interface{}"

	@for d in ./plugin/*/;do	\
		pushd . > /dev/null;	\
		cd $$d;					\
		if [[ -f "Makefile" ]]; then \
		echo Generate $$d;		\
		make --no-print-directory generate;	\
		fi; \
		popd > /dev/null;		\
	done
	@echo Success
	
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