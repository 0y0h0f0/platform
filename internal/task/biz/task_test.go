package biz_test

import (
	"context"
	"testing"

	"task-platform/internal/task/biz"
	"task-platform/internal/task/data"
)

func setupTaskBiz(t *testing.T) (projectBiz *biz.ProjectBiz, taskBiz *biz.TaskBiz, cleanup func(), caller, other string) {
	t.Helper()
	db, clean := setupBizDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	taskRepo := data.NewTaskRepository(db)
	userClient := &mockUserClient{exists: true, active: true}

	projectBiz = biz.NewProjectBiz(db, projectRepo, memberRepo, userClient)
	taskBiz = biz.NewTaskBiz(db, taskRepo, projectRepo, memberRepo, userClient)
	caller = uid()
	other = uid()
	cleanup = clean
	return
}

func TestCreateTask_Success(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "TaskProject", "")

	task, err := tb.CreateTask(context.Background(), project.ID, caller, "My Task", "Task content")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.ID == "" {
		t.Error("task ID should be set")
	}
	if task.CreatorID != caller {
		t.Errorf("creator = %s, want %s", task.CreatorID, caller)
	}
	if task.Status != data.TaskStatusTodo {
		t.Errorf("status = %d, want %d", task.Status, data.TaskStatusTodo)
	}
}

func TestCreateTask_EmptyTitle(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, err := tb.CreateTask(context.Background(), project.ID, caller, "", "")
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestCreateTask_NonMember(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	nonMember := uid()
	_, err := tb.CreateTask(context.Background(), project.ID, nonMember, "Task", "")
	if err == nil {
		t.Error("expected not found for non-member")
	}
}

func TestCreateTask_ArchivedProject(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.ArchiveProject(context.Background(), project.ID, caller)
	_, err := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")
	if err == nil {
		t.Error("expected failed precondition for archived project")
	}
}

func TestGetTask_Success(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	got, err := tb.GetTask(context.Background(), task.ID, caller)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.ID != task.ID {
		t.Errorf("id mismatch")
	}
}

func TestGetTask_NonMember(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	nonMember := uid()
	_, err := tb.GetTask(context.Background(), task.ID, nonMember)
	if err == nil {
		t.Error("expected not found for non-member")
	}
}

func TestUpdateTask_Success(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	updated, err := tb.UpdateTask(context.Background(), task.ID, caller, "New Title", "New Content", data.PriorityHigh, "", task.Version)
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.Title != "New Title" {
		t.Errorf("title = %s", updated.Title)
	}
	if updated.Priority != data.PriorityHigh {
		t.Errorf("priority = %d", updated.Priority)
	}
}

func TestUpdateTask_OwnerCanEditAny(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Task", "")

	_, err := tb.UpdateTask(context.Background(), task.ID, caller, "New", "", 0, "", task.Version)
	if err != nil {
		t.Errorf("owner should be able to edit any task: %v", err)
	}
}

func TestUpdateTask_MemberCanEditOwnTask(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Member's Task", "")

	_, err := tb.UpdateTask(context.Background(), task.ID, other, "Updated", "", 0, "", task.Version)
	if err != nil {
		t.Errorf("member should be able to edit own task: %v", err)
	}
}

func TestUpdateTask_MemberCannotEditOthersTask(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	u3 := uid()
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, u3, data.RoleMember)

	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Other's Task", "")
	_, err := tb.UpdateTask(context.Background(), task.ID, u3, "Hacked", "", 0, "", task.Version)
	if err == nil {
		t.Error("member should not be able to edit someone else's task")
	}
}

func TestUpdateTask_VersionConflict(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	_, err := tb.UpdateTask(context.Background(), task.ID, caller, "New", "", 0, "", 999)
	if err == nil {
		t.Error("expected version conflict")
	}
}

func TestUpdateTask_Archived(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")
	_, _ = pb.ArchiveProject(context.Background(), project.ID, caller)

	_, err := tb.UpdateTask(context.Background(), task.ID, caller, "New", "", 0, "", task.Version)
	if err == nil {
		t.Error("expected failed precondition for archived project")
	}
}

func TestDeleteTask_OwnerCanDeleteAny(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Member's Task", "")

	_, err := tb.DeleteTask(context.Background(), task.ID, caller)
	if err != nil {
		t.Errorf("owner should be able to delete any task: %v", err)
	}
}

func TestDeleteTask_MemberCanDeleteOwnTodo(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Todo Task", "")

	_, err := tb.DeleteTask(context.Background(), task.ID, other)
	if err != nil {
		t.Errorf("member should be able to delete own todo task: %v", err)
	}
}

func TestDeleteTask_MemberCannotDeleteOthers(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	u3 := uid()
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, u3, data.RoleMember)

	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Task", "")
	_, err := tb.DeleteTask(context.Background(), task.ID, u3)
	if err == nil {
		t.Error("member should not delete another's task")
	}
}

func TestDeleteTask_MemberCannotDeleteNonTodo(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Task", "")
	_, _ = tb.ChangeTaskStatus(context.Background(), task.ID, other, data.TaskStatusDoing, task.Version)

	_, _ = tb.GetTask(context.Background(), task.ID, other)
	_, err := tb.DeleteTask(context.Background(), task.ID, other)
	if err == nil {
		t.Error("member should not delete non-todo task")
	}
}

func TestAssignTask_OwnerCanAssignAny(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Task", "")

	assigned, err := tb.AssignTask(context.Background(), task.ID, caller, other)
	if err != nil {
		t.Fatalf("AssignTask: %v", err)
	}
	if assigned.AssigneeID == nil || *assigned.AssigneeID != other {
		t.Errorf("assignee should be %s", other)
	}
}

func TestAssignTask_TargetNotMember(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	nonMember := uid()
	_, err := tb.AssignTask(context.Background(), task.ID, caller, nonMember)
	if err == nil {
		t.Error("expected failed precondition for non-member assignee")
	}
}

func TestAssignTask_TargetDisabled(t *testing.T) {
	db, cleanup := setupBizDB(t)
	defer cleanup()
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	taskRepo := data.NewTaskRepository(db)
	userClient := &mockUserClient{exists: true, active: false}
	pb := biz.NewProjectBiz(db, projectRepo, memberRepo, userClient)
	tb := biz.NewTaskBiz(db, taskRepo, projectRepo, memberRepo, userClient)

	caller := uid()
	project, _ := pb.CreateProject(context.Background(), caller, "P", "")

	disabledID := uid()
	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: project.ID, UserID: disabledID, Role: data.RoleMember})

	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")
	_, err := tb.AssignTask(context.Background(), task.ID, caller, disabledID)
	if err == nil {
		t.Error("expected failed precondition for disabled assignee")
	}
}

func TestAssignTask_MemberCanAssignOwnTask(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	task, _ := tb.CreateTask(context.Background(), project.ID, other, "My Task", "")

	_, err := tb.AssignTask(context.Background(), task.ID, other, other)
	if err != nil {
		t.Errorf("member should be able to assign own task: %v", err)
	}
}

func TestAssignTask_MemberCannotAssignOthersTask(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	u3 := uid()
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, u3, data.RoleMember)

	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Task", "")
	_, err := tb.AssignTask(context.Background(), task.ID, u3, u3)
	if err == nil {
		t.Error("member should not assign someone else's task")
	}
}

func TestChangeTaskStatus_Success(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	changed, err := tb.ChangeTaskStatus(context.Background(), task.ID, caller, data.TaskStatusDoing, task.Version)
	if err != nil {
		t.Fatalf("ChangeTaskStatus: %v", err)
	}
	if changed.Status != data.TaskStatusDoing {
		t.Errorf("status = %d, want %d", changed.Status, data.TaskStatusDoing)
	}
}

func TestChangeTaskStatus_InvalidTransition(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")
	// todo → cancelled is valid
	changed, _ := tb.ChangeTaskStatus(context.Background(), task.ID, caller, data.TaskStatusCancelled, task.Version)
	// cancelled → doing is invalid
	_, err := tb.ChangeTaskStatus(context.Background(), task.ID, caller, data.TaskStatusDoing, changed.Version)
	if err == nil {
		t.Error("expected failed precondition for invalid transition (cancelled -> doing)")
	}
}

func TestChangeTaskStatus_SameState(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	task, _ := tb.CreateTask(context.Background(), project.ID, caller, "Task", "")

	_, err := tb.ChangeTaskStatus(context.Background(), task.ID, caller, data.TaskStatusTodo, task.Version)
	if err == nil {
		t.Error("expected failed precondition for same state transition")
	}
}

func TestListTasks_WithFilterHash(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = tb.CreateTask(context.Background(), project.ID, caller, "Task 1", "")
	_, _ = tb.CreateTask(context.Background(), project.ID, caller, "Task 2", "")

	tasks, cursor, err := tb.ListTasks(context.Background(), project.ID, caller, biz.TaskListFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("len = %d, want 2", len(tasks))
	}
	if cursor != "" {
		t.Logf("cursor: %s", cursor)
	}
}

func TestListTasks_FilterHashMismatch(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	for i := 0; i < 5; i++ {
		_, _ = tb.CreateTask(context.Background(), project.ID, caller, "Task", "")
	}

	_, cursor, _ := tb.ListTasks(context.Background(), project.ID, caller, biz.TaskListFilter{Limit: 2})

	if cursor == "" {
		t.Fatal("expected a cursor")
	}

	todoStatus := data.TaskStatusTodo
	_, _, err := tb.ListTasks(context.Background(), project.ID, caller, biz.TaskListFilter{
		Limit: 2, Cursor: cursor, Status: &todoStatus,
	})
	if err == nil {
		t.Error("expected invalid argument for filter hash mismatch")
	}
}

func TestListTasks_NonMember(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	nonMember := uid()

	_, _, err := tb.ListTasks(context.Background(), project.ID, nonMember, biz.TaskListFilter{Limit: 2})
	if err == nil {
		t.Error("expected not found for non-member")
	}
}

func TestChangeTaskStatus_MemberCanChangeOwn(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	task, _ := tb.CreateTask(context.Background(), project.ID, other, "My Task", "")

	_, err := tb.ChangeTaskStatus(context.Background(), task.ID, other, data.TaskStatusDoing, task.Version)
	if err != nil {
		t.Errorf("member should be able to change own task status: %v", err)
	}
}

func TestChangeTaskStatus_MemberCannotChangeOthers(t *testing.T) {
	pb, tb, cleanup, caller, other := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, other, data.RoleMember)
	u3 := uid()
	_, _ = pb.AddProjectMember(context.Background(), project.ID, caller, u3, data.RoleMember)

	task, _ := tb.CreateTask(context.Background(), project.ID, other, "Task", "")
	_, err := tb.ChangeTaskStatus(context.Background(), task.ID, u3, data.TaskStatusDoing, task.Version)
	if err == nil {
		t.Error("member should not change someone else's task status")
	}
}

func TestListTasks_FilterByStatus(t *testing.T) {
	pb, tb, cleanup, caller, _ := setupTaskBiz(t)
	defer cleanup()

	project, _ := pb.CreateProject(context.Background(), caller, "P", "")
	t1, _ := tb.CreateTask(context.Background(), project.ID, caller, "Todo", "")
	t2, _ := tb.CreateTask(context.Background(), project.ID, caller, "Doing", "")
	_, _ = tb.ChangeTaskStatus(context.Background(), t2.ID, caller, data.TaskStatusDoing, t2.Version)

	todoStatus := data.TaskStatusTodo
	tasks, _, err := tb.ListTasks(context.Background(), project.ID, caller, biz.TaskListFilter{
		Limit: 20, Status: &todoStatus,
	})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	for _, task := range tasks {
		if task.Status != data.TaskStatusTodo {
			t.Errorf("expected only todo tasks, got status=%d for %s", task.Status, task.ID)
		}
	}
	// t1 should be in results
	found := false
	for _, task := range tasks {
		if task.ID == t1.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("todo task not in filtered results")
	}
}
