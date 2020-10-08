package model

import (
	"time"
)

type Import struct {
	ID           uint      `gorm:"primary_key"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	FileName     string    `json:"file_name"`
	Status       string    `json:"status"`
	DeliveryDate string    `json:"delivery_date"`
	PickupDate   string    `json:"pickup_date"`
	Override     string    `json:"override"`
	ErrorData    string    `json:"error_data"`
	GeocodeTime    int64    `json:"geocode_time"`
	GeofenceTime    int64    `json:"geofence_time"`
	Username    string    `json:"username"`
	GeocodePickupStatus    string    `json:"geocode_pickup_status"`
	GeocodeDeliveryStatus    string    `json:"geocode_delivery_status"`
	GeofenceStatus    string    `json:"geofence_status"`
	StartGeocode int64 `json:"start_geocode"`
	EndGeocode int64 `json:"end_geocode"`
	StartGeofence int64 `json:"start_geofence"`
	EndGeofence int64 `json:"end_geofence"`
	NumberOrder int `json:"number_order"`
	AssignmentType string `json:"assignment_type"` //2digits hay geofencing
	UploadType string `json:"upload_type"` //pickup/delivery

}

func (i *Import) AfterFind() (err error) {
	i.PickupDate = i.PickupDate[0:10]
	i.DeliveryDate = i.DeliveryDate[0:10]
	return
}
