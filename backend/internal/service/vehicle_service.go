package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// VehicleService handles vehicle business logic.
type VehicleService struct {
	vehicleRepo repository.VehicleRepository
	logger      *slog.Logger
}

func NewVehicleService(vehicleRepo repository.VehicleRepository, logger *slog.Logger) *VehicleService {
	return &VehicleService{vehicleRepo: vehicleRepo, logger: logger}
}

type CreateVehicleInput struct {
	SaccoID        uuid.UUID          `json:"sacco_id" binding:"required"`
	RegistrationNo string             `json:"registration_no" binding:"required"`
	VehicleType    models.VehicleType `json:"vehicle_type" binding:"required"`
	RouteID        *uuid.UUID         `json:"route_id"`
	Capacity       int                `json:"capacity"`
}

func (s *VehicleService) CreateVehicle(ctx context.Context, input CreateVehicleInput) (*models.Vehicle, error) {
	vehicle := &models.Vehicle{
		SaccoID:        input.SaccoID,
		RegistrationNo: input.RegistrationNo,
		VehicleType:    input.VehicleType,
		RouteID:        input.RouteID,
		Capacity:       input.Capacity,
		IsActive:       true,
	}

	if err := s.vehicleRepo.Create(ctx, vehicle); err != nil {
		return nil, fmt.Errorf("create vehicle: %w", err)
	}

	s.logger.Info("vehicle created",
		slog.String("id", vehicle.ID.String()),
		slog.String("reg", vehicle.RegistrationNo),
	)
	return vehicle, nil
}

func (s *VehicleService) GetVehicle(ctx context.Context, id uuid.UUID) (*models.Vehicle, error) {
	return s.vehicleRepo.GetByID(ctx, id)
}

type UpdateVehicleInput struct {
	VehicleType *models.VehicleType `json:"vehicle_type"`
	RouteID     *uuid.UUID          `json:"route_id"`
	Capacity    *int                `json:"capacity"`
	IsActive    *bool               `json:"is_active"`
}

func (s *VehicleService) UpdateVehicle(ctx context.Context, id uuid.UUID, input UpdateVehicleInput) (*models.Vehicle, error) {
	vehicle, err := s.vehicleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.VehicleType != nil {
		vehicle.VehicleType = *input.VehicleType
	}
	if input.RouteID != nil {
		vehicle.RouteID = input.RouteID
	}
	if input.Capacity != nil {
		vehicle.Capacity = *input.Capacity
	}
	if input.IsActive != nil {
		vehicle.IsActive = *input.IsActive
	}

	if err := s.vehicleRepo.Update(ctx, vehicle); err != nil {
		return nil, fmt.Errorf("update vehicle: %w", err)
	}
	return vehicle, nil
}

func (s *VehicleService) DeleteVehicle(ctx context.Context, id uuid.UUID) error {
	return s.vehicleRepo.Delete(ctx, id)
}

func (s *VehicleService) ListVehicles(ctx context.Context, saccoID *uuid.UUID, page, perPage int) ([]models.Vehicle, int64, error) {
	return s.vehicleRepo.List(ctx, saccoID, page, perPage)
}
