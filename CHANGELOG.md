# plugin-v2024090102
## Components
- agent: 0.3.0 => 0.3.1
- skynet_api_agent: 0.3.0 => 0.3.1
- monitor: 0.2.4 => 0.2.5

## Changes
1. agent now use rust env to determine OS type.

## Bug fix
1. `agent`, `monitor`: Fix potential data corrupt.
2. `agent`: Fix potential dead lock.

# plugin-v2024090101
## Components
- monitor: 0.2.3 => 0.2.4

## Bug fix
1. Fix passive agent connect bug.

# plugin-v2024083102
## Components
- monitor: 0.2.2 => 0.2.3

## Bug fix
1. Fix postgres backend bug.

# v0.2.2
## Bug fix
1. Fix postgres backend bug.

# plugin-v2024083101
## Components
- agent: 0.2.1 => 0.3.0
- skynet_api_agent: 0.2.1 => 0.3.0
- monitor: 0.2.1 => 0.2.2

## New features
1. `agent`: Support `restart` option.
2. `monitor`, `skynet_api_agent`: Version check will depend on `agent` version instead of `skynet_api_agent` version.

# v0.2.1
## Bug fix
1. Fix `no process-level CryptoProvider` for rustls 0.23. 

# plugin-v0.2.1
## Bug fix
1. Fix `no process-level CryptoProvider` for rustls 0.23. 

# v0.2.0
## Breaking changes
1. `skynet` crate is split to `skynet` and `skynet_api`, plugins now should depend on `skynet_api`.
2. Based on new framework `actix-cloud`.
3. Most system is re-designed.

## Changes
1. Only warning and error logs will increase the unread count.
2. Allow change root username.
3. `/ping` is renamed to `/health`.

## Bug fix
1. Fix a bug that prevent success logs written to the database.

# plugin-v0.2.0
## Breaking changes
1. Compatible to the new `skynet_api` design.
2. Redesign C/S framework.

## New features
1. Support passive agents.

# plugin-v0.1.1
## Bug fix
1. Fix segmentation fault when loading plugins.
2. Fix monitor database foreign key error.

# v0.1.1
## Changes
1. `listen.ssl` will only raise warning when `proxy.enable` is `false`.

## Bug fix
1. Fix a IP parsing bug when proxy is enabled.
2. Fix CSP violation for reCAPTCHA.
3. Fix validation failed for reCAPTCHA.
4. Fix login db error when using PostgreSQL.
5. Fix plugin segmentation fault in some systems.
6. Fix segmentation fault when shared API is enabled.

# v0.1.0
First version of skynet!