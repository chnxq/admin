package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const EchoInvokeTarget = "system:echo"

type EchoInput struct {
	Message string `json:"message"`
}

type EchoResult struct {
	Message string `json:"message"`
	Source  string `json:"source"`
}

// EchoExecutor is a minimal example for non-cleanup executors.
// It is intentionally not registered into NewDefaultRegistry yet.
type EchoExecutor struct{}

func NewEchoExecutor() *EchoExecutor {
	return &EchoExecutor{}
}

func (e *EchoExecutor) InvokeTarget() string {
	return EchoInvokeTarget
}

func (e *EchoExecutor) Validate(_ context.Context, req ValidationRequest) error {
	payload, err := ParseEchoInput(req.Raw, req.Task)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.Message) == "" {
		return fmt.Errorf("message must not be empty")
	}
	return nil
}

func (e *EchoExecutor) Execute(ctx context.Context, req ExecuteRequest) (string, error) {
	if err := e.Validate(ctx, ValidationRequest{Task: req.Task, Raw: req.Input}); err != nil {
		return "", err
	}

	payload, _ := ParseEchoInput(req.Input, req.Task)
	result := EchoResult{
		Message: strings.TrimSpace(payload.Message),
		Source:  "echo-executor",
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func ParseEchoInput(raw string, taskItem any) (*EchoInput, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		switch t := taskItem.(type) {
		case interface{ GetArgs() string }:
			raw = strings.TrimSpace(t.GetArgs())
		}
	}
	if raw == "" {
		return nil, fmt.Errorf("task args are required")
	}

	var payload EchoInput
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("invalid echo args: %w", err)
	}
	return &payload, nil
}
