package task

import (
	"context"
	"encoding/json"
	"testing"

	taskv1 "admin/api/gen/task/v1"
)

type fakeTaskSummaryProvider struct {
	items []*taskv1.Task
}

func (f *fakeTaskSummaryProvider) ListTasksForRuntime(_ context.Context, _ *uint32) ([]*taskv1.Task, error) {
	return f.items, nil
}

func TestTaskRuntimeSummaryExecutor_Execute(t *testing.T) {
	running := taskv1.Task_RUNNING
	stopped := taskv1.Task_STOPPED
	functionType := taskv1.Task_FUNCTION
	apiType := taskv1.Task_API
	tenantID := uint32(101)

	executor := NewTaskRuntimeSummaryExecutor(RuntimeDeps{
		TaskSummaryProvider: &fakeTaskSummaryProvider{
			items: []*taskv1.Task{
				{
					Id:       uint64Ptr(10),
					TaskName: stringPtr("Cleanup"),
					Status:   &running,
					TaskType: &functionType,
					GroupId:  uint64Ptr(1501),
					TenantId: &tenantID,
				},
				{
					Id:       uint64Ptr(11),
					TaskName: stringPtr("Sync API"),
					Status:   &stopped,
					TaskType: &apiType,
					GroupId:  uint64Ptr(1502),
					TenantId: &tenantID,
				},
			},
		},
	})

	raw, err := executor.Execute(context.Background(), ExecuteRequest{
		Task: &taskv1.Task{
			TenantId: &tenantID,
			Args:     stringPtr(`{"tenantScope":"current"}`),
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	var result TaskRuntimeSummaryResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Total)
	}
	if result.ByStatus["RUNNING"] != 1 || result.ByStatus["STOPPED"] != 1 {
		t.Fatalf("unexpected status summary: %#v", result.ByStatus)
	}
	if len(result.Tasks) != 2 || result.Tasks[0].Name != "Cleanup" {
		t.Fatalf("unexpected task list: %#v", result.Tasks)
	}
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}
