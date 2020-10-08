package model

import "time"

type FirstMilePodOrder struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Image     string `json:"image"`
	OrderId      int `json:"order_id"`
}
