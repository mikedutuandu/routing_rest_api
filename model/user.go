package model

import (
	"github.com/dgrijalva/jwt-go"
	"time"
)

type User struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Email         string `json:"email,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	DriverLicense string `json:"driver_license,omitempty" form:"driver_license" validate:"required" `
	DriverName    string `json:"driver_name,omitempty" form:"driver_name" validate:"required" `
	LicensePlate  string `json:"license_plate,omitempty" form:"license_plate" validate:"required"`
	Phone         string `json:"phone,omitempty" form:"phone" validate:"required"`
	PostalCode    string `json:"postal_codes,omitempty" form:"postal_code" validate:"required"`
	Avatar        string `json:"avatar" form:"avatar"`
	IsAdmin       bool   `json:"is_admin,omitempty"`

	ShiftPickupStart   string `json:"shift_pickup_start"`
	ShiftPickupEnd     string `json:"shift_pickup_end"`
	ShiftDeliveryStart string `json:"shift_delivery_start"`
	ShiftDeliveryEnd   string `json:"shift_delivery_end"`
	DriverType         string `json:"driver_type"`
	Weight             string `json:"weight"`
	Volume             string `json:"volume"`
	Country            string `json:"country"`
	Timezone           string `json:"timezone"`
}


// jwtCustomClaims are custom claims extending default ones.
type JwtCustomClaims struct {
	User User
	jwt.StandardClaims
}
