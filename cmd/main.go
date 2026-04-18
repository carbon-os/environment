// cmd/main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/carbon-os/environment"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "env: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	cmd, rest := args[0], args[1:]

	switch cmd {
	case "create":
		return cmdCreate(rest)
	case "use":
		return cmdUse(rest)
	case "install":
		return cmdInstall(rest)
	case "remove":
		return cmdRemove(rest)
	case "list":
		return cmdList(rest)
	case "lock":
		return cmdLock(rest)
	case "sync":
		return cmdSync(rest)
	case "shell":
		return cmdShell(rest)
	case "run":
		return cmdRun(rest)
	case "destroy":
		return cmdDestroy(rest)
	case "config":
		return cmdConfig(rest)
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q — run 'env help' for usage", cmd)
	}
}

// ── commands ──────────────────────────────────────────────────────────────────

func cmdCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	customPath := fs.String("path", "", "create environment at a specific path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: env create <name> [--path <dir>]")
	}

	name := fs.Arg(0)
	params := environment.CreateParams{Path: *customPath}

	e, err := environment.New(name, params)
	if err != nil {
		return err
	}

	fmt.Printf("created environment %q\n  path: %s\n", e.Name, e.Path)
	return nil
}

func cmdUse(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: env use <name|path>")
	}

	target := args[0]

	// allow both a bare name ("myenv") and an explicit path ("./my-env", "/opt/envs/foo")
	var envPath string
	if filepath.IsAbs(target) || strings.HasPrefix(target, ".") {
		abs, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		envPath = abs
	} else {
		p, err := resolveEnvPath(target)
		if err != nil {
			return err
		}
		envPath = p
	}

	// validate before committing
	e, err := environment.Open(envPath)
	if err != nil {
		return fmt.Errorf("cannot open environment %q: %w", target, err)
	}

	if err := writeActive(envPath); err != nil {
		return err
	}

	fmt.Printf("now using %q (%s)\n", e.Name, e.Path)
	return nil
}

func cmdInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	platform := fs.String("platform", "", `target platform, e.g. debian:12 or ubuntu:22.04`)
	provider := fs.String("provider", "", "explicit provider override")
	downloadOnly := fs.Bool("download-only", false, "fetch package but skip exec and post-install steps")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: env install <pkg>[@<version>] [--platform=<os>:<ver>]")
	}

	e, err := openActive()
	if err != nil {
		return err
	}

	pkg, version := parsePkgArg(fs.Arg(0))

	// human-readable progress line
	label := pkg
	if version != "" {
		label += "@" + version
	}
	if *platform != "" {
		label += " (platform: " + *platform + ")"
	}
	fmt.Printf("installing %s ...\n", label)

	if err := e.Install(pkg, environment.InstallParams{
		Version:      version,
		Platform:     *platform,
		Provider:     *provider,
		DownloadOnly: *downloadOnly,
	}); err != nil {
		return err
	}

	if *downloadOnly {
		fmt.Printf("downloaded %s\n", label)
	} else {
		fmt.Printf("installed  %s\n", label)
	}
	return nil
}

func cmdRemove(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: env remove <pkg>")
	}

	e, err := openActive()
	if err != nil {
		return err
	}

	if err := e.Remove(args[0]); err != nil {
		return err
	}

	fmt.Printf("removed %s\n", args[0])
	return nil
}

func cmdList(_ []string) error {
	e, err := openActive()
	if err != nil {
		return err
	}

	idx, err := loadIndex(e.Path)
	if err != nil {
		return err
	}

	fmt.Printf("environment: %s\npath:        %s\n", e.Name, e.Path)

	if len(idx.Packages) == 0 {
		fmt.Println("\nno packages installed")
		return nil
	}

	// stable sort for deterministic output
	names := make([]string, 0, len(idx.Packages))
	for n := range idx.Packages {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Printf("\n%-24s %-16s %s\n", "package", "version", "platform")
	fmt.Println(strings.Repeat("─", 56))
	for _, n := range names {
		p := idx.Packages[n]
		ver := p.Version
		if ver == "" {
			ver = "(any)"
		}
		plat := p.Platform
		if plat == "" {
			plat = "(host)"
		}
		fmt.Printf("%-24s %-16s %s\n", n, ver, plat)
	}

	return nil
}

func cmdLock(_ []string) error {
	e, err := openActive()
	if err != nil {
		return err
	}

	if err := e.Lock(); err != nil {
		return err
	}

	fmt.Printf("locked → %s/index.lock\n", e.Path)
	return nil
}

func cmdSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "resolve without applying changes")
	if err := fs.Parse(args); err != nil {
		return err
	}

	e, err := openActive()
	if err != nil {
		return err
	}

	if *dryRun {
		fmt.Println("dry run — no packages will be installed")
	}

	if err := e.Sync(environment.SyncParams{DryRun: *dryRun}); err != nil {
		return err
	}

	if !*dryRun {
		fmt.Println("sync complete")
	}
	return nil
}

func cmdShell(_ []string) error {
	e, err := openActive()
	if err != nil {
		return err
	}
	return e.Shell()
}

func cmdRun(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: env run <command>")
	}

	e, err := openActive()
	if err != nil {
		return err
	}

	// re-join so "env run gcc --version" reaches the shell as "gcc --version"
	return e.Run(strings.Join(args, " "))
}

func cmdDestroy(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: env destroy <name>")
	}

	name := args[0]
	envPath, err := resolveEnvPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf("environment %q not found at %s", name, envPath)
	}

	if err := os.RemoveAll(envPath); err != nil {
		return fmt.Errorf("destroy: %w", err)
	}

	// if this was the active environment, clear the pointer
	if active, _ := readActivePath(); active == envPath {
		if p, err := activePath(); err == nil {
			os.Remove(p)
		}
	}

	fmt.Printf("destroyed environment %q\n", name)
	return nil
}

func cmdConfig(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: env config <set|get|unset> <key> [<value>]")
	}

	cfg, err := environment.Config()
	if err != nil {
		return err
	}

	sub, rest := args[0], args[1:]

	switch sub {
	case "set":
		if len(rest) < 2 {
			return fmt.Errorf("usage: env config set <key> <value>")
		}
		if err := cfg.Set(rest[0], rest[1]); err != nil {
			return err
		}
		fmt.Printf("%s = %s\n", rest[0], rest[1])

	case "get":
		val, err := cfg.Get(rest[0])
		if err != nil {
			return err
		}
		if val == "" {
			fmt.Println("(not set)")
		} else {
			fmt.Println(val)
		}

	case "unset":
		if err := cfg.Unset(rest[0]); err != nil {
			return err
		}
		fmt.Printf("unset %s\n", rest[0])

	default:
		return fmt.Errorf("unknown config subcommand %q — expected set, get, or unset", sub)
	}

	return nil
}

// ── active environment ────────────────────────────────────────────────────────

// activePath returns the path to the file that records the active environment.
func activePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".env", "active"), nil
}

// readActivePath reads the raw path stored in the active file.
func readActivePath() (string, error) {
	p, err := activePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("no active environment — run 'env use <name>' first")
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// writeActive persists envPath as the active environment.
func writeActive(envPath string) error {
	p, err := activePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(envPath), 0644)
}

// openActive reads the active pointer and opens the environment it points to.
func openActive() (*environment.Environment, error) {
	path, err := readActivePath()
	if err != nil {
		return nil, err
	}
	return environment.Open(path)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// parsePkgArg splits "pkg@version" into its two parts.
// An argument with no "@" returns the whole string as the package name.
func parsePkgArg(arg string) (pkg, version string) {
	if i := strings.IndexByte(arg, '@'); i >= 0 {
		return arg[:i], arg[i+1:]
	}
	return arg, ""
}

// resolveEnvPath mirrors the library's internal resolvePath logic:
// config base-path → default ~/.env/envs/<name>.
func resolveEnvPath(name string) (string, error) {
	cfg, err := environment.Config()
	if err == nil && cfg.BasePath != "" {
		return filepath.Join(cfg.BasePath, name), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".env", "envs", name), nil
}

// loadIndex decodes index.toml from an environment directory.
// Used by cmdList; readIndex is unexported in the library.
func loadIndex(envPath string) (*environment.Index, error) {
	data, err := os.ReadFile(filepath.Join(envPath, "index.toml"))
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	var idx environment.Index
	if _, err := toml.Decode(string(data), &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	return &idx, nil
}

// ── usage ─────────────────────────────────────────────────────────────────────

func printUsage() {
	fmt.Print(`env — isolated, reproducible package environments

USAGE
  env <command> [flags] [args]

COMMANDS
  create <name> [--path <dir>]                create a new environment
  use    <name|path>                          activate an environment
  install <pkg>[@<ver>] [--platform=<p>]      install a package
          [--provider=<p>] [--download-only]
  remove  <pkg>                               remove a package
  list                                        list installed packages
  lock                                        resolve and freeze to index.lock
  sync    [--dry-run]                         restore from index.lock
  shell                                       drop into an env-aware shell
  run     <command>                           run a command inside the environment
  destroy <name>                              delete an environment
  config  set   <key> <value>                 write a global config value
  config  get   <key>                         read a global config value
  config  unset <key>                         clear a global config value

CONFIG KEYS
  base-path    default directory for new environments (default: ~/.env/envs)
  apt.mirror   custom apt mirror URL

EXAMPLES
  env create myenv
  env use myenv
  env install gcc@13 --platform=ubuntu:22.04
  env install python@3.12
  env lock
  env sync
  env run gcc --version
`)
}