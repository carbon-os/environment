package environment

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// RunParams controls how a command is executed inside the environment.
type RunParams struct {
	Env    map[string]string
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

// Run executes a command inside the environment.
func (e *Environment) Run(command string, params ...RunParams) error {
	p := RunParams{}
	if len(params) > 0 {
		p = params[0]
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Env = e.buildEnv(p.Env)

	cmd.Stdout = firstWriter(p.Stdout, os.Stdout)
	cmd.Stderr = firstWriter(p.Stderr, os.Stderr)
	cmd.Stdin = firstReader(p.Stdin, os.Stdin)

	return cmd.Run()
}

// Shell drops into an interactive shell with the environment loaded.
// A PTY is allocated so that readline, job control, and full-screen programs
// (vim, htop, etc.) work correctly.
func (e *Environment) Shell() error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell)
	cmd.Env = e.buildEnv(nil)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("shell: start pty: %w", err)
	}
	defer ptmx.Close()

	// Put the host terminal into raw mode so every keystroke passes through
	// unmodified to the shell inside the PTY.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("shell: raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Forward SIGWINCH so the inner shell sees terminal resize events.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	defer func() {
		signal.Stop(ch)
		close(ch)
	}()
	go func() {
		for range ch {
			pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	ch <- syscall.SIGWINCH // sync initial size

	go io.Copy(ptmx, os.Stdin)
	io.Copy(os.Stdout, ptmx)

	return cmd.Wait()
}

// buildEnv constructs the command environment with the env's bin directory
// prepended to PATH, replacing any existing PATH entry rather than appending
// (appending would leave a duplicate and the system PATH would win).
func (e *Environment) buildEnv(extra map[string]string) []string {
	envPath := e.BinPath() + ":" + os.Getenv("PATH")

	env := make([]string, 0, len(os.Environ())+len(extra))
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "PATH=") {
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "PATH="+envPath)

	for k, v := range extra {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

func firstWriter(a, b io.Writer) io.Writer {
	if a != nil {
		return a
	}
	return b
}

func firstReader(a, b io.Reader) io.Reader {
	if a != nil {
		return a
	}
	return b
}