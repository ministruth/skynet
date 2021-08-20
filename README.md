# skynet

Skynet is a service integration and management system, especially optimized for personal and home-lab use. With plugin support, you can easily embed whatever software you want to satisfy your need.

## Quick start

Skynet master program only support Linux system, we recommand docker image for simple start up.

1. `mkdir skynet && cd skynet`
2. Copy `conf.yml` and `docker-compose.yml` to the folder
3. `vim conf.yml` to modify your config
4. `touch data.db` or copy your exist database
5. `docker-compose up -d` to start skynet
6. visit `localhost:8080`

### Create initial user

You need to use the command line to add your initial user, you may add more user in web UI or use command line for batch add.

    skynet user add $username

For security reasons, we do not support specify certain initial password, you must save the password shown on screen, or use

    skynet user reset $username

to reset the password.

## Built-in plugin

We provide some built-in plugins to simplify some operation:

- [Monitor](plugin/monitor): All-in-one agent for server monitor, file transfer and shell manage.
- [Task](plugin/task): Long time task support for skynet.

## More plugins

You can find more plugin in our [official support plugin](#) or other user shares.

**!!Please note that all plugins have the same privilege as skynet, use trusted plugins ONLY!!**

## Develop
### Create 

1. Create folder under `plugins`, e.g. `plugins/myplugin`
2. Copy `template.tpl` to `plugins/myplugin/main.go`
3. Read instructions in file to create your plugin
4. Run `go build -buildmode=plugin .` to build and test your plugin
5. Use `github.com/techknowlogick/xgo` command `xgo -buildmode=plugin -ldflags "-s -w" -targets linux/amd64 -dest $(OUTPUTDIR) -pkg $(PLUGINDIR) -out $(NAME) .` to make release version of your plugin, recommand to rename it to strip `-linux-amd64` suffix.

## Reference Project

see [go.mod](go.mod)
