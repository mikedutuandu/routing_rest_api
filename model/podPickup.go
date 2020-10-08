package model

import "time"

type PodPickup struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Image     string `json:"image"`
	PickupId      int `json:"pickup_id"`
}
