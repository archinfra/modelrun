package collect

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
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

type CommandStreamLine struct {
	Stream string
	Line   string
}

func (c *Collector) RunCommand(server domain.ServerConfig, jump *SSHConfig, command string) (CommandResult, error) {
	return c.RunCommandStream(server, jump, command, nil)
}

func (c *Collector) RunCommandStream(server domain.ServerConfig, jump *SSHConfig, command string, onLine func(CommandStreamLine)) (CommandResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return CommandResult{}, errors.New("command is required")
	}

	if IsMockServer(server) {
		target := firstNonEmpty(server.Name, server.ID, server.Host, "mock-server")
		result := CommandResult{
			Command:    command,
			Stdout:     fmt.Sprintf("mock executed on %s\n%s", target, command),
			DurationMs: 5,
		}
		if onLine != nil {
			for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				onLine(CommandStreamLine{Stream: "stdout", Line: line})
			}
		}
		return result, nil
	}

	client, closeFn, err := c.dial(FromServer(server), jump)
	if err != nil {
		return CommandResult{}, err
	}
	defer closeFn()

	return runCommand(client, command, onLine)
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

func runCommand(client sshRunner, command string, onLine func(CommandStreamLine)) (CommandResult, error) {
	session, err := client.NewSession()
	if err != nil {
		return CommandResult{}, err
	}
	defer session.Close()

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return CommandResult{}, err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return CommandResult{}, err
	}

	startedAt := time.Now()
	if err := session.Start(command); err != nil {
		return CommandResult{}, err
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var wg sync.WaitGroup
	var streamErrMu sync.Mutex
	streamErrors := []string{}

	readPipe := func(stream string, reader io.Reader, buffer *bytes.Buffer) {
		defer wg.Done()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		scanner.Split(splitStreamLines)
		for scanner.Scan() {
			line := scanner.Text()
			if buffer.Len() > 0 {
				buffer.WriteByte('\n')
			}
			buffer.WriteString(line)
			if onLine != nil {
				onLine(CommandStreamLine{Stream: stream, Line: line})
			}
		}
		if err := scanner.Err(); err != nil {
			streamErrMu.Lock()
			streamErrors = append(streamErrors, err.Error())
			streamErrMu.Unlock()
		}
	}

	wg.Add(2)
	go readPipe("stdout", stdoutPipe, &stdout)
	go readPipe("stderr", stderrPipe, &stderr)

	err = session.Wait()
	wg.Wait()

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
	if len(streamErrors) > 0 {
		if message != "" {
			message += "\n"
		}
		message += strings.Join(streamErrors, "\n")
	}
	if strings.Contains(message, "permission denied while trying to connect to the Docker daemon socket") {
		message += "\n" + "docker command requires sudo privileges for the current SSH user, or the user must be added to the docker group."
	}
	return result, errors.New(message)
}

func splitStreamLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '\n':
			return i + 1, dropTrailingCR(data[:i]), nil
		case '\r':
			if i+1 < len(data) && data[i+1] == '\n' {
				return i + 2, data[:i], nil
			}
			return i + 1, data[:i], nil
		}
	}

	if atEOF {
		return len(data), dropTrailingCR(data), nil
	}
	return 0, nil, nil
}

func dropTrailingCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[:len(data)-1]
	}
	return data
}
