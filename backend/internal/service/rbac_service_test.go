package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	mockRepo "github.com/kibsoft/amy-mis/internal/repository/mock"
)

func setupRBACTest() (*RBACService, *mockRepo.RBACRepo, *mockRepo.AuditRepo) {
	rbacRepo := mockRepo.NewRBACRepo()
	auditRepo := mockRepo.NewAuditRepo()
	auditSvc := NewAuditService(auditRepo, nil)
	svc := NewRBACService(rbacRepo, auditSvc, nil) // nil cache = DB-only path
	return svc, rbacRepo, auditRepo
}

func makeRole(name, industry string) *models.Role {
	return &models.Role{
		Name:         name,
		Description:  "Test role: " + name,
		IndustryType: industry,
		IsActive:     true,
	}
}

func TestCreateRole(t *testing.T) {
	svc, _, auditRepo := setupRBACTest()
	ctx := context.Background()

	role := makeRole("Test Admin", "TRANSPORT")
	err := svc.CreateRole(ctx, role)
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if role.ID == uuid.Nil {
		t.Error("expected role to have an ID assigned")
	}
	if role.Slug != "test-admin" {
		t.Errorf("expected slug 'test-admin', got '%s'", role.Slug)
	}
	if !role.IsActive {
		t.Error("expected new role to be active")
	}
	if len(auditRepo.Logs) == 0 {
		t.Error("expected audit log entry for role creation")
	}
}

func TestCreateRole_DuplicateSlug(t *testing.T) {
	svc, _, _ := setupRBACTest()
	ctx := context.Background()

	_ = svc.CreateRole(ctx, makeRole("Driver", "TRANSPORT"))
	err := svc.CreateRole(ctx, makeRole("Driver", "TRANSPORT"))
	if err == nil {
		t.Error("expected error for duplicate slug, got nil")
	}
}

func TestGetRole(t *testing.T) {
	svc, _, _ := setupRBACTest()
	ctx := context.Background()

	role := makeRole("Conductor", "TRANSPORT")
	_ = svc.CreateRole(ctx, role)

	fetched, err := svc.GetRole(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if fetched.ID != role.ID {
		t.Error("fetched role ID does not match")
	}
}

func TestDeleteRole_SystemProtection(t *testing.T) {
	svc, rbacRepo, _ := setupRBACTest()
	ctx := context.Background()

	sysRole := &models.Role{
		ID:       uuid.New(),
		Name:     "Platform Super Admin",
		Slug:     "platform-super-admin",
		IsSystem: true,
		IsActive: true,
	}
	_ = rbacRepo.CreateRole(ctx, sysRole)

	err := svc.DeleteRole(ctx, sysRole.ID, nil)
	if err == nil {
		t.Error("expected error when deleting system role, got nil")
	}
}

func TestDeleteRole_Success(t *testing.T) {
	svc, _, _ := setupRBACTest()
	ctx := context.Background()

	role := makeRole("Temp Role", "LOGISTICS")
	_ = svc.CreateRole(ctx, role)

	err := svc.DeleteRole(ctx, role.ID, nil)
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}

	_, err = svc.GetRole(ctx, role.ID)
	if err == nil {
		t.Error("expected error after deletion, got nil")
	}
}

func TestCloneRole(t *testing.T) {
	svc, _, _ := setupRBACTest()
	ctx := context.Background()

	original := makeRole("Supervisor", "CONSTRUCTION")
	_ = svc.CreateRole(ctx, original)

	cloned, err := svc.CloneRole(ctx, original.ID, "Senior Supervisor", nil, nil)
	if err != nil {
		t.Fatalf("CloneRole failed: %v", err)
	}
	if cloned.Name != "Senior Supervisor" {
		t.Errorf("expected name 'Senior Supervisor', got '%s'", cloned.Name)
	}
	if cloned.ID == original.ID {
		t.Error("cloned role should have a different ID")
	}
	if cloned.IsSystem {
		t.Error("cloned role should not be a system role")
	}
}

func TestSyncRegistryPermissions(t *testing.T) {
	svc, rbacRepo, _ := setupRBACTest()
	ctx := context.Background()

	err := svc.SyncRegistryPermissions(ctx)
	if err != nil {
		t.Fatalf("SyncRegistryPermissions failed: %v", err)
	}

	perms, _ := rbacRepo.ListPermissions(ctx, repository.PermissionFilter{})
	if len(perms) == 0 {
		t.Error("expected permissions to be synced, got 0")
	}
	t.Logf("synced %d permissions", len(perms))
}

func TestSetAndGetRolePermissions(t *testing.T) {
	svc, rbacRepo, _ := setupRBACTest()
	ctx := context.Background()

	_ = svc.SyncRegistryPermissions(ctx)

	role := makeRole("Test Worker", "TRANSPORT")
	_ = svc.CreateRole(ctx, role)

	// Verify workers.view exists
	matched, _ := rbacRepo.GetPermissionsByKeys(ctx, []string{"workers.view"})
	if len(matched) == 0 {
		t.Fatal("workers.view permission not found after sync")
	}

	err := svc.SetRolePermissions(ctx, role.ID, []string{"workers.view"}, nil)
	if err != nil {
		t.Fatalf("SetRolePermissions failed: %v", err)
	}

	rolePerms, err := svc.GetRolePermissions(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRolePermissions failed: %v", err)
	}
	found := false
	for _, p := range rolePerms {
		if p.Key == "workers.view" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected workers.view in role permissions")
	}
}

func TestAssignAndRevokeRole(t *testing.T) {
	svc, _, _ := setupRBACTest()
	ctx := context.Background()

	role := makeRole("Picker", "AGRICULTURE")
	_ = svc.CreateRole(ctx, role)
	userID := uuid.New()

	err := svc.AssignRole(ctx, userID, role.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}

	roles, err := svc.GetUserRoles(ctx, userID, nil)
	if err != nil {
		t.Fatalf("GetUserRoles failed: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}

	err = svc.RevokeRole(ctx, userID, role.ID, nil, nil)
	if err != nil {
		t.Fatalf("RevokeRole failed: %v", err)
	}

	roles, _ = svc.GetUserRoles(ctx, userID, nil)
	if len(roles) != 0 {
		t.Errorf("expected 0 roles after revocation, got %d", len(roles))
	}
}

func TestSyncTemplates(t *testing.T) {
	svc, rbacRepo, _ := setupRBACTest()
	ctx := context.Background()

	err := svc.SyncTemplates(ctx)
	if err != nil {
		t.Fatalf("SyncTemplates failed: %v", err)
	}

	templates, _ := rbacRepo.ListTemplates(ctx, "")
	if len(templates) == 0 {
		t.Error("expected templates to be synced, got 0")
	}
	t.Logf("synced %d templates", len(templates))
}

func TestHasPermission_DBFallback(t *testing.T) {
	svc, rbacRepo, _ := setupRBACTest()
	ctx := context.Background()

	_ = svc.SyncRegistryPermissions(ctx)

	role := makeRole("Admin", "PLATFORM")
	_ = svc.CreateRole(ctx, role)

	// Verify and assign workers.view
	matched, _ := rbacRepo.GetPermissionsByKeys(ctx, []string{"workers.view"})
	if len(matched) == 0 {
		t.Fatal("workers.view not found")
	}
	_ = svc.SetRolePermissions(ctx, role.ID, []string{"workers.view"}, nil)

	userID := uuid.New()
	_ = svc.AssignRole(ctx, userID, role.ID, nil, nil, nil)

	// No cache — falls back to DB
	has := svc.HasPermission(ctx, userID, nil, "workers.view")
	if !has {
		t.Error("expected user to have workers.view")
	}

	hasOther := svc.HasPermission(ctx, userID, nil, "payroll.run")
	if hasOther {
		t.Error("user should NOT have payroll.run")
	}
}

func TestListRoles_Filtering(t *testing.T) {
	svc, _, _ := setupRBACTest()
	ctx := context.Background()

	_ = svc.CreateRole(ctx, makeRole("Driver", "TRANSPORT"))
	_ = svc.CreateRole(ctx, makeRole("Laborer", "CONSTRUCTION"))
	_ = svc.CreateRole(ctx, makeRole("Rider", "LOGISTICS"))

	roles, total, err := svc.ListRoles(ctx, repository.RoleFilter{IndustryType: "TRANSPORT"}, 1, 10)
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 TRANSPORT role, got %d", total)
	}
	if len(roles) != 1 || roles[0].Name != "Driver" {
		t.Error("expected Driver role in results")
	}
}
