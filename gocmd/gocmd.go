// Package gocmd is a thin wrapper around calling 'go ...' commands.
package gocmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

//A GoCmd is not actually run until one of the functions on the GoCmd interface is called.
type GoCmd interface {
	Receive(action func(io.Reader) error) error
	StdOutLines() ([]string, error)
}

type goCmd struct {
	*exec.Cmd
}

//Prepare takes the sub and args and prepares a command like 'go sub arg1 arg2...'
func Prepare(sub string, args ...string) GoCmd {
	arguments := append([]string{sub}, args...)
	return &goCmd{exec.Command("go", arguments...)}
}

func (cmd *goCmd) Receive(action func(io.Reader) error) error {
	stdout, stdOutErr := cmd.StdoutPipe()
	if stdOutErr != nil {
		return stdOutErr
	}

	stderr, stdErrErr := cmd.StderrPipe()
	if stdErrErr != nil {
		return stdErrErr
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := action(stdout); err != nil {
		return err
	}

	return newGoError(cmd.Wait(), newStdError(stderr))
}

func (cmd *goCmd) StdOutLines() ([]string, error) {
	var results []string
	err := cmd.Receive(func(stdout io.Reader) error {
		var scanerr error
		results, scanerr = scanReader(stdout)
		return scanerr
	})

	return results, err
}

func scanReader(reader io.Reader) ([]string, error) {
	var results []string

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		results = append(results, scanner.Text())
	}

	return results, scanner.Err()
}

//GoError is returned when a go command fails during its execution.
type GoError struct {
	Exit   *exec.ExitError
	StdErr *StdError
}

func (s *GoError) Error() string {
	return fmt.Sprintf("%v\n%v", s.Exit, s.StdErr)
}

func newGoError(err error, stdErr *StdError) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return &GoError{exitErr, stdErr}
	}

	return err
}

//StdError is returned anytime a go command prints to stderr during execution.
type StdError struct {
	Output string
}

func (s *StdError) Error() string {
	return s.Output
}

func newStdError(stderr io.Reader) *StdError {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stderr)
	s := buf.String()

	if s != "" {
		return &StdError{s}
	}

	return nil
}
