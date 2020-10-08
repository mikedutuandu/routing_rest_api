package model

import (
	"time"
)

type OptimizeJob struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	JobType   string    `json:"job_type"`
	Status    string    `json:"status"`
	DriverId  int       `json:"driver_id"`
	JobId     string    `json:"job_id"`
}
