package handler

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	userv1 "task-platform/gen/go/user/v1"
	taskv1 "task-platform/gen/go/task/v1"
)

type enrichUserClient struct {
	userv1.UserServiceClient
	users map[string]*userv1.User
}

func (c *enrichUserClient) BatchGetUsers(_ context.Context, in *userv1.BatchGetUsersRequest, _ ...grpc.CallOption) (*userv1.BatchGetUsersResponse, error) {
	var users []*userv1.User
	for _, id := range in.UserIds {
		if u, ok := c.users[id]; ok {
			users = append(users, u)
		}
	}
	return &userv1.BatchGetUsersResponse{Users: users}, nil
}

func TestEnrichComments(t *testing.T) {
	client := &enrichUserClient{
		users: map[string]*userv1.User{
			"user-1": {Id: "user-1", Username: "alice", Nickname: "Alice", AvatarUrl: "/a.jpg"},
		},
	}

	res := &taskv1.ListTaskCommentsResponse{
		Comments: []*taskv1.TaskComment{
			{Id: "c1", TaskId: "t1", UserId: "user-1", Content: "hello"},
			{Id: "c2", TaskId: "t1", UserId: "user-2", Content: "world"},
		},
	}

	enriched := enrichComments(context.Background(), client, res)
	if len(enriched.Comments) != 2 {
		t.Fatalf("got %d comments, want 2", len(enriched.Comments))
	}
	if enriched.Comments[0].Username != "alice" {
		t.Errorf("username[0] = %q, want alice", enriched.Comments[0].Username)
	}
	if enriched.Comments[0].Nickname != "Alice" {
		t.Errorf("nickname[0] = %q, want Alice", enriched.Comments[0].Nickname)
	}
	if enriched.Comments[0].AvatarUrl != "/a.jpg" {
		t.Errorf("avatar_url[0] = %q, want /a.jpg", enriched.Comments[0].AvatarUrl)
	}
	if enriched.Comments[1].Username != "" {
		t.Errorf("username[1] should be empty, got %q", enriched.Comments[1].Username)
	}
	if enriched.Comments[1].UserId != "user-2" {
		t.Errorf("user_id[1] = %q, want user-2", enriched.Comments[1].UserId)
	}
}

func TestEnrichOperationLogs(t *testing.T) {
	client := &enrichUserClient{
		users: map[string]*userv1.User{
			"op-1": {Id: "op-1", Username: "bob", Nickname: "Bob", AvatarUrl: "/b.jpg"},
		},
	}

	res := &taskv1.ListOperationLogsResponse{
		Logs: []*taskv1.OperationLog{
			{Id: "l1", OperatorId: "op-1", Action: "task.create", DetailJson: "{}"},
			{Id: "l2", OperatorId: "op-2", Action: "task.update", DetailJson: "{}"},
		},
		NextCursor: "cursor-1",
	}

	enriched := enrichOperationLogs(context.Background(), client, res)
	if len(enriched.Logs) != 2 {
		t.Fatalf("got %d logs, want 2", len(enriched.Logs))
	}
	if enriched.Logs[0].Username != "bob" {
		t.Errorf("username[0] = %q, want bob", enriched.Logs[0].Username)
	}
	if enriched.Logs[1].Username != "" {
		t.Errorf("username[1] should be empty, got %q", enriched.Logs[1].Username)
	}
	if enriched.NextCursor != "cursor-1" {
		t.Errorf("next_cursor = %q, want cursor-1", enriched.NextCursor)
	}
}

func TestEnrichComments_NilClient(t *testing.T) {
	res := &taskv1.ListTaskCommentsResponse{
		Comments: []*taskv1.TaskComment{
			{Id: "c1", TaskId: "t1", UserId: "user-1", Content: "hello"},
		},
	}

	enriched := enrichComments(context.Background(), nil, res)
	if len(enriched.Comments) != 1 {
		t.Fatalf("got %d comments, want 1", len(enriched.Comments))
	}
	if enriched.Comments[0].Username != "" {
		t.Errorf("username should be empty with nil client")
	}
}

func TestEnrichOperationLogs_NilClient(t *testing.T) {
	res := &taskv1.ListOperationLogsResponse{
		Logs: []*taskv1.OperationLog{
			{Id: "l1", OperatorId: "op-1", Action: "task.create"},
		},
	}

	enriched := enrichOperationLogs(context.Background(), nil, res)
	if len(enriched.Logs) != 1 {
		t.Fatalf("got %d logs, want 1", len(enriched.Logs))
	}
	if enriched.Logs[0].Username != "" {
		t.Errorf("username should be empty with nil client")
	}
}

func TestFetchUserMap_NilClient(t *testing.T) {
	m := fetchUserMap(context.Background(), nil, []string{"user-1"})
	if m != nil {
		t.Error("expected nil map for nil client")
	}
}

func TestFetchUserMap_EmptyIDs(t *testing.T) {
	client := &enrichUserClient{users: map[string]*userv1.User{}}
	m := fetchUserMap(context.Background(), client, nil)
	if m != nil {
		t.Error("expected nil map for empty ids")
	}
}

func TestDedupeStrings(t *testing.T) {
	got := dedupeStrings([]string{"a", "b", "a", "", "c", "b"})
	if len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("got %v, want [a b c]", got)
	}
}

func TestApplyUserInfo_NotFound(t *testing.T) {
	username, nickname, avatar := applyUserInfo(nil, "user-1")
	if username != "" || nickname != "" || avatar != "" {
		t.Error("expected empty strings when user map is nil")
	}
}
