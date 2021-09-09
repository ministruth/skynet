# Skynet plugin - Task

This plugin provide asynchronous task execute and status tracking for skynet.

With `monitor` plugin enabled, it can also provide task execute in each agent.

## Information

- UUID: c1e81895-1f75-4988-9f10-52786b875ec7
- Name: task
- Version: 1.0.0
- SkynetVersion: >= 1.0, < 1.1
- Priority: 0

## Plugin API

Full API at [interface.go](shared/interface.go)

## Features

### Task status tracking

Task provides long time task status tracking with asynchronous task execute function.

### Task execute in agent

With `monitor` plugin enabled, Task can also support execute task in each agent.