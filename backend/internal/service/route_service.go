package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// RouteService handles route business logic.
type RouteService struct {
	routeRepo repository.RouteRepository
	logger    *slog.Logger
}

func NewRouteService(routeRepo repository.RouteRepository, logger *slog.Logger) *RouteService {
	return &RouteService{routeRepo: routeRepo, logger: logger}
}

type CreateRouteInput struct {
	Name                string  `json:"name" binding:"required"`
	StartPoint          string  `json:"start_point" binding:"required"`
	EndPoint            string  `json:"end_point" binding:"required"`
	EstimatedDistanceKm float64 `json:"estimated_distance_km"`
	BaseFareCents       int64   `json:"base_fare_cents"`
}

func (s *RouteService) CreateRoute(ctx context.Context, input CreateRouteInput) (*models.Route, error) {
	route := &models.Route{
		Name:                input.Name,
		StartPoint:          input.StartPoint,
		EndPoint:            input.EndPoint,
		EstimatedDistanceKm: input.EstimatedDistanceKm,
		BaseFareCents:       input.BaseFareCents,
		IsActive:            true,
	}

	if err := s.routeRepo.Create(ctx, route); err != nil {
		return nil, fmt.Errorf("create route: %w", err)
	}

	s.logger.Info("route created", slog.String("id", route.ID.String()), slog.String("name", route.Name))
	return route, nil
}

func (s *RouteService) GetRoute(ctx context.Context, id uuid.UUID) (*models.Route, error) {
	return s.routeRepo.GetByID(ctx, id)
}

type UpdateRouteInput struct {
	Name                *string  `json:"name"`
	StartPoint          *string  `json:"start_point"`
	EndPoint            *string  `json:"end_point"`
	EstimatedDistanceKm *float64 `json:"estimated_distance_km"`
	BaseFareCents       *int64   `json:"base_fare_cents"`
	IsActive            *bool    `json:"is_active"`
}

func (s *RouteService) UpdateRoute(ctx context.Context, id uuid.UUID, input UpdateRouteInput) (*models.Route, error) {
	route, err := s.routeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		route.Name = *input.Name
	}
	if input.StartPoint != nil {
		route.StartPoint = *input.StartPoint
	}
	if input.EndPoint != nil {
		route.EndPoint = *input.EndPoint
	}
	if input.EstimatedDistanceKm != nil {
		route.EstimatedDistanceKm = *input.EstimatedDistanceKm
	}
	if input.BaseFareCents != nil {
		route.BaseFareCents = *input.BaseFareCents
	}
	if input.IsActive != nil {
		route.IsActive = *input.IsActive
	}

	if err := s.routeRepo.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("update route: %w", err)
	}
	return route, nil
}

func (s *RouteService) DeleteRoute(ctx context.Context, id uuid.UUID) error {
	return s.routeRepo.Delete(ctx, id)
}

func (s *RouteService) ListRoutes(ctx context.Context, page, perPage int, search string) ([]models.Route, int64, error) {
	return s.routeRepo.List(ctx, page, perPage, search)
}
