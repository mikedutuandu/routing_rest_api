package api

type PayloadSuccess struct {
	Meta interface{} `json:"meta,omitempty"`
	Data interface{} `json:"data,omitempty"`
}
type PayloadError struct {
	Errors interface{} `json:"message"`
}


//for pickup model
//SCHEDULED
//IN_PROGRESS
//FAILED
//COMPLETED => 2 success, 1 failed

/*
====================ORDER STATUS LIST=======================
*/
//Order without driver: ORDER_INFO_RECEIVED
//Pickup List:  PICK_UP_SCHEDULED, PICK_UP_IN_PROGRESS, ORDER_PICKED_UP, PICKUP_RESCHEDULED, PICKUP_FAILED
//At warehouse: ORDER_RECEIVED_AT_LOCAL_SORTING_CENTER
//Delivery List: DELIVERY_SCHEDULED, DELIVERY_IN_PROGRESS, SUCCESS, DELIVERY_RESCHEDULED, DELIVERY_FAILED

const (
	layoutISO     = "2006-01-02"
	layoutUS      = "January 2, 2006"
	Host          = "http://68.183.230.245:8000"
	maxUploadSize = 10 // 2 MB
)
