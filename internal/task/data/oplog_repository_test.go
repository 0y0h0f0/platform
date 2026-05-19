package data_test

import (
	"context"
	"testing"

	"task-platform/internal/task/data"
)

func ptr(s string) *string { return &s }

func TestOpLogCreateBatch_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewOperationLogRepository(db)

	pid := uid()
	logs := []*data.OperationLog{
		{ProjectID: ptr(pid), OperatorID: uid(), Action: data.ActionTaskCreate, Detail: "{}"},
		{ProjectID: ptr(pid), OperatorID: uid(), Action: data.ActionTaskUpdate, Detail: "{}"},
	}

	err := repo.CreateBatch(context.Background(), logs)
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
}

func TestOpLogCreateBatch_Empty(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewOperationLogRepository(db)

	err := repo.CreateBatch(context.Background(), nil)
	if err != nil {
		t.Errorf("empty batch should not error: %v", err)
	}
}

func TestOpLogListByProject(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewOperationLogRepository(db)

	pid := uid()
	for i := 0; i < 5; i++ {
		if err := repo.CreateBatch(context.Background(), []*data.OperationLog{
			{ProjectID: ptr(pid), OperatorID: uid(), Action: data.ActionTaskCreate, Detail: "{}"},
		}); err != nil {
			t.Fatalf("create batch %d: %v", i, err)
		}
	}

	logs, nextCursor, err := repo.ListByProject(context.Background(), pid, 20, "")
	if err != nil {
		t.Fatalf("list by project: %v", err)
	}
	if len(logs) != 5 {
		t.Errorf("got %d logs, want 5", len(logs))
	}
	if nextCursor != "" {
		t.Error("nextCursor should be empty when results < limit")
	}
}

func TestOpLogListByProject_WithCursor(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewOperationLogRepository(db)

	pid := uid()
	for i := 0; i < 5; i++ {
		if err := repo.CreateBatch(context.Background(), []*data.OperationLog{
			{ProjectID: ptr(pid), OperatorID: uid(), Action: data.ActionTaskCreate, Detail: "{}"},
		}); err != nil {
			t.Fatalf("create batch %d: %v", i, err)
		}
	}

	logs, nextCursor, err := repo.ListByProject(context.Background(), pid, 3, "")
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("got %d logs, want 3", len(logs))
	}
	if nextCursor == "" {
		t.Error("nextCursor should be set when results >= limit")
	}

	logs2, nextCursor2, err := repo.ListByProject(context.Background(), pid, 3, nextCursor)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(logs2) != 2 {
		t.Errorf("got %d logs on second page, want 2", len(logs2))
	}
	if nextCursor2 != "" {
		t.Error("nextCursor should be empty on last page")
	}
}

func TestOpLogListByTask(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewOperationLogRepository(db)

	tid := uid()
	for i := 0; i < 3; i++ {
		if err := repo.CreateBatch(context.Background(), []*data.OperationLog{
			{TaskID: ptr(tid), OperatorID: uid(), Action: data.ActionCommentCreate, Detail: "{}"},
		}); err != nil {
			t.Fatalf("create batch %d: %v", i, err)
		}
	}

	logs, _, err := repo.ListByTask(context.Background(), tid, 20, "")
	if err != nil {
		t.Fatalf("list by task: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("got %d logs, want 3", len(logs))
	}
}

func TestOpLogListByTask_Empty(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewOperationLogRepository(db)

	logs, _, err := repo.ListByTask(context.Background(), uid(), 20, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("got %d logs, want 0", len(logs))
	}
}

func TestOpLogListByProject_InvalidCursor(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewOperationLogRepository(db)

	_, _, err := repo.ListByProject(context.Background(), uid(), 20, "not-valid-base64!")
	if err == nil {
		t.Error("expected error for invalid cursor")
	}
}
