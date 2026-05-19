package data_test

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

	"task-platform/internal/task/data"
)

func uid() string { return uuid.New().String() }

func setupTaskDB(t *testing.T) *gorm.DB {
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
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("pg connection string: %v", err)
	}

	db, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	rootDir := findTaskRoot()
	migrationSQL, err := os.ReadFile(filepath.Join(rootDir, "migrations", "task_svc", "000001_init.up.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	if err := db.Exec(string(migrationSQL)).Error; err != nil {
		t.Fatalf("run migration: %v", err)
	}

	return db
}

func findTaskRoot() string {
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

func TestProjectCreate_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewProjectRepository(db)

	u := uid()
	p := &data.Project{Name: "Test Project", Description: "A test project", OwnerID: u}
	err := repo.Create(context.Background(), p)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.ID == "" {
		t.Error("ID should be set after create")
	}
}

func TestProjectCreate_DuplicateName(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewProjectRepository(db)

	u := uid()
	p1 := &data.Project{Name: "My Project", OwnerID: u}
	if err := repo.Create(context.Background(), p1); err != nil {
		t.Fatalf("create: %v", err)
	}

	p2 := &data.Project{Name: "My Project", OwnerID: u}
	err := repo.Create(context.Background(), p2)
	if err == nil {
		t.Error("expected duplicate key error")
	}
}

func TestProjectFindByID_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewProjectRepository(db)

	p := &data.Project{Name: "FindMe", OwnerID: uid()}
	_ = repo.Create(context.Background(), p)

	found, err := repo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.Name != "FindMe" {
		t.Errorf("name = %s, want FindMe", found.Name)
	}
}

func TestProjectFindByID_NotFound(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewProjectRepository(db)

	_, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestProjectUpdate_Success(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewProjectRepository(db)

	p := &data.Project{Name: "Old Name", OwnerID: uid()}
	_ = repo.Create(context.Background(), p)

	updated, err := repo.Update(context.Background(), p.ID, p.Version, map[string]any{
		"name":    "New Name",
		"version": p.Version + 1,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("name = %s, want New Name", updated.Name)
	}
	if updated.Version != 1 {
		t.Errorf("version = %d, want 1", updated.Version)
	}
}

func TestProjectUpdate_VersionConflict(t *testing.T) {
	db := setupTaskDB(t)
	repo := data.NewProjectRepository(db)

	p := &data.Project{Name: "Conflict Test", OwnerID: uid()}
	_ = repo.Create(context.Background(), p)

	_, err := repo.Update(context.Background(), p.ID, 999, map[string]any{"name": "Conflict"})
	if err == nil {
		t.Error("expected version conflict error")
	}
}

func TestProjectFindByMemberID(t *testing.T) {
	db := setupTaskDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)

	o1 := uid()
	o2 := uid()
	ux := uid()

	p1 := &data.Project{Name: "Project 1", OwnerID: o1}
	p2 := &data.Project{Name: "Project 2", OwnerID: o2}
	_ = projectRepo.Create(context.Background(), p1)
	_ = projectRepo.Create(context.Background(), p2)

	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p1.ID, UserID: ux, Role: data.RoleMember})
	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p2.ID, UserID: ux, Role: data.RoleMember})

	projects, err := projectRepo.FindByMemberID(context.Background(), ux, false, 20, 0)
	if err != nil {
		t.Fatalf("find by member: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("len = %d, want 2", len(projects))
	}
}

func TestProjectFindByMemberID_ExcludeArchived(t *testing.T) {
	db := setupTaskDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)

	o := uid()
	ux := uid()

	p1 := &data.Project{Name: "Active", OwnerID: o}
	p2 := &data.Project{Name: "Archived", OwnerID: o, Status: data.ProjectStatusArchived}
	_ = projectRepo.Create(context.Background(), p1)
	_ = projectRepo.Create(context.Background(), p2)

	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p1.ID, UserID: ux, Role: data.RoleMember})
	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p2.ID, UserID: ux, Role: data.RoleMember})

	projects, err := projectRepo.FindByMemberID(context.Background(), ux, false, 20, 0)
	if err != nil {
		t.Fatalf("find by member: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("len = %d, want 1", len(projects))
	}
	if projects[0].Name != "Active" {
		t.Errorf("name = %s, want Active", projects[0].Name)
	}
}

func TestMemberAdd_Success(t *testing.T) {
	db := setupTaskDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)

	p := &data.Project{Name: "P", OwnerID: uid()}
	_ = projectRepo.Create(context.Background(), p)

	m := &data.ProjectMember{ProjectID: p.ID, UserID: uid(), Role: data.RoleMember}
	err := memberRepo.Add(context.Background(), m)
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if m.ID == "" {
		t.Error("ID should be set")
	}
}

func TestMemberAdd_Duplicate(t *testing.T) {
	db := setupTaskDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)

	p := &data.Project{Name: "P", OwnerID: uid()}
	_ = projectRepo.Create(context.Background(), p)

	u := uid()
	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p.ID, UserID: u, Role: data.RoleMember})
	err := memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p.ID, UserID: u, Role: data.RoleMember})
	if err == nil {
		t.Error("expected duplicate key error")
	}
}

func TestMemberFindByProjectAndUser(t *testing.T) {
	db := setupTaskDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)

	p := &data.Project{Name: "P", OwnerID: uid()}
	_ = projectRepo.Create(context.Background(), p)
	u := uid()
	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p.ID, UserID: u, Role: data.RoleMember})

	m, err := memberRepo.FindByProjectAndUser(context.Background(), p.ID, u)
	if err != nil {
		t.Fatalf("find member: %v", err)
	}
	if m == nil {
		t.Fatal("expected member, got nil")
	}
	if m.Role != data.RoleMember {
		t.Errorf("role = %d, want %d", m.Role, data.RoleMember)
	}
}

func TestMemberFindByProjectAndUser_NotFound(t *testing.T) {
	db := setupTaskDB(t)
	memberRepo := data.NewMemberRepository(db)

	nonExistentID := "00000000-0000-0000-0000-000000000001"
	m, err := memberRepo.FindByProjectAndUser(context.Background(), nonExistentID, nonExistentID)
	if err != nil {
		t.Fatalf("find member: %v", err)
	}
	if m != nil {
		t.Error("expected nil for non-existent member")
	}
}

func TestMemberListByProject(t *testing.T) {
	db := setupTaskDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)

	p := &data.Project{Name: "P", OwnerID: uid()}
	_ = projectRepo.Create(context.Background(), p)
	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p.ID, UserID: uid(), Role: data.RoleOwner})
	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p.ID, UserID: uid(), Role: data.RoleMember})

	members, err := memberRepo.ListByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("len = %d, want 2", len(members))
	}
}

func TestMemberRemove_Success(t *testing.T) {
	db := setupTaskDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)

	p := &data.Project{Name: "P", OwnerID: uid()}
	_ = projectRepo.Create(context.Background(), p)
	u := uid()
	_ = memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p.ID, UserID: u, Role: data.RoleMember})

	if err := memberRepo.Remove(context.Background(), p.ID, u); err != nil {
		t.Fatalf("remove: %v", err)
	}

	m, _ := memberRepo.FindByProjectAndUser(context.Background(), p.ID, u)
	if m != nil {
		t.Error("expected nil after remove")
	}
}

func TestMemberRemove_NotFound(t *testing.T) {
	db := setupTaskDB(t)
	memberRepo := data.NewMemberRepository(db)

	nonExistentID := "00000000-0000-0000-0000-000000000002"
	err := memberRepo.Remove(context.Background(), nonExistentID, nonExistentID)
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestMemberUpdateRole_Success(t *testing.T) {
	db := setupTaskDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)

	p := &data.Project{Name: "P", OwnerID: uid()}
	if err := projectRepo.Create(context.Background(), p); err != nil {
		t.Fatalf("create project: %v", err)
	}
	u := uid()
	if err := memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p.ID, UserID: u, Role: data.RoleMember}); err != nil {
		t.Fatalf("add member: %v", err)
	}

	m, err := memberRepo.UpdateRole(context.Background(), p.ID, u, data.RoleAdmin)
	if err != nil {
		t.Fatalf("update role: %v", err)
	}
	if m.Role != data.RoleAdmin {
		t.Errorf("role = %d, want %d", m.Role, data.RoleAdmin)
	}
}

func TestMemberFindOwnerByProject(t *testing.T) {
	db := setupTaskDB(t)
	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)

	o := uid()
	p := &data.Project{Name: "P", OwnerID: o}
	if err := projectRepo.Create(context.Background(), p); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if p.ID == "" {
		t.Fatal("project ID not set after create")
	}

	if err := memberRepo.Add(context.Background(), &data.ProjectMember{ProjectID: p.ID, UserID: o, Role: data.RoleOwner}); err != nil {
		t.Fatalf("add owner member: %v", err)
	}

	m, err := memberRepo.FindOwnerByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("find owner: %v", err)
	}
	if m == nil {
		t.Fatal("expected owner member")
	}
	if m.Role != data.RoleOwner {
		t.Errorf("role = %d", m.Role)
	}
}
