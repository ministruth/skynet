# Develop note

## Environment

[xgo](https://github.com/techknowlogick/xgo)
[apidoc](https://apidocjs.com/)
[genny](https://github.com/cheekybits/genny)
[protobuf](https://github.com/protocolbuffers/protobuf/

## API format

API parameters are in JSON format, so do return values.

We only support these http status code:
- HTTP 200: OK.
- HTTP 301/302: redirect.
- HTTP 400: parameters are not correct.
- HTTP 403: permission denied or path not found, in order to prevent plugin guessing.
- HTTP 500: internal error.

When status code is 200, return value is in general format:
```
{
    "code": 0,        // 0 for success, other for error code
    "msg": "Success", // message show on front end
    "data": ...,      // any structured data
    "total": ...,     // when using pagination
}
```

## Error handle

We use [tracerr](https://github.com/ztrue/tracerr) for stack trace and error wrapper, [logrus](https://github.com/sirupsen/logrus) for error logger.

- Error will be returned to top function to handle, or just don't return error if the function is fault tolerant or never fail(you must handle it inside for yourself).
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
            // handle or loafter thatg it for yourself
            return some_object
        }
        ...
        return some_other_object
    }
    ```
- All skynet function will return tracerr wrapped error to enable stack trace, all third-part or golang standard error should be thrown wrapped.
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
- You must use `utils.WithTrace` or `utils.WithLogTrace` for error log to trace stack.
- In http handler, prefer throw error to trigger 500 response other than `panic`, never use `log.Fatal`.

## Interface promise

- Pointer parameters passed usually **transfer ownership** to the function, that means you should not share or change the content after calling the function.
  - For example, you have slice `a []type` and passed to `AddSlice(a)`, the interface may take control of `a` so you shouldn't change `a` after that to prevent inconsistency.
- Pointer values returned usually **keeps ownership**, that means you should not share or change the content after calling the function, depending on the meaning.
  - For example, you call `GetSlice() []type`, that usually returns the inner structure, so you shouldn't change to prevent inconsistency. 
  - However, something like `NewSlice() []type` is certainly safe to change the value after that because it will create new one for you.

## Plugin Develop

### Makefile

**Plugin `bin` folder is reserved**