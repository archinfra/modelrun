package collect

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"modelrun/backend/internal/domain"

	"golang.org/x/crypto/ssh"
)

type CommandResult struct {
	Command    string `json:"command,omitempty"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exitCode,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
}

func (c *Collector) RunCommand(server domain.ServerConfig, jump *SSHConfig, command string) (CommandResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return CommandResult{}, errors.New("command is required")
	}

	if IsMockServer(server) {
		target := firstNonEmpty(server.Name, server.ID, server.Host, "mock-server")
		return CommandResult{
			Command:    command,
			Stdout:     fmt.Sprintf("mock executed on %s\n%s", target, command),
			DurationMs: 5,
		}, nil
	}

	client, closeFn, err := c.dial(FromServer(server), jump)
	if err != nil {
		return CommandResult{}, err
	}
	defer closeFn()

	return runCommand(client, command)
}

func BuildScriptURLCommand(scriptURL, scriptArgs string) (string, error) {
	scriptURL = strings.TrimSpace(scriptURL)
	if scriptURL == "" {
		return "", errors.New("scriptUrl is required")
	}

	command := "if command -v curl >/dev/null 2>&1; then " +
		"curl -fsSL " + shellQuote(scriptURL) + "; " +
		"elif command -v wget >/dev/null 2>&1; then " +
		"wget -q -O - " + shellQuote(scriptURL) + "; " +
		"else echo 'curl or wget is required to download remote scripts' >&2; exit 127; fi | bash -s --"
	if trimmed := strings.TrimSpace(scriptArgs); trimmed != "" {
		command += " " + trimmed
	}
	return command, nil
}

func runCommand(client sshRunner, command string) (CommandResult, error) {
	session, err := client.NewSession()
	if err != nil {
		return CommandResult{}, err
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	startedAt := time.Now()
	err = session.Run(command)

	result := CommandResult{
		Command:    command,
		Stdout:     strings.TrimSpace(stdout.String()),
		Stderr:     strings.TrimSpace(stderr.String()),
		DurationMs: time.Since(startedAt).Milliseconds(),
	}
	if err == nil {
		return result, nil
	}

	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitStatus()
	} else {
		result.ExitCode = 1
	}

	message := result.Stderr
	if message == "" {
		message = err.Error()
	}
	if strings.Contains(message, "permission denied while trying to connect to the Docker daemon socket") {
		message += "\n" + "docker command requires sudo privileges for the current SSH user, or the user must be added to the docker group."
	}
	return result, errors.New(message)
}
