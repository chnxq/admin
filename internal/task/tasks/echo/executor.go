package echo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	taskruntime "admin/internal/task/runtime"
)

const (
	EchoInvokeTarget = "system:echo"
)

type Input struct {
	Message string `json:"message"`
}

type Result struct {
	Message string `json:"message"`
	Source  string `json:"source"`
}

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) InvokeTarget() string {
	return EchoInvokeTarget
}

func (e *Executor) Validate(_ context.Context, req taskruntime.ValidationRequest) error {
	payload, err := ParseInput(req.Raw, req.Task)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.Message) == "" {
		return fmt.Errorf("message must not be empty")
	}
	return nil
}

func (e *Executor) Execute(ctx context.Context, req taskruntime.ExecuteRequest) (string, error) {
	if err := e.Validate(ctx, taskruntime.ValidationRequest{Task: req.Task, Raw: req.Input}); err != nil {
		return "", err
	}

	payload, _ := ParseInput(req.Input, req.Task)
	result := Result{
		Message: strings.TrimSpace(payload.Message),
		Source:  "echo-executor",
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func ParseInput(raw string, taskItem interface{ GetArgs() string }) (*Input, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" && taskItem != nil {
		raw = strings.TrimSpace(taskItem.GetArgs())
	}
	if raw == "" {
		return nil, fmt.Errorf("task args are required")
	}

	var payload Input
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("invalid echo args: %w", err)
	}
	return &payload, nil
}
