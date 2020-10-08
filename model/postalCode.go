package model

import (
	"time"
)

type PostalCode struct {
	ID          uint      `gorm:"primary_key"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CountryCode string    `json:"country_code" validate:"required"`
	PostalCode  string    `json:"postal_code" validate:"required"`
}
