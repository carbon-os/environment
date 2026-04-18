// cmd/main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"runtime"
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
	platform     := fs.String("platform",      "", "target platform, e.g. debian:12, ubuntu:22.04, macos, windows:11")
	provider     := fs.String("provider",      "", "explicit provider override (apt, brew, winget)")
	downloadOnly := fs.Bool("download-only",  false, "fetch package but skip exec and post-install steps")

	// Separate flag args from positional args so that pkg@version syntax
	// doesn't confuse the flag parser.
	var flagArgs, posArgs []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			flagArgs = append(flagArgs, a)
		} else {
			posArgs = append(posArgs, a)
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(posArgs) < 1 {
		return fmt.Errorf("usage: env install <pkg>[@<version>] [--platform=<os>:<ver>]")
	}

	e, err := openActive()
	if err != nil {
		return err
	}
	e.WithLogger(newCLILogger())

	pkg, version := parsePkgArg(posArgs[0])

	if err := e.Install(pkg, environment.InstallParams{
		Version:      version,
		Platform:     *platform,
		Provider:     *provider,
		DownloadOnly: *downloadOnly,
	}); err != nil {
		return err
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

	names := make([]string, 0, len(idx.Packages))
	for n := range idx.Packages {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Printf("\n%-32s %-16s %s\n", "package", "version", "platform")
	fmt.Println(strings.Repeat("─", 60))
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
		fmt.Printf("%-32s %-16s %s\n", n, ver, plat)
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
	e.WithLogger(newCLILogger())

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

// ── fancy CLI logger ──────────────────────────────────────────────────────────

// ANSI escape codes — no external dependencies.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
)

// cliLogger renders pip-style progress to stdout.
// Downloads and installs share a single animated line per package; the line is
// finalised with a newline and a ✔ once the package is installed.
type cliLogger struct {
	mu         sync.Mutex
	inProgress bool   // a \r progress line is currently on-screen
	lastPct    int    // last printed percentage — throttles DownloadProgress redraws
	lineTag    string // tag portion of the current progress line (for redraw)
	lineName   string
	lineVer    string
	lineTotal  int64
}

func newCLILogger() *cliLogger { return &cliLogger{} }

// pkgTag returns the colour-coded, fixed-width label for a package's role.
func pkgTag(isPre, isDep bool) string {
	switch {
	case isPre:
		return ansiYellow + "pre-dep" + ansiReset
	case isDep:
		return ansiCyan + "    dep" + ansiReset
	default:
		return ansiBold + ansiBlue + "install" + ansiReset
	}
}

// renderProgressLine builds the full \r line for a download in progress.
//
//	  pre-dep  libc6              2.36-9+deb12u3   ████████████░░░░░░░░  1.4/2.8 MB
func renderProgressLine(tag, name, ver string, received, total int64) string {
	const barWidth = 20
	bar := buildBar(received, total, barWidth)

	sizeStr := ""
	if total > 0 {
		sizeStr = fmt.Sprintf("%s / %s", humanBytes(received), humanBytes(total))
	} else {
		sizeStr = humanBytes(received)
	}

	return fmt.Sprintf("  %s  %-22s %-18s %s  %s",
		tag,
		truncate(name, 22),
		truncate(ver, 18),
		bar,
		sizeStr,
	)
}

func (l *cliLogger) Collecting(pkg, version, platform, arch string) {
	label := pkg
	if version != "" {
		label += "@" + version
	}
	fmt.Printf("%sCollecting%s %s%s%s  [%s · %s/%s]\n",
		ansiBold, ansiReset,
		ansiBold, label, ansiReset,
		platform, runtime.GOOS, arch,
	)
}

func (l *cliLogger) DepsResolved(pkg string, preDeps, deps int) {
	// For winget (preDeps == 0, deps == 0) skip the line entirely — it adds
	// no information and would look odd for a package manager with no dep graph.
	if preDeps == 0 && deps == 0 {
		return
	}
	total := 1 + preDeps + deps
	fmt.Printf("  %sResolved:%s  1 requested + %d pre-dep(s) + %d dep(s)  (%d total)\n\n",
		ansiDim, ansiReset, preDeps, deps, total)
}

func (l *cliLogger) Downloading(name, version string, sizeBytes int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lineName = name
	l.lineVer = version
	l.lineTotal = sizeBytes
	l.lastPct = -1
}

func (l *cliLogger) DownloadProgress(name string, received, total int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	pct := 0
	if total > 0 {
		pct = int(received * 100 / total)
	}
	if pct == l.lastPct {
		return
	}
	l.lastPct = pct
	l.inProgress = true

	line := renderProgressLine(l.lineTag, name, l.lineVer, received, total)
	fmt.Printf("\r%s", line)
}

func (l *cliLogger) DownloadDone(name, version string) {
	// Nothing to do here — Installed will finalise the line.
}

func (l *cliLogger) Installing(name, version string, isPre, isDep bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lineTag = pkgTag(isPre, isDep)
}

func (l *cliLogger) Installed(name, version string, isPre, isDep bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	tag := pkgTag(isPre, isDep)
	line := fmt.Sprintf("  %s  %-22s %-18s %s %s✔%s",
		tag,
		truncate(name, 22),
		truncate(version, 18),
		buildBar(1, 1, 20),
		ansiGreen, ansiReset,
	)
	if l.inProgress {
		fmt.Printf("\r%s\n", line)
	} else {
		fmt.Printf("%s\n", line)
	}
	l.inProgress = false
}

func (l *cliLogger) Warn(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inProgress {
		fmt.Print("\n")
		l.inProgress = false
	}
	fmt.Printf("  %swarn:%s %s\n", ansiYellow, ansiReset, msg)
}

// ── progress bar helpers ──────────────────────────────────────────────────────

func buildBar(received, total int64, width int) string {
	filled := 0
	if total > 0 {
		filled = int(float64(received) / float64(total) * float64(width))
		if filled > width {
			filled = width
		}
	}
	return ansiGreen +
		strings.Repeat("█", filled) +
		ansiDim +
		strings.Repeat("░", width-filled) +
		ansiReset
}

func humanBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f kB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// ── active environment ────────────────────────────────────────────────────────

func activePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".env", "active"), nil
}

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

func openActive() (*environment.Environment, error) {
	path, err := readActivePath()
	if err != nil {
		return nil, err
	}
	return environment.Open(path)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parsePkgArg(arg string) (pkg, version string) {
	if i := strings.IndexByte(arg, '@'); i >= 0 {
		return arg[:i], arg[i+1:]
	}
	return arg, ""
}

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

PROVIDERS
  apt     debian, ubuntu
  brew    macos, linux
  winget  windows

WINGET PACKAGES
  Packages use the Publisher.Package identifier format from the winget-pkgs
  repository. Set GITHUB_TOKEN to avoid unauthenticated rate limits.

EXAMPLES
  env create myenv
  env use myenv
  env install gcc@13 --platform=ubuntu:22.04
  env install python@3.12
  env install cmake --platform=macos
  env install Microsoft.PowerShell@7 --platform=windows:11
  env install Neovim.Neovim --platform=windows:11
  env lock
  env sync
  env run gcc --version
`)
}