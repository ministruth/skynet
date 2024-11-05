SHELL = /bin/bash
OUTPUT_DIR = $$(pwd)/bin
BUILD_TYPE = debug
TARGET_DIR = $$(pwd)/target/$(BUILD_TYPE)
PLUGIN_DIR = $$(pwd)/plugin
EXE_SUFFIX =
PLUGIN_SUFFIX =

ifeq ($(OS),Windows_NT)
	EXE_SUFFIX = .exe
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

.PHONY: check help build build_release clean static static_plugin run dev output build_plugin build_plugin_release

all: help

## check: Check code and style.
check:
	@cargo clippy -- -D clippy::all
	@cargo fmt --all -- --check

## build: Build skynet(debug).
build:
	@cargo build

## build_release: Build skynet(release).
build_release:
	@cargo build --locked --release

## build_plugin: Build plugin(debug).
build_plugin:
	@for d in `ls $(PLUGIN_DIR)`;do					\
 		if [[ -f $(PLUGIN_DIR)/$$d/config.yml ]];then	\
 			echo Building $$d...;					\
 			pushd . > /dev/null;					\
 			cd $(PLUGIN_DIR)/$$d;					\
 			cargo build; 							\
 			popd > /dev/null;						\
 		fi											\
 	done

## build_plugin_release: Build plugin(release).
build_plugin_release:
	@for d in `ls $(PLUGIN_DIR)`;do					\
 		if [[ -f $(PLUGIN_DIR)/$$d/config.yml ]];then	\
 			echo Building $$d...;					\
 			pushd . > /dev/null;					\
 			cd $(PLUGIN_DIR)/$$d;					\
 			cargo build --locked --release; 		\
 			popd > /dev/null;						\
 		fi											\
 	done

## output: Output build files from TARGET_DIR to OUTPUT_DIR (bin), not delete OUTPUT_DIR.
output:
	@echo OUTPUT_DIR=$(OUTPUT_DIR)
	@echo TARGET_DIR=$(TARGET_DIR)
	@echo Output Skynet...
	@mkdir -p $(OUTPUT_DIR)
	@cp conf.yml $(OUTPUT_DIR)
	@cp conf.schema.json $(OUTPUT_DIR)
	@cp default.webp $(OUTPUT_DIR)
	@cp $(TARGET_DIR)/skynet$(EXE_SUFFIX) $(OUTPUT_DIR)
	@rm -rf $(OUTPUT_DIR)/plugin && mkdir -p $(OUTPUT_DIR)/plugin
	@for d in `ls $(PLUGIN_DIR)`;do							\
		if [[ -f $(PLUGIN_DIR)/$$d/Makefile && -f $(PLUGIN_DIR)/$$d/config.yml ]];then			\
			echo Output $$d...;								\
			o=$(OUTPUT_DIR)/plugin;							\
			pushd . > /dev/null;							\
			cd $(PLUGIN_DIR)/$$d;							\
			make --no-print-directory output OUTPUT_DIR=$$o;\
			popd > /dev/null;								\
		fi													\
	done

## run: Run skynet (debug).
run: build build_plugin output
	@cp conf.dev.yml $(OUTPUT_DIR)/conf.yml
	@cd $(OUTPUT_DIR) && RUST_BACKTRACE=1 ./skynet run -v --persist-session --disable-csrf

## dev: Run dev server, auto reload on save.
dev:
	@cargo watch -i frontend -- make run 

## static: Build static files, delete assets folders.
static:
	@echo OUTPUT_DIR=$(OUTPUT_DIR)
	@cd ./skynet/frontend && yarn && yarn build
	@mkdir -p $(OUTPUT_DIR)
	@rm -rf $(OUTPUT_DIR)/assets
	@cp -r ./skynet/frontend/dist/. $(OUTPUT_DIR)/assets && mkdir $(OUTPUT_DIR)/assets/_plugin

## static_plugin: Build static plugin files, delete assets/_plugin folder.
static_plugin:
	@echo OUTPUT_DIR=$(OUTPUT_DIR)
	@rm -rf $(OUTPUT_DIR)/assets/_plugin && mkdir -p $(OUTPUT_DIR)/assets/_plugin
	@for d in `ls $(PLUGIN_DIR)`;do					\
		if [[ -f $(PLUGIN_DIR)/$$d/Makefile && -f $(PLUGIN_DIR)/$$d/config.yml ]];then	\
			echo Building $$d...;					\
		    id=`cat $(PLUGIN_DIR)/$$d/config.yml | head -n 1 | cut -d \" -f 2`;	\
			o=$(OUTPUT_DIR)/assets/_plugin;	\
			pushd . > /dev/null;					\
			cd $(PLUGIN_DIR)/$$d;					\
			make --no-print-directory static OUTPUT_DIR=$$o; 	\
			popd > /dev/null;						\
		fi											\
	done
	@mv $(OUTPUT_DIR)/assets/_plugin/assets/* $(OUTPUT_DIR)/assets/_plugin && rm -rf $(OUTPUT_DIR)/assets/_plugin/assets

## clean: Clean all build files.
clean:
	@rm -rf $(OUTPUT_DIR)
	@cargo clean
	@rm -rf skynet/frontend/dist && rm -rf skynet/frontend/node_modules 
	@for d in `ls $(PLUGIN_DIR)`;do					\
 		if [[ -f $(PLUGIN_DIR)/$$d/Makefile ]];then	\
 			echo Cleaning $$d...;					\
 			pushd . > /dev/null;					\
 			cd $(PLUGIN_DIR)/$$d;					\
 			make --no-print-directory clean; 		\
 			popd > /dev/null;						\
 		fi											\
 	done

## help: Show this help.
help: Makefile
	@echo Usage: make [command]
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
