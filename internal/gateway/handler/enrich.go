package handler

import (
	"context"

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
		deduped = deduped[:100]
	}
	res, err := client.BatchGetUsers(ctx, &userv1.BatchGetUsersRequest{UserIds: deduped})
	if err != nil {
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
