# 0.2.1
## Bug fix
1. Fix `no process-level CryptoProvider` for rustls 0.23. 

# 0.2.0
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

# plugin-0.2.0
## Breaking changes
1. Compatible to the new `skynet_api` design.
2. Redesign C/S framework.

## New features
1. Support passive agents.

# plugin-0.1.1
## Bug fix
1. Fix segmentation fault when loading plugins.
2. Fix monitor database foreign key error.

# 0.1.1
## Changes
1. `listen.ssl` will only raise warning when `proxy.enable` is `false`.

## Bug fix
1. Fix a IP parsing bug when proxy is enabled.
2. Fix CSP violation for reCAPTCHA.
3. Fix validation failed for reCAPTCHA.
4. Fix login db error when using PostgreSQL.
5. Fix plugin segmentation fault in some systems.
6. Fix segmentation fault when shared API is enabled.

# 0.1.0
First version of skynet!