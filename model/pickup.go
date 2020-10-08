package model

import (
	"time"
)

type Pickup struct {
	ID         uint      `gorm:"primary_key"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	PickupDate string    `json:"pickup_date"`

	//OrderID                       string  `json:"internal_tn"`
	//AdditionalData                string
	//AgentApplicationIDID          int
	//AgentIDID                     int
	//CodAmtToCollect               string
	//ConsigneeAddress              string `json:"consignee_address"`
	//ConsigneeCity                 string `json:"consignee_city"`
	//ConsigneeCountry              string `json:"consignee_country"`
	//ConsigneeEmail                string `json:"consignee_email"`
	//ConsigneeName                 string `json:"consignee_name"`
	//ConsigneeNumber               string `json:"consignee_number"`
	//ConsigneePostal               string `json:"consignee_postal"`
	//ConsigneeProvince             string `json:"consignee_province"`
	//ConsigneeState                string `json:"consignee_state"`
	//CreatedOn                     time.Time
	//DeliveryNote                  string `json:"delivery_note"`
	//HawbNo                        string
	//Incoterm                      string
	//InvoiceCreated                bool
	//IsProcessing                  bool
	//ModelLogLinkID                int
	//OrderHeight                   float32
	//OrderLabelURL                 string
	//OrderLength                   float32
	//OrderWeight                   float32
	//OrderWidth                    float32
	//PaymentType                   string
	PickupAddress       string `json:"pickup_address"`
	PickupCity          string `json:"pickup_city"`
	PickupContactName   string `json:"pickup_contact_name"`
	PickupContactNumber string `json:"pickup_contact_number"`
	PickupCountry       string `json:"pickup_country"`
	PickupPostal        string `json:"pickup_postal"`
	PickupProvince      string `json:"pickup_province"`
	PickupState         string `json:"pickup_state"`
	//PrintDefaultLabel             bool
	//PrintURL                      string
	//PrivateTrackerStatusCode      string
	//PrivateTrackerUpdatedOn       time.Time
	//ServiceIDID                   int
	//ShipperOrderID                string
	//ShipperSubAccountID           string
	//ShipperSubOrderID             string
	//StatusCodeStoreID             string
	//SubmitExternalServiceDatetime time.Time
	//SubmitFirstmileDatetime       time.Time
	//SubmitWarehouseDatetime       string
	//TrackerMainText               string
	//TrackerStatusCode             string
	//TrackerUpdatedOn              time.Time
	//TrackingNo                    string  `json:"external_tn"`
	//UpdatedOn                     time.Time
	//UploadBatchNo                 string
	//WarehouseAddress              string
	//FirstMileDriverId int `json:"first_mile_driver_id,omitempty"`
	//LastMileDriverId int `json:"last_mile_driver_id,omitempty"`
	//FirstMileDriverName string `json:"first_mile_driver_name,omitempty"`
	//LastMileDriverName string `json:"last_mile_driver_name,omitempty"`
	Status            string    `json:"status"`
	DriverNote        string    `json:"driver_note"`
	StatusNote        string    `json:"status_note"`
	RescheduleDate    time.Time `json:"reschedule_date"`
	CustomerSignature string    `json:"customer_signature"`
	DriverId          int       `json:"driver_id"`

	Orders []*Order `gorm:"foreignkey:PickupId"`

	Pods []*PodPickup `gorm:"foreignkey:PickupId",json:"pods"`

	PickupLat float64 `json:"pickup_lat"`
	PickupLng float64 `json:"pickup_lng"`
	AddLatLng string  `json:"add_lat_lng"`

	PickupStart   string `json:"pickup_start"`
	PickupEnd     string `json:"pickup_end"`
	SolutionOrder int32  `json:"solution_order"`
	ArrivalTime   string `json:"arrival_time"`
	FinishTime    string `json:"finish_time"`
	Duration      int32  `json:"duration"`

	ImportId          int       `json:"import_id"`
	TotalNumberOrder          int       `json:"total_number_order"`
	TotalNumberOrderPickup          int       `json:"total_number_order_pickup"`
}

func (p *Pickup) AfterFind() (err error) {
	p.PickupDate = p.PickupDate[0:10]
	return
}
