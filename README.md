# Docker Hub Tool

> [!WARNING]
> This is a custom fork of Docker's unmaintained `hub-tool`.

The Docker Hub Tool is a CLI tool for interacting with the
[Docker Hub](https://hub.docker.com).

## Building

### Prerequisites

- Go 1.26
- `make`

### Compiling

To build for your current platform, simply run `make` and the tool will be
output into the `./bin` directory:

```bash
$ make
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -ldflags=... -o bin/hub-tool_$(go env GOOS)_$(go env GOARCH) .
cp bin/hub-tool_$(go env GOOS)_$(go env GOARCH) bin/hub-tool

 $ ls bin/
 hub-tool
```

## Acknowledgments

- [Docker Hub Tool authors](https://github.com/docker/hub-tool/graphs/contributors)
