SHELL = /bin/bash
.SHELLFLAGS = -
OUTPUTDIR = ./bin
ID = 2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa

.ONESHELL:
.PHONY: static

static:
	@cd frontend &&	\
	yarn &&	\
	yarn build
	@rm -rf $(OUTPUTDIR)/assets
	@cp -r frontend/dist $(OUTPUTDIR)/assets
	@rm -rf ../../assets/_plugin/$(ID)
	@mkdir -p ../../assets/_plugin/$(ID)
	@cp -a frontend/dist/. ../../assets/_plugin/$(ID)