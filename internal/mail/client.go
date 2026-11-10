// Package mail drives Apple Mail through its scripting interface.
package mail

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

//go:embed script/*.js
var scripts embed.FS

const (
	prelude            = "prelude"
	notAuthorised      = "-1743"
	eventTimedOut      = "-1712"
	executionErrorMark = "execution error: "
	errorPrefix        = "Error: "
)

// DefaultTimeout bounds a single conversation with Mail.
const DefaultTimeout = 2 * time.Minute

// Client runs the embedded scripts against the local Mail application.
type Client struct {
	Timeout time.Duration
}

func (client Client) run(ctx context.Context, name string, request any, response any) error {
	source, err := source(name)
	if err != nil {
		return err
	}
	arguments := []string{"-l", "JavaScript", "-"}
	if request != nil {
		encoded, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("encode %s request: %w", name, err)
		}
		arguments = append(arguments, string(encoded))
	}

	timeout := client.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, "osascript", arguments...)
	command.Stdin = strings.NewReader(source)
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("Mail did not answer within %s", timeout)
		}
		return explain(stderr.String(), err)
	}
	if response == nil {
		return nil
	}
	if err := json.Unmarshal(stdout.Bytes(), response); err != nil {
		return fmt.Errorf("decode %s response: %w", name, err)
	}
	return nil
}

func source(name string) (string, error) {
	shared, err := scripts.ReadFile("script/" + prelude + ".js")
	if err != nil {
		return "", err
	}
	body, err := scripts.ReadFile("script/" + name + ".js")
	if err != nil {
		return "", err
	}
	return string(shared) + "\n" + string(body), nil
}

func explain(output string, cause error) error {
	message := strings.TrimSpace(output)
	switch {
	case message == "":
		return fmt.Errorf("run osascript: %w", cause)
	case strings.Contains(message, notAuthorised):
		return errors.New("this terminal is not allowed to control Mail; enable it under System Settings → Privacy & Security → Automation")
	case strings.Contains(message, eventTimedOut):
		return errors.New("Mail took too long to answer; narrow the request or try again")
	}
	if _, detail, found := strings.Cut(message, executionErrorMark); found {
		if cut := strings.LastIndex(detail, " ("); cut > 0 {
			detail = detail[:cut]
		}
		message = detail
	}
	for strings.HasPrefix(message, errorPrefix) {
		message = strings.TrimPrefix(message, errorPrefix)
	}
	return errors.New(message)
}
