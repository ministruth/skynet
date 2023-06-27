# Develop note

## Environment

- [swag](https://github.com/swaggo/swag)
- [protobuf](https://github.com/protocolbuffers/protobuf/)

## API format

API parameters are in JSON format, so do return values.

We only support these HTTP status codes:
- HTTP 200: OK.
- HTTP 301/302: redirect.
- HTTP 400: parameters are not correct.
- HTTP 403: permission denied.
- HTTP 404: path not found.
- HTTP 500: internal error.

When the status code is 200, the return value is in the general format:
```
{
    "code": 0,        // 0 for success, other for error code
    "msg": "Success", // code message
    "data": ...,      // any structured data
}
```

## Error handle

We use [tracerr](https://github.com/ztrue/tracerr) for stack trace and error wrapper, [logrus](https://github.com/sirupsen/logrus) for error logger.

You can use `utils/log` as the wrapper for `logrus`.

- Error will be returned to the top function to handle, or just don't return if the function is fault-tolerant or never fail(you must handle it inside for yourself).
    ```
    func ErrorFunc() error {
        ...
        return err // do not log error or handle it
    }

    func NeverFail() {
        ...
        if err != nil {
            // handle or log it for yourself
            panic(err)
        }
    }

    func FaultTolerant() object {
        ...
        if err != nil {
            // handle or log it for yourself
            return some_object
        }
        ...
        return some_other_object
    }
    ```
- All skynet functions will return `tracerr` wrapped error to enable stack trace, all third-part or golang standard error should be thrown wrapped.
    ```
    // no need to wrap
    err := skynet_func()
    if err != nil {
        return err
    }

    // need to wrap
    err := tracerr.Wrap(golang_or_thirdpard_func())
    if err != nil {
        return err
    }
    ```
- Error wrap should be closest to error function for better debugging
    ```
    // good
    err := tracerr.Wrap(error_func())

    ret, err := error_func()
    if err != nil {
        return tracerr.Wrap(err)
    }

    // bad
    err := error_func()
    if err != nil {
        return tracerr.Wrap(err)
    }
    ```
- In HTTP handler, prefer throw error to trigger 500 response other than `panic`, never use `log.Fatal`.
