package task

import (
	"context"
	"strings"
	"testing"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type fakeTaskRepo struct {
	getFunc           func(context.Context, *taskv1.GetTaskRequest) (*taskv1.Task, error)
	getRuntimeFunc    func(context.Context, uint64) (*taskv1.Task, error)
	listRunnableFunc  func(context.Context) ([]*taskv1.Task, error)
	updateRuntimeFunc func(context.Context, uint64, taskv1.Task_Status, *uint32) error
}

func (f *fakeTaskRepo) List(context.Context, *paginationv1.PagingRequest) (*taskv1.ListTaskResponse, error) {
	panic("not used")
}

func (f *fakeTaskRepo) Get(ctx context.Context, req *taskv1.GetTaskRequest) (*taskv1.Task, error) {
	return f.getFunc(ctx, req)
}

func (f *fakeTaskRepo) GetTaskByIDForRuntime(ctx context.Context, taskID uint64) (*taskv1.Task, error) {
	if f.getRuntimeFunc != nil {
		return f.getRuntimeFunc(ctx, taskID)
	}
	return nil, nil
}

func (f *fakeTaskRepo) Create(context.Context, *taskv1.CreateTaskRequest) (*emptypb.Empty, error) {
	panic("not used")
}

func (f *fakeTaskRepo) Update(context.Context, *taskv1.UpdateTaskRequest) (*emptypb.Empty, error) {
	panic("not used")
}

func (f *fakeTaskRepo) Delete(context.Context, *taskv1.DeleteTaskRequest) (*emptypb.Empty, error) {
	panic("not used")
}

func (f *fakeTaskRepo) ListRunnableTasks(ctx context.Context) ([]*taskv1.Task, error) {
	return f.listRunnableFunc(ctx)
}

func (f *fakeTaskRepo) ListTasksByGroupID(context.Context, uint64) ([]*taskv1.Task, error) {
	panic("not used")
}

func (f *fakeTaskRepo) ListTasksForRuntime(context.Context, *uint32) ([]*taskv1.Task, error) {
	panic("not used")
}

func (f *fakeTaskRepo) UpdateTaskRuntimeState(ctx context.Context, taskID uint64, status taskv1.Task_Status, entryID *uint32) error {
	return f.updateRuntimeFunc(ctx, taskID, status, entryID)
}

var _ repo.TaskRepo = (*fakeTaskRepo)(nil)
var _ repo.TaskRuntimeRepo = (*fakeTaskRepo)(nil)

func TestNewTaskRuntimeStore_RequiresRuntimeRepo(t *testing.T) {
	_, err := NewTaskRuntimeStore(nil)
	if err == nil || !strings.Contains(err.Error(), "task repo is not configured") {
		t.Fatalf("expected task repo error, got %v", err)
	}
}

func TestRuntimeStore_DelegatesToRepo(t *testing.T) {
	taskID := uint64(3001)
	status := taskv1.Task_RUNNING
	calledGet := false
	calledList := false
	calledUpdate := false
	repo := &fakeTaskRepo{
		getFunc: func(_ context.Context, req *taskv1.GetTaskRequest) (*taskv1.Task, error) {
			calledGet = true
			return &taskv1.Task{Id: &taskID}, nil
		},
		getRuntimeFunc: func(_ context.Context, gotTaskID uint64) (*taskv1.Task, error) {
			calledGet = gotTaskID == taskID
			return &taskv1.Task{Id: &taskID}, nil
		},
		listRunnableFunc: func(context.Context) ([]*taskv1.Task, error) {
			calledList = true
			return []*taskv1.Task{{Id: &taskID}}, nil
		},
		updateRuntimeFunc: func(_ context.Context, gotTaskID uint64, gotStatus taskv1.Task_Status, _ *uint32) error {
			calledUpdate = gotTaskID == taskID && gotStatus == status
			return nil
		},
	}

	store, err := NewTaskRuntimeStore(repo)
	if err != nil {
		t.Fatalf("NewTaskRuntimeStore failed: %v", err)
	}
	if _, err := store.GetTask(context.Background(), taskID); err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if _, err := store.ListRunnableTasks(context.Background()); err != nil {
		t.Fatalf("ListRunnableTasks failed: %v", err)
	}
	if err := store.UpdateTaskRuntimeState(context.Background(), taskID, status, nil); err != nil {
		t.Fatalf("UpdateTaskRuntimeState failed: %v", err)
	}
	if !calledGet || !calledList || !calledUpdate {
		t.Fatalf("expected all repo delegates to be called: get=%v list=%v update=%v", calledGet, calledList, calledUpdate)
	}
}
