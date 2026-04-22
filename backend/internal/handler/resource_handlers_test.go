package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/types"
)

func TestSACCOHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditSvc := service.NewAuditService(mock.NewAuditRepo(), logger)
	saccoSvc := service.NewSACCOService(mock.NewSACCORepo(), mock.NewMembershipRepo(), mock.NewSACCOFloatRepo(), auditSvc, logger)
	saccoHandler := NewSACCOHandler(saccoSvc)

	router.POST("/saccos", mockAuthMiddleware(types.RoleSystemAdmin, nil), saccoHandler.Create)

	reqBody := service.CreateSACCOInput{
		Name:            "Test SACCO",
		RegistrationNumber: "SAC-001",
		ContactEmail:       "test@sacco.com",
		ContactPhone:       "0700000000",
		County:             "Nairobi",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/saccos", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestVehicleHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	vehicleSvc := service.NewVehicleService(mock.NewVehicleRepo(), logger)
	vehicleHandler := NewVehicleHandler(vehicleSvc)

	router.POST("/vehicles", mockAuthMiddleware(types.RoleSystemAdmin, nil), vehicleHandler.Create)

	reqBody := service.CreateVehicleInput{
		RegistrationNo: "KAA 123A",
		VehicleType:    models.VehicleMatatu,
		Capacity:       14,
		SaccoID:        uuid.New(),
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/vehicles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestRouteHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	routeSvc := service.NewRouteService(mock.NewRouteRepo(), logger)
	routeHandler := NewRouteHandler(routeSvc)

	router.POST("/routes", mockAuthMiddleware(types.RoleSystemAdmin, nil), routeHandler.Create)

	reqBody := service.CreateRouteInput{
		Name:                "CBD - Westlands",
		StartPoint:          "CBD",
		EndPoint:            "Westlands",
		EstimatedDistanceKm: 5.5,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/routes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}
