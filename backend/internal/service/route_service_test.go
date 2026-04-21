package service_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
)

func TestRouteService_CreateRoute(t *testing.T) {
	repo := mock.NewRouteRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewRouteService(repo, logger)

	input := service.CreateRouteInput{
		Name:                "Nairobi - Nakuru",
		StartPoint:          "Nairobi",
		EndPoint:            "Nakuru",
		EstimatedDistanceKm: 160.5,
		BaseFareCents:       12000,
	}

	route, err := svc.CreateRoute(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if route.Name != "Nairobi - Nakuru" {
		t.Errorf("expected Nairobi - Nakuru, got %s", route.Name)
	}
	if route.IsActive != true {
		t.Errorf("expected route to be active")
	}
}

func TestRouteService_UpdateRoute(t *testing.T) {
	repo := mock.NewRouteRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewRouteService(repo, logger)

	route, _ := svc.CreateRoute(context.Background(), service.CreateRouteInput{
		Name:       "R1",
		StartPoint: "A",
		EndPoint:   "B",
	})

	newName := "Nairobi - Naivasha"
	updated, err := svc.UpdateRoute(context.Background(), route.ID, service.UpdateRouteInput{
		Name: &newName,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected %s, got %s", newName, updated.Name)
	}
}

func TestRouteService_ListRoutes(t *testing.T) {
	repo := mock.NewRouteRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewRouteService(repo, logger)

	svc.CreateRoute(context.Background(), service.CreateRouteInput{Name: "R1", StartPoint: "A", EndPoint: "B"})
	svc.CreateRoute(context.Background(), service.CreateRouteInput{Name: "R2", StartPoint: "C", EndPoint: "D"})

	routes, total, err := svc.ListRoutes(context.Background(), 1, 10, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 routes, got %d", total)
	}
	if len(routes) != 2 {
		t.Errorf("expected length 2, got %d", len(routes))
	}
}
