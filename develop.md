# Develop note

## Environment

- rust
- clippy
- [cargo watch](https://github.com/watchexec/cargo-watch)
- node/yarn

## API format

API parameters are in JSON format, so do return values.

We only support these HTTP status codes:
- HTTP 200: OK.
- HTTP 301/302: redirect.
- HTTP 400: parameters incorrect.
- HTTP 403: permission denied.
- HTTP 404: resource not found.
- HTTP 500: internal error.

When the status code is 200, the return value is in the general format:
```
{
    "code": 0,        // 0 for success, other for error code
    "msg": "Success", // code message, translated automatically through `lang` query param
    "data": ...,      // any structured data (optional)
}
```
