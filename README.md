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

The main package containing various utility functions. These functions might panic on unexpected input or errors.

### `safetool`

The `safetool` package provides alternative implementations of functions found in the `tool` package. These "safe" versions are designed to return an `error` instead of muting errors, allowing for more controlled error handling in your applications.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue if you have suggestions or find a bug.

1.  Fork the Project
2.  Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the Branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

## 📜 License

This project is licensed under the [MIT License](LICENSE).