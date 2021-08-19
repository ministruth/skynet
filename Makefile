SHELL = /bin/bash
OUTPUTDIR = ./bin

.ONESHELL:
.PHONY: generate plugin run help build docker

all: help

## docker: Build docker image
docker: build
	@docker build -t imwxz/skynet:latest .

## build: Build binary file
build:
	@rm -rf $(OUTPUTDIR)
	@echo Building skynet
	@mkdir -p $(OUTPUTDIR)
	@xgo -ldflags "-s -w" -targets linux/amd64 -dest $(OUTPUTDIR) .
	@mv $(OUTPUTDIR)/skynet-linux-amd64 $(OUTPUTDIR)/skynet
	@cp LICENSE $(OUTPUTDIR)
	@cp conf.yml $(OUTPUTDIR)
	@cp default.webp $(OUTPUTDIR)
	@rm -rf $(OUTPUTDIR)/assets && cp -r assets $(OUTPUTDIR)/assets
	@rm -rf $(OUTPUTDIR)/templates && cp -r templates $(OUTPUTDIR)/templates
	
	@echo Building plugins
	@for d in ./plugin/*/;do	\
		echo Building $$d;		\
		mkdir -p $(OUTPUTDIR)/$$d;	\
		name=$${d%/*};	\
		name=$${name##*/};	\
		xgo -buildmode=plugin -ldflags "-s -w" -targets linux/amd64 -dest $(OUTPUTDIR)/$$d -pkg $$d -out $$name .;	\
		sleep 3;	\
		mv $(OUTPUTDIR)/$$d$$name-linux-amd64 $(OUTPUTDIR)/$$d$$name.so;	\
		pushd . > /dev/null;	\
		cd $$d;					\
		if [[ -f "Makefile" ]]; then \
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
		go build -buildmode=plugin .;	\
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