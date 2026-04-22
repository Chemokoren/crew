package main

import (
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/config"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/pkg/types"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("running database seeder...")
	if err := SeedDatabase(db); err != nil {
		slog.Error("seeding failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("seeding completed successfully!")
}

// SeedDatabase inserts default administrative and testing data.
// It is perfectly idempotent and uses FirstOrCreate.
func SeedDatabase(db *gorm.DB) error {

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	passwordHash := string(hash)

	// 1. Create System Admin
	adminID := uuid.New()
	adminUser := models.User{
		ID:           adminID,
		Phone:        "+254700000000",
		Email:        "admin@amy.com",
		PasswordHash: passwordHash,
		SystemRole:   types.RoleSystemAdmin,
		IsActive:     true,
	}
	if err := db.FirstOrCreate(&adminUser, models.User{Phone: adminUser.Phone}).Error; err != nil {
		return err
	}

	// 2. Create SACCO
	saccoID := uuid.New()
	sacco := models.SACCO{
		ID:                 saccoID,
		Name:               "AMY SACCO LTD",
		RegistrationNumber: "REG-AMY-1234",
		County:             "Nairobi",
		ContactPhone:       "+254711000000",
		Currency:           "KES",
		IsActive:           true,
	}
	if err := db.FirstOrCreate(&sacco, models.SACCO{RegistrationNumber: sacco.RegistrationNumber}).Error; err != nil {
		return err
	}

	// Create SACCO Admin
	saccoAdminUser := models.User{
		ID:           uuid.New(),
		Phone:        "+254711111111",
		Email:        "sacco_admin@amy.com",
		PasswordHash: passwordHash,
		SystemRole:   types.RoleSaccoAdmin,
		SaccoID:      &sacco.ID,
		IsActive:     true,
	}
	if err := db.FirstOrCreate(&saccoAdminUser, models.User{Phone: saccoAdminUser.Phone}).Error; err != nil {
		return err
	}

	// 3. Create Route
	routeID := uuid.New()
	route := models.Route{
		ID:                  routeID,
		Name:                "CBD - KILIMANI",
		StartPoint:          "CBD",
		EndPoint:            "Kilimani",
		EstimatedDistanceKm: 15.5,
		BaseFareCents:       10000, // 100 KES
		IsActive:            true,
	}
	if err := db.FirstOrCreate(&route, models.Route{Name: route.Name}).Error; err != nil {
		return err
	}

	// 4. Create Vehicle
	vehicleID := uuid.New()
	vehicle := models.Vehicle{
		ID:             vehicleID,
		SaccoID:        sacco.ID,
		RegistrationNo: "KCX 123A",
		VehicleType:    models.VehicleMatatu,
		RouteID:        &route.ID,
		Capacity:       14,
		IsActive:       true,
	}
	if err := db.FirstOrCreate(&vehicle, models.Vehicle{RegistrationNo: vehicle.RegistrationNo}).Error; err != nil {
		return err
	}

	// 5. Create Crew Member
	crewID := uuid.New()
	crew := models.CrewMember{
		ID:         crewID,
		CrewID:     "CRW-0001",
		NationalID: "12345678",
		FirstName:  "John",
		LastName:   "Doe",
		KYCStatus:  models.KYCVerified,
		Role:       models.RoleDriver,
		IsActive:   true,
	}
	if err := db.FirstOrCreate(&crew, models.CrewMember{CrewID: crew.CrewID}).Error; err != nil {
		return err
	}

	// Create Crew User Login
	crewUser := models.User{
		ID:           uuid.New(),
		Phone:        "+254722000000",
		Email:        "john.doe@amy.com",
		PasswordHash: passwordHash,
		SystemRole:   types.RoleCrewUser,
		CrewMemberID: &crew.ID,
		IsActive:     true,
	}
	if err := db.FirstOrCreate(&crewUser, models.User{Phone: crewUser.Phone}).Error; err != nil {
		return err
	}

	// 6. Create Crew Wallet
	wallet := models.Wallet{
		ID:           uuid.New(),
		CrewMemberID: crew.ID,
		BalanceCents: 50000, // 500 KES
		Currency:     "KES",
		Version:      1,
		IsActive:     true,
	}
	if err := db.FirstOrCreate(&wallet, models.Wallet{CrewMemberID: wallet.CrewMemberID}).Error; err != nil {
		return err
	}

	return nil
}
