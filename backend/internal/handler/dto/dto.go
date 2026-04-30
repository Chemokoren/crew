// Package dto defines request/response Data Transfer Objects for the API.
// Raw GORM models are NEVER exposed directly — DTOs control exactly what
// enters and leaves the API boundary.
package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/pkg/types"
)

// --- Auth DTOs ---

type RegisterRequest struct {
	Phone      string           `json:"phone" binding:"required"`
	Email      string           `json:"email" binding:"omitempty,email"`
	Password   string           `json:"password" binding:"required,min=8"`
	Role       types.SystemRole `json:"role" binding:"required"`
	FirstName  string           `json:"first_name"`
	LastName   string           `json:"last_name"`
	NationalID string           `json:"national_id"`
	CrewRole   models.CrewRole  `json:"crew_role"`
}

type LoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	User   UserResponse `json:"user"`
	Tokens TokensDTO    `json:"tokens"`
}

type TokensDTO struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// --- User DTOs ---

type UserResponse struct {
	ID           uuid.UUID        `json:"id"`
	Phone        string           `json:"phone"`
	Email        string           `json:"email,omitempty"`
	SystemRole   types.SystemRole `json:"system_role"`
	CrewMemberID *uuid.UUID       `json:"crew_member_id,omitempty"`
	SaccoID      *uuid.UUID       `json:"sacco_id,omitempty"`
	IsActive     bool             `json:"is_active"`
	LastLoginAt  *time.Time       `json:"last_login_at,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
}

func UserToResponse(u *models.User) UserResponse {
	return UserResponse{
		ID:           u.ID,
		Phone:        u.Phone,
		Email:        u.Email,
		SystemRole:   u.SystemRole,
		CrewMemberID: u.CrewMemberID,
		SaccoID:      u.SaccoID,
		IsActive:     u.IsActive,
		LastLoginAt:  u.LastLoginAt,
		CreatedAt:    u.CreatedAt,
	}
}

// --- Crew Member DTOs ---

type CrewMemberResponse struct {
	ID            uuid.UUID        `json:"id"`
	CrewID        string           `json:"crew_id"`
	FirstName     string           `json:"first_name"`
	LastName      string           `json:"last_name"`
	FullName      string           `json:"full_name"`
	Role          models.CrewRole  `json:"role"`
	KYCStatus     models.KYCStatus `json:"kyc_status"`
	KYCVerifiedAt *time.Time       `json:"kyc_verified_at,omitempty"`
	PhotoURL      string           `json:"photo_url,omitempty"`
	IsActive      bool             `json:"is_active"`
	CreatedAt     time.Time        `json:"created_at"`
}

func CrewToResponse(c *models.CrewMember) CrewMemberResponse {
	return CrewMemberResponse{
		ID:            c.ID,
		CrewID:        c.CrewID,
		FirstName:     c.FirstName,
		LastName:      c.LastName,
		FullName:      c.FullName(),
		Role:          c.Role,
		KYCStatus:     c.KYCStatus,
		KYCVerifiedAt: c.KYCVerifiedAt,
		PhotoURL:      c.PhotoURL,
		IsActive:      c.IsActive,
		CreatedAt:     c.CreatedAt,
	}
}

func CrewListToResponse(members []models.CrewMember) []CrewMemberResponse {
	result := make([]CrewMemberResponse, len(members))
	for i, m := range members {
		result[i] = CrewToResponse(&m)
	}
	return result
}

type CreateCrewRequest struct {
	NationalID string          `json:"national_id" binding:"required"`
	FirstName  string          `json:"first_name" binding:"required"`
	LastName   string          `json:"last_name" binding:"required"`
	Role       models.CrewRole `json:"role" binding:"required,oneof=DRIVER CONDUCTOR RIDER OTHER"`
}

type UpdateKYCRequest struct {
	KYCStatus    models.KYCStatus `json:"kyc_status" binding:"required,oneof=PENDING VERIFIED REJECTED"`
	SerialNumber string           `json:"serial_number,omitempty"` // Required for IPRS verification when status is VERIFIED
}

// --- SACCO DTOs ---

type SACCOResponse struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	RegistrationNumber string    `json:"registration_number"`
	County             string    `json:"county"`
	SubCounty          string    `json:"sub_county,omitempty"`
	ContactPhone       string    `json:"contact_phone"`
	ContactEmail       string    `json:"contact_email,omitempty"`
	Currency           string    `json:"currency"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
}

func SACCOToResponse(s *models.SACCO) SACCOResponse {
	return SACCOResponse{
		ID:                 s.ID,
		Name:               s.Name,
		RegistrationNumber: s.RegistrationNumber,
		County:             s.County,
		SubCounty:          s.SubCounty,
		ContactPhone:       s.ContactPhone,
		ContactEmail:       s.ContactEmail,
		Currency:           s.Currency,
		IsActive:           s.IsActive,
		CreatedAt:          s.CreatedAt,
	}
}

func SACCOListToResponse(saccos []models.SACCO) []SACCOResponse {
	result := make([]SACCOResponse, len(saccos))
	for i, s := range saccos {
		result[i] = SACCOToResponse(&s)
	}
	return result
}

type CreateSACCORequest struct {
	Name               string `json:"name" binding:"required"`
	RegistrationNumber string `json:"registration_number" binding:"required"`
	County             string `json:"county" binding:"required"`
	SubCounty          string `json:"sub_county"`
	ContactPhone       string `json:"contact_phone" binding:"required"`
	ContactEmail       string `json:"contact_email" binding:"omitempty,email"`
}

type UpdateSACCORequest struct {
	Name         string `json:"name"`
	County       string `json:"county"`
	SubCounty    string `json:"sub_county"`
	ContactPhone string `json:"contact_phone"`
	ContactEmail string `json:"contact_email" binding:"omitempty,email"`
}

// --- Vehicle DTOs ---

type VehicleResponse struct {
	ID             uuid.UUID          `json:"id"`
	SaccoID        uuid.UUID          `json:"sacco_id"`
	RegistrationNo string             `json:"registration_no"`
	VehicleType    models.VehicleType `json:"vehicle_type"`
	RouteID        *uuid.UUID         `json:"route_id,omitempty"`
	Capacity       int                `json:"capacity"`
	IsActive       bool               `json:"is_active"`
	CreatedAt      time.Time          `json:"created_at"`
}

func VehicleToResponse(v *models.Vehicle) VehicleResponse {
	return VehicleResponse{
		ID:             v.ID,
		SaccoID:        v.SaccoID,
		RegistrationNo: v.RegistrationNo,
		VehicleType:    v.VehicleType,
		RouteID:        v.RouteID,
		Capacity:       v.Capacity,
		IsActive:       v.IsActive,
		CreatedAt:      v.CreatedAt,
	}
}

func VehicleListToResponse(vehicles []models.Vehicle) []VehicleResponse {
	result := make([]VehicleResponse, len(vehicles))
	for i, v := range vehicles {
		result[i] = VehicleToResponse(&v)
	}
	return result
}

type CreateVehicleRequest struct {
	SaccoID        uuid.UUID          `json:"sacco_id" binding:"required"`
	RegistrationNo string             `json:"registration_no" binding:"required"`
	VehicleType    models.VehicleType `json:"vehicle_type" binding:"required,oneof=MATATU BODA TUK_TUK"`
	RouteID        *uuid.UUID         `json:"route_id"`
	Capacity       int                `json:"capacity"`
}

// --- Assignment DTOs ---

type AssignmentResponse struct {
	ID                    uuid.UUID              `json:"id"`
	CrewMemberID          uuid.UUID              `json:"crew_member_id"`
	CrewMemberName        string                 `json:"crew_member_name"`
	VehicleID             uuid.UUID              `json:"vehicle_id"`
	VehicleRegistrationNo string                 `json:"vehicle_registration_no"`
	SaccoID               uuid.UUID              `json:"sacco_id"`
	SaccoName             string                 `json:"sacco_name"`
	RouteID               *uuid.UUID             `json:"route_id,omitempty"`
	RouteName             string                 `json:"route_name,omitempty"`
	ShiftDate             time.Time              `json:"shift_date"`
	ShiftStart            time.Time              `json:"shift_start"`
	ShiftEnd              *time.Time             `json:"shift_end,omitempty"`
	Status                models.AssignmentStatus `json:"status"`
	EarningModel          models.EarningModel    `json:"earning_model"`
	FixedAmountCents      int64                  `json:"fixed_amount_cents,omitempty"`
	CommissionRate        float64                `json:"commission_rate,omitempty"`
	HybridBaseCents       int64                  `json:"hybrid_base_cents,omitempty"`
	CommissionBasis       models.CommissionBasis  `json:"commission_basis,omitempty"`
	Notes                 string                 `json:"notes,omitempty"`
	CreatedByID           uuid.UUID              `json:"created_by_id"`
	CreatedAt             time.Time              `json:"created_at"`
}

func AssignmentToResponse(a *models.Assignment) AssignmentResponse {
	resp := AssignmentResponse{
		ID:               a.ID,
		CrewMemberID:     a.CrewMemberID,
		VehicleID:        a.VehicleID,
		SaccoID:          a.SaccoID,
		RouteID:          a.RouteID,
		ShiftDate:        a.ShiftDate,
		ShiftStart:       a.ShiftStart,
		ShiftEnd:         a.ShiftEnd,
		Status:           a.Status,
		EarningModel:     a.EarningModel,
		FixedAmountCents: a.FixedAmountCents,
		CommissionRate:   a.CommissionRate,
		HybridBaseCents:  a.HybridBaseCents,
		CommissionBasis:  a.CommissionBasis,
		Notes:            a.Notes,
		CreatedByID:      a.CreatedByID,
		CreatedAt:        a.CreatedAt,
	}

	// Resolve human-readable names from preloaded relations
	if a.CrewMember.ID != (uuid.UUID{}) {
		resp.CrewMemberName = a.CrewMember.FullName()
	}
	if a.Vehicle.ID != (uuid.UUID{}) {
		resp.VehicleRegistrationNo = a.Vehicle.RegistrationNo
	}
	if a.Sacco.ID != (uuid.UUID{}) {
		resp.SaccoName = a.Sacco.Name
	}
	if a.Route != nil && a.Route.ID != (uuid.UUID{}) {
		resp.RouteName = a.Route.Name
	}

	return resp
}

func AssignmentListToResponse(assignments []models.Assignment) []AssignmentResponse {
	result := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		result[i] = AssignmentToResponse(&a)
	}
	return result
}

type CreateAssignmentRequest struct {
	CrewMemberID     uuid.UUID              `json:"crew_member_id" binding:"required"`
	VehicleID        uuid.UUID              `json:"vehicle_id" binding:"required"`
	SaccoID          uuid.UUID              `json:"sacco_id" binding:"required"`
	RouteID          *uuid.UUID             `json:"route_id"`
	ShiftDate        string                 `json:"shift_date" binding:"required"`
	ShiftStart       string                 `json:"shift_start" binding:"required"`
	EarningModel     models.EarningModel    `json:"earning_model" binding:"required,oneof=FIXED COMMISSION HYBRID"`
	FixedAmountCents int64                  `json:"fixed_amount_cents"`
	CommissionRate   float64                `json:"commission_rate"`
	HybridBaseCents  int64                  `json:"hybrid_base_cents"`
	CommissionBasis  models.CommissionBasis `json:"commission_basis"`
	Notes            string                 `json:"notes"`
}

type CompleteAssignmentRequest struct {
	TotalRevenueCents int64 `json:"total_revenue_cents" binding:"required,min=0"`
}

// --- Wallet DTOs ---

type WalletResponse struct {
	ID                 uuid.UUID  `json:"id"`
	CrewMemberID       uuid.UUID  `json:"crew_member_id"`
	BalanceCents       int64      `json:"balance_cents"`
	BalanceFormatted   string     `json:"balance_formatted"`
	TotalCreditedCents int64      `json:"total_credited_cents"`
	TotalDebitedCents  int64      `json:"total_debited_cents"`
	Currency           string     `json:"currency"`
	IsActive           bool       `json:"is_active"`
	LastPayoutAt       *time.Time `json:"last_payout_at,omitempty"`
}

type WalletTransactionResponse struct {
	ID                uuid.UUID                   `json:"id"`
	TransactionType   models.TransactionType       `json:"transaction_type"`
	Category          models.TransactionCategory   `json:"category"`
	AmountCents       int64                        `json:"amount_cents"`
	BalanceAfterCents int64                        `json:"balance_after_cents"`
	Currency          string                       `json:"currency"`
	Reference         string                       `json:"reference,omitempty"`
	Description       string                       `json:"description,omitempty"`
	Status            models.TransactionStatus     `json:"status"`
	CreatedAt         time.Time                    `json:"created_at"`
}

func WalletTxToResponse(tx *models.WalletTransaction) WalletTransactionResponse {
	return WalletTransactionResponse{
		ID:                tx.ID,
		TransactionType:   tx.TransactionType,
		Category:          tx.Category,
		AmountCents:       tx.AmountCents,
		BalanceAfterCents: tx.BalanceAfterCents,
		Currency:          tx.Currency,
		Reference:         tx.Reference,
		Description:       tx.Description,
		Status:            tx.Status,
		CreatedAt:         tx.CreatedAt,
	}
}

func WalletTxListToResponse(txs []models.WalletTransaction) []WalletTransactionResponse {
	result := make([]WalletTransactionResponse, len(txs))
	for i, tx := range txs {
		result[i] = WalletTxToResponse(&tx)
	}
	return result
}

type CreditWalletRequest struct {
	CrewMemberID uuid.UUID                  `json:"crew_member_id" binding:"required"`
	AmountCents  int64                      `json:"amount_cents" binding:"required,min=1"`
	Category     models.TransactionCategory `json:"category" binding:"required"`
	Reference    string                     `json:"reference"`
	Description  string                     `json:"description"`
}

type DebitWalletRequest struct {
	CrewMemberID uuid.UUID                  `json:"crew_member_id" binding:"required"`
	AmountCents  int64                      `json:"amount_cents" binding:"required,min=1"`
	Category     models.TransactionCategory `json:"category" binding:"required"`
	Reference    string                     `json:"reference"`
	Description  string                     `json:"description"`
}

type PayoutWalletRequest struct {
	CrewMemberID   uuid.UUID `json:"crew_member_id" binding:"required"`
	AmountCents    int64     `json:"amount_cents" binding:"required,min=1"`
	Channel        string    `json:"channel" binding:"required,oneof=MOMO_B2C BANK MOMO_B2B"`
	RecipientName  string    `json:"recipient_name" binding:"required"`
	RecipientPhone string    `json:"recipient_phone"`
	BankAccount    string    `json:"bank_account"`
	BankCode       string    `json:"bank_code"`
	PaybillNumber  string    `json:"paybill_number"`
	PaybillRef     string    `json:"paybill_ref"`
}
