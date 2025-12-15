package utils

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

func ExecuteCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	return execute(command)
}

func ExecuteCommandInDir(cwd string, name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	command.Dir = cwd
	return execute(command)
}

func execute(command *exec.Cmd) (string, string, error) {
	slog.Info("executing", "cmd", command)
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()
	stdoutBuilder := &strings.Builder{}
	stdoutMultiWriter := io.MultiWriter(stdoutWriter, stdoutBuilder)
	command.Stdout = stdoutMultiWriter

	stderrReader, stderrWriter := io.Pipe()
	defer stderrReader.Close()
	stderrBuilder := &strings.Builder{}
	stderrMultiWriter := io.MultiWriter(stderrWriter, stderrBuilder)
	command.Stderr = stderrMultiWriter

	if err := command.Start(); err != nil {
		return "", "", fmt.Errorf("failed to start command: %v", err)
	}

	bufStdoutReader, bufStderrReader := bufio.NewScanner(stdoutReader), bufio.NewScanner(stderrReader)
	bufStdoutReader.Split(ScanUntil('\n', '\r'))
	bufStderrReader.Split(ScanUntil('\n', '\r'))

	stdoutDone, stderrDone := make(chan bool), make(chan bool)

	// TODO: Scanner.Err()

	go func() {
		for bufStdoutReader.Scan() {
			line := bufStdoutReader.Text()
			slog.Debug("stdout", "command-name", command.Path, "line", line)
			time.Sleep(10 * time.Millisecond)
		}
		stdoutDone <- true
	}()

	go func() {
		for bufStderrReader.Scan() {
			line := bufStderrReader.Text()
			slog.Debug("stderr", "command-name", command.Path, "line", line)
			time.Sleep(10 * time.Millisecond)
		}
		stderrDone <- true
	}()

	err := command.Wait()
	stdoutWriter.Close()
	stderrWriter.Close()
	<-stdoutDone
	<-stderrDone
	stdout := stdoutBuilder.String()
	stderr := stderrBuilder.String()
	slog.Debug("executed command", "name", command.Path, "args", command.Args, "stdout", stdout, "stderr", stderr, "err", err)
	return stdout, stderr, err
}

// TODO: Make shell configurable
func ExecuteShell(cmd string) (string, string, error) {
	return ExecuteCommand("bash", "-c", cmd)
}
