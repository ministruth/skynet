# Develop note
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

    func FaultTolerant() object{
        ...
        if err != nil {
            // handle or log it for yourself
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