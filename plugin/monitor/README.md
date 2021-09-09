# Skynet plugin - monitor

This plugin provide agent monitor and management for skynet, most other plugins depends on this.

## Information

- UUID: 2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa
- Name: monitor
- Version: 1.0.0
- SkynetVersion: >= 1.0, < 1.1
- Priority: 0

## Quick start

Monitor is designed for All operating system, currently only support Linux. To monitor your machine and make command feature available, the agent program should NOT run in docker.

1. Generate a connect token at Skynet for safety
2. Download your platform agent
3. Connect to skynet like `agent-linux-amd64 -t $token localhost:8080`

## Plugin API

Full API at [interface.go](shared/interface.go)

## Features

### Self update

The agent can update itself when master updates, no user interaction needed.

### Agent monitor

Monitor provides agent status monitor such as CPU/Memory/Disk usage and so on.

### Agent setting

Monitor provides settings for each agent, you can use it to boost your plugin.

### File distribution

Monitor provides simple file distribution function, you can transfer files to certain agent.
**Note that this function is designed for small file transfer, you may meet other function error when transfering large files. We suggest transfer a shell file and download large files using command.**

### Agent command

Monitor provides asynchronous or synchronize command run in the agent, also provides a web-based shell for user.

## Security

Communication security is protected by SSL, all your traffic is plain text when ssl is not enabled, add option `-s` to use SSL(skynet must run in ssl mode).

Connection is also protected by connection token, you should generate a token when enabling the plugin(a warning will be shown when token is empty).

**Do NOT connect to any untrusted skynet for they will have full access to your system.**