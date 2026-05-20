//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func flushRateLimitKeys(t *testing.T) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()
	ctx := context.Background()
	iter := rdb.Scan(ctx, 0, "ratelimit:*", 0).Iterator()
	for iter.Next(ctx) {
		rdb.Del(ctx, iter.Val())
	}
}

func TestIdempotency_DuplicateCreateProject(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "idem_alice")

	body := `{"name":"idem-project","description":"test"}`

	// First request with idempotency key — should create
	req1, _ := http.NewRequest(http.MethodPost, "http://localhost/api/v1/projects", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	req1.Header.Set("Idempotency-Key", "idem-key-1")
	res1, err := client.Do(req1)
	require.NoError(t, err)
	env1, _ := decodeEnvelope(t, res1)
	assert.Equal(t, http.StatusCreated, res1.StatusCode)
	assert.Equal(t, "OK", env1.Code)

	// Duplicate request with same key — should return 200 with cached response
	req2, _ := http.NewRequest(http.MethodPost, "http://localhost/api/v1/projects", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Idempotency-Key", "idem-key-1")
	res2, err := client.Do(req2)
	require.NoError(t, err)
	env2, _ := decodeEnvelope(t, res2)
	assert.Equal(t, http.StatusOK, res2.StatusCode, "duplicate request should return 200 OK")
	assert.Equal(t, "OK", env2.Code)
}

func TestIdempotency_DifferentKeysAllowRequests(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "idem2_alice")

	body1 := `{"name":"idem-proj-A","description":"A"}`
	req1, _ := http.NewRequest(http.MethodPost, "http://localhost/api/v1/projects", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	req1.Header.Set("Idempotency-Key", "idem-key-A")
	res1, _ := client.Do(req1)
	assert.Equal(t, http.StatusCreated, res1.StatusCode)
	io.Copy(io.Discard, res1.Body)
	res1.Body.Close()

	body2 := `{"name":"idem-proj-B","description":"B"}`
	req2, _ := http.NewRequest(http.MethodPost, "http://localhost/api/v1/projects", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Idempotency-Key", "idem-key-B")
	res2, _ := client.Do(req2)
	assert.Equal(t, http.StatusCreated, res2.StatusCode, "different key should allow new request")
	io.Copy(io.Discard, res2.Body)
	res2.Body.Close()
}

func TestIdempotency_InvalidBodyFreesKey(t *testing.T) {
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "idem3_alice")

	// Invalid body with idempotency key — should be rejected
	req1, _ := http.NewRequest(http.MethodPost, "http://localhost/api/v1/projects", bytes.NewReader([]byte(`{bad`)))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	req1.Header.Set("Idempotency-Key", "idem-key-invalid")
	res1, _ := client.Do(req1)
	io.Copy(io.Discard, res1.Body)
	res1.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res1.StatusCode, "invalid body should return 400")

	// Same key with valid body should succeed (key freed after error)
	validBody := `{"name":"idem-proj-valid","description":"ok"}`
	req2, _ := http.NewRequest(http.MethodPost, "http://localhost/api/v1/projects", strings.NewReader(validBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Idempotency-Key", "idem-key-invalid")
	res2, _ := client.Do(req2)
	io.Copy(io.Discard, res2.Body)
	res2.Body.Close()
	assert.Equal(t, http.StatusCreated, res2.StatusCode, "same key should work after invalid request freed it")
}

func TestRateLimit_AnonymousEndpoints(t *testing.T) {
	flushRateLimitKeys(t)
	defer flushRateLimitKeys(t)
	client := newGatewayEngine(t)

	body := `{"username":"rl_user","email":"rl@test.com","password":"SecurePass123!"}`
	gotLimited := false
	for i := 0; i < 20; i++ {
		res, _ := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(body))
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode == http.StatusTooManyRequests {
			gotLimited = true
			break
		}
	}
	assert.True(t, gotLimited, "rate limiting should block excessive anonymous requests")
}

func TestRateLimit_AuthenticatedEndpoints(t *testing.T) {
	flushRateLimitKeys(t)
	defer flushRateLimitKeys(t)
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "rl_auth")
	require.NotEmpty(t, token, "register must succeed for rate limit test")

	gotLimited := false
	for i := 0; i < 120; i++ {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost/api/v1/projects?offset=%d", i), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res, _ := client.Do(req)
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode == http.StatusTooManyRequests {
			gotLimited = true
			break
		}
	}
	assert.True(t, gotLimited, "authenticated rate limiting should block excessive requests")
}

func TestCache_ProjectGetAndInvalidation(t *testing.T) {
	flushRateLimitKeys(t)
	client := newGatewayEngine(t)
	token := registerAndGetToken(t, client, "cache_alice")

	// Create a project
	body := `{"name":"cache-proj","description":"cache test"}`
	resCreate := authedDo(t, client, http.MethodPost, "/api/v1/projects", body, token)
	_, createData := decodeEnvelope(t, resCreate)
	assert.Equal(t, http.StatusCreated, resCreate.StatusCode, "create project")

	var createResp struct {
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	}
	json.Unmarshal(createData, &createResp)
	projectID := createResp.Project.ID
	require.NotEmpty(t, projectID)

	// GET #1 — cache miss, fetches from DB
	res1 := authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projectID, "", token)
	assert.Equal(t, http.StatusOK, res1.StatusCode, "GET #1")
	decodeEnvelope(t, res1)

	// GET #2 — cache hit
	res2 := authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projectID, "", token)
	assert.Equal(t, http.StatusOK, res2.StatusCode, "GET #2 (cache hit)")
	decodeEnvelope(t, res2)

	// Update project — should invalidate cache
	updateBody := `{"name":"cache-proj-updated","version":0}`
	resUpdate := authedDo(t, client, http.MethodPut, "/api/v1/projects/"+projectID, updateBody, token)
	assert.Equal(t, http.StatusOK, resUpdate.StatusCode, "update project")
	decodeEnvelope(t, resUpdate)

	// GET #3 — cache miss (was invalidated), fresh data
	res3 := authedDo(t, client, http.MethodGet, "/api/v1/projects/"+projectID, "", token)
	assert.Equal(t, http.StatusOK, res3.StatusCode, "GET #3 (after invalidation)")
	decodeEnvelope(t, res3)
}
