package service_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
)

func TestSACCOService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	saccoRepo := mock.NewSACCORepo()
	membershipRepo := mock.NewMembershipRepo()
	floatRepo := mock.NewSACCOFloatRepo()

	auditRepo := mock.NewAuditRepo()
	auditSvc := service.NewAuditService(auditRepo, logger)

	svc := service.NewSACCOService(saccoRepo, membershipRepo, floatRepo, auditSvc, logger)

	t.Run("Create and Get SACCO", func(t *testing.T) {
		ctx := context.Background()
		sacco, err := svc.CreateSACCO(ctx, service.CreateSACCOInput{
			Name:               "Test SACCO",
			RegistrationNumber: "REG123",
			County:             "Nairobi",
			ContactPhone:       "+254700000000",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sacco.ID == uuid.Nil {
			t.Errorf("expected valid UUID")
		}
		if sacco.Name != "Test SACCO" {
			t.Errorf("expected Name to be Test SACCO, got %v", sacco.Name)
		}

		fetched, err := svc.GetSACCO(ctx, sacco.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fetched.Name != sacco.Name {
			t.Errorf("expected fetched sacco to match")
		}
	})

	t.Run("Add Member", func(t *testing.T) {
		ctx := context.Background()
		saccoID := uuid.New()
		crewID := uuid.New()

		m, err := svc.AddMember(ctx, service.AddMemberInput{
			CrewMemberID: crewID,
			SaccoID:      saccoID,
			Role:         models.SACCORoleMember,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.CrewMemberID != crewID {
			t.Errorf("expected matching crew ID")
		}

		// Add again should fail
		_, err = svc.AddMember(ctx, service.AddMemberInput{
			CrewMemberID: crewID,
			SaccoID:      saccoID,
			Role:         models.SACCORoleMember,
		})
		if err == nil {
			t.Errorf("expected error when adding duplicate active member")
		}
	})

	t.Run("Float Operations", func(t *testing.T) {
		ctx := context.Background()
		saccoID := uuid.New()

		// Credit
		tx, err := svc.CreditFloat(ctx, service.FloatOperationInput{
			SaccoID:        saccoID,
			AmountCents:    1000,
			IdempotencyKey: "credit-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.BalanceAfterCents != 1000 {
			t.Errorf("expected balance to be 1000, got %d", tx.BalanceAfterCents)
		}

		// Debit
		tx2, err := svc.DebitFloat(ctx, service.FloatOperationInput{
			SaccoID:        saccoID,
			AmountCents:    400,
			IdempotencyKey: "debit-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx2.BalanceAfterCents != 600 {
			t.Errorf("expected balance to be 600, got %d", tx2.BalanceAfterCents)
		}

		// Debit too much
		_, err = svc.DebitFloat(ctx, service.FloatOperationInput{
			SaccoID:        saccoID,
			AmountCents:    1000,
			IdempotencyKey: "debit-2",
		})
		if err == nil {
			t.Errorf("expected error for insufficient balance")
		}
	})
}
