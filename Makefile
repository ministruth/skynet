SHELL = /bin/bash
.SHELLFLAGS = -
OUTPUTDIR = ./bin
TARGETS = darwin/arm64

.ONESHELL:
.PHONY: generate run help build build_plugin docker coverage test packer clean docs static static_plugin package

all: help

## docs: Generate documents
docs:
	@apidoc -f ".*\\.go$$" -i api/ -o docs/ -c apidoc.json -v

## clean: Clean build files
clean:
	@rm -rf $(OUTPUTDIR)
	@rm -rf assets docs
	@rm -f coverage.html coverage.out

## packer: Build plugin packer
packer:
	@xgo -ldflags "-s -w" -targets $(TARGETS) -dest $(OUTPUTDIR) -pkg packer -out packer .

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
docker: build build_plugin
	@docker build -t imwxz/skynet:latest .

## build_plugin: Build plugin binary file
build_plugin:
	@echo Building plugins
	@for d in `ls ./plugin | grep "^[^_]"`;do	\
		if [ -d ./plugin/$$d ];then	\
			rm -rf ./plugin/$$d/bin;	\
			mkdir -p ./plugin/$$d/bin;	\
			echo Building $$d;		\
			xgo -buildmode=plugin -ldflags "-s -w" -targets $(TARGETS) -dest ./plugin/$$d/bin -pkg ./plugin/$$d -out $$d .;	\
			pushd . > /dev/null;	\
			cd ./plugin/$$d;		\
			find ./bin -type f -maxdepth 1 -name "$$d*" -exec mv {} {}.so \;; \
			if [[ -f "Makefile" ]]; then \
			make --no-print-directory build;	\
			fi; \
			popd > /dev/null;		\
		fi	\
	done
	@echo Success

## build: Build skynet binary file
build:
	@rm -rf $(OUTPUTDIR)
	@echo Building skynet
	@mkdir -p $(OUTPUTDIR)
	@mkdir -p $(OUTPUTDIR)/plugin
	@mkdir -p $(OUTPUTDIR)/assets/_plugin
	@xgo -ldflags "-s -w" -targets $(TARGETS) -dest $(OUTPUTDIR) .
	@cp LICENSE $(OUTPUTDIR)
	@cp conf.yml $(OUTPUTDIR)
	@cp default.webp $(OUTPUTDIR)
	@cp -r frontend/dist $(OUTPUTDIR)/assets
	@echo Success

## generate: Generate dynamic source code.
generate:
	@echo Generate code
	@go generate ./...
	@echo Success

## static_plugin: Build plugin static file.
static_plugin:
	@for d in `ls ./plugin | grep "^[^_]"`;do	\
		if [ -d ./plugin/$$d ];then	\
			pushd . > /dev/null;	\
			cd ./plugin/$$d;		\
			if [[ -f "Makefile" ]]; then \
			echo Building static file $$d;		\
			make --no-print-directory static;	\
			fi; \
			popd > /dev/null;		\
		fi	\
	done

## static: Build static file.
static:
	@cd frontend && \
	yarn && \
	yarn build
	@rm -rf assets
	@cp -r frontend/dist assets
 
## package: Package plugin.
package: build_plugin static_plugin packer
	@for d in `ls ./plugin | grep "^[^_]"`;do	\
		if [ -d ./plugin/$$d ];then	\
			pushd . > /dev/null;	\
			cd ./plugin/$$d;		\
			echo Packaging $$d;		\
			if [[ -f "Makefile" ]]; then \
			make --no-print-directory package;	\
			fi; \
			popd > /dev/null;		\
			./bin/packer* ./plugin/$$d/bin $(OUTPUTDIR)/$$d;	\
		fi	\
	done

## run: Run skynet(for develop use)
run:
	@for d in `ls ./plugin | grep "^[^_]"`;do	\
		if [ -d ./plugin/$$d ];then	\
			pushd . > /dev/null;	\
			cd ./plugin/$$d;					\
			echo Building $$d;		\
			go build -buildmode=plugin .;	\
			if [[ -f "Makefile" ]]; then \
			make --no-print-directory run;	\
			fi; \
			popd > /dev/null;		\
		fi	\
	done
	@echo Success
	@go run . run -v

## help: Show this help.
help: Makefile
	@echo Usage: make [command]
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
