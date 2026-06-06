package taskruntime

import (
	"context"
	"errors"
	"strings"
	"testing"

	taskv1 "admin/api/gen/task/v1"
)

type testLogWriter struct {
	logs []*taskv1.TaskLog
	err  error
}

func (w *testLogWriter) WriteTaskLog(_ context.Context, data *taskv1.TaskLog) error {
	if w.err != nil {
		return w.err
	}
	w.logs = append(w.logs, data)
	return nil
}

func TestRunner_SupportsInvokeTarget(t *testing.T) {
	runner := NewRunner(MustNewRegistry(testExecutor{target: "demo:task"}), nil)
	if !runner.SupportsInvokeTarget("demo:task") {
		t.Fatalf("expected invoke target to be supported")
	}
	if runner.SupportsInvokeTarget("missing:task") {
		t.Fatalf("expected invoke target to be unsupported")
	}
}

func TestRunner_RunTaskWritesSuccessLog(t *testing.T) {
	writer := &testLogWriter{}
	runner := NewRunner(MustNewRegistry(testExecutor{target: "demo:task"}), writer)

	taskID := uint64(1001)
	target := "demo:task"
	taskName := "demo"
	err := runner.RunTask(context.Background(), &taskv1.Task{
		Id:           &taskID,
		TaskName:     &taskName,
		InvokeTarget: &target,
		Args:         stringPtr("payload"),
	}, "")
	if err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}
	if len(writer.logs) != 1 {
		t.Fatalf("expected 1 task log, got %d", len(writer.logs))
	}
	if writer.logs[0].GetStatus() != taskv1.TaskLog_SUCCESS {
		t.Fatalf("expected success log status, got %v", writer.logs[0].GetStatus())
	}
	if writer.logs[0].GetOutput() != "demo:payload" {
		t.Fatalf("unexpected log output: %q", writer.logs[0].GetOutput())
	}
}

func TestRunner_RunTaskReturnsExecAndLogError(t *testing.T) {
	writer := &testLogWriter{err: errors.New("write failed")}
	runner := NewRunner(MustNewRegistry(testExecutor{target: "demo:task"}), writer)

	taskID := uint64(1002)
	target := "missing:task"
	taskName := "demo"
	err := runner.RunTask(context.Background(), &taskv1.Task{
		Id:           &taskID,
		TaskName:     &taskName,
		InvokeTarget: &target,
	}, "")
	if err == nil || !strings.Contains(err.Error(), "write task log") {
		t.Fatalf("expected combined execution and log error, got %v", err)
	}
}
