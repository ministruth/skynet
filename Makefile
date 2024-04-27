SHELL = /bin/bash
.SHELLFLAGS = -
OUTPUTDIR = ./bin
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
.PHONY: run build help

all: help

## check: Check code using clippy.
check:
	@cargo clippy -- -W clippy::all -W clippy::pedantic -W clippy::nursery -W clippy::restriction \
	-A clippy::future_not_send -A clippy::type_repetition_in_bounds -A clippy::module_name_repetitions \
	-A clippy::single_call_fn -A clippy::shadow_reuse -A clippy::multiple_unsafe_ops_per_block -A clippy::pattern_type_mismatch \
	-A clippy::unwrap_used -A clippy::question_mark_used -A clippy::min_ident_chars -A clippy::implicit_return \
	-A clippy::std_instead_of_core -A clippy::indexing_slicing -A clippy::let_underscore_untyped \
	-A clippy::clone-on-ref-ptr -A clippy::let_underscore_must_use -A clippy::missing_inline_in_public_items \
	-A clippy::unreachable -A clippy::std_instead_of_alloc -A clippy::mod_module_files -A clippy::missing_trait_methods \
	-A clippy::string_add -A clippy::exhaustive_structs -A clippy::exhaustive_enums -A clippy::shadow_unrelated \
	-A clippy::arithmetic_side_effects -A clippy::shadow_same -A clippy::error_impl_error -A clippy::unwrap_in_result \
	-A clippy::panic -A clippy::wildcard_enum_match_arm -A clippy::default_numeric_fallback -A clippy::single_char_lifetime_names \
	-A clippy::partial_pub_fields -A clippy::missing_docs_in_private_items -A clippy::pub_use -A clippy::expect_used \
	-A clippy::print_stdout -A clippy::blanket_clippy_restriction_lints -A clippy::should_implement_trait -A clippy::similar_names \
	-A clippy::as_conversions -A clippy::significant_drop_in_scrutinee -A clippy::use_debug -A clippy::match_wildcard_for_single_variants \
	-A clippy::separated_literal_suffix -A clippy::significant_drop_tightening -A clippy::too-many-arguments \
	-A clippy::iter-over-hash-type -A clippy::no-effect-underscore-binding -A clippy::redundant-else

## build: Build skynet(dev).
build:
	@echo Building...
	@cargo build
	@mkdir -p $(OUTPUTDIR)
	@cp conf.yml $(OUTPUTDIR)
	@cp conf.schema.json $(OUTPUTDIR)
	@cp default.webp $(OUTPUTDIR)
	@cp ./target/debug/skynet $(OUTPUTDIR)
	@mkdir -p $(OUTPUTDIR)/plugin
	@for d in `ls ./plugin`;do	\
		if [ -f ./plugin/$$d/config.yml ];then		\
			mkdir -p $(OUTPUTDIR)/plugin/$$d; 		\
			cp ./target/debug/lib$$d$(PLUGIN_SUFFIX) $(OUTPUTDIR)/plugin/$$d; \
			cp ./plugin/$$d/config.yml $(OUTPUTDIR)/plugin/$$d;	\
			if [ -f ./plugin/$$d/Makefile ];then	\
				pushd . > /dev/null;				\
				cd ./plugin/$$d;					\
				make --no-print-directory build; 	\
				popd > /dev/null;					\
			fi										\
		fi											\
	done
	@echo Success

## run: Run skynet(dev).
run: build
	@cd $(OUTPUTDIR) && ./skynet run -v --persist-session --disable-csrf

## dev: Run dev server, auto reload on save.
dev:
	@cargo watch -i frontend -- make run 

## static: make static files.
static:
	@cd ./skynet/frontend && yarn && yarn build
	@mkdir -p $(OUTPUTDIR)
	@rm -rf $(OUTPUTDIR)/assets
	@cp -r ./skynet/frontend/dist/. $(OUTPUTDIR)/assets && mkdir $(OUTPUTDIR)/assets/_plugin
	@for d in `ls ./plugin`;do	\
		if [ -d ./plugin/$$d/frontend ];then								\
		    id=`cat ./plugin/$$d/config.yml | head -n 1 | cut -d \" -f 2`;	\
		    mkdir -p $(OUTPUTDIR)/assets/_plugin/$$id;	\
			pushd . > /dev/null;						\
			cd ./plugin/$$d/frontend;					\
			yarn build; 								\
			popd > /dev/null;							\
			cp -r ./plugin/$$d/frontend/dist/. $(OUTPUTDIR)/assets/_plugin/$$id; \
		fi												\
	done

## clean: clean all build files
clean:
	@rm -rf $(OUTPUTDIR)
	@cargo clean

## help: Show this help.
help: Makefile
	@echo Usage: make [command]
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
