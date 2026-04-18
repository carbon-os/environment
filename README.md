# environment

A cross-platform, language-agnostic package and environment manager — at the OS level.

The binary is `env`. The repo is `carbon-os/environment`.

---

`env` creates isolated, reproducible package environments scoped to a project
or machine. Define your dependencies once, lock them, and reproduce the exact
same environment anywhere — on any OS, any arch, any layer of the stack.

Environments are first-class citizens. Each one is a completely self-contained
scope with its own packages, versions, and lock file — fully independent from
your host system and from every other environment. Switch between projects,
switch environments. Nothing conflicts. Nothing bleeds.

- **Host-safe** — never modifies your system. Install, remove, and rebuild
  environments freely without touching anything outside of them.

- **IDE and workflow ready** — colocate an environment with any project, activate
  it per-repo, and your entire team works from the same dependency scope.

- **Dependency isolation, solved** — mixing dependencies across projects breaks
  hosts. `env` scopes everything cleanly so each project gets exactly what it
  needs, nothing more.

- **Works at any layer** — bare metal, inside a container, inside a VM. `env`
  runs on the real OS and solves isolation at whatever level it lands on.

- **Cross-platform targeting** — install packages for any target platform from
  any host. Running macOS? Install a Debian package. Running Linux? Install a
  Windows package. It's just downloading and unpacking files.

- **Backed by native providers** — brew, apt, winget. Package platforms you
  already trust, now managed in a controlled and reproducible way.

---

## Install

**macOS / Linux**
```bash
wget -qO- https://raw.githubusercontent.com/carbon-os/environment/main/install.sh | sh
```

**Windows (PowerShell)**
```powershell
irm https://raw.githubusercontent.com/carbon-os/environment/main/install.ps1 | iex
```

Or download the binary directly from the [releases page](https://github.com/carbon-os/environment/releases).

The binary is a single static Go binary. No runtime dependencies.

---

## Concepts

### Environment
An isolated collection of packages tied to a name. Environments live in
`~/.env/envs/<name>/` by default, or at a custom path if specified at
create time. Each has its own `index.toml`, `index.lock`, and `bin/`.

### index.toml
The source of truth. Declares packages, versions, which provider to use
per platform, and the environment's resolved path. You edit this.

### index.lock
The resolved output. Stamps exact versions and providers per platform after
running `env lock`. You commit this. You never hand-edit it.

---

## Quick Start

```bash
env create myenv        # create a new environment
env use myenv           # activate it

env install gcc         # install a package (provider auto-detected from host)
env install gcc@13      # install a pinned version

env list                # show installed packages
env lock                # freeze to index.lock
```

---

## Commands

```bash
env create <name>                          # create a new environment
env create <name> --path <dir>             # create at a specific path
env use <name>                             # activate an environment
env install <pkg>                          # install a package
env install <pkg>@<version>                # install a pinned version
env install <pkg> --platform=<os>:<version>          # target a specific platform
env install <pkg>@<version> --platform=<os>:<version> # pinned pkg + platform target
env list                                   # list installed packages
env lock                                   # resolve and freeze to index.lock
env sync                                   # restore environment from index.lock
env shell                                  # drop into a shell with env loaded
env run <cmd>                              # run a command inside the environment
env remove <pkg>                           # remove a package
env destroy <name>                         # delete an environment entirely
env config set <key> <value>               # set a global config value
env config get <key>                       # inspect a global config value
env config unset <key>                     # remove a global config value
```

---

## Environment Paths

By default, environments are stored in `~/.env/envs/<name>/`.

### Override path for a single environment

```bash
env create myenv --path /some/folder

# colocate with a project
env create myenv --path ./.env
```

### Change the global default base path

```bash
env config set base-path /opt/envs
env config get base-path
env config unset base-path
```

### Precedence

| Priority | Source |
|----------|--------|
| 1 (highest) | `--path` flag on `env create` |
| 2 | `base-path` set via `env config` |
| 3 (default) | `~/.env/envs/<name>/` |

---

## Platform Targeting

The `--platform` flag targets a specific OS and version for package installation.
Since `env` is just downloading and unpacking files, you can target any platform
from any host — no virtualisation required.

```bash
# from any host OS
env install gcc --platform=debian:12
env install gcc@13 --platform=ubuntu:22.04
env install cmake --platform=macos
env install cmake --platform=windows:11
```

No `--platform` means use the host OS and auto-detect the provider.

### Supported platforms

| Platform | Provider |
|----------|----------|
| `debian:11` | `apt` |
| `debian:12` | `apt` |
| `ubuntu:20.04` | `apt` |
| `ubuntu:22.04` | `apt` |
| `ubuntu:24.04` | `apt` |
| `macos` | `brew` |
| `windows:11` | `winget` |

### index.toml representation

```toml
[packages]
python = { version = "3.12" }
gcc    = { version = "13", platform = "ubuntu:22.04" }
cmake  = { version = "3.28", platform = "debian:11" }
```

### index.lock representation

```toml
[platform.linux.amd64]
python = { version = "3.12.2", provider = "apt" }

[platform.linux.amd64.ubuntu.22_04]
gcc = { version = "13.2.0", provider = "apt" }

[platform.linux.amd64.debian.11]
cmake = { version = "3.28.1", provider = "apt" }
```

---

## index.toml

```toml
[env]
name = "myenv"
path = "/some/folder"

[providers]
darwin.arm64  = "brew"
darwin.amd64  = "brew"
linux.amd64   = "apt"
linux.arm64   = "apt"
windows.amd64 = "winget"

[packages]
gcc    = { version = "13", platform = "ubuntu:22.04" }
cmake  = { version = "3.28", platform = "debian:11" }
python = { version = "3.12" }
```

---

## index.lock

```toml
[platform.darwin.arm64]
gcc    = { version = "13.2.0", provider = "brew" }
cmake  = { version = "3.28.1", provider = "brew" }
python = { version = "3.12.2", provider = "brew" }

[platform.linux.amd64]
python = { version = "3.12.2", provider = "apt" }

[platform.linux.amd64.ubuntu.22_04]
gcc = { version = "13.2.0", provider = "apt" }

[platform.linux.amd64.debian.11]
cmake = { version = "3.28.1", provider = "apt" }
```

---

## Supported Providers

| Provider | Platform |
|----------|----------|
| `brew`   | macOS, Linux |
| `apt`    | Debian, Ubuntu |
| `winget` | Windows |

---

## Reproducible Environments

```bash
# developer machine
env create myenv
env use myenv
env install gcc@13 --platform=ubuntu:22.04
env install cmake@3.28 --platform=debian:11
env install python@3.12
env lock
# commit both index.toml and index.lock

# new machine or CI
env sync
```

---

## Go Library

```bash
go get github.com/carbon-os/environment
```

```go
import "github.com/carbon-os/environment"

e, err := environment.New("myenv")

e.Install("gcc", environment.InstallParams{
    Version:  "13",
    Platform: "ubuntu:22.04",
})

e.Install("python", environment.InstallParams{
    Version: "3.12",
})

e.Lock()
e.Run("gcc --version", environment.RunParams{})
```

---

## Project Structure

```
carbon-os/environment
  environment.go      # New, Open
  install.go          # Install, Remove, provider resolution
  lock.go             # Lock, Sync
  lockfile.go         # index.lock read/write
  run.go              # Run, Shell
  config.go           # global config read/write
  platform.go         # OS/arch detection, provider defaults
  provider.go         # Provider interface, ProviderParams
  index.go            # index.toml read/write
  provider/
    apt/
      apt.go          # Apt provider
      image.go        # platform constants, mirror URLs
      index.go        # Packages index fetch and parse
      download.go     # .deb download
      unpack.go       # ar + tar extraction
```

---

## License

MIT