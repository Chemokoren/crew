package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/pkg/jwt"
)

// TestCrewService_CreateWithJobType validates that crew members can be created
// with a tenant-specific job type, ensuring the generalized worker taxonomy works.
func TestCrewService_CreateWithJobType(t *testing.T) {
	crewRepo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := NewCrewService(crewRepo, nil, nil, logger)
	ctx := context.Background()

	jtID := uuid.New()
	crew, err := svc.CreateCrewMember(ctx, CreateCrewInput{
		NationalID: "12345678",
		FirstName:  "James",
		LastName:   "Mwangi",
		Role:       models.RoleOther,
		JobTypeID:  &jtID,
		JobTitle:   "Mason",
	})
	if err != nil {
		t.Fatalf("CreateCrewMember with job type failed: %v", err)
	}
	if crew.JobTypeID == nil || *crew.JobTypeID != jtID {
		t.Errorf("expected job_type_id %s, got %v", jtID, crew.JobTypeID)
	}
	if crew.JobTitle != "Mason" {
		t.Errorf("expected job_title Mason, got %s", crew.JobTitle)
	}
}

// TestCrewService_CreateWithoutJobType verifies backward compatibility —
// crew members can still be created without a job type (transport default).
func TestCrewService_CreateWithoutJobType(t *testing.T) {
	crewRepo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := NewCrewService(crewRepo, nil, nil, logger)
	ctx := context.Background()

	crew, err := svc.CreateCrewMember(ctx, CreateCrewInput{
		NationalID: "87654321",
		FirstName:  "John",
		LastName:   "Kamau",
		Role:       models.RoleDriver,
	})
	if err != nil {
		t.Fatalf("CreateCrewMember without job type failed: %v", err)
	}
	if crew.JobTypeID != nil {
		t.Errorf("expected nil job_type_id, got %v", crew.JobTypeID)
	}
	if crew.JobTitle != "" {
		t.Errorf("expected empty job_title, got %q", crew.JobTitle)
	}
}

// TestCrewService_BulkImportWithJobType validates that bulk import passes job type through.
func TestCrewService_BulkImportWithJobType(t *testing.T) {
	crewRepo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := NewCrewService(crewRepo, nil, nil, logger)
	ctx := context.Background()

	jtID := uuid.New()
	result, err := svc.BulkImport(ctx, []CreateCrewInput{
		{NationalID: "11111111", FirstName: "Alice", LastName: "Wanjiru", Role: models.RoleOther, JobTypeID: &jtID, JobTitle: "Plumber"},
		{NationalID: "22222222", FirstName: "Bob", LastName: "Ochieng", Role: models.RoleOther, JobTypeID: &jtID, JobTitle: "Electrician"},
		{NationalID: "33333333", FirstName: "Carol", LastName: "Muthoni", Role: models.RoleDriver}, // No job type — backward compat
	})
	if err != nil {
		t.Fatalf("BulkImport with job types failed: %v", err)
	}
	if result.Imported != 3 {
		t.Errorf("expected 3 imported, got %d", result.Imported)
	}
}

// TestAuthService_RegisterWithJobType validates that crew users can register
// with a tenant-specific job type during account creation.
func TestAuthService_RegisterWithJobType(t *testing.T) {
	userRepo := mock.NewUserRepo()
	crewRepo := mock.NewCrewRepo()
	jwtMgr := jwt.NewManager("test-secret-key-that-is-at-least-32-chars-long!", 15, 7)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	authSvc := NewAuthService(userRepo, crewRepo, jwtMgr, nil, logger)
	ctx := context.Background()

	jtID := uuid.New()
	result, err := authSvc.Register(ctx, RegisterInput{
		Phone:     "+254700100200",
		Password:  "securepass123",
		Role:      "EMPLOYEE",
		FirstName: "Grace",
		LastName:  "Akinyi",
		JobTypeID: &jtID,
	})
	if err != nil {
		t.Fatalf("Register with job type failed: %v", err)
	}
	if result.CrewMember == nil {
		t.Fatal("expected crew member to be created")
	}
	if result.CrewMember.JobTypeID == nil || *result.CrewMember.JobTypeID != jtID {
		t.Errorf("expected job_type_id %s, got %v", jtID, result.CrewMember.JobTypeID)
	}
}
