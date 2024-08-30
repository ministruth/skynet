SHELL = /bin/bash
OUTPUT_DIR = $$(pwd)/bin
TARGET_DIR = $$(pwd)/target
PLUGIN_DIR = $$(pwd)/plugin
PLUGIN_SUFFIX =

ifeq ($(OS),Windows_NT)
    PLUGIN_SUFFIX = .dll
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        PLUGIN_SUFFIX = .so
    endif
    ifeq ($(UNAME_S),Darwin)
        PLUGIN_SUFFIX = .dylib
    endif
endif

.ONESHELL:
.PHONY: check build run dev static clean help

all: help

## check: Check code and style.
check:
	@cargo clippy -- -D clippy::all
	@cargo fmt --all -- --check

## build: Build skynet(dev).
build:
	@echo Building...
	@cargo build
	@mkdir -p $(OUTPUT_DIR)
	@cp conf.dev.yml $(OUTPUT_DIR)/conf.yml
	@cp conf.schema.json $(OUTPUT_DIR)
	@cp default.webp $(OUTPUT_DIR)
	@cp $(TARGET_DIR)/debug/skynet $(OUTPUT_DIR)
	@mkdir -p $(OUTPUT_DIR)/plugin
	@for d in `ls $(PLUGIN_DIR)`;do	\
		if [ -f $(PLUGIN_DIR)/$$d/config.yml ];then		\
			mkdir -p $(OUTPUT_DIR)/plugin/$$d; 		\
			cp $(TARGET_DIR)/debug/lib$$d$(PLUGIN_SUFFIX) $(OUTPUT_DIR)/plugin/$$d; \
			cp $(PLUGIN_DIR)/$$d/config.yml $(OUTPUT_DIR)/plugin/$$d;	\
			if [ -f $(PLUGIN_DIR)/$$d/Makefile ];then	\
				t=$(TARGET_DIR);					\
				o=$(OUTPUT_DIR)/plugin/$$d;			\
				pushd . > /dev/null;				\
				cd $(PLUGIN_DIR)/$$d;					\
				make --no-print-directory build TARGET_DIR=$$t OUTPUT_DIR=$$o; 	\
				popd > /dev/null;					\
			fi										\
		fi											\
	done
	@echo Success

## run: Run skynet (dev).
run: build
	@cd $(OUTPUT_DIR) && RUST_BACKTRACE=1 ./skynet run -v --persist-session --disable-csrf

## dev: Run dev server, auto reload on save.
dev:
	@cargo watch -i frontend -- make run 

## static: Make static files.
static:
	@cd ./skynet/frontend && yarn && yarn build
	@mkdir -p $(OUTPUT_DIR)
	@rm -rf $(OUTPUT_DIR)/assets
	@cp -r ./skynet/frontend/dist/. $(OUTPUT_DIR)/assets && mkdir $(OUTPUT_DIR)/assets/_plugin
	@for d in `ls $(PLUGIN_DIR)`;do	\
		if [ -d $(PLUGIN_DIR)/$$d/frontend ];then								\
		    id=`cat $(PLUGIN_DIR)/$$d/config.yml | head -n 1 | cut -d \" -f 2`;	\
		    mkdir -p $(OUTPUT_DIR)/assets/_plugin/$$id;	\
			pushd . > /dev/null;						\
			cd $(PLUGIN_DIR)/$$d/frontend;					\
			yarn build; 								\
			popd > /dev/null;							\
			cp -r $(PLUGIN_DIR)/$$d/frontend/dist/. $(OUTPUT_DIR)/assets/_plugin/$$id; \
		fi												\
	done

## clean: Clean all build files.
clean:
	@rm -rf $(OUTPUT_DIR)
	@cargo clean

## help: Show this help.
help: Makefile
	@echo Usage: make [command]
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
