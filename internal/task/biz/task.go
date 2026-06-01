package biz

import (
	"context"

	"gorm.io/gorm"

	"task-platform/internal/task/data"
	"task-platform/pkg/xcursor"
	"task-platform/pkg/xerr"
)

type TaskListFilter struct {
	Status     *int32
	AssigneeID string
	Keyword    string
	Limit      int
	Cursor     string
}

type TaskBiz struct {
	db          *gorm.DB
	taskRepo    data.TaskRepository
	projectRepo data.ProjectRepository
	memberRepo  data.MemberRepository
	userClient  UserServiceClient
	logWriter   *LogWriter
}

func NewTaskBiz(db *gorm.DB, taskRepo data.TaskRepository, projectRepo data.ProjectRepository, memberRepo data.MemberRepository, userClient UserServiceClient, logWriter *LogWriter) *TaskBiz {
	return &TaskBiz{
		db:          db,
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		userClient:  userClient,
		logWriter:   logWriter,
	}
}

var validTransitions = map[int32][]int32{
	data.TaskStatusTodo:      {data.TaskStatusDoing, data.TaskStatusDone, data.TaskStatusCancelled},
	data.TaskStatusDoing:     {data.TaskStatusDone, data.TaskStatusCancelled, data.TaskStatusTodo},
	data.TaskStatusDone:      {data.TaskStatusDoing},
	data.TaskStatusCancelled: {data.TaskStatusTodo},
}

func isValidTransition(from, to int32) bool {
	for _, valid := range validTransitions[from] {
		if valid == to {
			return true
		}
	}
	return false
}

func (b *TaskBiz) CreateTask(ctx context.Context, projectID, callerID, title, content string) (*data.Task, error) {
	if !data.IsValidTaskTitle(title) {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "task title must be 1-200 characters")
	}

	project, member, err := b.getProjectAndMember(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}
	_ = member

	task := &data.Task{
		ProjectID: projectID,
		Title:     title,
		Content:   content,
		CreatorID: callerID,
		Status:    data.TaskStatusTodo,
		Priority:  data.PriorityNormal,
		Extra:     "{}",
	}
	if err := b.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		TaskID:     &task.ID,
		OperatorID: callerID,
		Action:     data.ActionTaskCreate,
		Detail:     `{}`,
	})
	return task, nil
}

func (b *TaskBiz) GetTask(ctx context.Context, taskID, callerID string) (*data.Task, error) {
	task, err := b.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	member, err := b.memberRepo.FindByProjectAndUser(ctx, task.ProjectID, callerID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, xerr.NewError(xerr.CodeNotFound, "task not found")
	}

	return task, nil
}

func (b *TaskBiz) UpdateTask(ctx context.Context, taskID, callerID, title, content string, priority int32, dueTime string, version int64) (*data.Task, error) {
	if title != "" && !data.IsValidTaskTitle(title) {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "task title must be 1-200 characters")
	}

	task, err := b.getTaskWithPermission(ctx, taskID, callerID)
	if err != nil {
		return nil, err
	}

	if err := b.requireWriteAccess(ctx, task, callerID); err != nil {
		return nil, err
	}

	updates := make(map[string]any)
	if title != "" {
		updates["title"] = title
	}
	updates["content"] = content
	updates["priority"] = priority
	if dueTime != "" {
		updates["due_time"] = dueTime
	}
	updates["version"] = task.Version + 1

	updated, err := b.taskRepo.Update(ctx, taskID, version, updates)
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &task.ProjectID,
		TaskID:     &taskID,
		OperatorID: callerID,
		Action:     data.ActionTaskUpdate,
		Detail:     `{}`,
	})
	return updated, nil
}

func (b *TaskBiz) DeleteTask(ctx context.Context, taskID, callerID string) (*data.Task, error) {
	task, err := b.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	project, err := b.projectRepo.FindByID(ctx, task.ProjectID)
	if err != nil {
		return nil, err
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}

	member, err := b.memberRepo.FindByProjectAndUser(ctx, task.ProjectID, callerID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, xerr.NewError(xerr.CodeNotFound, "task not found")
	}

	if member.Role == data.RoleMember {
		if task.CreatorID != callerID {
			return nil, xerr.NewError(xerr.CodePermissionDenied, "members can only delete their own tasks")
		}
		if task.Status != data.TaskStatusTodo {
			return nil, xerr.NewError(xerr.CodePermissionDenied, "members can only delete tasks in todo status")
		}
	}

	deleted, err := b.taskRepo.Delete(ctx, taskID)
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &task.ProjectID,
		TaskID:     &taskID,
		OperatorID: callerID,
		Action:     data.ActionTaskDelete,
		Detail:     `{}`,
	})
	return deleted, nil
}

func (b *TaskBiz) ListTasks(ctx context.Context, projectID, callerID string, filter TaskListFilter) ([]*data.Task, string, error) {
	member, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
	if err != nil {
		return nil, "", err
	}
	if member == nil {
		return nil, "", xerr.NewError(xerr.CodeNotFound, "project not found")
	}

	statusStr := ""
	if filter.Status != nil {
		statusStr = string(rune(*filter.Status + '0'))
	}
	filterHash := xcursor.ComputeFilterHash(projectID, statusStr, filter.AssigneeID, filter.Keyword)

	if filter.Cursor != "" {
		fields, err := xcursor.DecodeCursor(filter.Cursor)
		if err != nil {
			return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "invalid cursor")
		}
		if storedHash, ok := fields["filter_hash"].(string); ok {
			if storedHash != filterHash {
				return nil, "", xerr.NewError(xerr.CodeInvalidArgument, "filter parameters changed, please re-fetch from first page")
			}
		}
	}

	if filter.Limit <= 0 || filter.Limit > 50 {
		filter.Limit = 20
	}

	tasks, nextCursor, err := b.taskRepo.List(ctx, data.TaskFilter{
		ProjectID:  projectID,
		Status:     filter.Status,
		AssigneeID: filter.AssigneeID,
		Keyword:    filter.Keyword,
		Limit:      filter.Limit,
		Cursor:     filter.Cursor,
		FilterHash: filterHash,
	})
	if err != nil {
		return nil, "", err
	}

	if nextCursor != "" {
		fields, err := xcursor.DecodeCursor(nextCursor)
		if err == nil {
			fields["filter_hash"] = filterHash
			var encErr error
			nextCursor, encErr = xcursor.EncodeCursor(fields)
			if encErr != nil {
				nextCursor = ""
			}
		}
	}

	return tasks, nextCursor, nil
}

func (b *TaskBiz) AssignTask(ctx context.Context, taskID, callerID, assigneeID string) (*data.Task, error) {
	task, err := b.getTaskWithPermission(ctx, taskID, callerID)
	if err != nil {
		return nil, err
	}

	member, err := b.memberRepo.FindByProjectAndUser(ctx, task.ProjectID, callerID)
	if err != nil {
		return nil, err
	}
	if member.Role == data.RoleMember && task.CreatorID != callerID {
		return nil, xerr.NewError(xerr.CodePermissionDenied, "members can only assign their own tasks")
	}

	targetMember, err := b.memberRepo.FindByProjectAndUser(ctx, task.ProjectID, assigneeID)
	if err != nil {
		return nil, err
	}
	if targetMember == nil {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "assignee is not a project member")
	}

	exists, active, err := b.userClient.GetUser(ctx, assigneeID)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "validate assignee failed")
	}
	if !exists || !active {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "assignee does not exist or is disabled")
	}

	updated, err := b.taskRepo.Update(ctx, taskID, task.Version, map[string]any{
		"assignee_id": assigneeID,
		"version":     task.Version + 1,
	})
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &task.ProjectID,
		TaskID:     &taskID,
		OperatorID: callerID,
		Action:     data.ActionTaskAssign,
		Detail:     jsonDetail(map[string]string{"assignee_id": assigneeID}),
	})
	return updated, nil
}

func (b *TaskBiz) ChangeTaskStatus(ctx context.Context, taskID, callerID string, status int32, version int64) (*data.Task, error) {
	task, err := b.getTaskWithPermission(ctx, taskID, callerID)
	if err != nil {
		return nil, err
	}

	member, err := b.memberRepo.FindByProjectAndUser(ctx, task.ProjectID, callerID)
	if err != nil {
		return nil, err
	}
	if member.Role == data.RoleMember && task.CreatorID != callerID {
		return nil, xerr.NewError(xerr.CodePermissionDenied, "members can only change status of their own tasks")
	}

	if task.Status == status {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "task is already in this status")
	}
	if !isValidTransition(task.Status, status) {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "invalid status transition")
	}

	updated, err := b.taskRepo.Update(ctx, taskID, version, map[string]any{
		"status":  status,
		"version": task.Version + 1,
	})
	if err != nil {
		return nil, err
	}
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &task.ProjectID,
		TaskID:     &taskID,
		OperatorID: callerID,
		Action:     data.ActionTaskStatusChange,
		Detail:     jsonDetail(map[string]int32{"from_status": task.Status, "to_status": status}),
	})
	return updated, nil
}

func (b *TaskBiz) getTaskWithPermission(ctx context.Context, taskID, callerID string) (*data.Task, error) {
	task, err := b.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	project, err := b.projectRepo.FindByID(ctx, task.ProjectID)
	if err != nil {
		return nil, err
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}

	member, err := b.memberRepo.FindByProjectAndUser(ctx, task.ProjectID, callerID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, xerr.NewError(xerr.CodeNotFound, "task not found")
	}

	return task, nil
}

func (b *TaskBiz) requireWriteAccess(ctx context.Context, task *data.Task, callerID string) error {
	member, err := b.memberRepo.FindByProjectAndUser(ctx, task.ProjectID, callerID)
	if err != nil {
		return err
	}
	if member.Role == data.RoleMember && task.CreatorID != callerID {
		return xerr.NewError(xerr.CodePermissionDenied, "members can only edit their own tasks")
	}
	return nil
}

func (b *TaskBiz) getProjectAndMember(ctx context.Context, projectID, callerID string) (*data.Project, *data.ProjectMember, error) {
	project, err := b.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}

	if project.Status == data.ProjectStatusArchived {
		member, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
		if err != nil {
			return nil, nil, err
		}
		if member == nil {
			return nil, nil, xerr.NewError(xerr.CodeNotFound, "project not found")
		}
		return project, member, nil
	}

	member, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
	if err != nil {
		return nil, nil, err
	}
	if member == nil {
		return nil, nil, xerr.NewError(xerr.CodeNotFound, "project not found")
	}

	return project, member, nil
}
