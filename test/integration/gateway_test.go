//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	gwserver "task-platform/internal/gateway/server"
	"task-platform/pkg/xerr"
	"task-platform/pkg/xjwt"
)

func newGatewayEngine(t *testing.T) *http.Client {
	t.Helper()

	logger := zap.NewNop()
	ready := &atomic.Bool{}

	cfg := gwserver.Config{
		UserServiceAddr: grpcLisAddr,
		RedisAddr:       redisAddr,
		RedisPassword:   "",
		JWTSecret:       testJWTSecret,
		InternalToken:   testInternalToken,
	}

	engine, cleanup, err := gwserver.NewEngine("test-gateway", ready, logger, cfg)
	if err != nil {
		t.Fatalf("create gateway engine: %v", err)
	}
	t.Cleanup(func() { cleanup() })

	return &http.Client{
		Transport: &localRoundTripper{engine: engine},
	}
}

// localRoundTripper calls the gin engine directly without a real HTTP server.
type localRoundTripper struct {
	engine http.Handler
}

func (rt *localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	rt.engine.ServeHTTP(w, req)
	return w.Result(), nil
}

func TestGateway_Register(t *testing.T) {
	client := newGatewayEngine(t)

	body := `{"username":"gwtest1","email":"gwtest1@example.com","password":"secret123"}`
	resp, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var envelope xerr.HTTPResponse
	json.NewDecoder(resp.Body).Decode(&envelope)
	if envelope.Code != xerr.CodeOK {
		t.Errorf("code = %s", envelope.Code)
	}
	if envelope.Data == nil {
		t.Fatal("data should not be nil")
	}
}

func TestGateway_Register_WeakPassword(t *testing.T) {
	client := newGatewayEngine(t)

	body := `{"username":"gwtest2","email":"gwtest2@example.com","password":"12345678"}`
	resp, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGateway_Register_DuplicateUsername(t *testing.T) {
	client := newGatewayEngine(t)

	body1 := `{"username":"gwdup1","email":"gwdup1a@example.com","password":"secret123"}`
	resp1, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(body1))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first register: status = %d", resp1.StatusCode)
	}

	body2 := `{"username":"gwdup1","email":"gwdup1b@example.com","password":"secret123"}`
	resp2, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(body2))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp2.StatusCode, http.StatusConflict)
	}
}

func TestGateway_Login(t *testing.T) {
	client := newGatewayEngine(t)

	// Register first
	body1 := `{"username":"gwlogin1","email":"gwlogin1@example.com","password":"secret123"}`
	resp1, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(body1))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("register failed: status = %d", resp1.StatusCode)
	}

	// Login
	body2 := `{"account":"gwlogin1","password":"secret123"}`
	resp2, err := client.Post("http://localhost/api/v1/auth/login", "application/json", strings.NewReader(body2))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp2.StatusCode, http.StatusOK)
	}

	var envelope xerr.HTTPResponse
	json.NewDecoder(resp2.Body).Decode(&envelope)
	if envelope.Code != xerr.CodeOK {
		t.Errorf("code = %s", envelope.Code)
	}
}

func TestGateway_Login_WrongPassword(t *testing.T) {
	client := newGatewayEngine(t)

	// Register first
	body1 := `{"username":"gwbadpw","email":"gwbadpw@example.com","password":"secret123"}`
	resp1, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(body1))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("register failed: status = %d", resp1.StatusCode)
	}

	// Login with wrong password
	body2 := `{"account":"gwbadpw","password":"wrongpassword"}`
	resp2, err := client.Post("http://localhost/api/v1/auth/login", "application/json", strings.NewReader(body2))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp2.StatusCode, http.StatusUnauthorized)
	}
}

func TestGateway_Me(t *testing.T) {
	client := newGatewayEngine(t)

	// Register
	regBody := `{"username":"gwme1","email":"gwme1@example.com","password":"secret123"}`
	regResp, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(regBody))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	var regEnvelope xerr.HTTPResponse
	json.NewDecoder(regResp.Body).Decode(&regEnvelope)
	regResp.Body.Close()
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register failed: status = %d", regResp.StatusCode)
	}

	regData, _ := json.Marshal(regEnvelope.Data)
	var regParsed struct {
		AccessToken string `json:"access_token"`
	}
	json.Unmarshal(regData, &regParsed)
	token := regParsed.AccessToken
	if token == "" {
		t.Fatal("no access token in register response")
	}

	// Me request
	meReq, _ := http.NewRequest(http.MethodGet, "http://localhost/api/v1/users/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meResp, err := client.Do(meReq)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	defer meResp.Body.Close()

	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", meResp.StatusCode, http.StatusOK)
	}

	var meEnvelope xerr.HTTPResponse
	json.NewDecoder(meResp.Body).Decode(&meEnvelope)
	meData, _ := json.Marshal(meEnvelope.Data)
	var meParsed struct {
		User struct {
			Username string `json:"username"`
			Email    string `json:"email"`
		} `json:"user"`
	}
	json.Unmarshal(meData, &meParsed)
	if meParsed.User.Username != "gwme1" {
		t.Errorf("username = %s, want gwme1", meParsed.User.Username)
	}
	if meParsed.User.Email != "gwme1@example.com" {
		t.Errorf("email = %s", meParsed.User.Email)
	}
}

func TestGateway_Me_Unauthorized(t *testing.T) {
	client := newGatewayEngine(t)

	req, _ := http.NewRequest(http.MethodGet, "http://localhost/api/v1/users/me", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestGateway_Logout(t *testing.T) {
	client := newGatewayEngine(t)

	// Register
	regBody := `{"username":"gwlogout1","email":"gwlogout1@example.com","password":"secret123"}`
	regResp, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(regBody))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	var regEnvelope xerr.HTTPResponse
	json.NewDecoder(regResp.Body).Decode(&regEnvelope)
	regResp.Body.Close()
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register failed: status = %d", regResp.StatusCode)
	}

	regData, _ := json.Marshal(regEnvelope.Data)
	var regParsed struct {
		AccessToken string `json:"access_token"`
	}
	json.Unmarshal(regData, &regParsed)
	token := regParsed.AccessToken

	// Logout
	logoutReq, _ := http.NewRequest(http.MethodPost, "http://localhost/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+token)
	logoutResp, err := client.Do(logoutReq)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	logoutResp.Body.Close()

	if logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logoutResp.StatusCode, http.StatusOK)
	}

	// Me after logout should fail (token blacklisted)
	meReq, _ := http.NewRequest(http.MethodGet, "http://localhost/api/v1/users/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meResp, err := client.Do(meReq)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	defer meResp.Body.Close()

	if meResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("me after logout status = %d, want %d", meResp.StatusCode, http.StatusUnauthorized)
	}
}

func TestGateway_Auth_ExpiredToken(t *testing.T) {
	client := newGatewayEngine(t)

	jwtMgr := xjwt.NewManager(testJWTSecret)
	expiredToken, _, err := jwtMgr.Generate("test-user", "testuser", -1*time.Hour)
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, "http://localhost/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestGateway_Auth_InvalidToken(t *testing.T) {
	client := newGatewayEngine(t)

	req, _ := http.NewRequest(http.MethodGet, "http://localhost/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestGateway_Register_InvalidJSON(t *testing.T) {
	client := newGatewayEngine(t)

	body := `not json`
	resp, err := client.Post("http://localhost/api/v1/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGateway_Healthz(t *testing.T) {
	client := newGatewayEngine(t)

	resp, err := client.Get("http://localhost/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
