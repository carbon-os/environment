package environment

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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
	cmd.Env = append(os.Environ(), "PATH="+e.BinPath()+":"+os.Getenv("PATH"))

	for k, v := range p.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd.Stdout = firstWriter(p.Stdout, os.Stdout)
	cmd.Stderr = firstWriter(p.Stderr, os.Stderr)
	cmd.Stdin = firstReader(p.Stdin, os.Stdin)

	return cmd.Run()
}

// Shell drops into an interactive shell with the environment loaded.
func (e *Environment) Shell() error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return e.Run(shell)
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