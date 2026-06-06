package task

import (
	"context"
	"strings"
	"testing"

	taskv1 "admin/api/gen/task/v1"
)

type testExecutor struct {
	target string
}

func (e testExecutor) InvokeTarget() string {
	return e.target
}

func (e testExecutor) Validate(_ context.Context, req ValidationRequest) error {
	if strings.TrimSpace(req.Raw) == "" {
		return errTestValidation
	}
	return nil
}

func (e testExecutor) Execute(_ context.Context, req ExecuteRequest) (string, error) {
	if req.Task == nil {
		return "", errTestValidation
	}
	return req.Task.GetTaskName() + ":" + strings.TrimSpace(req.Input), nil
}

func TestNewRegistry_RejectsDuplicateInvokeTarget(t *testing.T) {
	_, err := NewRegistry(testExecutor{target: "demo:task"}, testExecutor{target: "demo:task"})
	if err == nil || !strings.Contains(err.Error(), "duplicate task executor") {
		t.Fatalf("expected duplicate executor error, got %v", err)
	}
}

func TestRegistry_GetAndExecute(t *testing.T) {
	registry, err := NewRegistry(testExecutor{target: "demo:task"})
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}
	executor, ok := registry.Get("demo:task")
	if !ok || executor == nil {
		t.Fatalf("expected executor to exist")
	}

	result, execErr := registry.Execute(context.Background(), &taskv1.Task{
		TaskName:     stringPtr("sample"),
		InvokeTarget: stringPtr("demo:task"),
	}, "payload")
	if execErr != nil {
		t.Fatalf("Execute failed: %v", execErr)
	}
	if result != "sample:payload" {
		t.Fatalf("unexpected execute result: %q", result)
	}
}

func TestRegistry_ValidateRejectsUnknownTarget(t *testing.T) {
	registry := MustNewRegistry(testExecutor{target: "demo:task"})
	err := registry.Validate(context.Background(), &taskv1.Task{
		InvokeTarget: stringPtr("missing:task"),
	}, "payload")
	if err == nil || !strings.Contains(err.Error(), "unsupported task invoke target") {
		t.Fatalf("expected unsupported target error, got %v", err)
	}
}

func TestRegistry_ValidateDelegatesToExecutor(t *testing.T) {
	registry := MustNewRegistry(testExecutor{target: "demo:task"})
	err := registry.Validate(context.Background(), &taskv1.Task{
		InvokeTarget: stringPtr("demo:task"),
	}, "")
	if err == nil || err != errTestValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}
