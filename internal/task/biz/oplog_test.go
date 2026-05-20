package biz_test

import (
	"context"
	"testing"

	"task-platform/internal/task/biz"
	"task-platform/internal/task/data"
)

func setupOpLogBiz(t *testing.T) (projectBiz *biz.ProjectBiz, taskBiz *biz.TaskBiz, opLogBiz *biz.OpLogBiz, opLogRepo data.OperationLogRepository, cleanup func(), caller, other string) {
	t.Helper()
	db, clean := setupBizDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	taskRepo := data.NewTaskRepository(db)
	opLogRepo = data.NewOperationLogRepository(db)
	userClient := &mockUserClient{exists: true, active: true}

	projectBiz = biz.NewProjectBiz(db, projectRepo, memberRepo, userClient, nil, nil)
	taskBiz = biz.NewTaskBiz(db, taskRepo, projectRepo, memberRepo, userClient, nil)
	opLogBiz = biz.NewOpLogBiz(opLogRepo, projectRepo, taskRepo, memberRepo)
	caller = uid()
	other = uid()
	cleanup = clean
	return
}

func createProjectAndMember(t *testing.T, pb *biz.ProjectBiz, caller string) string {
	t.Helper()
	project, err := pb.CreateProject(context.Background(), caller, "OpLogTestProject", "")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	return project.ID
}

func insertOpLogs(t *testing.T, repo data.OperationLogRepository, projectID string, taskID *string, n int) {
	t.Helper()
	logs := make([]*data.OperationLog, n)
	for i := 0; i < n; i++ {
		logs[i] = &data.OperationLog{
			ProjectID:  &projectID,
			TaskID:     taskID,
			OperatorID: uid(),
			Action:     "test.action",
			Detail:     "{}",
		}
	}
	if err := repo.CreateBatch(context.Background(), logs); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
}

func TestNewOpLogBiz(t *testing.T) {
	db, clean := setupBizDB(t)
	defer clean()

	b := biz.NewOpLogBiz(
		data.NewOperationLogRepository(db),
		data.NewProjectRepository(db),
		data.NewTaskRepository(db),
		data.NewMemberRepository(db),
	)
	if b == nil {
		t.Fatal("NewOpLogBiz returned nil")
	}
}

func TestListProjectLogs_Success(t *testing.T) {
	pb, _, ob, opLogRepo, cleanup, caller, _ := setupOpLogBiz(t)
	defer cleanup()

	projectID := createProjectAndMember(t, pb, caller)
	insertOpLogs(t, opLogRepo, projectID, nil, 5)

	logs, nextCursor, err := ob.ListProjectLogs(context.Background(), projectID, caller, 20, "")
	if err != nil {
		t.Fatalf("ListProjectLogs: %v", err)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs, got %d", len(logs))
	}
	_ = nextCursor
}

func TestListProjectLogs_WithCursor(t *testing.T) {
	pb, _, ob, opLogRepo, cleanup, caller, _ := setupOpLogBiz(t)
	defer cleanup()

	projectID := createProjectAndMember(t, pb, caller)
	insertOpLogs(t, opLogRepo, projectID, nil, 5)

	logs, nextCursor, err := ob.ListProjectLogs(context.Background(), projectID, caller, 3, "")
	if err != nil {
		t.Fatalf("ListProjectLogs page 1: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("page 1: expected 3 logs, got %d", len(logs))
	}

	logs2, _, err := ob.ListProjectLogs(context.Background(), projectID, caller, 3, nextCursor)
	if err != nil {
		t.Fatalf("ListProjectLogs page 2: %v", err)
	}
	if len(logs2) != 2 {
		t.Errorf("page 2: expected 2 logs, got %d", len(logs2))
	}
}

func TestListProjectLogs_NonMember(t *testing.T) {
	pb, _, ob, _, cleanup, caller, other := setupOpLogBiz(t)
	defer cleanup()

	projectID := createProjectAndMember(t, pb, caller)

	_, _, err := ob.ListProjectLogs(context.Background(), projectID, other, 20, "")
	if err == nil {
		t.Error("expected error for non-member")
	}
}

func TestListProjectLogs_ProjectNotFound(t *testing.T) {
	_, _, ob, _, cleanup, _, _ := setupOpLogBiz(t)
	defer cleanup()

	_, _, err := ob.ListProjectLogs(context.Background(), uid(), uid(), 20, "")
	if err == nil {
		t.Error("expected error for non-existent project")
	}
}

func TestListTaskLogs_Success(t *testing.T) {
	pb, tb, ob, opLogRepo, cleanup, caller, _ := setupOpLogBiz(t)
	defer cleanup()

	projectID := createProjectAndMember(t, pb, caller)
	task, err := tb.CreateTask(context.Background(), projectID, caller, "Test task", "")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	taskID := task.ID
	insertOpLogs(t, opLogRepo, projectID, &taskID, 3)

	logs, _, err := ob.ListTaskLogs(context.Background(), task.ID, caller, 20, "")
	if err != nil {
		t.Fatalf("ListTaskLogs: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}
}

func TestListTaskLogs_NonMember(t *testing.T) {
	pb, tb, ob, _, cleanup, caller, other := setupOpLogBiz(t)
	defer cleanup()

	projectID := createProjectAndMember(t, pb, caller)
	task, err := tb.CreateTask(context.Background(), projectID, caller, "Test task", "")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	_, _, err = ob.ListTaskLogs(context.Background(), task.ID, other, 20, "")
	if err == nil {
		t.Error("expected error for non-member")
	}
}

func TestListTaskLogs_TaskNotFound(t *testing.T) {
	_, _, ob, _, cleanup, caller, _ := setupOpLogBiz(t)
	defer cleanup()

	_, _, err := ob.ListTaskLogs(context.Background(), uid(), caller, 20, "")
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}
