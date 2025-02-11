# skynet

![version](https://img.shields.io/badge/version-0.2.5-blue?style=flat-square) ![api](https://img.shields.io/badge/api-0.2.9-light_green?style=flat-square) ![rustc](https://img.shields.io/badge/rustc-1.56+-red?style=flat-square) ![license](https://img.shields.io/github/license/ministruth/skynet?style=flat-square)

Skynet is a service integration and management system, specially optimized for personal and home-lab use. With plugin support, you can easily embed whatever software you want to satisfy your need.

Security is considered as **TOP** priority in Skynet, we will not consider features that conflict with our security policy. If you find vulnerabilities, please report ASAP.

## Quick start

### Run in docker
We offer pre-built `x86_64` and `aarch64` docker images.

1. Copy `docker-compose.yml` and `conf.yml` to your folder.
2. `docker-compose up`
3. Visit `localhost:8080`.

### Run natively
**We do not recommend this method, use at your own risk!**

You can download pre-built libraries in our [release](https://github.com/MXWXZ/skynet/releases) page.

We offer `linux-{x86_64,i686,aarch64}`, `darwin-{x86_64,aarch64}` and `windows-x86_64` binaries. You might build from source if your platform is not included.

1. Download the release and extract.
2. `vim conf.yml` to modify your config.
3. `touch data.db` or copy your existing database.
4. `./skynet check` to verify your config.
5. `./skynet run` to start up the server.
6. Visit `localhost:8080`.

### Build from source

```
make build_release
make output BUILD_TYPE=release
make static
```

You are ready to go with files in `bin` :)

### Create initial user

You must use the command line to initialize your root user:

    skynet user init

Like linux, root user can ignore all built-in privilege checkers. Remember your initial randomized password, you can change it after login.

You may add more users in the web UI or use the command line for batch add:

    skynet user add <USERNAME>

Note that no permission is allowed for these users.

## Optional features
### Redis
You can enable redis to replace the built-in memory database.

1. Install redis
2. Change `redis.enable` to `true`.
3. Modify `redis.dsn` to connect your database.

### SSL
Enable SSL to secure your connection.

1. Get your SSL certificate (certificate `*.crt` and key `*.key`)
2. Change `listen.ssl` to `true`.
3. Modify `listen.ssl_cert` and `listen.ssl_key` to file path.

### Proxy
You need to enable this if Skynet is behind some kind of proxy (Nginx, Load balancer, etc.).
Otherwise, you cannot obtain users' real IP.

1. Change `proxy.enable` to `true`.
2. Modify `proxy.header` to the `ip:port` header set by the proxy server.
   
   Config example:
   ```
   proxy:
     header: "X-Real-Address"
   ```
   The proxy server should pass the following header (peer ip/port):
   ```
   X-Real-Address: 192.168.0.1:12345
   ```

### Recaptcha
You can enable Google recaptcha to protect against brute force attack.

1. Register recaptcha site on https://www.google.com/recaptcha/about/ (Type: V2 Checkbox).
2. Change `recaptcha.enable` to `true`.
3. Modify `recaptcha.sitekey` and `recaptcha.secret`.

### Geoip
For every IP address, you can enable geoip to view the country directly.

1. Get `GeoLite2-Country.mmdb` from Github or maxmind official website.
2. Change `geoip.enable` to `true`.
3. Modify `geoip.database` to the `.mmdb` file path.

## Plugins

You can find plugins in our [organization repositories](https://github.com/ministruth) or other user shares.

Use our script to download all official plugins:

```
./get_offical_plugins.sh
```

**!!Please note that all plugins have the same privilege as skynet, use trusted plugins ONLY!!**

**In other words, run untrusted plugin = RCE**

## Develop

See [develop note](develop.md).

## FAQ

### How to trace logs?

Skynet provides pretty formatted or JSON formatted logs. You can get `WARN` and `ERROR` logs from `stderr` and others from `stdout`. Execute `skynet run -h` for more details. We only guarantee the order of printed logs. Please do not rely on notifications in database (see below).

### Why do timestamps of notifications in database differ from those in console?

To avoid database deadlock and performance issues, notifications are written to database asynchronously. We do not guarantee the order and success of notifications.
