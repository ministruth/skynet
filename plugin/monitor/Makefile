SHELL = /bin/bash
OUTPUTDIR = ./bin/plugin/monitor

.ONESHELL:
.PHONY: generate build

generate:
	@genny -in=../../sn/utils/map_tpl.go -out=./shared/map_cmdres_gen.go -pkg=shared gen "MPrefix=CMDRes MTypeA=uuid.UUID MTypeB=*CMDRes"
	@genny -in=../../sn/utils/map_tpl.go -out=./shared/map_socket_gen.go -pkg=shared gen "MPrefix=Socket MTypeA=uuid.UUID MTypeB=*Websocket"
	@genny -in=../../sn/utils/map_tpl.go -out=./shared/map_agent_gen.go -pkg=shared gen "MPrefix=Agent MTypeA=int MTypeB=*AgentInfo"

build:
	@pushd . > /dev/null
	@cd ../../
	@rm -rf $(OUTPUTDIR)/assets && cp -r ./plugin/monitor/assets $(OUTPUTDIR)/assets
	@rm -rf $(OUTPUTDIR)/templates && cp -r ./plugin/monitor/templates $(OUTPUTDIR)/templates
	@xgo -ldflags "-s -w" -targets linux/amd64 -dest $(OUTPUTDIR) -out agent -pkg ./plugin/monitor/agent .
	@popd > /dev/null