SHELL = /bin/bash
OUTPUTDIR = ./bin

.ONESHELL:
.PHONY: generate run help build build_plugin docker coverage test packer clean

all: help

## clean: Clean build files
clean:
	@rm -rf $(OUTPUTDIR)
	@rm -f coverage.html coverage.out

## packer: Build plugin packer
packer:
	@xgo -ldflags "-s -w" -targets linux/amd64 -dest $(OUTPUTDIR) -pkg packer -out packer .
	@mv $(OUTPUTDIR)/packer-linux-amd64 $(OUTPUTDIR)/packer

## test: Go test
test: 
	@go test ./...

## coverage: Make coverage
coverage:
	@go test -coverprofile=coverage_temp.out ./...
	@cat coverage_temp.out | grep -v "_gen.go" > coverage.out && rm coverage_temp.out
	@go tool cover -html=coverage.out -o coverage.html
	@xdg-open coverage.html >/dev/null 2>&1

## docker: Build docker image
docker: build
	@docker build -t imwxz/skynet:latest .

## build_plugin: Build plugin binary file
build_plugin:
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

## build: Build skynet binary file
build:
	@rm -rf $(OUTPUTDIR)
	@echo Building skynet
	@mkdir -p $(OUTPUTDIR)
	@mkdir -p $(OUTPUTDIR)/plugin
	@xgo -ldflags "-s -w" -targets linux/amd64 -dest $(OUTPUTDIR) .
	@mv $(OUTPUTDIR)/skynet-linux-amd64 $(OUTPUTDIR)/skynet
	@cp LICENSE $(OUTPUTDIR)
	@cp conf.yml $(OUTPUTDIR)
	@cp default.webp $(OUTPUTDIR)
	@rm -rf $(OUTPUTDIR)/assets && cp -r assets $(OUTPUTDIR)/assets
	@rm -rf $(OUTPUTDIR)/templates && cp -r templates $(OUTPUTDIR)/templates
	@echo Success

## generate: Generate dynamic source code.
generate:
	@echo Generate utils;
	@genny -in=./sn/utils/map_tpl.go -out=./sn/utils/map_int_gen.go gen "MPrefix=Int MTypeA=int MTypeB=interface{}"
	@genny -in=./sn/utils/map_tpl.go -out=./sn/utils/map_string_gen.go gen "MPrefix=String MTypeA=string MTypeB=interface{}"
	@genny -in=./sn/utils/map_tpl.go -out=./sn/utils/map_uuid_gen.go gen "MPrefix=UUID MTypeA=uuid.UUID MTypeB=interface{}"
	@genny -in=./sn/utils/map_tpl.go -out=./handler/map_plugin_gen.go -pkg=handler gen "MPrefix=Plugin MTypeA=uuid.UUID MTypeB=*PluginLoad"

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

## run: Run skynet(for develop use)
run:
	@for d in ./plugin/*/;do	\
		pushd . > /dev/null;	\
		cd $$d;					\
		echo Building $$d;		\
		go build -buildmode=plugin .;	\
		popd > /dev/null;		\
	done
	@echo Success
	@go run . run

## help: Show this help.
help: Makefile
	@echo Usage: make [command]
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'