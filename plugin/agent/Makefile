SHELL = /bin/bash
.SHELLFLAGS = -
OUTPUTDIR = ../../bin/plugin/agent
PLATFORM =
MACHINE =
SUFFIX =

ifeq ($(OS),Windows_NT)
	PLATFORM = windows
	SUFFIX = .exe
	ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
		MACHINE = x64
	endif
	ifeq ($(PROCESSOR_ARCHITECTURE),x86)
		MACHINE = x86
	endif
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		PLATFORM = linux
	endif
	ifeq ($(UNAME_S),Darwin)
		PLATFORM = osx
	endif
	UNAME_M := $(shell uname -m)
	ifeq ($(UNAME_M),x86_64)
		MACHINE = x64
	endif
	ifneq ($(filter %86,$(UNAME_M)),)
		MACHINE = x86
	endif
	ifeq ($(UNAME_M),arm64)
		MACHINE = arm64
	endif
	ifeq ($(UNAME_M),arm)
		MACHINE = arm
	endif
endif

build:
	@mkdir -p $(OUTPUTDIR)/bin
	@cp ../../target/debug/agent $(OUTPUTDIR)/bin/agent_$(PLATFORM)_$(MACHINE)$(SUFFIX);