package biz

import (
	"context"

	"task-platform/internal/task/data"
	"task-platform/pkg/xerr"
)

type OpLogBiz struct {
	opLogRepo   data.OperationLogRepository
	projectRepo data.ProjectRepository
	taskRepo    data.TaskRepository
	memberRepo  data.MemberRepository
}

func NewOpLogBiz(opLogRepo data.OperationLogRepository, projectRepo data.ProjectRepository, taskRepo data.TaskRepository, memberRepo data.MemberRepository) *OpLogBiz {
	return &OpLogBiz{
		opLogRepo:   opLogRepo,
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
		memberRepo:  memberRepo,
	}
}

func (b *OpLogBiz) ListProjectLogs(ctx context.Context, projectID, callerID string, limit int, cursor string) ([]*data.OperationLog, string, error) {
	member, err := b.memberRepo.FindByProjectAndUser(ctx, projectID, callerID)
	if err != nil {
		return nil, "", xerr.NewError(xerr.CodeNotFound, "project not found")
	}
	if member == nil {
		return nil, "", xerr.NewError(xerr.CodeNotFound, "project not found")
	}
	return b.opLogRepo.ListByProject(ctx, projectID, limit, cursor)
}

func (b *OpLogBiz) ListTaskLogs(ctx context.Context, taskID, callerID string, limit int, cursor string) ([]*data.OperationLog, string, error) {
	task, err := b.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, "", xerr.NewError(xerr.CodeNotFound, "task not found")
	}
	member, err := b.memberRepo.FindByProjectAndUser(ctx, task.ProjectID, callerID)
	if err != nil {
		return nil, "", xerr.NewError(xerr.CodeNotFound, "task not found")
	}
	if member == nil {
		return nil, "", xerr.NewError(xerr.CodeNotFound, "task not found")
	}
	return b.opLogRepo.ListByTask(ctx, taskID, limit, cursor)
}
