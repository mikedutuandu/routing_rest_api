package model

import (
	"time"
)

type OrderLog struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy uint      `json:"created_by"`
	OrderId   string    `json:"order_id"`
	content   string    `json:"content"`
}
