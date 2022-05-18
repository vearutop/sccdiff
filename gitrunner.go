package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

// runCmd runs cmd sending its stdout and stderr to debug.Write().
func runCmd(cmd *exec.Cmd, debug *log.Logger) error {
	if debug == nil {
		debug = log.New(ioutil.Discard, "", 0)
	}

	var bufStderr bytes.Buffer

	stderr := io.MultiWriter(&bufStderr, debug.Writer())

	if cmd.Stderr != nil {
		stderr = io.MultiWriter(cmd.Stderr, stderr)
	}

	cmd.Stderr = stderr
	stdout := debug.Writer()

	if cmd.Stdout != nil {
		stdout = io.MultiWriter(cmd.Stdout, stdout)
	}

	cmd.Stdout = stdout

	debug.Printf("+ %s", cmd)

	err := cmd.Run()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		err = fmt.Errorf(`error running command: %s
exit code: %d
stderr: %s`, cmd.String(), exitErr.ExitCode(), bufStderr.String())
	}

	return err
}

func runGitCmd(debug *log.Logger, gitCmd, repoPath string, args ...string) ([]byte, error) {
	var stdout bytes.Buffer

	cmd := exec.Command(gitCmd, args...)
	cmd.Stdout = &stdout
	cmd.Dir = repoPath

	err := runCmd(cmd, debug)

	return bytes.TrimSpace(stdout.Bytes()), err
}

func runAtGitRef(debug *log.Logger, gitCmd, repoPath, ref string, fn func(path string)) error {
	worktree, err := ioutil.TempDir("", "sccdiff")
	if err != nil {
		return err
	}

	defer func() {
		rErr := os.RemoveAll(worktree)
		if rErr != nil {
			fmt.Printf("Could not delete temp directory: %s\n", worktree)
		}
	}()

	_, err = runGitCmd(debug, gitCmd, repoPath, "worktree", "add", "--quiet", "--detach", worktree, ref)
	if err != nil {
		return err
	}

	defer func() {
		_, cerr := runGitCmd(debug, gitCmd, repoPath, "worktree", "remove", worktree)
		if cerr != nil {
			var exitErr *exec.ExitError
			if errors.As(cerr, &exitErr) {
				fmt.Println(string(exitErr.Stderr))
			}

			fmt.Println(cerr)
		}
	}()
	fn(worktree)

	return nil
}
