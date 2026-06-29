// Package run handles executing shell commands with timing and output capture.
package run

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Result struct {
	ExitCode   int
	Duration   time.Duration
	Stdout     string
	Stderr     string
	Combined   string
	TimedOut   bool
}

// Options for running a command.
type Options struct {
	Timeout time.Duration // 0 = no timeout
	Dir     string        // working directory (empty = current)
	Env     []string      // additional env vars (K=V format)
}

// Run executes a shell command and captures all output.
// WARNING: Uses shell execution. Only pass trusted commands (never user input directly).
func Run(command string, opts Options) *Result {
	if command == "" {
		return &Result{ExitCode: -1, Combined: "error: empty command"}
	}

	res := &Result{}

	// Parse the command
	var cmd *exec.Cmd
	if opts.Dir != "" {
		cmd = exec.Command("sh", "-c", command)
		cmd.Dir = opts.Dir
	} else {
		wd, err := os.Getwd()
		if err == nil {
			cmd = exec.Command("sh", "-c", command)
			cmd.Dir = wd
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
	}

	// Set env
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	} else {
		cmd.Env = os.Environ()
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Handle timeout
	done := make(chan error, 1)
	start := time.Now()

	go func() {
		done <- cmd.Run()
	}()

	if opts.Timeout > 0 {
		timer := time.NewTimer(opts.Timeout)
		select {
		case err := <-done:
			timer.Stop()
			res.Duration = time.Since(start)
			res.Stdout = stdout.String()
			res.Stderr = stderr.String()
			res.Combined = res.Stdout + res.Stderr
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					res.ExitCode = exitErr.ExitCode()
				} else {
					res.ExitCode = -1
				}
			} else {
				res.ExitCode = 0
			}
		case <-timer.C:
			cmd.Process.Kill()
			res.Duration = opts.Timeout
			res.Stdout = stdout.String()
			res.Stderr = stderr.String()
			res.Combined = res.Stdout + res.Stderr
			res.ExitCode = -1
			res.TimedOut = true
		}
	} else {
		err := <-done
		res.Duration = time.Since(start)
		res.Stdout = stdout.String()
		res.Stderr = stderr.String()
		res.Combined = res.Stdout + res.Stderr
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				res.ExitCode = exitErr.ExitCode()
			} else {
				res.ExitCode = -1
			}
		} else {
			res.ExitCode = 0
		}
	}

	return res
}

// Tail returns the last N lines of combined output.
func (r Result) Tail(n int) string {
	lines := strings.Split(r.Combined, "\n")
	if len(lines) <= n {
		return r.Combined
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// FormatDuration returns a human-readable duration string.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
