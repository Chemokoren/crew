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

func TestVehicleService_CreateVehicle(t *testing.T) {
	repo := mock.NewVehicleRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewVehicleService(repo, logger)

	saccoID := uuid.New()
	input := service.CreateVehicleInput{
		RegistrationNo: "KAA 123A",
		Capacity:       14,
		VehicleType:    models.VehicleType("MATATU"),
		SaccoID:        saccoID,
	}

	vehicle, err := svc.CreateVehicle(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if vehicle.RegistrationNo != "KAA 123A" {
		t.Errorf("expected KAA 123A, got %s", vehicle.RegistrationNo)
	}
	if vehicle.IsActive != true {
		t.Errorf("expected vehicle to be active")
	}
}

func TestVehicleService_UpdateVehicle(t *testing.T) {
	repo := mock.NewVehicleRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewVehicleService(repo, logger)

	saccoID := uuid.New()
	vehicle, _ := svc.CreateVehicle(context.Background(), service.CreateVehicleInput{
		RegistrationNo: "KAA 123A",
		SaccoID:        saccoID,
	})

	newStatus := false
	updated, err := svc.UpdateVehicle(context.Background(), vehicle.ID, service.UpdateVehicleInput{
		IsActive: &newStatus,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.IsActive != false {
		t.Errorf("expected inactive, got %v", updated.IsActive)
	}
}

func TestVehicleService_ListVehicles(t *testing.T) {
	repo := mock.NewVehicleRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewVehicleService(repo, logger)

	saccoID := uuid.New()
	svc.CreateVehicle(context.Background(), service.CreateVehicleInput{RegistrationNo: "V1", SaccoID: saccoID})
	svc.CreateVehicle(context.Background(), service.CreateVehicleInput{RegistrationNo: "V2", SaccoID: saccoID})

	vehicles, total, err := svc.ListVehicles(context.Background(), &saccoID, 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 vehicles, got %d", total)
	}
	if len(vehicles) != 2 {
		t.Errorf("expected length 2, got %d", len(vehicles))
	}
}

func TestVehicleService_GetVehicle(t *testing.T) {
	repo := mock.NewVehicleRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewVehicleService(repo, logger)

	saccoID := uuid.New()
	vehicle, _ := svc.CreateVehicle(context.Background(), service.CreateVehicleInput{
		RegistrationNo: "KAZ 123Z",
		SaccoID:        saccoID,
	})

	fetched, err := svc.GetVehicle(context.Background(), vehicle.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fetched.RegistrationNo != "KAZ 123Z" {
		t.Errorf("expected KAZ 123Z, got %s", fetched.RegistrationNo)
	}
}

func TestVehicleService_DeleteVehicle(t *testing.T) {
	repo := mock.NewVehicleRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewVehicleService(repo, logger)

	saccoID := uuid.New()
	vehicle, _ := svc.CreateVehicle(context.Background(), service.CreateVehicleInput{
		RegistrationNo: "KAB 123B",
		SaccoID:        saccoID,
	})

	err := svc.DeleteVehicle(context.Background(), vehicle.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = svc.GetVehicle(context.Background(), vehicle.ID)
	if err == nil {
		t.Errorf("expected error when getting deleted vehicle")
	}
}
