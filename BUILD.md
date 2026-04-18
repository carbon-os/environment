# BUILD.md

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Delve](https://github.com/go-delve/delve) (optional, for debugging)

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

---

## Repository Layout

```
carbon-os/environment/
  go.mod
  *.go                 # library
  provider/
    apt/
      *.go             # apt provider
  cmd/
    main.go            # env CLI
```

---

## Running Locally

```bash
# compile and run without producing a binary
go run ./cmd/main.go <command>

# examples
go run ./cmd/main.go create myenv
go run ./cmd/main.go install gcc@13 --platform=ubuntu:22.04
```

---

## Building a Binary

```bash
# output binary named 'env' in the repo root
go build -o env ./cmd/main.go

./env create myenv
./env install gcc@13 --platform=ubuntu:22.04
./env lock
```

---

## Installing to $PATH

```bash
go install ./cmd/main.go
```

Installs the binary to `$(go env GOPATH)/bin`. Make sure that directory is in
your `$PATH`:

```bash
# add to ~/.zshrc or ~/.bashrc if not already present
export PATH="$(go env GOPATH)/bin:$PATH"
```

Then call it directly from anywhere:

```bash
env create myenv
env use myenv
```

---

## Dependencies

```bash
# sync go.sum and remove unused deps
go mod tidy
```

---

## Testing

```bash
go test ./...              # all packages
go test ./... -v           # verbose output
go test ./provider/apt/    # apt provider only
go test ./... -run TestName # single test by name
```

---

## Debugging

```bash
# debug the CLI — flags and args go after --
dlv debug ./cmd/main.go -- install gcc --platform=debian:12

# debug a specific package's tests
dlv test ./provider/apt/
```

For quick print-style debugging, use `log.Printf` — it writes to stderr and
won't interfere with CLI stdout:

```go
import "log"
log.Printf("meta: %+v", meta)
```

---

## Inner Loop

```bash
go build -o env ./cmd/main.go && ./env <command>
```

Rebuild and run in one step. If the build fails the command never runs.

---

## Lint and Vet

```bash
go vet ./...
```

Optional — install [staticcheck](https://staticcheck.io/) for deeper analysis:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```