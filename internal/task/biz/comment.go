package biz

import (
	"context"

	"gorm.io/gorm"

	"task-platform/internal/task/data"
	"task-platform/pkg/xerr"
)

type CommentBiz struct {
	db          *gorm.DB
	commentRepo data.CommentRepository
	taskRepo    data.TaskRepository
	projectRepo data.ProjectRepository
	memberRepo  data.MemberRepository
	logWriter   *LogWriter
}

func NewCommentBiz(db *gorm.DB, commentRepo data.CommentRepository, taskRepo data.TaskRepository, projectRepo data.ProjectRepository, memberRepo data.MemberRepository, logWriter *LogWriter) *CommentBiz {
	return &CommentBiz{
		db:          db,
		commentRepo: commentRepo,
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		logWriter:   logWriter,
	}
}

func (b *CommentBiz) CreateComment(ctx context.Context, taskID, callerID, content string) (*data.TaskComment, error) {
	if content == "" {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "comment content is required")
	}
	if !data.IsValidContent(content) {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "comment content must be at most 10000 characters")
	}

	task, _, err := b.getTaskAndMemberForWrite(ctx, taskID, callerID)
	if err != nil {
		return nil, err
	}

	comment := &data.TaskComment{
		TaskID:  taskID,
		UserID:  callerID,
		Content: content,
	}
	if err := b.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	projectID := task.ProjectID
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		TaskID:     &taskID,
		OperatorID: callerID,
		Action:     data.ActionCommentCreate,
		Detail:     jsonDetail(map[string]string{"comment_id": comment.ID}),
	})

	return comment, nil
}

func (b *CommentBiz) DeleteComment(ctx context.Context, taskID, commentID, callerID string) (*data.TaskComment, error) {
	task, member, err := b.getTaskAndMemberForWrite(ctx, taskID, callerID)
	if err != nil {
		return nil, err
	}

	comment, err := b.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment.TaskID != taskID {
		return nil, xerr.NewError(xerr.CodeNotFound, "comment not found")
	}

	if comment.UserID != callerID && member.Role != data.RoleOwner && member.Role != data.RoleAdmin {
		return nil, xerr.NewError(xerr.CodePermissionDenied, "no permission to delete this comment")
	}

	deleted, err := b.commentRepo.Delete(ctx, commentID)
	if err != nil {
		return nil, err
	}

	projectID := task.ProjectID
	b.logWriter.Enqueue(ctx, &data.OperationLog{
		ProjectID:  &projectID,
		TaskID:     &taskID,
		OperatorID: callerID,
		Action:     data.ActionCommentDelete,
		Detail:     jsonDetail(map[string]string{"comment_id": commentID}),
	})

	return deleted, nil
}

func (b *CommentBiz) ListComments(ctx context.Context, taskID, callerID string, limit int32, afterID string) ([]*data.TaskComment, error) {
	_, _, err := b.getTaskAndMember(ctx, taskID, callerID)
	if err != nil {
		return nil, err
	}

	return b.commentRepo.ListByTask(ctx, taskID, limit, afterID)
}

func (b *CommentBiz) getTaskAndMemberForWrite(ctx context.Context, taskID, callerID string) (*data.Task, *data.ProjectMember, error) {
	task, member, err := b.getTaskAndMember(ctx, taskID, callerID)
	if err != nil {
		return nil, nil, err
	}

	project, err := b.projectRepo.FindByID(ctx, task.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if project.Status == data.ProjectStatusArchived {
		return nil, nil, xerr.NewError(xerr.CodeFailedPrecondition, "archived project is read-only")
	}

	return task, member, nil
}

func (b *CommentBiz) getTaskAndMember(ctx context.Context, taskID, callerID string) (*data.Task, *data.ProjectMember, error) {
	task, err := b.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, nil, err
	}

	member, err := b.memberRepo.FindByProjectAndUser(ctx, task.ProjectID, callerID)
	if err != nil {
		return nil, nil, err
	}
	if member == nil {
		return nil, nil, xerr.NewError(xerr.CodeNotFound, "task not found")
	}

	return task, member, nil
}
