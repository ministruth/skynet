# skynet

![status](https://img.shields.io/badge/status-dev-lightgrey?style=flat-square) ![rustc](https://img.shields.io/badge/rustc-1.56+-red?style=flat-square)

Skynet is a service integration and management system, specially optimized for personal and home-lab use. With plugin support, you can easily embed whatever software you want to satisfy your need.

Security is considered as **TOP** priority in Skynet, we will not consider features that conflict with our security policy. If you find vulnerabilities, please report ASAP.

## Quick start

### Run in docker

### Run natively

**We do not recommend this method, use at your own risk!**

1. Download the release and extract.
2. Install redis on your machine.
3. `vim conf.yml` to modify your config.
4. `touch data.db` or copy your existing database.
5. `./skynet check` to verify your config.
6. `./skynet run` to start up the server.
7. Visit `localhost:8080`.

### Create initial user

You must use the command line to initialize your root user:

    skynet user init

Like linux, root user can ignore all built-in privilege checkers. Remember your initial randomized password, you can change it after login.

You may add more users in the web UI or use the command line for batch add:

    skynet user add <USERNAME>

Note that no permission is allowed for these users.

## Plugins

You can find plugins in our [official support plugin](https://github.com/MXWXZ/skynet/plugin) or other user shares.

**!!Please note that all plugins have the same privilege as skynet, use trusted plugins ONLY!!**

## Develop

See [develop note](develop.md).
