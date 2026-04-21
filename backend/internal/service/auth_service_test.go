package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/types"
)

func newTestAuthService() (*AuthService, *mock.UserRepo, *mock.CrewRepo) {
	userRepo := mock.NewUserRepo()
	crewRepo := mock.NewCrewRepo()
	jwtMgr := jwt.NewManager("test-secret-key-that-is-at-least-32-chars-long!", 15, 7)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewAuthService(userRepo, crewRepo, jwtMgr, nil, logger), userRepo, crewRepo
}

func TestRegisterCrewUser(t *testing.T) {
	svc, _, _ := newTestAuthService()
	ctx := context.Background()

	result, err := svc.Register(ctx, RegisterInput{
		Phone:      "+254712345678",
		Password:   "SecurePass123!",
		Role:       types.RoleCrewUser,
		FirstName:  "John",
		LastName:   "Kamau",
		NationalID: "12345678",
		CrewRole:   models.RoleDriver,
	})

	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if result.User == nil {
		t.Fatal("User is nil")
	}
	if result.User.Phone != "+254712345678" {
		t.Errorf("Phone = %q, want +254712345678", result.User.Phone)
	}
	if result.User.SystemRole != types.RoleCrewUser {
		t.Errorf("Role = %q, want CREW", result.User.SystemRole)
	}
	if result.User.PasswordHash == "SecurePass123!" {
		t.Error("PasswordHash should NOT be plaintext")
	}
	if result.CrewMember == nil {
		t.Fatal("CrewMember should be created for CREW role")
	}
	if result.CrewMember.CrewID != "CRW-00001" {
		t.Errorf("CrewID = %q, want CRW-00001", result.CrewMember.CrewID)
	}
	if result.Tokens == nil || result.Tokens.AccessToken == "" {
		t.Error("Tokens should be generated")
	}
}

func TestRegisterSaccoAdmin(t *testing.T) {
	svc, _, _ := newTestAuthService()
	ctx := context.Background()

	result, err := svc.Register(ctx, RegisterInput{
		Phone:    "+254700000001",
		Password: "SecurePass123!",
		Role:     types.RoleSaccoAdmin,
	})

	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if result.CrewMember != nil {
		t.Error("SACCO_ADMIN should NOT have a crew member profile")
	}
}

func TestRegisterDuplicatePhone(t *testing.T) {
	svc, _, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{
		Phone:      "+254712345678",
		Password:   "SecurePass123!",
		Role:       types.RoleCrewUser,
		FirstName:  "John",
		LastName:   "Kamau",
		NationalID: "12345678",
		CrewRole:   models.RoleDriver,
	})
	if err != nil {
		t.Fatalf("first Register: %v", err)
	}

	_, err = svc.Register(ctx, RegisterInput{
		Phone:      "+254712345678",
		Password:   "AnotherPass123!",
		Role:       types.RoleCrewUser,
		FirstName:  "Jane",
		LastName:   "Wanjiku",
		NationalID: "87654321",
		CrewRole:   models.RoleConductor,
	})
	if err != ErrPhoneAlreadyExists {
		t.Errorf("expected ErrPhoneAlreadyExists, got %v", err)
	}
}

func TestRegisterInvalidRole(t *testing.T) {
	svc, _, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{
		Phone:    "+254700000002",
		Password: "SecurePass123!",
		Role:     "INVALID_ROLE",
	})
	if err == nil {
		t.Error("expected error for invalid role")
	}
}

func TestLoginSuccess(t *testing.T) {
	svc, _, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{
		Phone:      "+254712345678",
		Password:   "SecurePass123!",
		Role:       types.RoleCrewUser,
		FirstName:  "John",
		LastName:   "Kamau",
		NationalID: "12345678",
		CrewRole:   models.RoleDriver,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	result, err := svc.Login(ctx, LoginInput{
		Phone:    "+254712345678",
		Password: "SecurePass123!",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if result.User == nil {
		t.Fatal("User is nil")
	}
	if result.Tokens == nil || result.Tokens.AccessToken == "" {
		t.Error("Tokens should be generated on login")
	}
	if result.User.LastLoginAt == nil {
		t.Error("LastLoginAt should be set after login")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc, _, _ := newTestAuthService()
	ctx := context.Background()

	_, _ = svc.Register(ctx, RegisterInput{
		Phone:      "+254712345678",
		Password:   "SecurePass123!",
		Role:       types.RoleCrewUser,
		FirstName:  "John",
		LastName:   "Kamau",
		NationalID: "12345678",
		CrewRole:   models.RoleDriver,
	})

	_, err := svc.Login(ctx, LoginInput{
		Phone:    "+254712345678",
		Password: "WrongPassword!",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginNonExistentUser(t *testing.T) {
	svc, _, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.Login(ctx, LoginInput{
		Phone:    "+254700000099",
		Password: "Whatever123!",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginDisabledAccount(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, RegisterInput{
		Phone:      "+254712345678",
		Password:   "SecurePass123!",
		Role:       types.RoleSaccoAdmin,
	})

	// Disable the account
	result.User.IsActive = false
	_ = userRepo.Update(ctx, result.User)

	_, err := svc.Login(ctx, LoginInput{
		Phone:    "+254712345678",
		Password: "SecurePass123!",
	})
	if err != ErrAccountDisabled {
		t.Errorf("expected ErrAccountDisabled, got %v", err)
	}
}

func TestRefreshToken(t *testing.T) {
	svc, _, _ := newTestAuthService()
	ctx := context.Background()

	reg, _ := svc.Register(ctx, RegisterInput{
		Phone:      "+254712345678",
		Password:   "SecurePass123!",
		Role:       types.RoleSaccoAdmin,
	})

	newTokens, err := svc.RefreshToken(ctx, reg.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if newTokens.AccessToken == "" {
		t.Error("new AccessToken should not be empty")
	}
	if newTokens.AccessToken == reg.Tokens.AccessToken {
		t.Error("new AccessToken should differ from the original")
	}
}
