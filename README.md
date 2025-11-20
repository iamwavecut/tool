# 🛠️ Tool Utilities

[![Go Reference](https://pkg.go.dev/badge/github.com/iamwavecut/tool.svg)](https://pkg.go.dev/github.com/iamwavecut/tool)
[![Go Report Card](https://goreportcard.com/badge/github.com/iamwavecut/tool)](https://goreportcard.com/report/github.com/iamwavecut/tool)
[![Test Coverage](https://img.shields.io/badge/coverage-TBD-lightgrey)](https://goreportcard.com/report/github.com/iamwavecut/tool) 

This library provides a collection of utility functions designed to simplify common tasks and improve code robustness. It includes a main `tool` package and a `safetool` sub-package, where functions are designed to return errors instead of panicking.

## ✨ Features

-   **Convenient Utilities**: A collection of helper functions for various tasks.
-   **Safe Alternatives**: The `safetool` sub-package offers functions that return errors, promoting more resilient error handling.
-   **Well-Tested**: Includes a comprehensive test suite to ensure reliability.

## 🚀 Installation

To install the `tool` package, use `go get`:

```bash
go get -u github.com/iamwavecut/tool
```

## 📦 Packages

This repository contains the following main packages:

### `tool`

The main package containing various utility functions. These functions will return zero values or errors instead of panicking on unexpected input or errors.

### `safetool`

The `safetool` package provides alternative implementations of functions found in the `tool` package. These "safe" versions are designed to return an `error` instead of muting errors, allowing for more controlled error handling in your applications.

---

#### Common exported functions (with identical interfaces in both packages):

*   `In(needle, ...haystack) bool` - deprecated in favor of `slices.Contains`
*   `IsZero(comparable) bool` - returns true if the value is the zero value for its type
*   `NilPtr(comparable) *comparable` - returns a pointer to the input value if it's not zero, otherwise returns nil
*   `NonZero(comparable) comparable` - returns the first non-zero value from the input arguments
*   `Ptr(any) *any` - returns a pointer to the input value
*   `RetryFunc(attempts int, sleep time.Duration, f func() error) error` - retries a function with exponential backoff
*   `Strtr(subject string, oldToNew map[string]string) string` - replaces substrings in a string based on a mapping
*   `Val(*any) any` - returns the value pointed to by the pointer, or zero value if pointer is nil
*   `ZeroVal(any) any` - returns the zero value of the type, regardless of the input value

---

#### `tool`-specific exported functions:

*   `Catch(func(error))` - catches panics and calls a function with the error
*   `Console(...any)` - prints to the console
*   `ConvertSlice([]T, Y) []Y` - converts a slice of one type to another
*   `Err(...any) error` - returns an error with the input arguments
*   `ExecTemplate(string, []|map) string` - executes a template
*   `Jsonify(any) safetool.Varchar` - returns a Varchar from the input value
*   `MultiMute(...any) []any` - returns a slice of the input arguments with no latest error
*   `Must(error, verbose ?bool)` - panics if the error is not nil
*   `MustReturn(any, error) any` - panics if the error is not nil
*   `Objectify(in any, target any) bool` - decodes a JSON string into the target object
*   `RandInt(min, max int) int` - returns a random integer between min and max
*   `Recoverer(maxPanics int, f func(), jobID ...string) error` - recovers from panics and returns the error
*   `Return(any, _ error) any` - returns the input value
*   `SetLogger(l StdLogger)` - sets the logger
*   `Try(err error, verbose ?bool) bool` - returns true if the error is nil

---

#### `safetool`-specific exported functions:

*   `ConvertSlice([]T, Y) ([]Y, error)` - converts a slice of one type to another
*   `ExecTemplate(string, []|map) (string, error)` - executes a template
*   `FindRootCaller() string` - returns the root caller filepath of the application
*   `GetRelativePath(string) (string, error)` - returns a relative path from the directory of the root caller to the given filePath
*   `Jsonify(any) (safetool.Varchar, error)` - returns an encoded JSON string from the input value represented as a `Varchar`
*   `Objectify(in any, target any) error` - decodes a JSON string into the target object
*   `RandInt(min, max int) (int, error)` - returns a random integer between min and max
*   `Zero(any) any` - returns the zero value for the input type

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue if you have suggestions or find a bug.

1.  Fork the Project
2.  Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the Branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

## 📜 License

This project is licensed under the [MIT License](LICENSE).
