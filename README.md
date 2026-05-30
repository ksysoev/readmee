# ReadMee

[![Tests](https://github.com/ksysoev/readmee/actions/workflows/tests.yml/badge.svg)](https://github.com/ksysoev/readmee/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ksysoev/readmee)](https://goreportcard.com/report/github.com/ksysoev/readmee)
[![Go Reference](https://pkg.go.dev/badge/github.com/ksysoev/readmee.svg)](https://pkg.go.dev/github.com/ksysoev/readmee)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

SSH platform for hosting developers profiles

## Installation

## Building from Source

```sh
RUN CGO_ENABLED=0 go build -o readmee -ldflags "-X main.version=dev -X main.name=readmee" ./cmd/readmee/main.go
```

### Using Go

If you have Go installed, you can install ReadMee directly:

```sh
go install github.com/ksysoev/readmee/cmd/readmee@latest
```


## Using

```sh
readmee --log-level=debug --log-text=true --config=runtime/config.yml
```

## License

ReadMee is licensed under the MIT License. See the LICENSE file for more details.
