package biz_test

import (
	"context"
	"testing"

	"task-platform/internal/task/biz"
	"task-platform/internal/task/data"
)

func setupCommentBiz(t *testing.T) (projectBiz *biz.ProjectBiz, taskBiz *biz.TaskBiz, commentBiz *biz.CommentBiz, cleanup func(), caller, other string) {
	t.Helper()
	db, clean := setupBizDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	taskRepo := data.NewTaskRepository(db)
	commentRepo := data.NewCommentRepository(db)
	userClient := &mockUserClient{exists: true, active: true}

	projectBiz = biz.NewProjectBiz(db, projectRepo, memberRepo, userClient, nil)
	taskBiz = biz.NewTaskBiz(db, taskRepo, projectRepo, memberRepo, userClient, nil)
	commentBiz = biz.NewCommentBiz(db, commentRepo, taskRepo, projectRepo, memberRepo, nil)
	caller = uid()
	other = uid()
	cleanup = clean
	return
}

func TestCreateComment_Success(t *testing.T) {
	pb, tb, cb, cleanup, caller, _ := setupCommentBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "CommentProject", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task for comments", "")

	comment, err := cb.CreateComment(context.Background(), task.ID, caller, "test comment")
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if comment.ID == "" {
		t.Error("comment ID should be set")
	}
	if comment.UserID != caller {
		t.Errorf("user_id = %s, want %s", comment.UserID, caller)
	}
}

func TestCreateComment_EmptyContent(t *testing.T) {
	_, _, cb, cleanup, caller, _ := setupCommentBiz(t)
	defer cleanup()

	_, err := cb.CreateComment(context.Background(), "task-id", caller, "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestCreateComment_NonMember(t *testing.T) {
	pb, tb, cb, cleanup, caller, other := setupCommentBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	_, err := cb.CreateComment(context.Background(), task.ID, other, "test comment")
	if err == nil {
		t.Error("expected error for non-member")
	}
}

func TestDeleteComment_Author(t *testing.T) {
	pb, tb, cb, cleanup, caller, _ := setupCommentBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	comment, err := cb.CreateComment(context.Background(), task.ID, caller, "my comment")
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}

	_, err = cb.DeleteComment(context.Background(), task.ID, comment.ID, caller)
	if err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
}

func TestDeleteComment_ByOwner(t *testing.T) {
	pb, tb, cb, cleanup, caller, other := setupCommentBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, err := pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	if err != nil {
		t.Fatalf("AddProjectMember: %v", err)
	}
	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Task", "")

	comment, _ := cb.CreateComment(context.Background(), task.ID, other, "user comment")

	_, err = cb.DeleteComment(context.Background(), task.ID, comment.ID, caller)
	if err != nil {
		t.Fatalf("owner should delete any comment: %v", err)
	}
}

func TestDeleteComment_NoPermission(t *testing.T) {
	pb, tb, cb, cleanup, caller, other := setupCommentBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, err := pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	if err != nil {
		t.Fatalf("AddProjectMember: %v", err)
	}
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	comment, _ := cb.CreateComment(context.Background(), task.ID, caller, "owner comment")

	_, err = cb.DeleteComment(context.Background(), task.ID, comment.ID, other)
	if err == nil {
		t.Error("expected error for non-author member deleting another's comment")
	}
}

func TestListComments(t *testing.T) {
	pb, tb, cb, cleanup, caller, _ := setupCommentBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	if _, err := cb.CreateComment(context.Background(), task.ID, caller, "comment 1"); err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if _, err := cb.CreateComment(context.Background(), task.ID, caller, "comment 2"); err != nil {
		t.Fatalf("CreateComment: %v", err)
	}

	comments, err := cb.ListComments(context.Background(), task.ID, caller, 20, "")
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("got %d comments, want 2", len(comments))
	}
}

func TestCreateComment_ArchivedProject(t *testing.T) {
	pb, tb, cb, cleanup, caller, _ := setupCommentBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	_, err := pb.ArchiveProject(context.Background(), project.ID, caller)
	if err != nil {
		t.Fatalf("ArchiveProject: %v", err)
	}

	_, err = cb.CreateComment(context.Background(), task.ID, caller, "comment")
	if err == nil {
		t.Error("expected error for archived project")
	}
}

func TestListComments_NonMember(t *testing.T) {
	pb, tb, cb, cleanup, caller, other := setupCommentBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	_, err := cb.ListComments(context.Background(), task.ID, other, 20, "")
	if err == nil {
		t.Error("expected error for non-member")
	}
}

func TestDeleteComment_WrongTask(t *testing.T) {
	pb, tb, cb, cleanup, caller, _ := setupCommentBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task1, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task 1", "")
	task2, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task 2", "")

	comment, _ := cb.CreateComment(context.Background(), task1.ID, caller, "comment on task 1")

	_, err := cb.DeleteComment(context.Background(), task2.ID, comment.ID, caller)
	if err == nil {
		t.Error("expected error when deleting comment with wrong task ID")
	}
}
