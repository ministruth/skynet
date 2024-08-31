SHELL = /bin/bash

static:
	@rm -rf $(OUTPUT_DIR) && mkdir -p $(OUTPUT_DIR)
	@cd frontend && yarn && yarn build
	@cp -r frontend/dist/. $(OUTPUT_DIR)