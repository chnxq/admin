package task

import (
	"context"
	"strings"
	"testing"

	taskv1 "admin/api/gen/task/v1"
)

func TestEchoExecutor_ValidateRejectsEmptyMessage(t *testing.T) {
	executor := NewEchoExecutor()
	err := executor.Validate(context.Background(), ValidationRequest{
		Raw: `{"message":"   "}`,
	})
	if err == nil || !strings.Contains(err.Error(), "message must not be empty") {
		t.Fatalf("expected empty message error, got %v", err)
	}
}

func TestEchoExecutor_ExecuteReturnsMessagePayload(t *testing.T) {
	executor := NewEchoExecutor()
	result, err := executor.Execute(context.Background(), ExecuteRequest{
		Task: &taskv1.Task{
			Args: stringPtr(`{"message":"hello"}`),
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, `"message":"hello"`) {
		t.Fatalf("unexpected execute result: %s", result)
	}
	if !strings.Contains(result, `"source":"echo-executor"`) {
		t.Fatalf("unexpected execute source: %s", result)
	}
}

func TestParseEchoInput_UsesTaskArgsFallback(t *testing.T) {
	payload, err := ParseEchoInput("", &taskv1.Task{
		Args: stringPtr(`{"message":"from-task"}`),
	})
	if err != nil {
		t.Fatalf("ParseEchoInput failed: %v", err)
	}
	if payload.Message != "from-task" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
