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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

//go:embed script/*.js
var scripts embed.FS

// DefaultTimeout bounds a single conversation with Mail.
const DefaultTimeout = 2 * time.Minute

// script names one embedded JavaScript for Automation file.
type script string

const (
	prelude           script = "prelude"
	pingScript        script = "ping"
	accountsScript    script = "accounts"
	mailboxesScript   script = "mailboxes"
	listScript        script = "list"
	searchScript      script = "search"
	resolveScript     script = "resolve"
	readScript        script = "read"
	sourceScript      script = "source"
	attachmentsScript script = "attachments"
	markScript        script = "mark"
	flagScript        script = "flag"
	moveScript        script = "move"
	composeScript     script = "compose"
	checkScript       script = "check"
)

// kind is the category a script attaches to an error it throws.
type kind string

const (
	messageKind   kind = "message"
	mailboxKind   kind = "mailbox"
	accountKind   kind = "account"
	signatureKind kind = "signature"
	refusedKind   kind = "refused"
)

// Apple event error numbers, from AppleEvents.h and MacErrors.h.
const (
	notPermitted      = -1743
	timedOut          = -1712
	missingObject     = -1728
	applicationAbsent = -600
)

const (
	executionErrorMark = "execution error: "
	errorPrefix        = "Error: "
	emptyRequest       = "{}"
)

// trailingNumber matches the error number osascript prints last, which, unlike the text, is not localised.
var trailingNumber = regexp.MustCompile(`\((-?\d+)\)\s*$`)

// Client runs the embedded scripts against the local Mail application.
type Client struct {
	Timeout time.Duration
}

var _ Backend = Client{}

type envelope struct {
	Result json.RawMessage `json:"result"`
	Error  *fault          `json:"error"`
}

// fault is an error thrown inside a script, as the prelude reports it.
type fault struct {
	Number  int    `json:"number"`
	Message string `json:"message"`
	Kind    kind   `json:"kind"`
}

// run executes a script with a JSON request and decodes its result into response.
// The target is used only to explain a message that could not be found.
func (client Client) run(ctx context.Context, name script, target Target, request any, response any) error {
	program, err := source(string(name))
	if err != nil {
		return err
	}
	encoded := []byte(emptyRequest)
	if request != nil {
		if encoded, err = json.Marshal(request); err != nil {
			return fmt.Errorf("encode %s request: %w", name, err)
		}
	}

	timeout := client.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-", string(encoded))
	command.Stdin = strings.NewReader(program)
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Unanswered(timeout)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return explain(stderr.String(), err, target, timeout)
	}

	var answer envelope
	if err := json.Unmarshal(stdout.Bytes(), &answer); err != nil {
		return fmt.Errorf("decode %s response: %w", name, err)
	}
	if answer.Error != nil {
		return classify(*answer.Error, target, timeout)
	}
	if response == nil || len(answer.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(answer.Result, response); err != nil {
		return fmt.Errorf("decode %s result: %w", name, err)
	}
	return nil
}

func source(name string) (string, error) {
	shared, err := scripts.ReadFile("script/" + string(prelude) + ".js")
	if err != nil {
		return "", err
	}
	body, err := scripts.ReadFile("script/" + name + ".js")
	if err != nil {
		return "", err
	}
	return string(shared) + "\n" + string(body), nil
}

// Settle replaces the retry advice of an unanswered send, which Mail may still carry out, with a warning to check Sent first.
func Settle(err error, send bool) error {
	var problem *failure.Error
	if !send || !errors.As(err, &problem) || problem.Code != failure.Busy {
		return err
	}
	return MaybeSent(problem)
}

// classify turns a script error into a failure by its number and kind, never by its text.
func classify(problem fault, target Target, timeout time.Duration) error {
	switch {
	case problem.Number == notPermitted:
		return PermissionDenied()
	case problem.Number == timedOut:
		return Unanswered(timeout)
	case problem.Kind == messageKind:
		return MessageNotFound(target, problem.Message)
	case problem.Kind == mailboxKind:
		return failure.New(failure.NotFound, problem.Message, "list them with `mailslot mailboxes`")
	case problem.Kind == accountKind:
		return failure.New(failure.NotFound, problem.Message, "list them with `mailslot accounts`")
	case problem.Kind == signatureKind, problem.Number == missingObject:
		return failure.New(failure.NotFound, problem.Message, "")
	case problem.Kind == refusedKind:
		return failure.New(failure.Refused, problem.Message, "")
	case problem.Number == applicationAbsent:
		return failure.New(failure.General, problem.Message, "Mail could not be started")
	}
	return failure.New(failure.General, problem.Message, "")
}

// explain classifies an osascript failure that produced no envelope, such as a script that did not compile.
func explain(output string, cause error, target Target, timeout time.Duration) error {
	message := strings.TrimSpace(output)
	if message == "" {
		return fmt.Errorf("run osascript: %w", cause)
	}
	problem := fault{Message: message}
	if match := trailingNumber.FindStringSubmatch(message); match != nil {
		problem.Number, _ = strconv.Atoi(match[1])
		problem.Message = strings.TrimSpace(strings.TrimSuffix(message, match[0]))
	}
	if _, detail, found := strings.Cut(problem.Message, executionErrorMark); found {
		problem.Message = detail
	}
	for strings.HasPrefix(problem.Message, errorPrefix) {
		problem.Message = strings.TrimPrefix(problem.Message, errorPrefix)
	}
	return classify(problem, target, timeout)
}
