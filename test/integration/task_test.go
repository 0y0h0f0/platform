//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	taskv1 "task-platform/gen/go/task/v1"
	userv1 "task-platform/gen/go/user/v1"
)

// ---------- gRPC: CreateTask + GetTask ----------

func TestIntegration_CreateTask_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register user
	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_tasker", Email: "grpc_tasker@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	// Create project
	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "TaskProject"})
	require.NoError(t, err)

	// Create task
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	tctx := withUser(ctx2, regRes.User.Id, regRes.User.Username)
	taskRes, err := taskGrpcClient.CreateTask(tctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id,
		Title:     "My First Task",
		Content:   "task content",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, taskRes.Task.Id)
	assert.Equal(t, "My First Task", taskRes.Task.Title)
	assert.Equal(t, int32(0), taskRes.Task.Status) // todo
	assert.Equal(t, regRes.User.Id, taskRes.Task.CreatorId)
}

func TestIntegration_GetTask_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_gettasker", Email: "grpc_gettasker@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "GetTaskProj"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Find Me",
	})
	require.NoError(t, err)

	getRes, err := taskGrpcClient.GetTask(pctx, &taskv1.GetTaskRequest{TaskId: taskRes.Task.Id})
	require.NoError(t, err)
	assert.Equal(t, "Find Me", getRes.Task.Title)
	assert.Equal(t, projRes.Project.Id, getRes.Task.ProjectId)
}

func TestIntegration_GetTask_NonMember(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ownerRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_gt_owner", Email: "grpc_gt_owner@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	strangerRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_gt_stranger", Email: "grpc_gt_stranger@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, ownerRes.User.Id, ownerRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "SecretProj"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Secret Task",
	})
	require.NoError(t, err)

	// Stranger cannot access
	sctx := withUser(context.Background(), strangerRes.User.Id, strangerRes.User.Username)
	_, err = taskGrpcClient.GetTask(sctx, &taskv1.GetTaskRequest{TaskId: taskRes.Task.Id})
	require.Error(t, err, "non-member should get not found")
}

// ---------- gRPC: UpdateTask ----------

func TestIntegration_UpdateTask_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_updater", Email: "grpc_updater@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "UpdateProj"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Old Title",
	})
	require.NoError(t, err)

	// Update task
	updRes, err := taskGrpcClient.UpdateTask(pctx, &taskv1.UpdateTaskRequest{
		TaskId:   taskRes.Task.Id,
		Title:    "New Title",
		Content:  "new content",
		Priority: 2, // high
		Version:  taskRes.Task.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, "New Title", updRes.Task.Title)
	assert.Equal(t, int32(2), updRes.Task.Priority)
	assert.Equal(t, int64(1), updRes.Task.Version)
}

func TestIntegration_UpdateTask_VersionConflict(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_vconflict", Email: "grpc_vconflict@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "VConflictProj"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Task",
	})
	require.NoError(t, err)

	_, err = taskGrpcClient.UpdateTask(pctx, &taskv1.UpdateTaskRequest{
		TaskId:  taskRes.Task.Id,
		Title:   "Hacked",
		Version: 999,
	})
	require.Error(t, err, "version conflict expected")
}

func TestIntegration_UpdateTask_ConcurrentOptimisticLock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_concurrent", Email: "grpc_concurrent@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "ConcurrentProj"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Concurrent Target",
	})
	require.NoError(t, err)

	// Both goroutines race with the same initial version.
	var barrier sync.WaitGroup
	barrier.Add(2)

	type result struct {
		title string
		err   error
	}
	ch := make(chan result, 2)

	updateFn := func(newTitle string) {
		barrier.Done()
		barrier.Wait() // release both at the same time
		res, err := taskGrpcClient.UpdateTask(pctx, &taskv1.UpdateTaskRequest{
			TaskId:  taskRes.Task.Id,
			Title:   newTitle,
			Version: taskRes.Task.Version,
		})
		r := result{err: err}
		if res != nil {
			r.title = res.Task.Title
		}
		ch <- r
	}

	go updateFn("A")
	go updateFn("B")

	r1 := <-ch
	r2 := <-ch

	// Exactly one must succeed; the other must fail with Aborted.
	successes := 0
	failures := 0

	for _, r := range []result{r1, r2} {
		if r.err == nil {
			successes++
		} else {
			failures++
			st, ok := status.FromError(r.err)
			require.True(t, ok)
			assert.Equal(t, codes.Aborted, st.Code(), "concurrent loser must be Aborted")
		}
	}

	assert.Equal(t, 1, successes, "exactly one concurrent update must succeed")
	assert.Equal(t, 1, failures, "exactly one concurrent update must fail")
}

// ---------- gRPC: DeleteTask ----------

func TestIntegration_DeleteTask_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_deleter", Email: "grpc_deleter@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "DeleteProj"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "To Delete",
	})
	require.NoError(t, err)

	delRes, err := taskGrpcClient.DeleteTask(pctx, &taskv1.DeleteTaskRequest{TaskId: taskRes.Task.Id})
	require.NoError(t, err)
	assert.Equal(t, taskRes.Task.Id, delRes.Task.Id)

	// Verify deleted — non-member check should return NotFound
	_, err = taskGrpcClient.GetTask(pctx, &taskv1.GetTaskRequest{TaskId: taskRes.Task.Id})
	require.Error(t, err)
}

// ---------- gRPC: ListTasks ----------

func TestIntegration_ListTasks_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_lister_t", Email: "grpc_lister_t@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "ListProj"})
	require.NoError(t, err)

	// Create 3 tasks
	for i := 0; i < 3; i++ {
		_, err = taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
			ProjectId: projRes.Project.Id, Title: "Task " + string(rune('A'+i)),
		})
		require.NoError(t, err)
	}

	listRes, err := taskGrpcClient.ListTasks(pctx, &taskv1.ListTasksRequest{
		ProjectId: projRes.Project.Id, Limit: 20,
	})
	require.NoError(t, err)
	assert.Len(t, listRes.Tasks, 3)

	// Test cursor pagination
	listRes2, err := taskGrpcClient.ListTasks(pctx, &taskv1.ListTasksRequest{
		ProjectId: projRes.Project.Id, Limit: 2,
	})
	require.NoError(t, err)
	assert.Len(t, listRes2.Tasks, 2)
	assert.NotEmpty(t, listRes2.NextCursor)
}

// ---------- gRPC: AssignTask ----------

func TestIntegration_AssignTask_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ownerRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_assign_o", Email: "grpc_assign_o@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	memberRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_assign_m", Email: "grpc_assign_m@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, ownerRes.User.Id, ownerRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "AssignProj"})
	require.NoError(t, err)

	// Add member
	_, err = taskGrpcClient.AddProjectMember(pctx, &taskv1.AddProjectMemberRequest{
		ProjectId: projRes.Project.Id, UserId: memberRes.User.Id, Role: 2,
	})
	require.NoError(t, err)

	// Create task
	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Assignable",
	})
	require.NoError(t, err)

	// Assign
	aCtx := withUser(context.Background(), ownerRes.User.Id, ownerRes.User.Username)
	assignRes, err := taskGrpcClient.AssignTask(aCtx, &taskv1.AssignTaskRequest{
		TaskId: taskRes.Task.Id, AssigneeId: memberRes.User.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, memberRes.User.Id, assignRes.Task.AssigneeId)
}

func TestIntegration_AssignTask_NonMemberAssignee(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_asgnm_o", Email: "grpc_asgnm_o@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	nonMemberRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_asgnm_s", Email: "grpc_asgnm_s@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "NoMemberAssign"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "CannotAssign",
	})
	require.NoError(t, err)

	// Assign to non-member should fail
	aCtx := withUser(context.Background(), regRes.User.Id, regRes.User.Username)
	_, err = taskGrpcClient.AssignTask(aCtx, &taskv1.AssignTaskRequest{
		TaskId: taskRes.Task.Id, AssigneeId: nonMemberRes.User.Id,
	})
	require.Error(t, err)
}

// ---------- gRPC: ChangeTaskStatus ----------

func TestIntegration_ChangeTaskStatus_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_status", Email: "grpc_status@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "StatusProj"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Status Task",
	})
	require.NoError(t, err)

	// todo -> doing
	changed, err := taskGrpcClient.ChangeTaskStatus(pctx, &taskv1.ChangeTaskStatusRequest{
		TaskId: taskRes.Task.Id, Status: 1, Version: taskRes.Task.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), changed.Task.Status)
}

func TestIntegration_ChangeTaskStatus_InvalidTransition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_badtrans", Email: "grpc_badtrans@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "BadTransProj"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Transition Test",
	})
	require.NoError(t, err)

	// todo -> cancelled
	changed, err := taskGrpcClient.ChangeTaskStatus(pctx, &taskv1.ChangeTaskStatusRequest{
		TaskId: taskRes.Task.Id, Status: 3, Version: taskRes.Task.Version,
	})
	require.NoError(t, err)

	// cancelled -> doing should fail (invalid transition)
	_, err = taskGrpcClient.ChangeTaskStatus(pctx, &taskv1.ChangeTaskStatusRequest{
		TaskId: taskRes.Task.Id, Status: 1, Version: changed.Task.Version,
	})
	require.Error(t, err, "invalid transition expected")
}

// ---------- gRPC: Member permissions ----------

func TestIntegration_Task_MemberPermissions_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ownerRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_tm_o", Email: "grpc_tm_o@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	memberRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_tm_m", Email: "grpc_tm_m@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	otherMemberRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_tm_o2", Email: "grpc_tm_o2@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, ownerRes.User.Id, ownerRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "TaskPermProj"})
	require.NoError(t, err)

	// Add both as members
	_, err = taskGrpcClient.AddProjectMember(pctx, &taskv1.AddProjectMemberRequest{
		ProjectId: projRes.Project.Id, UserId: memberRes.User.Id, Role: 2,
	})
	require.NoError(t, err)
	_, err = taskGrpcClient.AddProjectMember(pctx, &taskv1.AddProjectMemberRequest{
		ProjectId: projRes.Project.Id, UserId: otherMemberRes.User.Id, Role: 2,
	})
	require.NoError(t, err)

	// Member creates own task
	mctx := withUser(context.Background(), memberRes.User.Id, memberRes.User.Username)
	taskRes, err := taskGrpcClient.CreateTask(mctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Member's Task",
	})
	require.NoError(t, err)

	// Other member cannot edit member's task
	octx := withUser(context.Background(), otherMemberRes.User.Id, otherMemberRes.User.Username)
	_, err = taskGrpcClient.UpdateTask(octx, &taskv1.UpdateTaskRequest{
		TaskId: taskRes.Task.Id, Title: "Hacked", Version: taskRes.Task.Version,
	})
	require.Error(t, err, "member should not edit another's task")

	// Other member cannot change status
	_, err = taskGrpcClient.ChangeTaskStatus(octx, &taskv1.ChangeTaskStatusRequest{
		TaskId: taskRes.Task.Id, Status: 1, Version: taskRes.Task.Version,
	})
	require.Error(t, err, "member should not change another's task status")

	// Other member cannot delete member's task
	_, err = taskGrpcClient.DeleteTask(octx, &taskv1.DeleteTaskRequest{TaskId: taskRes.Task.Id})
	require.Error(t, err, "member should not delete another's task")

	// Owner can edit any task
	octx2 := withUser(context.Background(), ownerRes.User.Id, ownerRes.User.Username)
	_, err = taskGrpcClient.UpdateTask(octx2, &taskv1.UpdateTaskRequest{
		TaskId: taskRes.Task.Id, Title: "Owner Edits", Version: taskRes.Task.Version,
	})
	require.NoError(t, err)
}

// ---------- gRPC: Archived project rejects task writes ----------

func TestIntegration_Task_ArchivedRejectsWrites_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_ta_o", Email: "grpc_ta_o@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	pctx := withUser(ctx, regRes.User.Id, regRes.User.Username)
	projRes, err := taskGrpcClient.CreateProject(pctx, &taskv1.CreateProjectRequest{Name: "ArchTaskProj"})
	require.NoError(t, err)

	taskRes, err := taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Task",
	})
	require.NoError(t, err)

	// Archive project
	_, err = taskGrpcClient.ArchiveProject(pctx, &taskv1.ArchiveProjectRequest{ProjectId: projRes.Project.Id})
	require.NoError(t, err)

	// Create task on archived should fail
	_, err = taskGrpcClient.CreateTask(pctx, &taskv1.CreateTaskRequest{
		ProjectId: projRes.Project.Id, Title: "Should Fail",
	})
	require.Error(t, err)

	// Update task on archived should fail
	_, err = taskGrpcClient.UpdateTask(pctx, &taskv1.UpdateTaskRequest{
		TaskId: taskRes.Task.Id, Title: "Hacked", Version: taskRes.Task.Version,
	})
	require.Error(t, err)

	// Delete task on archived should fail
	_, err = taskGrpcClient.DeleteTask(pctx, &taskv1.DeleteTaskRequest{TaskId: taskRes.Task.Id})
	require.Error(t, err)

	// But read still works
	getRes, err := taskGrpcClient.GetTask(pctx, &taskv1.GetTaskRequest{TaskId: taskRes.Task.Id})
	require.NoError(t, err)
	assert.Equal(t, "Task", getRes.Task.Title)
}

// ---------- HTTP: Create + Get Task ----------

func TestE2E_CreateAndGetTask(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_ctasker")

	// Create project
	projID := createProject(t, client, token, "TaskProject")

	// Create task
	resp := authedDo(t, client, http.MethodPost, "/api/v1/tasks",
		`{"project_id":"`+projID+`","title":"My Task","content":"task body"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Task struct {
			Id        string `json:"id"`
			Title     string `json:"title"`
			Status    int32  `json:"status"`
			CreatorId string `json:"creator_id"`
		} `json:"task"`
	}
	json.Unmarshal(data, &createRes)
	assert.Equal(t, "My Task", createRes.Task.Title)
	assert.Equal(t, int32(0), createRes.Task.Status)
	assert.NotEmpty(t, createRes.Task.Id)
	taskID := createRes.Task.Id

	// Get task
	resp = authedDo(t, client, http.MethodGet, "/api/v1/tasks/"+taskID, "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var getRes struct {
		Task struct {
			Title string `json:"title"`
		} `json:"task"`
	}
	json.Unmarshal(data, &getRes)
	assert.Equal(t, "My Task", getRes.Task.Title)
}

// ---------- HTTP: Update Task ----------

func TestE2E_UpdateTask(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_updtasker")

	projID := createProject(t, client, token, "UpdateTaskProj")

	// Create task
	resp := authedDo(t, client, http.MethodPost, "/api/v1/tasks",
		`{"project_id":"`+projID+`","title":"Old"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Task struct {
			Id      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"task"`
	}
	json.Unmarshal(data, &createRes)

	// Update
	resp = authedDo(t, client, http.MethodPut, "/api/v1/tasks/"+createRes.Task.Id,
		`{"title":"New","content":"updated","priority":2,"version":`+formatInt(createRes.Task.Version)+`}`, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var updRes struct {
		Task struct {
			Title    string `json:"title"`
			Priority int32  `json:"priority"`
			Version  int64  `json:"version"`
		} `json:"task"`
	}
	json.Unmarshal(data, &updRes)
	assert.Equal(t, "New", updRes.Task.Title)
	assert.Equal(t, int32(2), updRes.Task.Priority)
	assert.Equal(t, int64(1), updRes.Task.Version)
}

// ---------- HTTP: Delete Task ----------

func TestE2E_DeleteTask(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_deltasker")

	projID := createProject(t, client, token, "DeleteTaskProj")

	resp := authedDo(t, client, http.MethodPost, "/api/v1/tasks",
		`{"project_id":"`+projID+`","title":"Delete Me"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Task struct{ Id string `json:"id"` }
	}
	json.Unmarshal(data, &createRes)

	// Delete
	resp = authedDo(t, client, http.MethodDelete, "/api/v1/tasks/"+createRes.Task.Id, "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// ---------- HTTP: List Tasks ----------

func TestE2E_ListTasks(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_listtasker")

	projID := createProject(t, client, token, "ListTaskProj")

	// Create 2 tasks
	for _, title := range []string{"Task A", "Task B"} {
		resp := authedDo(t, client, http.MethodPost, "/api/v1/tasks",
			`{"project_id":"`+projID+`","title":"`+title+`"}`, token)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	}

	// List
	resp := authedDo(t, client, http.MethodGet, "/api/v1/tasks?project_id="+projID+"&limit=20", "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var listRes struct {
		Tasks []struct {
			Title string `json:"title"`
		} `json:"tasks"`
	}
	json.Unmarshal(data, &listRes)
	assert.Len(t, listRes.Tasks, 2)
}

// ---------- HTTP: List Tasks missing project_id ----------

func TestE2E_ListTasks_MissingProjectID(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_list_nopid")

	resp := authedDo(t, client, http.MethodGet, "/api/v1/tasks", "", token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

// ---------- HTTP: Assign Task ----------

func TestE2E_AssignTask(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_assignt_o")
	memberToken := registerAndGetToken(t, client, "e2e_assignt_m")

	projID := createProject(t, client, ownerToken, "AssignTaskProj")
	memberID := getMyUserID(t, client, memberToken)

	// Add member
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+memberID+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Create task
	resp = authedDo(t, client, http.MethodPost, "/api/v1/tasks",
		`{"project_id":"`+projID+`","title":"To Assign"}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Task struct{ Id string `json:"id"` }
	}
	json.Unmarshal(data, &createRes)

	// Assign
	resp = authedDo(t, client, http.MethodPost, "/api/v1/tasks/"+createRes.Task.Id+"/assign",
		`{"assignee_id":"`+memberID+`"}`, ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var assignRes struct {
		Task struct {
			AssigneeId string `json:"assignee_id"`
		} `json:"task"`
	}
	json.Unmarshal(data, &assignRes)
	assert.Equal(t, memberID, assignRes.Task.AssigneeId)
}

// ---------- HTTP: Change Task Status ----------

func TestE2E_ChangeTaskStatus(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_chgstatus")

	projID := createProject(t, client, token, "ChangeStatusProj")

	resp := authedDo(t, client, http.MethodPost, "/api/v1/tasks",
		`{"project_id":"`+projID+`","title":"Status Test"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Task struct {
			Id      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"task"`
	}
	json.Unmarshal(data, &createRes)

	// todo -> doing
	resp = authedDo(t, client, http.MethodPost, "/api/v1/tasks/"+createRes.Task.Id+"/status",
		`{"status":1,"version":`+formatInt(createRes.Task.Version)+`}`, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var statusRes struct {
		Task struct {
			Status int32 `json:"status"`
		} `json:"task"`
	}
	json.Unmarshal(data, &statusRes)
	assert.Equal(t, int32(1), statusRes.Task.Status)
}

// ---------- HTTP: Non-member access returns 404 ----------

func TestE2E_Task_NonMemberAccess(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_tnm_o")
	strangerToken := registerAndGetToken(t, client, "e2e_tnm_s")

	projID := createProject(t, client, ownerToken, "SecretTaskProj")

	resp := authedDo(t, client, http.MethodPost, "/api/v1/tasks",
		`{"project_id":"`+projID+`","title":"Secret Task"}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Task struct{ Id string `json:"id"` }
	}
	json.Unmarshal(data, &createRes)

	// Stranger gets 404
	resp = authedDo(t, client, http.MethodGet, "/api/v1/tasks/"+createRes.Task.Id, "", strangerToken)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

// ---------- HTTP: Member delete only own todo ----------

func TestE2E_Task_MemberDeletePermissions(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_tmd_o")
	memberToken := registerAndGetToken(t, client, "e2e_tmd_m")

	projID := createProject(t, client, ownerToken, "MemberDelProj")
	memberID := getMyUserID(t, client, memberToken)

	// Add member
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+memberID+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Owner creates task
	resp = authedDo(t, client, http.MethodPost, "/api/v1/tasks",
		`{"project_id":"`+projID+`","title":"Owner's Task"}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Task struct {
			Id      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"task"`
	}
	json.Unmarshal(data, &createRes)
	ownerTaskID := createRes.Task.Id

	// Member cannot delete owner's task
	resp = authedDo(t, client, http.MethodDelete, "/api/v1/tasks/"+ownerTaskID, "", memberToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Member can delete own todo task
	resp = authedDo(t, client, http.MethodPost, "/api/v1/tasks",
		`{"project_id":"`+projID+`","title":"Member's Own"}`, memberToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var membRes struct {
		Task struct{ Id string `json:"id"` }
	}
	json.Unmarshal(data, &membRes)

	resp = authedDo(t, client, http.MethodDelete, "/api/v1/tasks/"+membRes.Task.Id, "", memberToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Member cannot delete own non-todo task
	// Create another task, change status to doing, try to delete
	resp = authedDo(t, client, http.MethodPost, "/api/v1/tasks",
		`{"project_id":"`+projID+`","title":"My Doing"}`, memberToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var doingRes struct {
		Task struct {
			Id      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"task"`
	}
	json.Unmarshal(data, &doingRes)

	// Change to doing
	resp = authedDo(t, client, http.MethodPost, "/api/v1/tasks/"+doingRes.Task.Id+"/status",
		`{"status":1,"version":`+formatInt(doingRes.Task.Version)+`}`, memberToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Try to delete doing task
	resp = authedDo(t, client, http.MethodDelete, "/api/v1/tasks/"+doingRes.Task.Id, "", memberToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

// ---------- helpers ----------

func formatInt(v int64) string { return fmt.Sprintf("%d", v) }
