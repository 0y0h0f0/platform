package data_test

import (
	"context"
	"testing"

	"task-platform/internal/task/data"
	"task-platform/pkg/xcursor"
)

func TestTaskCreate_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	task := &data.Task{ProjectID: uid(), Title: "Test Task", Content: "Content", CreatorID: uid()}
	err := repo.Create(context.Background(), task)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if task.ID == "" {
		t.Error("ID should be set")
	}
}

func TestTaskFindByID_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	task := &data.Task{ProjectID: uid(), Title: "FindMe", CreatorID: uid()}
	_ = repo.Create(context.Background(), task)

	found, err := repo.FindByID(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.Title != "FindMe" {
		t.Errorf("title = %s, want FindMe", found.Title)
	}
}

func TestTaskFindByID_NotFound(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	_, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestTaskUpdate_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	task := &data.Task{ProjectID: uid(), Title: "Old Title", CreatorID: uid()}
	_ = repo.Create(context.Background(), task)

	updated, err := repo.Update(context.Background(), task.ID, task.Version, map[string]any{
		"title":   "New Title",
		"version": task.Version + 1,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "New Title" {
		t.Errorf("title = %s, want New Title", updated.Title)
	}
	if updated.Version != 1 {
		t.Errorf("version = %d, want 1", updated.Version)
	}
}

func TestTaskUpdate_VersionConflict(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	task := &data.Task{ProjectID: uid(), Title: "Conflict", CreatorID: uid()}
	_ = repo.Create(context.Background(), task)

	_, err := repo.Update(context.Background(), task.ID, 999, map[string]any{"title": "Hacked"})
	if err == nil {
		t.Error("expected version conflict error")
	}
}

func TestTaskDelete_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	task := &data.Task{ProjectID: uid(), Title: "To Delete", CreatorID: uid()}
	_ = repo.Create(context.Background(), task)

	deleted, err := repo.Delete(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted.ID != task.ID {
		t.Errorf("id mismatch")
	}

	_, err = repo.FindByID(context.Background(), task.ID)
	if err == nil {
		t.Error("expected not found after soft delete")
	}
}

func TestTaskList_NoCursor(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	projID := uid()
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "Task 1", CreatorID: uid()})
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "Task 2", CreatorID: uid()})

	tasks, nextCursor, err := repo.List(context.Background(), data.TaskFilter{
		ProjectID: projID, Limit: 20,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("len = %d, want 2", len(tasks))
	}
	if nextCursor != "" {
		t.Error("next_cursor should be empty when all items fit")
	}
}

func TestTaskList_CursorPagination(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	projID := uid()
	for i := 0; i < 5; i++ {
		_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "Task", CreatorID: uid()})
	}

	page1, cursor, err := repo.List(context.Background(), data.TaskFilter{
		ProjectID: projID, Limit: 2,
	})
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("page1 len = %d, want 2", len(page1))
	}
	if cursor == "" {
		t.Fatal("expected next_cursor")
	}

	page2, _, err := repo.List(context.Background(), data.TaskFilter{
		ProjectID: projID, Limit: 2, Cursor: cursor,
	})
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("page2 len = %d, want 2", len(page2))
	}

	for _, t1 := range page1 {
		for _, t2 := range page2 {
			if t1.ID == t2.ID {
				t.Errorf("duplicate task across pages: %s", t1.ID)
			}
		}
	}
}

func TestTaskList_FilterByStatus(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	projID := uid()
	todoStatus := data.TaskStatusTodo
	doingStatus := data.TaskStatusDoing
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "Todo Task", CreatorID: uid(), Status: data.TaskStatusTodo})
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "Doing Task", CreatorID: uid(), Status: data.TaskStatusDoing})

	tasks, _, err := repo.List(context.Background(), data.TaskFilter{
		ProjectID: projID, Limit: 20, Status: &todoStatus,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Todo Task" {
		t.Errorf("expected 1 todo task")
	}

	tasks, _, err = repo.List(context.Background(), data.TaskFilter{
		ProjectID: projID, Limit: 20, Status: &doingStatus,
	})
	if err != nil {
		t.Fatalf("list doing: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Doing Task" {
		t.Errorf("expected 1 doing task")
	}
}

func TestTaskList_FilterByAssignee(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	projID := uid()
	assignee := uid()
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "Assigned", CreatorID: uid(), AssigneeID: &assignee})
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "Unassigned", CreatorID: uid()})

	tasks, _, err := repo.List(context.Background(), data.TaskFilter{
		ProjectID: projID, Limit: 20, AssigneeID: assignee,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Assigned" {
		t.Errorf("expected 1 assigned task")
	}
}

func TestTaskList_InvalidCursor(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	_, _, err := repo.List(context.Background(), data.TaskFilter{
		ProjectID: uid(), Cursor: "bad-cursor!!!",
	})
	if err == nil {
		t.Error("expected invalid cursor error")
	}
}

func TestTaskList_KeywordFilter(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	projID := uid()
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "Fix login bug", CreatorID: uid()})
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "Update docs", CreatorID: uid()})

	tasks, _, err := repo.List(context.Background(), data.TaskFilter{
		ProjectID: projID, Limit: 20, Keyword: "login",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Fix login bug" {
		t.Errorf("expected 1 task matching 'login', got %d", len(tasks))
	}
}

func TestTaskList_ZeroLimitDefaults(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	projID := uid()
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "T1", CreatorID: uid()})
	_ = repo.Create(context.Background(), &data.Task{ProjectID: projID, Title: "T2", CreatorID: uid()})

	tasks, _, err := repo.List(context.Background(), data.TaskFilter{
		ProjectID: projID, Limit: 0,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("len = %d, want 2 (limit=0 defaults to 20)", len(tasks))
	}
}

func TestTaskList_CursorMissingFields(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewTaskRepository(db)

	// cursor with valid base64+json but missing required fields
	badCursor, _ := xcursor.EncodeCursor(map[string]any{"foo": "bar"})
	_, _, err := repo.List(context.Background(), data.TaskFilter{
		ProjectID: uid(), Cursor: badCursor,
	})
	if err == nil {
		t.Error("expected error for cursor missing required fields")
	}
}

func TestIsValidTaskTitle(t *testing.T) {
	if !data.IsValidTaskTitle("x") {
		t.Error("'x' should be valid")
	}
	if !data.IsValidTaskTitle(string(make([]byte, 200))) {
		t.Error("200 chars should be valid")
	}
	if data.IsValidTaskTitle("") {
		t.Error("empty title should be invalid")
	}
	if data.IsValidTaskTitle(string(make([]byte, 201))) {
		t.Error("201 chars should be invalid")
	}
}
