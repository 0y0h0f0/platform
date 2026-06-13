package handler

import (
	"context"
	"log"

	userv1 "task-platform/gen/go/user/v1"
	taskv1 "task-platform/gen/go/task/v1"
)

type enrichedComment struct {
	Id        string `json:"id"`
	TaskId    string `json:"task_id"`
	UserId    string `json:"user_id"`
	Username  string `json:"username,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarUrl string `json:"avatar_url,omitempty"`
	Content   string `json:"content"`
}

type enrichedOperationLog struct {
	Id         string `json:"id"`
	ProjectId  string `json:"project_id,omitempty"`
	TaskId     string `json:"task_id,omitempty"`
	OperatorId string `json:"operator_id"`
	Username   string `json:"username,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarUrl  string `json:"avatar_url,omitempty"`
	Action     string `json:"action"`
	DetailJson string `json:"detail_json"`
}

type enrichedListCommentsResponse struct {
	Comments   []enrichedComment `json:"comments"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type enrichedListOpLogsResponse struct {
	Logs       []enrichedOperationLog `json:"logs"`
	NextCursor string                 `json:"next_cursor"`
}

func fetchUserMap(ctx context.Context, client userv1.UserServiceClient, ids []string) map[string]*userv1.User {
	if client == nil || len(ids) == 0 {
		return nil
	}
	deduped := dedupeStrings(ids)
	if len(deduped) > 100 {
		log.Printf("WARN: user enrichment truncated from %d to 100 IDs", len(deduped))
		deduped = deduped[:100]
	}
	res, err := client.BatchGetUsers(ctx, &userv1.BatchGetUsersRequest{UserIds: deduped})
	if err != nil {
		log.Printf("WARN: fetchUserMap failed, enrichment skipped: %v", err)
		return nil
	}
	m := make(map[string]*userv1.User, len(res.Users))
	for _, u := range res.Users {
		m[u.Id] = u
	}
	return m
}

func applyUserInfo(userMap map[string]*userv1.User, userID string) (string, string, string) {
	if u, ok := userMap[userID]; ok {
		return u.Username, u.Nickname, u.AvatarUrl
	}
	return "", "", ""
}

func enrichComments(ctx context.Context, client userv1.UserServiceClient, res *taskv1.ListTaskCommentsResponse) *enrichedListCommentsResponse {
	ids := make([]string, 0, len(res.Comments))
	for _, c := range res.Comments {
		ids = append(ids, c.UserId)
	}
	userMap := fetchUserMap(ctx, client, ids)

	comments := make([]enrichedComment, len(res.Comments))
	for i, c := range res.Comments {
		username, nickname, avatarUrl := applyUserInfo(userMap, c.UserId)
		comments[i] = enrichedComment{
			Id:        c.Id,
			TaskId:    c.TaskId,
			UserId:    c.UserId,
			Username:  username,
			Nickname:  nickname,
			AvatarUrl: avatarUrl,
			Content:   c.Content,
		}
	}
	return &enrichedListCommentsResponse{Comments: comments}
}

func enrichOperationLogs(ctx context.Context, client userv1.UserServiceClient, res *taskv1.ListOperationLogsResponse) *enrichedListOpLogsResponse {
	ids := make([]string, 0, len(res.Logs))
	for _, l := range res.Logs {
		ids = append(ids, l.OperatorId)
	}
	userMap := fetchUserMap(ctx, client, ids)

	logs := make([]enrichedOperationLog, len(res.Logs))
	for i, l := range res.Logs {
		username, nickname, avatarUrl := applyUserInfo(userMap, l.OperatorId)
		logs[i] = enrichedOperationLog{
			Id:         l.Id,
			ProjectId:  l.ProjectId,
			TaskId:     l.TaskId,
			OperatorId: l.OperatorId,
			Username:   username,
			Nickname:   nickname,
			AvatarUrl:  avatarUrl,
			Action:     l.Action,
			DetailJson: l.DetailJson,
		}
	}
	return &enrichedListOpLogsResponse{
		Logs:       logs,
		NextCursor: res.NextCursor,
	}
}

func dedupeStrings(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}



type enrichedProject struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	OwnerId        string `json:"owner_id"`
	OwnerUsername  string `json:"owner_username,omitempty"`
	OwnerNickname  string `json:"owner_nickname,omitempty"`
	OwnerAvatarUrl string `json:"owner_avatar_url,omitempty"`
	Status         int32  `json:"status"`
	Version        int64  `json:"version"`
}

type enrichedTask struct {
	Id                string `json:"id"`
	ProjectId         string `json:"project_id"`
	Title             string `json:"title"`
	Content           string `json:"content"`
	Status            int32  `json:"status"`
	Priority          int32  `json:"priority"`
	AssigneeId        string `json:"assignee_id"`
	AssigneeUsername  string `json:"assignee_username,omitempty"`
	AssigneeNickname  string `json:"assignee_nickname,omitempty"`
	AssigneeAvatarUrl string `json:"assignee_avatar_url,omitempty"`
	CreatorId         string `json:"creator_id"`
	CreatorUsername   string `json:"creator_username,omitempty"`
	CreatorNickname   string `json:"creator_nickname,omitempty"`
	CreatorAvatarUrl  string `json:"creator_avatar_url,omitempty"`
	DueTime           string `json:"due_time"`
	Version           int64  `json:"version"`
}

type enrichedProjectMember struct {
	Id        string `json:"id"`
	ProjectId string `json:"project_id"`
	UserId    string `json:"user_id"`
	Username  string `json:"username,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarUrl string `json:"avatar_url,omitempty"`
	Role      int32  `json:"role"`
}

type enrichedListProjectsResponse struct {
	Projects []enrichedProject `json:"projects"`
}

type enrichedGetProjectResponse struct {
	Project *enrichedProject `json:"project"`
}

type enrichedListTasksResponse struct {
	Tasks      []enrichedTask `json:"tasks"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

type enrichedGetTaskResponse struct {
	Task *enrichedTask `json:"task"`
}

type enrichedListMembersResponse struct {
	Members []enrichedProjectMember `json:"members"`
}

func enrichProjects(ctx context.Context, client userv1.UserServiceClient, res *taskv1.ListProjectsResponse) *enrichedListProjectsResponse {
	ids := make([]string, 0, len(res.Projects))
	for _, p := range res.Projects {
		ids = append(ids, p.OwnerId)
	}
	userMap := fetchUserMap(ctx, client, ids)

	projects := make([]enrichedProject, len(res.Projects))
	for i, p := range res.Projects {
		username, nickname, avatarUrl := applyUserInfo(userMap, p.OwnerId)
		projects[i] = enrichedProject{
			Id:             p.Id,
			Name:           p.Name,
			Description:    p.Description,
			OwnerId:        p.OwnerId,
			OwnerUsername:  username,
			OwnerNickname:  nickname,
			OwnerAvatarUrl: avatarUrl,
			Status:         p.Status,
			Version:        p.Version,
		}
	}
	return &enrichedListProjectsResponse{Projects: projects}
}

// enrichProtoProject enriches a single proto Project with owner user info.
// Used by both read (GetProject) and write (Create/Update/Archive/Unarchive/Transfer) handlers.
func enrichProtoProject(ctx context.Context, client userv1.UserServiceClient, p *taskv1.Project) *enrichedGetProjectResponse {
	userMap := fetchUserMap(ctx, client, []string{p.OwnerId})
	username, nickname, avatarUrl := applyUserInfo(userMap, p.OwnerId)
	return &enrichedGetProjectResponse{
		Project: &enrichedProject{
			Id:             p.Id,
			Name:           p.Name,
			Description:    p.Description,
			OwnerId:        p.OwnerId,
			OwnerUsername:  username,
			OwnerNickname:  nickname,
			OwnerAvatarUrl: avatarUrl,
			Status:         p.Status,
			Version:        p.Version,
		},
	}
}

func enrichProject(ctx context.Context, client userv1.UserServiceClient, res *taskv1.GetProjectResponse) *enrichedGetProjectResponse {
	return enrichProtoProject(ctx, client, res.Project)
}

func enrichTasks(ctx context.Context, client userv1.UserServiceClient, res *taskv1.ListTasksResponse) *enrichedListTasksResponse {
	ids := make([]string, 0, len(res.Tasks)*2)
	for _, t := range res.Tasks {
		ids = append(ids, t.AssigneeId, t.CreatorId)
	}
	userMap := fetchUserMap(ctx, client, ids)

	tasks := make([]enrichedTask, len(res.Tasks))
	for i, t := range res.Tasks {
		assigneeUsername, assigneeNickname, assigneeAvatar := applyUserInfo(userMap, t.AssigneeId)
		creatorUsername, creatorNickname, creatorAvatar := applyUserInfo(userMap, t.CreatorId)
		tasks[i] = enrichedTask{
			Id:                t.Id,
			ProjectId:         t.ProjectId,
			Title:             t.Title,
			Content:           t.Content,
			Status:            t.Status,
			Priority:          t.Priority,
			AssigneeId:        t.AssigneeId,
			AssigneeUsername:  assigneeUsername,
			AssigneeNickname:  assigneeNickname,
			AssigneeAvatarUrl: assigneeAvatar,
			CreatorId:         t.CreatorId,
			CreatorUsername:   creatorUsername,
			CreatorNickname:   creatorNickname,
			CreatorAvatarUrl:  creatorAvatar,
			DueTime:           t.DueTime,
			Version:           t.Version,
		}
	}
	return &enrichedListTasksResponse{Tasks: tasks, NextCursor: res.NextCursor}
}

// enrichProtoTask enriches a single proto Task with assignee and creator user info.
// Used by both read (GetTask) and write (Create/Update/Delete/Assign/ChangeStatus) handlers.
func enrichProtoTask(ctx context.Context, client userv1.UserServiceClient, t *taskv1.Task) *enrichedGetTaskResponse {
	userMap := fetchUserMap(ctx, client, []string{t.AssigneeId, t.CreatorId})
	assigneeUsername, assigneeNickname, assigneeAvatar := applyUserInfo(userMap, t.AssigneeId)
	creatorUsername, creatorNickname, creatorAvatar := applyUserInfo(userMap, t.CreatorId)
	return &enrichedGetTaskResponse{
		Task: &enrichedTask{
			Id:                t.Id,
			ProjectId:         t.ProjectId,
			Title:             t.Title,
			Content:           t.Content,
			Status:            t.Status,
			Priority:          t.Priority,
			AssigneeId:        t.AssigneeId,
			AssigneeUsername:  assigneeUsername,
			AssigneeNickname:  assigneeNickname,
			AssigneeAvatarUrl: assigneeAvatar,
			CreatorId:         t.CreatorId,
			CreatorUsername:   creatorUsername,
			CreatorNickname:   creatorNickname,
			CreatorAvatarUrl:  creatorAvatar,
			DueTime:           t.DueTime,
			Version:           t.Version,
		},
	}
}

func enrichTask(ctx context.Context, client userv1.UserServiceClient, res *taskv1.GetTaskResponse) *enrichedGetTaskResponse {
	return enrichProtoTask(ctx, client, res.Task)
}

type enrichedSingleMemberResponse struct {
	Member *enrichedProjectMember `json:"member"`
}

// enrichProtoMember enriches a single proto ProjectMember with user info.
// Used by AddMember/RemoveMember/UpdateRole/Leave handlers to avoid proto
// omitempty dropping role=0 (Owner) from the JSON response.
func enrichProtoMember(ctx context.Context, client userv1.UserServiceClient, m *taskv1.ProjectMember) *enrichedSingleMemberResponse {
	userMap := fetchUserMap(ctx, client, []string{m.UserId})
	username, nickname, avatarUrl := applyUserInfo(userMap, m.UserId)
	return &enrichedSingleMemberResponse{
		Member: &enrichedProjectMember{
			Id:        m.Id,
			ProjectId: m.ProjectId,
			UserId:    m.UserId,
			Username:  username,
			Nickname:  nickname,
			AvatarUrl: avatarUrl,
			Role:      m.Role,
		},
	}
}

func enrichMembers(ctx context.Context, client userv1.UserServiceClient, res *taskv1.ListProjectMembersResponse) *enrichedListMembersResponse {
	ids := make([]string, 0, len(res.Members))
	for _, m := range res.Members {
		ids = append(ids, m.UserId)
	}
	userMap := fetchUserMap(ctx, client, ids)

	members := make([]enrichedProjectMember, len(res.Members))
	for i, m := range res.Members {
		username, nickname, avatarUrl := applyUserInfo(userMap, m.UserId)
		members[i] = enrichedProjectMember{
			Id:        m.Id,
			ProjectId: m.ProjectId,
			UserId:    m.UserId,
			Username:  username,
			Nickname:  nickname,
			AvatarUrl: avatarUrl,
			Role:      m.Role,
		}
	}
	return &enrichedListMembersResponse{Members: members}
}
