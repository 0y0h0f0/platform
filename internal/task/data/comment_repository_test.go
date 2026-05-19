package data_test

import (
	"context"
	"testing"

	"task-platform/internal/task/data"
)

func TestCommentCreate_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewCommentRepository(db)

	comment := &data.TaskComment{TaskID: uid(), UserID: uid(), Content: "hello"}
	err := repo.Create(context.Background(), comment)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if comment.ID == "" {
		t.Error("ID should be set")
	}
}

func TestCommentFindByID_NotFound(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewCommentRepository(db)

	_, err := repo.FindByID(context.Background(), uid())
	if err == nil {
		t.Error("expected error for non-existent comment")
	}
}

func TestCommentFindByID_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewCommentRepository(db)

	comment := &data.TaskComment{TaskID: uid(), UserID: uid(), Content: "findable"}
	_ = repo.Create(context.Background(), comment)

	found, err := repo.FindByID(context.Background(), comment.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.Content != "findable" {
		t.Errorf("content = %s, want findable", found.Content)
	}
}

func TestCommentDelete_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewCommentRepository(db)

	comment := &data.TaskComment{TaskID: uid(), UserID: uid(), Content: "delete me"}
	_ = repo.Create(context.Background(), comment)

	deleted, err := repo.Delete(context.Background(), comment.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted.ID != comment.ID {
		t.Error("deleted comment ID mismatch")
	}

	_, err = repo.FindByID(context.Background(), comment.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestCommentDelete_NotFound(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewCommentRepository(db)

	_, err := repo.Delete(context.Background(), uid())
	if err == nil {
		t.Error("expected error for non-existent comment")
	}
}

func TestCommentListByTask_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewCommentRepository(db)

	taskID := uid()
	if err := repo.Create(context.Background(), &data.TaskComment{TaskID: taskID, UserID: uid(), Content: "c1"}); err != nil {
		t.Fatalf("create c1: %v", err)
	}
	if err := repo.Create(context.Background(), &data.TaskComment{TaskID: taskID, UserID: uid(), Content: "c2"}); err != nil {
		t.Fatalf("create c2: %v", err)
	}
	if err := repo.Create(context.Background(), &data.TaskComment{TaskID: taskID, UserID: uid(), Content: "c3"}); err != nil {
		t.Fatalf("create c3: %v", err)
	}

	comments, err := repo.ListByTask(context.Background(), taskID, 20, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(comments) != 3 {
		t.Errorf("got %d comments, want 3", len(comments))
	}
}

func TestCommentListByTask_WithAfterID(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewCommentRepository(db)

	taskID := uid()
	c1, _ := createComment(t, repo, taskID, "first")
	createComment(t, repo, taskID, "second")

	comments, err := repo.ListByTask(context.Background(), taskID, 20, c1.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("got %d comments, want 1", len(comments))
	}
}

func TestCommentListByTask_Empty(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewCommentRepository(db)

	comments, err := repo.ListByTask(context.Background(), uid(), 20, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("got %d comments, want 0", len(comments))
	}
}

func TestCommentListByTask_InvalidAfterID(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewCommentRepository(db)

	_, err := repo.ListByTask(context.Background(), uid(), 20, "nonexistent-id")
	if err == nil {
		t.Error("expected error for invalid after_id")
	}
}

func createComment(t *testing.T, repo data.CommentRepository, taskID, content string) (*data.TaskComment, string) {
	t.Helper()
	c := &data.TaskComment{TaskID: taskID, UserID: uid(), Content: content}
	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("create comment: %v", err)
	}
	return c, c.ID
}
