//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	userv1 "task-platform/gen/go/user/v1"
	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/pkg/xerr"
)

// ---------- helpers ----------

func registerAndGetToken(t *testing.T, client *http.Client, username string) string {
	t.Helper()
	body := `{"username":"` + username + `","email":"` + username + `@test.com","password":"secret123"}`
	resp, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	var envelope xerr.HTTPResponse
	json.NewDecoder(resp.Body).Decode(&envelope)
	data, _ := json.Marshal(envelope.Data)
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	json.Unmarshal(data, &parsed)
	return parsed.AccessToken
}

func authedDo(t *testing.T, client *http.Client, method, path, body, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, "http://localhost"+path, bodyReader)
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func decodeEnvelope(t *testing.T, resp *http.Response) (*xerr.HTTPResponse, []byte) {
	t.Helper()
	defer resp.Body.Close()
	var envelope xerr.HTTPResponse
	json.NewDecoder(resp.Body).Decode(&envelope)
	data, _ := json.Marshal(envelope.Data)
	return &envelope, data
}

// ---------- create project ----------

func TestE2E_CreateProject(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_creator")

	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects",
		`{"name":"My Project","description":"test"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	env, data := decodeEnvelope(t, resp)
	assert.Equal(t, xerr.CodeOK, env.Code)

	var proj struct {
		Project struct {
			Id      string `json:"id"`
			Name    string `json:"name"`
			OwnerId string `json:"owner_id"`
		} `json:"project"`
	}
	json.Unmarshal(data, &proj)
	assert.Equal(t, "My Project", proj.Project.Name)
	assert.NotEmpty(t, proj.Project.Id)
	assert.NotEmpty(t, proj.Project.OwnerId)
}

// ---------- create → list → get ----------

func TestE2E_CreateListGetProject(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_clg")

	// Create
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects",
		`{"name":"Test Project"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Project struct {
			Id string `json:"id"`
		} `json:"project"`
	}
	json.Unmarshal(data, &createRes)
	projID := createRes.Project.Id

	// List (should include the project)
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects", "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var listRes struct {
		Projects []struct {
			Id string `json:"id"`
		} `json:"projects"`
	}
	json.Unmarshal(data, &listRes)
	assert.GreaterOrEqual(t, len(listRes.Projects), 1)

	// Get
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projID, "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var getRes struct {
		Project struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"project"`
	}
	json.Unmarshal(data, &getRes)
	assert.Equal(t, "Test Project", getRes.Project.Name)
}

// ---------- archive / unarchive ----------

func TestE2E_ArchiveUnarchiveProject(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_archiver")

	// Create
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects",
		`{"name":"Archive Test"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Project struct {
			Id string `json:"id"`
		} `json:"project"`
	}
	json.Unmarshal(data, &createRes)
	projID := createRes.Project.Id

	// Archive
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/archive", "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var archiveRes struct {
		Project struct {
			Status int32 `json:"status"`
		} `json:"project"`
	}
	json.Unmarshal(data, &archiveRes)
	assert.Equal(t, int32(1), archiveRes.Project.Status)

	// Update on archived should fail
	resp = authedDo(t, client, http.MethodPut, "/api/v1/projects/"+projID,
		`{"name":"Hacked","version":0}`, token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	// Get on archived should succeed
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projID, "", token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Unarchive
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/unarchive", "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var unarchiveRes struct {
		Project struct {
			Status int32 `json:"status"`
		} `json:"project"`
	}
	json.Unmarshal(data, &unarchiveRes)
	assert.Equal(t, int32(0), unarchiveRes.Project.Status)
}

// ---------- add member + list members ----------

func TestE2E_AddAndListMembers(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_owner1")
	memberToken := registerAndGetToken(t, client, "e2e_member1")

	// Owner creates project
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects",
		`{"name":"Team Project"}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Project struct {
			Id string `json:"id"`
		} `json:"project"`
	}
	json.Unmarshal(data, &createRes)
	projID := createRes.Project.Id

	// Get member's user ID via Me
	meResp := authedDo(t, client, http.MethodGet, "/api/v1/users/me", "", memberToken)
	require.Equal(t, http.StatusOK, meResp.StatusCode)
	_, meData := decodeEnvelope(t, meResp)
	var meRes struct {
		User struct {
			Id string `json:"id"`
		} `json:"user"`
	}
	json.Unmarshal(meData, &meRes)
	memberUserID := meRes.User.Id

	// Owner adds member (role=2)
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+memberUserID+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// List members
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projID+"/members", "", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var listRes struct {
		Members []struct {
			UserId string `json:"user_id"`
			Role   int32  `json:"role"`
		} `json:"members"`
	}
	json.Unmarshal(data, &listRes)
	assert.Len(t, listRes.Members, 2)

	// Member can view the project
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projID, "", memberToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Member cannot add members
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+memberUserID+`","role":2}`, memberToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

// ---------- member leave ----------

func TestE2E_MemberLeave(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_owner_leave")
	memberToken := registerAndGetToken(t, client, "e2e_leaver")

	// Create project
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects",
		`{"name":"Leave Test"}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Project struct {
			Id string `json:"id"`
		} `json:"project"`
	}
	json.Unmarshal(data, &createRes)
	projID := createRes.Project.Id

	// Get member's user ID
	meResp := authedDo(t, client, http.MethodGet, "/api/v1/users/me", "", memberToken)
	require.Equal(t, http.StatusOK, meResp.StatusCode)
	_, meData := decodeEnvelope(t, meResp)
	var meRes struct {
		User struct{ Id string `json:"id"` }
	}
	json.Unmarshal(meData, &meRes)

	// Owner adds member
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+meRes.User.Id+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Member leaves
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members/me/leave", "", memberToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// After leaving, member cannot view project
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projID, "", memberToken)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

// ---------- transfer ownership ----------

func TestE2E_TransferOwnership(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_xfer_owner")
	otherToken := registerAndGetToken(t, client, "e2e_xfer_other")

	// Create project
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects",
		`{"name":"Transfer Test"}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Project struct {
			Id string `json:"id"`
		} `json:"project"`
	}
	json.Unmarshal(data, &createRes)
	projID := createRes.Project.Id

	// Get other user's ID
	meResp := authedDo(t, client, http.MethodGet, "/api/v1/users/me", "", otherToken)
	require.Equal(t, http.StatusOK, meResp.StatusCode)
	_, meData := decodeEnvelope(t, meResp)
	var meRes struct {
		User struct{ Id string `json:"id"` }
	}
	json.Unmarshal(meData, &meRes)
	otherUserID := meRes.User.Id

	// Add other as admin
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+otherUserID+`","role":1}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Get owner's user ID
	ownerMeResp := authedDo(t, client, http.MethodGet, "/api/v1/users/me", "", ownerToken)
	require.Equal(t, http.StatusOK, ownerMeResp.StatusCode)
	_, ownerMeData := decodeEnvelope(t, ownerMeResp)
	var ownerMeRes struct {
		User struct{ Id string `json:"id"` }
	}
	json.Unmarshal(ownerMeData, &ownerMeRes)
	ownerUserID := ownerMeRes.User.Id

	// Transfer ownership to other
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/transfer",
		`{"target_user_id":"`+otherUserID+`"}`, ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var xferRes struct {
		Project struct {
			OwnerId string `json:"owner_id"`
		} `json:"project"`
	}
	json.Unmarshal(data, &xferRes)
	assert.Equal(t, otherUserID, xferRes.Project.OwnerId)

	// Verify old owner is now admin via members list
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projID+"/members", "", otherToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, memberData := decodeEnvelope(t, resp)
	var memberListRes struct {
		Members []struct {
			UserId string `json:"user_id"`
			Role   int32  `json:"role"`
		} `json:"members"`
	}
	json.Unmarshal(memberData, &memberListRes)

	var oldOwnerRole, newOwnerRole int32 = -1, -1
	for _, m := range memberListRes.Members {
		if m.UserId == ownerUserID {
			oldOwnerRole = m.Role
		}
		if m.UserId == otherUserID {
			newOwnerRole = m.Role
		}
	}
	assert.Equal(t, int32(1), oldOwnerRole, "old owner should be admin (role=1)")
	assert.Equal(t, int32(0), newOwnerRole, "new owner should be owner (role=0)")
}

// ---------- non-member access ----------

func TestE2E_NonMemberAccess(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_nonmem_owner")
	strangerToken := registerAndGetToken(t, client, "e2e_nonmem_stranger")

	// Owner creates project
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects",
		`{"name":"Secret Project"}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Project struct {
			Id string `json:"id"`
		} `json:"project"`
	}
	json.Unmarshal(data, &createRes)
	projID := createRes.Project.Id

	// Stranger cannot access
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projID, "", strangerToken)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// Stranger not in list
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects", "", strangerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var listRes struct {
		Projects []struct {
			Id string `json:"id"`
		} `json:"projects"`
	}
	json.Unmarshal(data, &listRes)
	assert.Len(t, listRes.Projects, 0)
}

// ---------- update project ----------

func TestE2E_UpdateProject(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "e2e_updater")

	// Create
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects",
		`{"name":"Old Name"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var createRes struct {
		Project struct {
			Id      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"project"`
	}
	json.Unmarshal(data, &createRes)
	projID := createRes.Project.Id

	// Update
	resp = authedDo(t, client, http.MethodPut, "/api/v1/projects/"+projID,
		`{"name":"New Name","version":0}`, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var updateRes struct {
		Project struct {
			Name string `json:"name"`
		} `json:"project"`
	}
	json.Unmarshal(data, &updateRes)
	assert.Equal(t, "New Name", updateRes.Project.Name)
}

// ---------- helpers ----------

func getMyUserID(t *testing.T, client *http.Client, token string) string {
	t.Helper()
	resp := authedDo(t, client, http.MethodGet, "/api/v1/users/me", "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var meRes struct {
		User struct{ Id string `json:"id"` }
	}
	json.Unmarshal(data, &meRes)
	return meRes.User.Id
}

func createProject(t *testing.T, client *http.Client, token, name string) string {
	t.Helper()
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects",
		`{"name":"`+name+`"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var cr struct {
		Project struct{ Id string `json:"id"` }
	}
	json.Unmarshal(data, &cr)
	return cr.Project.Id
}

// ---------- admin permission boundaries ----------

func TestE2E_AdminPermissions(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_admin_owner")
	adminToken := registerAndGetToken(t, client, "e2e_admin_admin")
	thirdToken := registerAndGetToken(t, client, "e2e_admin_third")

	projID := createProject(t, client, ownerToken, "AdminBoundary")
	ownerID := getMyUserID(t, client, ownerToken)
	adminID := getMyUserID(t, client, adminToken)
	thirdID := getMyUserID(t, client, thirdToken)

	// Owner adds admin (role=1)
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+adminID+`","role":1}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Admin CAN add member (role=2)
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+thirdID+`","role":2}`, adminToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Admin CANNOT update project info
	resp = authedDo(t, client, http.MethodPut, "/api/v1/projects/"+projID,
		`{"name":"AdminHack","version":0}`, adminToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Admin CANNOT archive project
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/archive",
		"", adminToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Admin CANNOT remove owner
	resp = authedDo(t, client, http.MethodDelete, "/api/v1/projects/"+projID+"/members/"+ownerID,
		"", adminToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Admin CANNOT add admin (role=1)
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+thirdID+`","role":1}`, adminToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

// ---------- member permission boundaries ----------

func TestE2E_MemberPermissions(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_memb_owner")
	member1Token := registerAndGetToken(t, client, "e2e_memb_m1")
	member2Token := registerAndGetToken(t, client, "e2e_memb_m2")

	projID := createProject(t, client, ownerToken, "MemberBoundary")
	member1ID := getMyUserID(t, client, member1Token)
	member2ID := getMyUserID(t, client, member2Token)

	// Owner adds both as members
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+member1ID+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+member2ID+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Member1 CANNOT remove another member
	resp = authedDo(t, client, http.MethodDelete, "/api/v1/projects/"+projID+"/members/"+member2ID,
		"", member1Token)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Member1 CANNOT change another member's role
	resp = authedDo(t, client, http.MethodPut, "/api/v1/projects/"+projID+"/members/"+member2ID,
		`{"role":1}`, member1Token)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

// ---------- owner positive permissions ----------

func TestE2E_OwnerCanRemoveAdminAndChangeRole(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_ownpos_owner")
	adminToken := registerAndGetToken(t, client, "e2e_ownpos_admin")
	memberToken := registerAndGetToken(t, client, "e2e_ownpos_member")

	projID := createProject(t, client, ownerToken, "OwnerPositive")
	adminID := getMyUserID(t, client, adminToken)
	memberID := getMyUserID(t, client, memberToken)

	// Owner adds admin (role=1) and member (role=2)
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+adminID+`","role":1}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+memberID+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Owner CAN remove admin
	resp = authedDo(t, client, http.MethodDelete, "/api/v1/projects/"+projID+"/members/"+adminID,
		"", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Owner CAN change member role to admin
	resp = authedDo(t, client, http.MethodPut, "/api/v1/projects/"+projID+"/members/"+memberID,
		`{"role":1}`, ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify member is now admin via members list
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projID+"/members",
		"", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var listRes struct {
		Members []struct {
			UserId string `json:"user_id"`
			Role   int32  `json:"role"`
		} `json:"members"`
	}
	json.Unmarshal(data, &listRes)
	for _, m := range listRes.Members {
		if m.UserId == memberID {
			assert.Equal(t, int32(1), m.Role, "member should now be admin")
		}
	}
}

// ---------- archived project rejects all writes ----------

func TestE2E_ArchivedRejectsAllWrites(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_archw_owner")
	memberToken := registerAndGetToken(t, client, "e2e_archw_member")
	thirdToken := registerAndGetToken(t, client, "e2e_archw_third")

	projID := createProject(t, client, ownerToken, "ArchWriteBlock")
	memberID := getMyUserID(t, client, memberToken)
	thirdID := getMyUserID(t, client, thirdToken)

	// Owner adds member
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+memberID+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Archive project
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/archive",
		"", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Cannot add member on archived
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+thirdID+`","role":2}`, ownerToken)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	// Cannot remove member on archived
	resp = authedDo(t, client, http.MethodDelete, "/api/v1/projects/"+projID+"/members/"+memberID,
		"", ownerToken)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	// Cannot change member role on archived
	resp = authedDo(t, client, http.MethodPut, "/api/v1/projects/"+projID+"/members/"+memberID,
		`{"role":1}`, ownerToken)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

// ---------- include_archived query parameter ----------

func TestE2E_ListProjectsIncludeArchived(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_listarch_owner")

	projID := createProject(t, client, ownerToken, "ArchiveFilterTest")

	// Archive the project
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/archive",
		"", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// List without include_archived — project should NOT appear
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects", "", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data := decodeEnvelope(t, resp)
	var listRes struct {
		Projects []struct {
			Id string `json:"id"`
		} `json:"projects"`
	}
	json.Unmarshal(data, &listRes)
	for _, p := range listRes.Projects {
		assert.NotEqual(t, projID, p.Id, "archived project should not appear without include_archived")
	}

	// List with include_archived=true — project SHOULD appear
	resp = authedDo(t, client, http.MethodGet, "/api/v1/projects?include_archived=true", "", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, data = decodeEnvelope(t, resp)
	var listArchRes struct {
		Projects []struct {
			Id string `json:"id"`
		} `json:"projects"`
	}
	json.Unmarshal(data, &listArchRes)
	found := false
	for _, p := range listArchRes.Projects {
		if p.Id == projID {
			found = true
			break
		}
	}
	assert.True(t, found, "archived project should appear with include_archived=true")
}

// ---------- member write denials ----------

func TestE2E_MemberWriteDenials(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_mwd_owner")
	memberToken := registerAndGetToken(t, client, "e2e_mwd_member")

	projID := createProject(t, client, ownerToken, "MemberWriteDeny")
	ownerID := getMyUserID(t, client, ownerToken)
	memberID := getMyUserID(t, client, memberToken)

	// Owner adds member
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+memberID+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Member cannot update project
	resp = authedDo(t, client, http.MethodPut, "/api/v1/projects/"+projID,
		`{"name":"MemberHack","version":0}`, memberToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Member cannot archive
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/archive",
		"", memberToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Owner archives, then member cannot unarchive
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/archive",
		"", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/unarchive",
		"", memberToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
	// Owner unarchives for next test
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/unarchive",
		"", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Member cannot transfer ownership (use ownerID as target to avoid self-transfer check)
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/transfer",
		`{"target_user_id":"`+ownerID+`"}`, memberToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

// ---------- admin remaining boundaries ----------

func TestE2E_AdminBoundaries(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_ab_owner")
	adminToken := registerAndGetToken(t, client, "e2e_ab_admin")
	admin2Token := registerAndGetToken(t, client, "e2e_ab_admin2")
	memberToken := registerAndGetToken(t, client, "e2e_ab_member")

	projID := createProject(t, client, ownerToken, "AdminBoundary2")
	adminID := getMyUserID(t, client, adminToken)
	admin2ID := getMyUserID(t, client, admin2Token)
	memberID := getMyUserID(t, client, memberToken)

	// Owner adds admin (role=1), admin2 (role=1), member (role=2)
	resp := authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+adminID+`","role":1}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+admin2ID+`","role":1}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members",
		`{"user_id":"`+memberID+`","role":2}`, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Admin CANNOT unarchive
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/archive",
		"", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/unarchive",
		"", adminToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/unarchive",
		"", ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Admin CANNOT transfer ownership
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/transfer",
		`{"target_user_id":"`+admin2ID+`"}`, adminToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Admin CANNOT change member role
	resp = authedDo(t, client, http.MethodPut, "/api/v1/projects/"+projID+"/members/"+memberID,
		`{"role":1}`, adminToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Admin CANNOT remove another admin
	resp = authedDo(t, client, http.MethodDelete, "/api/v1/projects/"+projID+"/members/"+admin2ID,
		"", adminToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Admin CAN remove member
	resp = authedDo(t, client, http.MethodDelete, "/api/v1/projects/"+projID+"/members/"+memberID,
		"", adminToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Admin CAN leave
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members/me/leave",
		"", adminToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// ---------- owner self-boundaries ----------

func TestE2E_OwnerSelfBoundaries(t *testing.T) {
	client := newGatewayEngine(t)
	ownerToken := registerAndGetToken(t, client, "e2e_osb_owner")

	projID := createProject(t, client, ownerToken, "OwnerSelfTest")
	ownerID := getMyUserID(t, client, ownerToken)

	// Owner cannot remove self
	resp := authedDo(t, client, http.MethodDelete, "/api/v1/projects/"+projID+"/members/"+ownerID,
		"", ownerToken)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	// Owner cannot leave
	resp = authedDo(t, client, http.MethodPost, "/api/v1/projects/"+projID+"/members/me/leave",
		"", ownerToken)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

// ---------- gRPC direct tests ----------

func TestIntegration_CreateProject_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register a user first so we have a valid user ID
	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_proj_creator", Email: "grpc_proj_creator@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	ctx = withUser(ctx, regRes.User.Id, regRes.User.Username)
	res, err := taskGrpcClient.CreateProject(ctx, &taskv1.CreateProjectRequest{
		Name: "GRPC Project", Description: "via gRPC",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Project.Id)
	assert.Equal(t, "GRPC Project", res.Project.Name)
	assert.Equal(t, regRes.User.Id, res.Project.OwnerId)
}

func TestIntegration_ListProjects_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_lister", Email: "grpc_lister@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	ctx = withUser(ctx, regRes.User.Id, regRes.User.Username)

	_, err = taskGrpcClient.CreateProject(ctx, &taskv1.CreateProjectRequest{Name: "P1"})
	require.NoError(t, err)
	_, err = taskGrpcClient.CreateProject(ctx, &taskv1.CreateProjectRequest{Name: "P2"})
	require.NoError(t, err)

	listRes, err := taskGrpcClient.ListProjects(ctx, &taskv1.ListProjectsRequest{Limit: 20})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listRes.Projects), 2)
}

func TestIntegration_ArchiveProject_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_archiver", Email: "grpc_archiver@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	ctx = withUser(ctx, regRes.User.Id, regRes.User.Username)

	createRes, err := taskGrpcClient.CreateProject(ctx, &taskv1.CreateProjectRequest{Name: "To Archive"})
	require.NoError(t, err)
	projID := createRes.Project.Id

	archiveRes, err := taskGrpcClient.ArchiveProject(ctx, &taskv1.ArchiveProjectRequest{ProjectId: projID})
	require.NoError(t, err)
	assert.Equal(t, int32(1), archiveRes.Project.Status)

	// Read still works
	getRes, err := taskGrpcClient.GetProject(ctx, &taskv1.GetProjectRequest{ProjectId: projID})
	require.NoError(t, err)
	assert.Equal(t, int32(1), getRes.Project.Status)

	// Write on archived fails
	_, err = taskGrpcClient.UpdateProject(ctx, &taskv1.UpdateProjectRequest{
		ProjectId: projID, Name: "Hacked", Version: archiveRes.Project.Version,
	})
	require.Error(t, err)
}

func TestIntegration_TransferOwnership_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register two users
	ownerRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_xfer_o", Email: "grpc_xfer_o@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	targetRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_xfer_t", Email: "grpc_xfer_t@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	ctx = withUser(ctx, ownerRes.User.Id, ownerRes.User.Username)

	// Create project
	createRes, err := taskGrpcClient.CreateProject(ctx, &taskv1.CreateProjectRequest{Name: "Xfer Project"})
	require.NoError(t, err)
	projID := createRes.Project.Id

	// Add target as admin
	addCtx := withUser(context.Background(), ownerRes.User.Id, ownerRes.User.Username)
	_, err = taskGrpcClient.AddProjectMember(addCtx, &taskv1.AddProjectMemberRequest{
		ProjectId: projID, UserId: targetRes.User.Id, Role: 1,
	})
	require.NoError(t, err)

	// Transfer ownership
	xferCtx := withUser(context.Background(), ownerRes.User.Id, ownerRes.User.Username)
	xferRes, err := taskGrpcClient.TransferProjectOwnership(xferCtx, &taskv1.TransferProjectOwnershipRequest{
		ProjectId: projID, TargetUserId: targetRes.User.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, targetRes.User.Id, xferRes.Project.OwnerId)

	// Verify old owner is admin, new owner is owner
	checkCtx := withUser(context.Background(), targetRes.User.Id, targetRes.User.Username)
	membersRes, err := taskGrpcClient.ListProjectMembers(checkCtx, &taskv1.ListProjectMembersRequest{ProjectId: projID})
	require.NoError(t, err)

	var oldOwnerRole, newOwnerRole int32 = -1, -1
	for _, m := range membersRes.Members {
		if m.UserId == ownerRes.User.Id {
			oldOwnerRole = m.Role
		}
		if m.UserId == targetRes.User.Id {
			newOwnerRole = m.Role
		}
	}
	assert.Equal(t, int32(1), oldOwnerRole, "old owner should be admin")
	assert.Equal(t, int32(0), newOwnerRole, "new owner should be owner")
}

func TestIntegration_AdminCannotAddAdmin_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ownerRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_aaa_o", Email: "grpc_aaa_o@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	adminRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_aaa_a", Email: "grpc_aaa_a@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	thirdRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_aaa_3", Email: "grpc_aaa_3@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	ctx = withUser(ctx, ownerRes.User.Id, ownerRes.User.Username)
	createRes, err := taskGrpcClient.CreateProject(ctx, &taskv1.CreateProjectRequest{Name: "AAA Project"})
	require.NoError(t, err)
	projID := createRes.Project.Id

	// Owner adds admin
	_, err = taskGrpcClient.AddProjectMember(ctx, &taskv1.AddProjectMemberRequest{
		ProjectId: projID, UserId: adminRes.User.Id, Role: 1,
	})
	require.NoError(t, err)

	// Admin tries to add another admin — should fail
	adminCtx := withUser(context.Background(), adminRes.User.Id, adminRes.User.Username)
	_, err = taskGrpcClient.AddProjectMember(adminCtx, &taskv1.AddProjectMemberRequest{
		ProjectId: projID, UserId: thirdRes.User.Id, Role: 1,
	})
	require.Error(t, err, "admin should not be able to add another admin")
}

func TestIntegration_CheckProjectMember_GRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ownerRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_chk_o", Email: "grpc_chk_o@test.com", Password: "secret123",
	})
	require.NoError(t, err)
	otherRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grpc_chk_s", Email: "grpc_chk_s@test.com", Password: "secret123",
	})
	require.NoError(t, err)

	ctx = withUser(ctx, ownerRes.User.Id, ownerRes.User.Username)
	createRes, err := taskGrpcClient.CreateProject(ctx, &taskv1.CreateProjectRequest{Name: "Check Project"})
	require.NoError(t, err)
	projID := createRes.Project.Id

	// Owner is member
	checkRes, err := taskGrpcClient.CheckProjectMember(ctx, &taskv1.CheckProjectMemberRequest{
		ProjectId: projID, UserId: ownerRes.User.Id,
	})
	require.NoError(t, err)
	assert.True(t, checkRes.IsMember)
	assert.Equal(t, int32(0), checkRes.Role)

	// Stranger is not member
	checkRes, err = taskGrpcClient.CheckProjectMember(ctx, &taskv1.CheckProjectMemberRequest{
		ProjectId: projID, UserId: otherRes.User.Id,
	})
	require.NoError(t, err)
	assert.False(t, checkRes.IsMember)
}
