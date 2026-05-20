package biz_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"task-platform/internal/task/biz"
	"task-platform/internal/task/data"
)

func uid() string { return uuid.New().String() }

type mockUserClient struct {
	exists bool
	active bool
	err    error
}

func (m *mockUserClient) GetUser(_ context.Context, _ string) (bool, bool, error) {
	return m.exists, m.active, m.err
}

func setupBizDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("task_platform"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("pg connection string: %v", err)
	}

	db, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	rootDir := findBizRoot()
	migrationSQL, err := os.ReadFile(filepath.Join(rootDir, "migrations", "task_svc", "000001_init.up.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if err := db.Exec(string(migrationSQL)).Error; err != nil {
		t.Fatalf("run migration: %v", err)
	}

	cleanup := func() { _ = pgContainer.Terminate(ctx) }
	return db, cleanup
}

func findBizRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func setupBiz(t *testing.T) (*biz.ProjectBiz, func(), string, string) {
	t.Helper()
	db, cleanup := setupBizDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	userClient := &mockUserClient{exists: true, active: true}
	b := biz.NewProjectBiz(db, projectRepo, memberRepo, userClient, nil, nil)
	caller := uid()
	other := uid()
	return b, cleanup, caller, other
}

func TestCreateProject_Success(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, err := b.CreateProject(context.Background(), caller, "My Project", "A description")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.ID == "" {
		t.Error("ID should be set")
	}
	if p.Name != "My Project" {
		t.Errorf("name = %s", p.Name)
	}
	if p.OwnerID != caller {
		t.Errorf("owner = %s", p.OwnerID)
	}
}

func TestCreateProject_InvalidName(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	_, err := b.CreateProject(context.Background(), caller, "", "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestGetProject_Success(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	got, err := b.GetProject(context.Background(), p.ID, caller)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("id mismatch")
	}
}

func TestGetProject_NotMember(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	nonMember := uid()
	_, err := b.GetProject(context.Background(), p.ID, nonMember)
	if err == nil {
		t.Error("expected error for non-member")
	}
}

func TestGetProject_ArchivedProject_MemberCanRead(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.ArchiveProject(context.Background(), p.ID, caller)

	got, err := b.GetProject(context.Background(), p.ID, caller)
	if err != nil {
		t.Fatalf("member should be able to read archived project: %v", err)
	}
	if got.Status != data.ProjectStatusArchived {
		t.Error("expected archived status")
	}
}

func TestGetProject_ArchivedProject_NonMemberNotFound(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.ArchiveProject(context.Background(), p.ID, caller)

	nonMember := uid()
	_, err := b.GetProject(context.Background(), p.ID, nonMember)
	if err == nil {
		t.Error("expected not found for non-member of archived project")
	}
}

func TestUpdateProject_Success(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Old", "Old desc")
	updated, err := b.UpdateProject(context.Background(), p.ID, caller, "New", "New desc", p.Version)
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	if updated.Name != "New" {
		t.Errorf("name = %s", updated.Name)
	}
}

func TestUpdateProject_NotOwner(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)
	if err != nil {
		t.Fatalf("add member for test: %v", err)
	}

	_, err = b.UpdateProject(context.Background(), p.ID, other, "Hacked", "", p.Version)
	if err == nil {
		t.Error("expected permission denied for non-owner")
	}
}

func TestUpdateProject_Archived(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	ap, _ := b.ArchiveProject(context.Background(), p.ID, caller)

	_, err := b.UpdateProject(context.Background(), p.ID, caller, "New", "", ap.Version)
	if err == nil {
		t.Error("expected failed precondition for archived project")
	}
}

func TestUpdateProject_VersionConflict(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.UpdateProject(context.Background(), p.ID, caller, "New", "", 999)
	if err == nil {
		t.Error("expected version conflict")
	}
}

func TestArchiveProject_Success(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	archived, err := b.ArchiveProject(context.Background(), p.ID, caller)
	if err != nil {
		t.Fatalf("ArchiveProject: %v", err)
	}
	if archived.Status != data.ProjectStatusArchived {
		t.Errorf("status = %d, want %d", archived.Status, data.ProjectStatusArchived)
	}
}

func TestArchiveProject_NotOwner(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	_, err = b.ArchiveProject(context.Background(), p.ID, other)
	if err == nil {
		t.Error("expected permission denied")
	}
}

func TestUnarchiveProject_Success(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.ArchiveProject(context.Background(), p.ID, caller)

	unarchived, err := b.UnarchiveProject(context.Background(), p.ID, caller)
	if err != nil {
		t.Fatalf("UnarchiveProject: %v", err)
	}
	if unarchived.Status != data.ProjectStatusActive {
		t.Errorf("status = %d, want %d", unarchived.Status, data.ProjectStatusActive)
	}
}

func TestTransferOwnership_Success(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	transferred, err := b.TransferOwnership(context.Background(), p.ID, caller, other)
	if err != nil {
		t.Fatalf("TransferOwnership: %v", err)
	}
	if transferred.OwnerID != other {
		t.Errorf("owner = %s, want %s", transferred.OwnerID, other)
	}

	// Assert old owner is now admin
	isMember, role, err := b.CheckProjectMember(context.Background(), p.ID, caller)
	if err != nil {
		t.Fatalf("CheckProjectMember: %v", err)
	}
	if !isMember {
		t.Error("old owner should still be a member")
	}
	if role != data.RoleAdmin {
		t.Errorf("old owner role = %d, want %d (admin)", role, data.RoleAdmin)
	}

	// Assert new owner is owner
	_, newOwnerRole, err := b.CheckProjectMember(context.Background(), p.ID, other)
	if err != nil {
		t.Fatalf("CheckProjectMember: %v", err)
	}
	if newOwnerRole != data.RoleOwner {
		t.Errorf("new owner role = %d, want %d (owner)", newOwnerRole, data.RoleOwner)
	}
}

func TestTransferOwnership_ToSelf(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.TransferOwnership(context.Background(), p.ID, caller, caller)
	if err == nil {
		t.Error("expected error for self-transfer")
	}
}

func TestTransferOwnership_NotOwner(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	_, err = b.TransferOwnership(context.Background(), p.ID, other, caller)
	if err == nil {
		t.Error("expected permission denied")
	}
}

func TestTransferOwnership_TargetNotMember(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	randomID := uid()
	_, err := b.TransferOwnership(context.Background(), p.ID, caller, randomID)
	if err == nil {
		t.Error("expected failed precondition for non-member target")
	}
}

func TestAddProjectMember_Success(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	m, err := b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)
	if err != nil {
		t.Fatalf("AddProjectMember: %v", err)
	}
	if m.Role != data.RoleMember {
		t.Errorf("role = %d", m.Role)
	}
}

func TestAddProjectMember_AdminCanAddMember(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)

	m, err := b.AddProjectMember(context.Background(), p.ID, other, uid(), data.RoleMember)
	if err != nil {
		t.Fatalf("admin should be able to add member: %v", err)
	}
	if m.Role != data.RoleMember {
		t.Errorf("role = %d", m.Role)
	}
}

func TestAddProjectMember_AdminCannotAddAdmin(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)

	_, err := b.AddProjectMember(context.Background(), p.ID, other, uid(), data.RoleAdmin)
	if err == nil {
		t.Error("admin should not be able to add another admin")
	}
}

func TestAddProjectMember_MemberCannotAdd(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)

	_, err := b.AddProjectMember(context.Background(), p.ID, other, uid(), data.RoleMember)
	if err == nil {
		t.Error("member should not be able to add members")
	}
}

func TestAddProjectMember_UserNotActive(t *testing.T) {
	db, cleanup := setupBizDB(t)
	defer cleanup()
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	userClient := &mockUserClient{exists: true, active: false}
	b := biz.NewProjectBiz(db, projectRepo, memberRepo, userClient, nil, nil)

	caller := uid()
	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.AddProjectMember(context.Background(), p.ID, caller, uid(), data.RoleMember)
	if err == nil {
		t.Error("expected failed precondition for disabled user")
	}
}

func TestRemoveProjectMember_Success(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)

	m, err := b.RemoveProjectMember(context.Background(), p.ID, caller, other)
	if err != nil {
		t.Fatalf("RemoveProjectMember: %v", err)
	}
	if m.UserID != other {
		t.Errorf("user_id = %s", m.UserID)
	}
}

func TestRemoveProjectMember_CannotRemoveSelf(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.RemoveProjectMember(context.Background(), p.ID, caller, caller)
	if err == nil {
		t.Error("expected error for self-remove")
	}
}

func TestRemoveProjectMember_AdminCannotRemoveAdmin(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)
	u3 := uid()
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, u3, data.RoleAdmin)

	_, err := b.RemoveProjectMember(context.Background(), p.ID, other, u3)
	if err == nil {
		t.Error("admin should not be able to remove admin")
	}
}

func TestRemoveProjectMember_MemberCannotRemove(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)
	u3 := uid()
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, u3, data.RoleMember)

	_, err := b.RemoveProjectMember(context.Background(), p.ID, other, u3)
	if err == nil {
		t.Error("member should not be able to remove members")
	}
}

func TestUpdateProjectMemberRole_Success(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)

	m, err := b.UpdateProjectMemberRole(context.Background(), p.ID, caller, other, data.RoleAdmin)
	if err != nil {
		t.Fatalf("UpdateProjectMemberRole: %v", err)
	}
	if m.Role != data.RoleAdmin {
		t.Errorf("role = %d", m.Role)
	}
}

func TestUpdateProjectMemberRole_OnlyOwner(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)
	u3 := uid()
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, u3, data.RoleMember)

	_, err := b.UpdateProjectMemberRole(context.Background(), p.ID, other, u3, data.RoleAdmin)
	if err == nil {
		t.Error("admin should not be able to change roles")
	}
}

func TestLeaveProject_Success(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)

	m, err := b.LeaveProject(context.Background(), p.ID, other)
	if err != nil {
		t.Fatalf("LeaveProject: %v", err)
	}
	if m.UserID != other {
		t.Errorf("user_id = %s", m.UserID)
	}
}

func TestLeaveProject_OwnerCannotLeave(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.LeaveProject(context.Background(), p.ID, caller)
	if err == nil {
		t.Error("owner should not be able to leave")
	}
}

func TestListProjectMembers(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)

	members, err := b.ListProjectMembers(context.Background(), p.ID, caller)
	if err != nil {
		t.Fatalf("ListProjectMembers: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("len = %d, want 2", len(members))
	}
}

func TestListProjectMembers_NonMember(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	nonMember := uid()
	_, err := b.ListProjectMembers(context.Background(), p.ID, nonMember)
	if err == nil {
		t.Error("expected not found for non-member")
	}
}

func TestCheckProjectMember(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")

	isMember, role, err := b.CheckProjectMember(context.Background(), p.ID, caller)
	if err != nil {
		t.Fatalf("CheckProjectMember: %v", err)
	}
	if !isMember {
		t.Error("expected member")
	}
	if role != data.RoleOwner {
		t.Errorf("role = %d, want %d", role, data.RoleOwner)
	}
}

func TestCheckProjectMember_NotMember(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	nonMember := uid()

	isMember, _, err := b.CheckProjectMember(context.Background(), p.ID, nonMember)
	if err != nil {
		t.Fatalf("CheckProjectMember: %v", err)
	}
	if isMember {
		t.Error("expected not member")
	}
}

func TestListProjects(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	_, _ = b.CreateProject(context.Background(), caller, "Project 1", "")
	_, _ = b.CreateProject(context.Background(), caller, "Project 2", "")

	projects, err := b.ListProjects(context.Background(), caller, false, 20, 0)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) < 2 {
		t.Errorf("len = %d, want at least 2", len(projects))
	}
}

func TestUnarchiveProject_NotOwner(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.ArchiveProject(context.Background(), p.ID, caller)
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)

	_, err := b.UnarchiveProject(context.Background(), p.ID, other)
	if err == nil {
		t.Error("expected permission denied for non-owner unarchiving")
	}
}

func TestTransferOwnership_TargetDisabled(t *testing.T) {
	db, cleanup := setupBizDB(t)
	defer cleanup()
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	userClient := &mockUserClient{exists: true, active: false}
	b := biz.NewProjectBiz(db, projectRepo, memberRepo, userClient, nil, nil)

	caller := uid()
	other := uid()
	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)

	_, err := b.TransferOwnership(context.Background(), p.ID, caller, other)
	if err == nil {
		t.Error("expected failed precondition for disabled target")
	}
}

func TestAddProjectMember_InvalidRole(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.AddProjectMember(context.Background(), p.ID, caller, uid(), 99)
	if err == nil {
		t.Error("expected invalid argument for invalid role")
	}
}

func TestAddProjectMember_UserNotExists(t *testing.T) {
	db, cleanup := setupBizDB(t)
	defer cleanup()
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	userClient := &mockUserClient{exists: false, active: false}
	b := biz.NewProjectBiz(db, projectRepo, memberRepo, userClient, nil, nil)

	caller := uid()
	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.AddProjectMember(context.Background(), p.ID, caller, uid(), data.RoleMember)
	if err == nil {
		t.Error("expected failed precondition for non-existent user")
	}
}

func TestRemoveProjectMember_TargetNotMember(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.RemoveProjectMember(context.Background(), p.ID, caller, uid())
	if err == nil {
		t.Error("expected not found for non-member target")
	}
}

func TestUpdateProjectMemberRole_TargetNotMember(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, err := b.UpdateProjectMemberRole(context.Background(), p.ID, caller, uid(), data.RoleAdmin)
	if err == nil {
		t.Error("expected not found for non-member target")
	}
}

func TestLeaveProject_NotMember(t *testing.T) {
	db, cleanup := setupBizDB(t)
	defer cleanup()
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	userClient := &mockUserClient{exists: true, active: true}
	b2 := biz.NewProjectBiz(db, projectRepo, memberRepo, userClient, nil, nil)

	_, err := b2.LeaveProject(context.Background(), uid(), uid())
	if err == nil {
		t.Error("expected not found for non-member")
	}
}

func TestListProjects_WithLimitZero(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	projects, err := b.ListProjects(context.Background(), caller, false, 0, 0)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) > 0 {
		t.Log("projects returned despite limit 0 (should use default)")
	}
}

func TestListProjects_IncludeArchived(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	_, _ = b.CreateProject(context.Background(), caller, "Active Project", "")
	p2, _ := b.CreateProject(context.Background(), caller, "Archived Project", "")
	_, _ = b.ArchiveProject(context.Background(), p2.ID, caller)

	projects, err := b.ListProjects(context.Background(), caller, true, 20, 0)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) < 2 {
		t.Errorf("len = %d, want at least 2 with archived included", len(projects))
	}

	projectsNoArchived, err := b.ListProjects(context.Background(), caller, false, 20, 0)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	for _, p := range projectsNoArchived {
		if p.Status == data.ProjectStatusArchived {
			t.Error("should not include archived when include_archived=false")
		}
	}
}

func TestTransferOwnership_Archived(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleAdmin)
	_, _ = b.ArchiveProject(context.Background(), p.ID, caller)

	_, err := b.TransferOwnership(context.Background(), p.ID, caller, other)
	if err == nil {
		t.Error("expected failed precondition for archived project")
	}
}

func TestAddProjectMember_Archived(t *testing.T) {
	b, cleanup, caller, _ := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.ArchiveProject(context.Background(), p.ID, caller)

	_, err := b.AddProjectMember(context.Background(), p.ID, caller, uid(), data.RoleMember)
	if err == nil {
		t.Error("expected failed precondition for archived project")
	}
}

func TestRemoveProjectMember_Archived(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)
	_, _ = b.ArchiveProject(context.Background(), p.ID, caller)

	_, err := b.RemoveProjectMember(context.Background(), p.ID, caller, other)
	if err == nil {
		t.Error("expected failed precondition for archived project")
	}
}

func TestUpdateProjectMemberRole_Archived(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)
	_, _ = b.ArchiveProject(context.Background(), p.ID, caller)

	_, err := b.UpdateProjectMemberRole(context.Background(), p.ID, caller, other, data.RoleAdmin)
	if err == nil {
		t.Error("expected failed precondition for archived project")
	}
}

func TestLeaveProject_Archived(t *testing.T) {
	b, cleanup, caller, other := setupBiz(t)
	defer cleanup()

	p, _ := b.CreateProject(context.Background(), caller, "Project", "")
	_, _ = b.AddProjectMember(context.Background(), p.ID, caller, other, data.RoleMember)
	_, _ = b.ArchiveProject(context.Background(), p.ID, caller)

	_, err := b.LeaveProject(context.Background(), p.ID, other)
	if err == nil {
		t.Error("expected failed precondition for archived project")
	}
}
