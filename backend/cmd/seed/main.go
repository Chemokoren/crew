package main

import (
	"log/slog"
	"os"
	"time"

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
	db, err := database.Connect(cfg.DatabaseURL, cfg.IsDevelopment(), database.PoolConfig{})
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
// It is perfectly idempotent — uses FirstOrCreate keyed on natural unique fields.
// IDs are NOT pre-set so GORM doesn't include them in the WHERE clause.
func SeedDatabase(db *gorm.DB) error {

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	passwordHash := string(hash)

	// ========================================
	// 1. SYSTEM ADMIN — Full platform access
	// ========================================
	adminUser := models.User{
		Phone:        "+254700000000",
		Email:        "admin@amy.com",
		PasswordHash: passwordHash,
		SystemRole:   types.RoleSystemAdmin,
		IsActive:     true,
	}
	if err := db.Where(models.User{Phone: adminUser.Phone}).FirstOrCreate(&adminUser).Error; err != nil {
		return err
	}
	slog.Info("✅ SYSTEM_ADMIN", slog.String("phone", adminUser.Phone), slog.String("password", "password123"))

	// ========================================
	// 2. SACCOs — Organization entities
	// ========================================
	sacco := models.SACCO{
		Name:               "AMY SACCO LTD",
		RegistrationNumber: "REG-AMY-1234",
		County:             "Nairobi",
		ContactPhone:       "+254711000000",
		Currency:           "KES",
		IsActive:           true,
	}
	if err := db.Where(models.SACCO{RegistrationNumber: sacco.RegistrationNumber}).FirstOrCreate(&sacco).Error; err != nil {
		return err
	}

	sacco2 := models.SACCO{
		Name:               "CITY SHUTTLE SACCO",
		RegistrationNumber: "REG-CSH-5678",
		County:             "Mombasa",
		SubCounty:          "Mvita",
		ContactPhone:       "+254733000000",
		ContactEmail:       "info@cityshuttle.co.ke",
		Currency:           "KES",
		IsActive:           true,
	}
	if err := db.Where(models.SACCO{RegistrationNumber: sacco2.RegistrationNumber}).FirstOrCreate(&sacco2).Error; err != nil {
		return err
	}

	// ========================================
	// 3. SACCO ADMIN — Manages SACCO operations
	// ========================================
	saccoAdminUser := models.User{
		Phone:        "+254711111111",
		Email:        "sacco_admin@amy.com",
		PasswordHash: passwordHash,
		SystemRole:   types.RoleSaccoAdmin,
		SaccoID:      &sacco.ID,
		IsActive:     true,
	}
	if err := db.Where(models.User{Phone: saccoAdminUser.Phone}).FirstOrCreate(&saccoAdminUser).Error; err != nil {
		return err
	}
	slog.Info("✅ SACCO_ADMIN", slog.String("phone", saccoAdminUser.Phone), slog.String("password", "password123"), slog.String("sacco", sacco.Name))

	// ========================================
	// 4. ROUTES
	// ========================================
	route := models.Route{
		Name:                "CBD - KILIMANI",
		StartPoint:          "CBD",
		EndPoint:            "Kilimani",
		EstimatedDistanceKm: 15.5,
		BaseFareCents:       10000, // 100 KES
		IsActive:            true,
	}
	if err := db.Where(models.Route{Name: route.Name}).FirstOrCreate(&route).Error; err != nil {
		return err
	}

	route2 := models.Route{
		Name:                "WESTLANDS - KAREN",
		StartPoint:          "Westlands",
		EndPoint:            "Karen",
		EstimatedDistanceKm: 22.0,
		BaseFareCents:       15000, // 150 KES
		IsActive:            true,
	}
	if err := db.Where(models.Route{Name: route2.Name}).FirstOrCreate(&route2).Error; err != nil {
		return err
	}

	// ========================================
	// 5. VEHICLES
	// ========================================
	vehicle := models.Vehicle{
		SaccoID:        sacco.ID,
		RegistrationNo: "KCX 123A",
		VehicleType:    models.VehicleMatatu,
		RouteID:        &route.ID,
		Capacity:       14,
		IsActive:       true,
	}
	if err := db.Where(models.Vehicle{RegistrationNo: vehicle.RegistrationNo}).FirstOrCreate(&vehicle).Error; err != nil {
		return err
	}

	vehicle2 := models.Vehicle{
		SaccoID:        sacco.ID,
		RegistrationNo: "KDG 456B",
		VehicleType:    models.VehicleMatatu,
		RouteID:        &route2.ID,
		Capacity:       33,
		IsActive:       true,
	}
	if err := db.Where(models.Vehicle{RegistrationNo: vehicle2.RegistrationNo}).FirstOrCreate(&vehicle2).Error; err != nil {
		return err
	}

	vehicle3 := models.Vehicle{
		SaccoID:        sacco2.ID,
		RegistrationNo: "KBZ 789C",
		VehicleType:    models.VehicleMatatu,
		RouteID:        &route.ID,
		Capacity:       14,
		IsActive:       true,
	}
	if err := db.Where(models.Vehicle{RegistrationNo: vehicle3.RegistrationNo}).FirstOrCreate(&vehicle3).Error; err != nil {
		return err
	}

	// ========================================
	// 6. CREW MEMBERS — Profile entities
	// ========================================

	// Crew Member 1 — Driver (John Doe)
	crew := models.CrewMember{
		CrewID:     "CRW-0001",
		NationalID: "12345678",
		FirstName:  "John",
		LastName:   "Doe",
		KYCStatus:  models.KYCVerified,
		Role:       models.RoleDriver,
		IsActive:   true,
	}
	if err := db.Where(models.CrewMember{CrewID: crew.CrewID}).FirstOrCreate(&crew).Error; err != nil {
		return err
	}

	// Crew Member 2 — Conductor (Jane Muthoni)
	crew2 := models.CrewMember{
		CrewID:     "CRW-0002",
		NationalID: "23456789",
		FirstName:  "Jane",
		LastName:   "Muthoni",
		KYCStatus:  models.KYCVerified,
		Role:       models.RoleConductor,
		IsActive:   true,
	}
	if err := db.Where(models.CrewMember{CrewID: crew2.CrewID}).FirstOrCreate(&crew2).Error; err != nil {
		return err
	}

	// Crew Member 3 — Rider (Peter Kamau)
	crew3 := models.CrewMember{
		CrewID:     "CRW-0003",
		NationalID: "34567890",
		FirstName:  "Peter",
		LastName:   "Kamau",
		KYCStatus:  models.KYCPending,
		Role:       models.RoleRider,
		IsActive:   true,
	}
	if err := db.Where(models.CrewMember{CrewID: crew3.CrewID}).FirstOrCreate(&crew3).Error; err != nil {
		return err
	}

	// ========================================
	// 7. CREW USERS — Logged-in crew members
	// ========================================
	crewUser := models.User{
		Phone:        "+254722000000",
		Email:        "john.doe@amy.com",
		PasswordHash: passwordHash,
		SystemRole:   types.RoleCrewUser,
		CrewMemberID: &crew.ID,
		IsActive:     true,
	}
	if err := db.Where(models.User{Phone: crewUser.Phone}).FirstOrCreate(&crewUser).Error; err != nil {
		return err
	}
	slog.Info("✅ CREW (Driver)", slog.String("phone", crewUser.Phone), slog.String("password", "password123"), slog.String("name", "John Doe"))

	crewUser2 := models.User{
		Phone:        "+254722111111",
		Email:        "jane.muthoni@amy.com",
		PasswordHash: passwordHash,
		SystemRole:   types.RoleCrewUser,
		CrewMemberID: &crew2.ID,
		IsActive:     true,
	}
	if err := db.Where(models.User{Phone: crewUser2.Phone}).FirstOrCreate(&crewUser2).Error; err != nil {
		return err
	}
	slog.Info("✅ CREW (Conductor)", slog.String("phone", crewUser2.Phone), slog.String("password", "password123"), slog.String("name", "Jane Muthoni"))

	// ========================================
	// 8. LENDER — Financial services partner
	// ========================================
	lenderUser := models.User{
		Phone:        "+254733333333",
		Email:        "lender@amyfinance.com",
		PasswordHash: passwordHash,
		SystemRole:   types.RoleLender,
		IsActive:     true,
	}
	if err := db.Where(models.User{Phone: lenderUser.Phone}).FirstOrCreate(&lenderUser).Error; err != nil {
		return err
	}
	slog.Info("✅ LENDER", slog.String("phone", lenderUser.Phone), slog.String("password", "password123"))

	// ========================================
	// 9. INSURER — Insurance partner
	// ========================================
	insurerUser := models.User{
		Phone:        "+254744444444",
		Email:        "insurer@amycover.com",
		PasswordHash: passwordHash,
		SystemRole:   types.RoleInsurer,
		IsActive:     true,
	}
	if err := db.Where(models.User{Phone: insurerUser.Phone}).FirstOrCreate(&insurerUser).Error; err != nil {
		return err
	}
	slog.Info("✅ INSURER", slog.String("phone", insurerUser.Phone), slog.String("password", "password123"))

	// ========================================
	// 10. SACCO Memberships
	// ========================================
	membership1 := models.CrewSACCOMembership{
		CrewMemberID: crew.ID,
		SaccoID:      sacco.ID,
		RoleInSacco:  models.SACCORoleMember,
		JoinedAt:     time.Now(),
		IsActive:     true,
	}
	if err := db.Where("crew_member_id = ? AND sacco_id = ?", crew.ID, sacco.ID).FirstOrCreate(&membership1).Error; err != nil {
		return err
	}

	membership2 := models.CrewSACCOMembership{
		CrewMemberID: crew2.ID,
		SaccoID:      sacco.ID,
		RoleInSacco:  models.SACCORoleMember,
		JoinedAt:     time.Now(),
		IsActive:     true,
	}
	if err := db.Where("crew_member_id = ? AND sacco_id = ?", crew2.ID, sacco.ID).FirstOrCreate(&membership2).Error; err != nil {
		return err
	}

	membership3 := models.CrewSACCOMembership{
		CrewMemberID: crew3.ID,
		SaccoID:      sacco2.ID,
		RoleInSacco:  models.SACCORoleMember,
		JoinedAt:     time.Now(),
		IsActive:     true,
	}
	if err := db.Where("crew_member_id = ? AND sacco_id = ?", crew3.ID, sacco2.ID).FirstOrCreate(&membership3).Error; err != nil {
		return err
	}

	// ========================================
	// 11. WALLETS
	// ========================================
	wallet := models.Wallet{
		CrewMemberID: crew.ID,
		BalanceCents: 50000, // 500 KES
		Currency:     "KES",
		Version:      1,
		IsActive:     true,
	}
	if err := db.Where(models.Wallet{CrewMemberID: crew.ID}).FirstOrCreate(&wallet).Error; err != nil {
		return err
	}

	wallet2 := models.Wallet{
		CrewMemberID: crew2.ID,
		BalanceCents: 32500, // 325 KES
		Currency:     "KES",
		Version:      1,
		IsActive:     true,
	}
	if err := db.Where(models.Wallet{CrewMemberID: crew2.ID}).FirstOrCreate(&wallet2).Error; err != nil {
		return err
	}

	wallet3 := models.Wallet{
		CrewMemberID: crew3.ID,
		BalanceCents: 12000, // 120 KES
		Currency:     "KES",
		Version:      1,
		IsActive:     true,
	}
	if err := db.Where(models.Wallet{CrewMemberID: crew3.ID}).FirstOrCreate(&wallet3).Error; err != nil {
		return err
	}

	// ========================================
	// 12. SAMPLE ASSIGNMENTS
	// ========================================
	today := time.Now().Truncate(24 * time.Hour)
	shiftStart := today.Add(6 * time.Hour) // 6 AM

	assignment1 := models.Assignment{
		CrewMemberID:     crew.ID,
		VehicleID:        vehicle.ID,
		SaccoID:          sacco.ID,
		RouteID:          &route.ID,
		ShiftDate:        today,
		ShiftStart:       shiftStart,
		Status:           models.AssignmentActive,
		EarningModel:     models.EarningFixed,
		FixedAmountCents: 200000, // 2000 KES
		Notes:            "Morning shift — CBD to Kilimani",
		CreatedByID:      adminUser.ID,
	}
	if err := db.Omit("CommissionBasis").Where("crew_member_id = ? AND vehicle_id = ? AND shift_date = ?",
		crew.ID, vehicle.ID, today).FirstOrCreate(&assignment1).Error; err != nil {
		return err
	}

	assignment2 := models.Assignment{
		CrewMemberID:    crew2.ID,
		VehicleID:       vehicle2.ID,
		SaccoID:         sacco.ID,
		RouteID:         &route2.ID,
		ShiftDate:       today,
		ShiftStart:      shiftStart.Add(1 * time.Hour),
		Status:          models.AssignmentScheduled,
		EarningModel:    models.EarningCommission,
		CommissionRate:  0.15, // 15%
		CommissionBasis: models.CommissionOnRevenue,
		Notes:           "Afternoon shift — Westlands to Karen",
		CreatedByID:     adminUser.ID,
	}
	if err := db.Where("crew_member_id = ? AND vehicle_id = ? AND shift_date = ?",
		crew2.ID, vehicle2.ID, today).FirstOrCreate(&assignment2).Error; err != nil {
		return err
	}

	// ========================================
	// SUMMARY
	// ========================================
	slog.Info("═══════════════════════════════════════════")
	slog.Info("  TEST ACCOUNTS (password: password123)")
	slog.Info("═══════════════════════════════════════════")
	slog.Info("  SYSTEM_ADMIN   +254700000000")
	slog.Info("  SACCO_ADMIN    +254711111111")
	slog.Info("  CREW (Driver)  +254722000000")
	slog.Info("  CREW (Cond.)   +254722111111")
	slog.Info("  LENDER         +254733333333")
	slog.Info("  INSURER        +254744444444")
	slog.Info("═══════════════════════════════════════════")

	return nil
}
