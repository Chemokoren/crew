package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Route represents a transport route with base fare.
type Route struct {
	ID                  uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name                string         `json:"name" gorm:"not null" validate:"required"`
	StartPoint          string         `json:"start_point" gorm:"not null"`
	EndPoint            string         `json:"end_point" gorm:"not null"`
	EstimatedDistanceKm float64        `json:"estimated_distance_km"`
	BaseFareCents       int64          `json:"base_fare_cents" gorm:"type:bigint;default:0"`
	IsActive            bool           `json:"is_active" gorm:"default:true"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Route) TableName() string { return "routes" }
